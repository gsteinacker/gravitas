//go:build darwin && cgo

package bh

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Metal -framework Foundation
#import <Metal/Metal.h>
#include <stdint.h>
#include <string.h>
#include <stdlib.h>

static id<MTLDevice> dev;
static id<MTLCommandQueue> queue;
static id<MTLComputePipelineState> pso;

// gpu_init compiles the kernel; returns 0 on success, otherwise writes a message to err.
static int gpu_init(const char *src, char *err, int errlen) {
	dev = MTLCreateSystemDefaultDevice();
	if (!dev) { strncpy(err, "no Metal device", errlen); return 1; }
	NSError *e = nil;
	id<MTLLibrary> lib = [dev newLibraryWithSource:[NSString stringWithUTF8String:src] options:nil error:&e];
	if (!lib) { strncpy(err, [[e localizedDescription] UTF8String], errlen); return 2; }
	pso = [dev newComputePipelineStateWithFunction:[lib newFunctionWithName:@"walk"] error:&e];
	if (!pso) { strncpy(err, [[e localizedDescription] UTF8String], errlen); return 3; }
	queue = [dev newCommandQueue];
	return 0;
}

// gpu_buffer allocates a CPU/GPU shared buffer; the returned handle owns one retain.
static void *gpu_buffer(size_t bytes) {
	id<MTLBuffer> b = [dev newBufferWithLength:bytes options:MTLResourceStorageModeShared];
	return (__bridge_retained void *)b;
}

static void *gpu_contents(void *buf) { return [(__bridge id<MTLBuffer>)buf contents]; }

static void gpu_release(void *buf) { if (buf) { id<MTLBuffer> b = (__bridge_transfer id<MTLBuffer>)buf; b = nil; } }

// gpu_walk runs the kernel for n stars and waits; returns the GPU time in seconds.
static double gpu_walk(void *nodes, void *stars, void *acc, uint32_t n, float eps2) {
	id<MTLCommandBuffer> cb = [queue commandBuffer];
	id<MTLComputeCommandEncoder> enc = [cb computeCommandEncoder];
	[enc setComputePipelineState:pso];
	[enc setBuffer:(__bridge id<MTLBuffer>)nodes offset:0 atIndex:0];
	[enc setBuffer:(__bridge id<MTLBuffer>)stars offset:0 atIndex:1];
	[enc setBuffer:(__bridge id<MTLBuffer>)acc offset:0 atIndex:2];
	[enc setBytes:&n length:sizeof(n) atIndex:3];
	[enc setBytes:&eps2 length:sizeof(eps2) atIndex:4];
	NSUInteger tg = MIN((NSUInteger)256, pso.maxTotalThreadsPerThreadgroup);
	[enc dispatchThreads:MTLSizeMake(n, 1, 1) threadsPerThreadgroup:MTLSizeMake(tg, 1, 1)];
	[enc endEncoding];
	[cb commit];
	[cb waitUntilCompleted];
	return cb.GPUEndTime - cb.GPUStartTime;
}
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"

	"galaxy/internal/sim"
)

// walkKernel is the GPU version of Solver.walk (skip-pointer traversal with quadrupoles), one
// thread per star in Morton order, so neighboring threads take similar paths through the tree.
const walkKernel = `
#include <metal_stdlib>
using namespace metal;
struct GNode { float4 com; float4 q1; float4 q2; int4 links; };

kernel void walk(device const GNode *nodes [[buffer(0)]], device const float4 *stars [[buffer(1)]],
                 device float4 *acc [[buffer(2)]], constant uint &n [[buffer(3)]],
                 constant float &eps2 [[buffer(4)]], uint k [[thread_position_in_grid]]) {
	if (k >= n) return;
	float3 p = stars[k].xyz;
	float3 a = 0;
	int i = 0;
	while (i >= 0) {
		GNode nd = nodes[i];
		float3 d = nd.com.xyz - p;
		float d2 = dot(d, d);
		if (d2 >= nd.q2.z) {
			float inv = 1.0f / (d2 + eps2);
			float inv3 = inv * sqrt(inv);
			float3 qd = float3(nd.q1.x * d.x + nd.q1.w * d.y + nd.q2.x * d.z,
			                   nd.q1.w * d.x + nd.q1.y * d.y + nd.q2.y * d.z,
			                   nd.q2.x * d.x + nd.q2.y * d.y + nd.q1.z * d.z);
			float inv5 = inv3 * inv;
			a += d * (nd.com.w * inv3 + 2.5f * dot(d, qd) * inv5 * inv) - qd * inv5;
			i = nd.links.y;
			continue;
		}
		if (nd.links.x >= 0) {
			i = nd.links.x;
			continue;
		}
		for (int j = nd.links.z; j < nd.links.w; j++) {
			if (j == int(k)) continue;
			float4 s = stars[j];
			float3 dd = s.xyz - p;
			float ir = rsqrt(dot(dd, dd) + eps2);
			a += dd * (s.w * ir * ir * ir);
		}
		i = nd.links.y;
	}
	acc[k] = float4(a, 0);
}
`

// gNode mirrors the kernel's GNode (64 bytes).
type gNode struct {
	com, q1, q2 [4]float32 // (x, y, z, m), (qxx, qyy, qzz, qxy), (qxz, qyz, crit², 0)
	links       [4]int32   // first, next, lo, hi
}

var (
	gpuOnce sync.Once
	gpuErr  error
)

// GPUAvailable reports whether the Metal tree walk can be used, compiling it on first use.
func GPUAvailable() error {
	gpuOnce.Do(func() {
		src := C.CString(walkKernel)
		defer C.free(unsafe.Pointer(src))
		var msg [512]C.char
		if C.gpu_init(src, &msg[0], C.int(len(msg))) != 0 {
			gpuErr = errors.New("metal: " + C.GoString(&msg[0]))
		}
	})
	return gpuErr
}

// gpuBuffers are the shared buffers of one Solver.
type gpuBuffers struct {
	nodes, stars, acc unsafe.Pointer
	nodeCap, starCap  int
	lastSeconds       float64
}

func (g *gpuBuffers) ensure(nodes, stars int) {
	if nodes > g.nodeCap {
		C.gpu_release(g.nodes)
		g.nodeCap = nodes + nodes/4
		g.nodes = C.gpu_buffer(C.size_t(g.nodeCap * int(unsafe.Sizeof(gNode{}))))
	}
	if stars > g.starCap {
		C.gpu_release(g.stars)
		C.gpu_release(g.acc)
		g.starCap = stars + stars/4
		g.stars = C.gpu_buffer(C.size_t(g.starCap * 16))
		g.acc = C.gpu_buffer(C.size_t(g.starCap * 16))
	}
}

// walkGPU computes the star–star accelerations of all stars in sorted order into b.ax/ay/az.
func (b *Solver) walkGPU(eps2 float64) {
	n := len(b.order)
	g := &b.gpu
	g.ensure(len(b.nodes), n)
	nodes := unsafe.Slice((*gNode)(C.gpu_contents(g.nodes)), len(b.nodes))
	stars := unsafe.Slice((*[4]float32)(C.gpu_contents(g.stars)), n)
	acc := unsafe.Slice((*[4]float32)(C.gpu_contents(g.acc)), n)
	sim.ParallelFor(len(b.nodes), 4096, b.Workers, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			nd := &b.nodes[i]
			nodes[i] = gNode{
				com:   [4]float32{float32(nd.x), float32(nd.y), float32(nd.z), float32(nd.m)},
				q1:    [4]float32{float32(nd.qxx), float32(nd.qyy), float32(nd.qzz), float32(nd.qxy)},
				q2:    [4]float32{float32(nd.qxz), float32(nd.qyz), float32(nd.crit2), 0},
				links: [4]int32{nd.first, nd.next, nd.lo, nd.hi},
			}
		}
	})
	sim.ParallelFor(n, 8192, b.Workers, func(lo, hi int) {
		for k := lo; k < hi; k++ {
			stars[k] = [4]float32{float32(b.sx[k]), float32(b.sy[k]), float32(b.sz[k]), float32(b.sm[k])}
		}
	})
	g.lastSeconds = float64(C.gpu_walk(g.nodes, g.stars, g.acc, C.uint32_t(n), C.float(eps2)))
	b.ax, b.ay, b.az = resize(b.ax, n), resize(b.ay, n), resize(b.az, n)
	sim.ParallelFor(n, 8192, b.Workers, func(lo, hi int) {
		for k := lo; k < hi; k++ {
			b.ax[k], b.ay[k], b.az[k] = float64(acc[k][0]), float64(acc[k][1]), float64(acc[k][2])
		}
	})
}
