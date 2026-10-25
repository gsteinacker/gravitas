// Metal shaders of the SwiftUI app, compiled to Galaxy.metallib by `make app` (xcrun metal). They
// port internal/render: projection (view.Camera), star colors, Doppler shift, Hα emission, trails,
// gravitational lens, bloom and the black holes. Rendering is in float16 (HDR) and tone mapped at
// the end instead of clipping in 8 bits.
#include <metal_stdlib>
using namespace metal;

struct Uniforms {
    float4 view;   // width, height (render pixels), focal length, pixel scale
    float4 target; // camera target xyz, yaw
    float4 cam;    // pitch, distance, mean star mass, speed of light (0 = Newtonian)
    float4 eye;    // camera position xyz, exposure
    float4 misc;   // effect bits, young age, emission normalization, dust opacity κ
    float4 haze;   // dark matter alpha, gas map alpha, number of world holes, unused
};

struct Body { float4 pos; float4 vel; float4 gas; }; // xyz mass | v kind | age ion h rho

struct Sprite {
    float4 position [[position]];
    float2 uv;
    float4 color; // premultiplied, additive
};

constant float2 corners[4] = { float2(-1, -1), float2(1, -1), float2(-1, 1), float2(1, 1) };

// project returns screen x, y and depth; depth < 0.05 means behind the camera.
float3 project(float3 p, constant Uniforms &u) {
    p -= u.target.xyz;
    float sy = sin(u.target.w), cy = cos(u.target.w);
    float sp = sin(u.cam.x), cp = cos(u.cam.x);
    float x1 = cy * p.x - sy * p.y;
    float y1 = sy * p.x + cy * p.y;
    float fwd = cp * y1 - sp * p.z;
    float up = sp * y1 + cp * p.z;
    float depth = u.cam.y + fwd;
    float f = u.view.z / max(depth, 1e-4);
    return float3(u.view.x / 2 + f * x1, u.view.y / 2 - f * up, depth);
}

float3 starColor(float rel) {
    float t = clamp(0.5 + 0.25 * log2(rel), 0.0, 1.0);
    return float3(1 - 0.35 * t, 0.75 + 0.15 * t, 0.45 + 0.55 * t);
}

// gravShift is √(1 − r_s/r), clamped so stars right at a horizon stay faintly visible.
float gravShift(float r, float rs) {
    return rs <= 0 ? 1.0 : sqrt(max(0.02, 1 - rs / r));
}

// frequencyShift is ν_observed/ν_emitted for velocity v seen from direction n at speed of light c,
// times the gravitational redshift grav of the holes (render/shift.go).
float frequencyShift(float3 v, float3 n, float c, float grav) {
    if (c <= 0) return grav;
    float3 b = v / c;
    float b2 = min(dot(b, b), 0.9999);
    float gamma = 1 / sqrt(1 - b2);
    return grav / (gamma * (1 - dot(b, n)));
}

// shiftColor tints by the frequency factor and returns the beaming factor in w.
float4 shiftColor(float3 c, float f) {
    float t = clamp(2 * log2(f), -1.0, 1.0);
    float3 tint = t >= 0 ? float3(0.55, 0.7, 1.0) : float3(1.0, 0.3, 0.12);
    return float4(mix(c, tint, fabs(t)), clamp(f * f * f, 0.25, 2.5));
}

Sprite makeSprite(float3 s, float radius, float4 color, uint vid, constant Uniforms &u) {
    Sprite o;
    float2 c = corners[vid];
    float2 p = s.xy + c * radius;
    o.position = float4(p.x / u.view.x * 2 - 1, 1 - p.y / u.view.y * 2, 0, 1);
    o.uv = c;
    o.color = color;
    return o;
}

Sprite culled() {
    Sprite o;
    o.position = float4(2, 2, 2, 1); // outside the clip volume
    o.uv = 0;
    o.color = 0;
    return o;
}

vertex Sprite spriteVertex(uint vid [[vertex_id]], uint iid [[instance_id]],
                           const device Body *bodies [[buffer(0)]], constant Uniforms &u [[buffer(1)]],
                           const device float4 *attn [[buffer(2)]], const device float4 *holes [[buffer(3)]]) {
    Body b = bodies[iid];
    bool dusty = (int(u.misc.x) & 64) != 0;
    int kind = int(b.vel.w + 0.5);
    int fx = int(u.misc.x);
    float ps = u.view.w;
    float3 s = project(b.pos.xyz, u);
    if (s.z < 0.05) return culled();
    float focal = u.view.z;
    if (kind == 0) { // star
        float rel = b.pos.w / max(u.cam.z, 1e-12);
        float size = clamp(0.05 * pow(rel, 1.0 / 3.0) * focal / s.z, 3.5 * ps, 28 * ps);
        float3 col = starColor(rel);
        float alpha = min(1.0, 0.55 + 0.2 * rel);
        float age = b.gas.x;
        if (age >= 0 && age < u.misc.y) { // young stars from gas: hot, blue, bright
            float f = 1 - age / u.misc.y;
            col = mix(col, float3(0.6, 0.78, 1.0), f);
            alpha *= 1 + 1.5 * f;
            size *= 1 + 0.5 * f;
        }
        if ((fx & 2) != 0 && u.cam.w > 0) {
            float3 n = normalize(u.eye.xyz - b.pos.xyz);
            float grav = 1;
            for (int h = 0; h < int(u.haze.z); h++) {
                grav *= gravShift(length(b.pos.xyz - holes[h].xyz), holes[h].w);
            }
            float4 sc = shiftColor(col, frequencyShift(b.vel.xyz, n, u.cam.w, grav));
            col = sc.xyz;
            alpha *= sc.w;
        }
        if (dusty) { // dust in front (computed in the Go core): reddened and dimmed (render/stars.go)
            float4 t = attn[iid];
            col *= t.rgb;
            alpha *= t.g;
        }
        return makeSprite(s, size / 2, float4(col * alpha * u.eye.w, 0), vid, u);
    }
    if ((kind == 2 && (fx & 16) != 0) || (kind == 3 && (fx & 32) != 0)) {
        // Haze: dark matter violet, the gas map in false colors (neutral blue, partly ionized
        // green, ionized red); opacity from the projected density (Go core).
        float size = clamp(0.12 * focal / s.z, 6 * ps, 48 * ps);
        float3 tint = float3(0.55, 0.35, 1.0);
        float alpha = u.haze.x;
        if (kind == 3) {
            float ion = b.gas.y;
            tint = float3(0.25 + 0.75 * ion, 0.45 + 0.5 * (1 - 2 * fabs(ion - 0.5)), 1 - 0.8 * ion);
            alpha = u.haze.y;
        }
        return makeSprite(s, size / 2, float4(tint * alpha, 0), vid, u);
    }
    if (kind == 3 && b.gas.y > 0.2) { // photoionized gas: Hα emission
        float size = clamp(3 * b.gas.z * focal / s.z, 3 * ps, 120 * ps);
        float ion = b.gas.y;
        float brightness = ion * ion * b.pos.w * b.gas.w / (size * size);
        if (dusty) brightness *= attn[iid].r; // Hα is red
        float a = 1 - exp(-u.misc.z * brightness);
        return makeSprite(s, size / 2, float4(float3(1.0, 0.16, 0.22) * a, 0), vid, u);
    }
    return culled();
}

// dustVertex draws the neutral gas as column density into the dust image: alpha accumulates
// "over", the background is attenuated by exp(−alpha · DustStrength · k) (render/stars.go DrawDust).
vertex Sprite dustVertex(uint vid [[vertex_id]], uint iid [[instance_id]],
                         const device Body *bodies [[buffer(0)]], constant Uniforms &u [[buffer(1)]]) {
    Body b = bodies[iid];
    if (int(b.vel.w + 0.5) != 3) return culled();
    float3 s = project(b.pos.xyz, u);
    if (s.z < 0.05) return culled();
    float focal = u.view.z, ps = u.view.w;
    float size = clamp(2 * b.gas.z * focal / s.z, 2 * ps, 80 * ps);
    float k = focal / s.z;
    float tauArea = u.misc.w * b.pos.w * (1 - b.gas.y) * k * k;
    float a = min(1.0, tauArea / (0.10972 * size * size) / 2.5);
    return makeSprite(s, size / 2, float4(a), vid, u);
}

fragment float4 spriteFragment(Sprite in [[stage_in]]) {
    float d2 = dot(in.uv, in.uv);
    if (d2 >= 1) discard_fragment();
    float a = min(1.0, exp(-d2 * 12) + 0.25 * exp(-d2 * 3));
    return in.color * a;
}

// ---- full screen passes ----

struct Screen { float4 position [[position]]; float2 uv; };

vertex Screen screenVertex(uint vid [[vertex_id]]) {
    Screen o;
    float2 c = corners[vid];
    o.position = float4(c, 0, 1);
    o.uv = float2(c.x * 0.5 + 0.5, 0.5 - c.y * 0.5);
    return o;
}

constexpr sampler linearSampler(filter::linear, address::clamp_to_edge);

// trailFragment keeps a fading history that is never brighter than the stars (render/post.go).
fragment float4 trailFragment(Screen in [[stage_in]], texture2d<float> stars [[texture(0)]],
                              texture2d<float> previous [[texture(1)]]) {
    float4 p = previous.sample(linearSampler, in.uv);
    float4 s = stars.sample(linearSampler, in.uv);
    return max(max(p * 0.93 - 0.002, 0.0), s);
}

// composeFragment puts the stars over the static sky, which the dust in front attenuates per
// color channel (interstellar reddening; the stars were attenuated individually).
fragment float4 composeFragment(Screen in [[stage_in]], texture2d<float> stars [[texture(0)]],
                                texture2d<float> sky [[texture(1)]], texture2d<float> dust [[texture(2)]]) {
    float tau = dust.sample(linearSampler, in.uv).a * 2.5;
    float4 c = sky.sample(linearSampler, in.uv);
    return float4(c.rgb * exp(-tau * float3(0.55, 0.75, 1.0)), c.a) + stars.sample(linearSampler, in.uv);
}

struct Lens { float4 holes[16]; float4 info; }; // x, y, θ_E² | count, width, height

// lensFragment displaces every pixel by the point-lens equation, summed over the holes.
fragment float4 lensFragment(Screen in [[stage_in]], texture2d<float> src [[texture(0)]], constant Lens &l [[buffer(0)]]) {
    float2 size = l.info.yz;
    float2 p = in.uv * size;
    float2 q = p;
    for (int i = 0; i < int(l.info.x); i++) {
        float2 d = p - l.holes[i].xy;
        q -= d * (l.holes[i].z / max(dot(d, d), 1.0));
    }
    return src.sample(linearSampler, q / size);
}

fragment float4 copyFragment(Screen in [[stage_in]], texture2d<float> src [[texture(0)]]) {
    return src.sample(linearSampler, in.uv);
}

// blurFragment is one direction of the separable 9-tap Gaussian of the bloom.
fragment float4 blurFragment(Screen in [[stage_in]], texture2d<float> src [[texture(0)]], constant float2 &dir [[buffer(0)]]) {
    const float w[5] = { 0.227027, 0.1945946, 0.1216216, 0.054054, 0.016216 };
    float2 texel = dir / float2(src.get_width(), src.get_height());
    float4 c = src.sample(linearSampler, in.uv) * w[0];
    for (int i = 1; i < 5; i++) {
        float2 o = texel * float(i) * 1.5;
        c += (src.sample(linearSampler, in.uv + o) + src.sample(linearSampler, in.uv - o)) * w[i];
    }
    return c;
}

// finalFragment adds the bloom and maps the HDR image softly to the display.
fragment float4 finalFragment(Screen in [[stage_in]], texture2d<float> scene [[texture(0)]],
                              texture2d<float> bloom [[texture(1)]], constant float &bloomStrength [[buffer(0)]]) {
    float3 c = scene.sample(linearSampler, in.uv).rgb;
    if (bloomStrength > 0) { // without glow the bloom image is not drawn: never read it (0 · NaN = NaN)
        c += bloomStrength * bloom.sample(linearSampler, in.uv).rgb;
    }
    c = 1 - exp(-1.6 * c);
    return float4(c, 1);
}

// ---- black holes ----

struct HoleOut { float4 position [[position]]; float2 uv; float r; float rings; };

// Hole instances: x, y (render pixels), drawn radius R, ISCO ring flag.
vertex HoleOut holeVertex(uint vid [[vertex_id]], uint iid [[instance_id]], const device float4 *holes [[buffer(0)]],
                          constant Uniforms &u [[buffer(1)]]) {
    float4 h = holes[iid];
    float ext = 3.2 * h.z;
    float2 c = corners[vid];
    float2 p = h.xy + c * ext;
    HoleOut o;
    o.position = float4(p.x / u.view.x * 2 - 1, 1 - p.y / u.view.y * 2, 0, 1);
    o.uv = c * 3.2; // in units of R
    o.r = h.z;
    o.rings = h.w;
    return o;
}

// holeFragment: glow out to the shadow edge (2.6 R), ISCO ring at 3 R, photon ring at 1.5 R and
// the black horizon (premultiplied; glow has alpha 0, so it adds light).
fragment float4 holeFragment(HoleOut in [[stage_in]]) {
    float d = length(in.uv);
    float px = 1 / max(in.r, 1e-3); // one pixel in units of R
    if (d < 1) return float4(0, 0, 0, 1);
    float4 c = 0;
    if (d < 2.6) c += float4(18, 7, 2, 0) / 255;
    float ring = max(1.5 * px, 0.25);
    float a = 1 - smoothstep(ring / 2, ring / 2 + px, fabs(d - 1.5));
    c = c * (1 - a * 0.86) + float4(1.0, 0.59, 0.2, 1) * a * 0.86;
    if (in.rings > 0) {
        float b = 1 - smoothstep(0.5 * px, 1.5 * px, fabs(d - 3));
        c = c * (1 - b * 0.47) + float4(0.35, 0.24, 0.16, 1) * b * 0.47;
    }
    return c;
}
