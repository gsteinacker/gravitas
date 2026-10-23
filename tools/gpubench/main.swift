// GPU direct-sum N-body benchmark (Metal). Run: swiftc -O tools/gpubench/main.swift -o bin/gpubench && bin/gpubench
// Reference for docs/analysis/01-tech-stack.md, section 6.
import Metal
import Foundation

let src = """
#include <metal_stdlib>
using namespace metal;
constant uint TILE = 256;
kernel void acc(device const float4* pos [[buffer(0)]], device float4* out [[buffer(1)]],
                constant uint& n [[buffer(2)]], constant float& eps2 [[buffer(3)]],
                uint i [[thread_position_in_grid]], uint lid [[thread_position_in_threadgroup]]) {
    threadgroup float4 tile[TILE];
    float4 p = pos[min(i, n-1)];
    float3 a = 0;
    for (uint base = 0; base < n; base += TILE) {
        tile[lid] = pos[min(base + lid, n-1)] * float4(1,1,1, (base+lid < n) ? 1 : 0);
        threadgroup_barrier(mem_flags::mem_threadgroup);
        for (uint j = 0; j < TILE; j++) {
            float4 q = tile[j];
            float3 d = q.xyz - p.xyz;
            float r2 = dot(d, d) + eps2;
            float inv = rsqrt(r2);
            a += d * (q.w * inv * inv * inv);
        }
        threadgroup_barrier(mem_flags::mem_threadgroup);
    }
    if (i < n) out[i] = float4(a, 0);
}
"""
let dev = MTLCreateSystemDefaultDevice()!
let lib = try! dev.makeLibrary(source: src, options: nil)
let pso = try! dev.makeComputePipelineState(function: lib.makeFunction(name: "acc")!)
let q = dev.makeCommandQueue()!
for n in [16_384, 100_000, 262_144, 1_000_000] {
    var p = [SIMD4<Float>](repeating: .zero, count: n)
    for k in 0..<n { p[k] = SIMD4(Float.random(in: -1...1), Float.random(in: -1...1), Float.random(in: -1...1), 1.0/Float(n)) }
    let pb = dev.makeBuffer(bytes: p, length: n*16)!, ob = dev.makeBuffer(length: n*16)!
    var nn = UInt32(n); var e: Float = 1e-4
    func run() -> Double {
        let cb = q.makeCommandBuffer()!, enc = cb.makeComputeCommandEncoder()!
        enc.setComputePipelineState(pso); enc.setBuffer(pb, offset: 0, index: 0); enc.setBuffer(ob, offset: 0, index: 1)
        enc.setBytes(&nn, length: 4, index: 2); enc.setBytes(&e, length: 4, index: 3)
        enc.dispatchThreadgroups(MTLSize(width: (n+255)/256, height: 1, depth: 1), threadsPerThreadgroup: MTLSize(width: 256, height: 1, depth: 1))
        enc.endEncoding(); cb.commit(); cb.waitUntilCompleted()
        return cb.gpuEndTime - cb.gpuStartTime
    }
    _ = run()
    let reps = n >= 1_000_000 ? 2 : 5
    var best = 1e9; for _ in 0..<reps { best = min(best, run()) }
    print(String(format: "N=%8d  %8.2f ms/step  %6.1f Gpairs/s", n, best*1000, Double(n)*Double(n)/best/1e9))
}
