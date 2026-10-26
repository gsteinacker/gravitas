// Metal shaders of the SwiftUI app, compiled to Galaxy.metallib by `make app` (xcrun metal). They
// draw what the Go core prepares: projection (view.Camera), star colors, Doppler shift, Hα emission, trails,
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
    float4 haze;   // dark matter alpha, gas map alpha, number of world holes, cosmic web point brightness
    float4 pop;    // population radius (0: particles are stars), star formation density, its light, unused
};

struct Body { float4 pos; float4 vel; float4 gas; }; // xyz mass | v kind | age ion h rho

struct Sprite {
    float4 position [[position]];
    float2 uv;
    float4 color;  // premultiplied, additive
    float gas;     // 1: smooth SPH-like kernel (gas), 0: star glow
};

// DISK_OUTER is the outer radius of accretion disks in r_g (T-065).
constant float DISK_OUTER = 30;

// KERNEL_MEAN is the mean of the gas kernel (1 − d²)³ over its quad: π/16 (engine.kernelMean).
constant float KERNEL_MEAN = 0.19635;

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
// times the gravitational redshift grav of the holes.
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
    o.gas = 0;
    return o;
}

// gasSprite is a gas body as a smooth kernel of diameter size. Larger than gasCap (engine.gasCap)
// it is drawn at that size: callers compute the surface brightness from the true size, so a huge
// cloud close to the camera stays as bright per pixel and only its far edge is cut.
Sprite gasSprite(float3 s, float size, float4 color, uint vid, constant Uniforms &u) {
    Sprite o = makeSprite(s, min(size, 1.5 * max(u.view.x, u.view.y)) / 2, color, vid, u);
    o.gas = 1;
    return o;
}

Sprite culled() {
    Sprite o;
    o.position = float4(2, 2, 2, 1); // outside the clip volume
    o.uv = 0;
    o.color = 0;
    o.gas = 0;
    return o;
}

// Every body draws POPULATION sprites (Renderer.populationSprites). In galaxy scenes a star particle
// is a whole stellar population (engine.Population): a core, a soft glow of its unresolved stars
// as large as the population (from afar the glows merge into the diffuse light of the disk) and
// MEMBERS single stars scattered within it, which resolve when zoomed in. The sprites share the
// light of the particle, so the exposure of the Go core is unchanged.
constant uint POPULATION = 8;
constant uint MEMBERS = POPULATION - 2;

uint hash(uint x) {
    x ^= x >> 16; x *= 0x7feb352du;
    x ^= x >> 15; x *= 0x846ca68bu;
    x ^= x >> 16;
    return x;
}

float unit(uint h) { return float(h & 0xffffffu) / 16777216.0; }

// shade applies Doppler shift, beaming, gravitational redshift and the dust in front.
float4 shade(Body b, float3 col, float alpha, int fx, uint bi, const device float4 *attn,
             const device float4 *holes, constant Uniforms &u) {
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
    if ((fx & 64) != 0) { // dust in front (computed in the Go core): reddened and dimmed
        float4 t = attn[bi];
        col *= t.rgb;
        alpha *= t.g;
    }
    return float4(col, alpha);
}

// population returns sprite sub of a population at screen position s whose whole light is that of
// one sprite of the given size with color and alpha c; core is the share of the central sprite,
// tint colors the glow.
Sprite population(Body b, float3 s, float4 c, float3 tint, float size, float core, uint sub, uint vid,
                  constant Uniforms &u) {
    float light = c.w * u.eye.w * size * size;
    float r = u.pop.x, ps = u.view.w;
    if (sub == 0) {
        if (r > 0) core *= 1 - smoothstep(4 * ps, 12 * ps, size);
        if (core <= 0) return culled();
        return makeSprite(s, size / 2, float4(c.rgb * core * c.w * u.eye.w, 0), vid, u);
    }
    if (r <= 0) return culled();
    float share = (1 - core) / 2; // glow and members
    // A population larger than a few pixels is resolved: its core goes to the members.
    float members = share + core * smoothstep(4 * ps, 12 * ps, size);
    if (sub == 1) {
        // Constant surface brightness: beyond the cap the outer glow is dropped, the members carry it.
        float g = max(size, 5 * r * u.view.z / s.z); // Plummer softening: most of the mass within 2.5 ε
        float a = share * light / (g * g);
        return makeSprite(s, min(g, 240 * ps) / 2, float4(c.rgb * tint * a, 0), vid, u);
    }
    uint h = hash(uint(b.gas.y) * POPULATION + sub); // the body's identity (engine.Frame), not its mass
    float3 o = float3(unit(h) + unit(hash(h + 1)), unit(hash(h + 2)) + unit(hash(h + 3)),
                      unit(hash(h + 4)) + unit(hash(h + 5))) - 1;
    float3 m = project(b.pos.xyz + 1.5 * r * o, u);
    if (m.z < 0.05) return culled();
    // Luminosity function: many faint stars and a few bright giants (w = 7u⁶, mean 1). Brighter
    // stars are drawn larger, as their image wings rise above the background, with the same light;
    // at least 3 pixels, so they do not twinkle between pixels.
    float w = 7 * pow(unit(hash(h + 6)), 6.0);
    float ms = clamp(3 * ps * (0.8 + 0.5 * sqrt(w)), 3 * ps, 7 * ps);
    // Each star its own temperature around the color of the population: cooler or hotter.
    float t = unit(hash(h + 7));
    float3 col = c.rgb * mix(float3(1.08, 0.9, 0.74), float3(0.86, 0.97, 1.22), t);
    return makeSprite(m, ms / 2, float4(col * members * w / MEMBERS * light / (ms * ms), 0), vid, u);
}

vertex Sprite spriteVertex(uint vid [[vertex_id]], uint iid [[instance_id]],
                           const device Body *bodies [[buffer(0)]], constant Uniforms &u [[buffer(1)]],
                           const device float4 *attn [[buffer(2)]], const device float4 *holes [[buffer(3)]]) {
    uint bi = iid / POPULATION, sub = iid % POPULATION;
    Body b = bodies[bi];
    bool dusty = (int(u.misc.x) & 64) != 0;
    int kind = int(b.vel.w + 0.5);
    int fx = int(u.misc.x);
    float ps = u.view.w;
    float3 s = project(b.pos.xyz, u);
    if (s.z < 0.05) return culled();
    float focal = u.view.z;
    float rel = b.pos.w / max(u.cam.z, 1e-12);
    float size = clamp(0.05 * pow(rel, 1.0 / 3.0) * focal / s.z, 3.5 * ps, 28 * ps);
    float alpha = min(1.0, 0.55 + 0.2 * rel);
    if (kind == 0) { // star
        // A population's color is that of its age, not of its particle mass: old ones are yellowish white.
        float3 col = starColor(u.pop.x > 0 ? clamp(rel, 0.35, 1.0) : rel);
        float3 tint = 1;
        float age = b.gas.x;
        if (age >= 0 && age < u.misc.y) { // young stars from gas: hot, blue, bright (engine.youngBoost)
            float f = 1 - age / u.misc.y;
            col = mix(col, float3(0.6, 0.78, 1.0), f);
            tint = mix(float3(1), float3(1.0, 0.45, 0.65) / col, 0.5 * f * f); // still in its H II region
            alpha *= 1 + 3 * f;
            size *= 1 + 0.6 * f;
        }
        float4 c = shade(b, col, alpha, fx, bi, attn, holes, u);
        return population(b, s, c, tint, size, u.pop.x > 0 ? 0.4 : 1.0, sub, vid, u);
    }
    if (kind == 3 && sub == 1) {
        // Star-forming gas shines with the young stars it forms below the resolution
        // (engine.Population.starFormingLight): a soft bluish veil as large as the gas particle
        // (smoothing length h), with constant surface brightness, no points.
        float rho = b.gas.w, ion = b.gas.y;
        if (u.pop.z <= 0 || rho <= u.pop.y || ion >= 0.5) return culled();
        float w = min(20.0, u.pop.z * sqrt(rho)) * smoothstep(u.pop.y, 2 * u.pop.y, rho);
        float3 col = mix(float3(0.45, 0.65, 1.0), float3(1.0, 0.45, 0.65), 0.15);
        float4 c = shade(b, col, alpha * w, fx & ~64, bi, attn, holes, u);
        if (dusty) c.w *= attn[bi].b; // young stars are blue
        float g = max(size, max(5 * u.pop.x, 2 * b.gas.z) * focal / s.z);
        float a = c.w * u.eye.w * size * size / (g * g);
        return makeSprite(s, min(g, 240 * ps) / 2, float4(c.rgb * a, 0), vid, u);
    }
    if (sub != 0) return culled();
    if (kind == 2 && u.haze.w > 0) {
        // Cosmic web (engine.Look.WebAlpha): dark matter is all there is, so every particle is a
        // faint point and their sum shows filaments, halos and voids; the tone map compresses it.
        float3 col = float3(0.62, 0.5, 1.0);
        return makeSprite(s, 1.5 * ps, float4(col * u.haze.w, 0), vid, u);
    }
    if (kind == 3 && (fx & 32) != 0) {
        // Gas map in false colors (neutral blue, partly ionized green, ionized red): column density
        // of a kernel of diameter 2h, normalized by the Go core (engine.gasMapSize).
        float size = max(2 * b.gas.z * focal / s.z, 4 * ps);
        float ion = b.gas.y;
        float3 tint = float3(0.25 + 0.75 * ion, 0.45 + 0.5 * (1 - 2 * fabs(ion - 0.5)), 1 - 0.8 * ion);
        float a = u.haze.y * rel * 100 * ps * ps / (KERNEL_MEAN * size * size);
        return gasSprite(s, size, float4(tint * a, 0), vid, u);
    }
    if (kind == 3 && b.gas.y > 0.2) { // photoionized gas: Hα emission
        float size = max(3 * b.gas.z * focal / s.z, 3 * ps);
        float ion = b.gas.y;
        float brightness = ion * ion * b.pos.w * b.gas.w / (size * size);
        if (dusty) brightness *= attn[bi].r; // Hα is red
        float a = 1 - exp(-u.misc.z * brightness);
        return gasSprite(s, size, float4(float3(1.0, 0.16, 0.22) * a, 0), vid, u);
    }
    return culled();
}

// dustVertex draws the neutral gas as column density into the dust image: alpha accumulates
// "over", the background is attenuated by exp(−alpha · DustStrength · k).
vertex Sprite dustVertex(uint vid [[vertex_id]], uint iid [[instance_id]],
                         const device Body *bodies [[buffer(0)]], constant Uniforms &u [[buffer(1)]]) {
    Body b = bodies[iid];
    if (int(b.vel.w + 0.5) != 3) return culled();
    float3 s = project(b.pos.xyz, u);
    if (s.z < 0.05) return culled();
    float focal = u.view.z, ps = u.view.w;
    float size = max(2 * b.gas.z * focal / s.z, 2 * ps);
    float k = focal / s.z;
    float tauArea = u.misc.w * b.pos.w * (1 - b.gas.y) * k * k;
    float a = min(1.0, tauArea / (KERNEL_MEAN * size * size) / 2.5);
    return gasSprite(s, size, float4(a), vid, u);
}

// fogVertex draws dark matter as projected density into a quarter-resolution image with large,
// soft sprites; the compose pass blurs and saturates it into a faint fog behind the stars.
vertex Sprite fogVertex(uint vid [[vertex_id]], uint iid [[instance_id]],
                        const device Body *bodies [[buffer(0)]], constant Uniforms &u [[buffer(1)]]) {
    Body b = bodies[iid];
    if (int(b.vel.w + 0.5) != 2) return culled();
    float3 s = project(b.pos.xyz, u);
    if (s.z < 0.05) return culled();
    float ps = u.view.w;
    // Three times the haze sprite of the Go core's statistics: nine times the area, so a ninth of the
    // opacity keeps the same projected density (smoother, without particle noise).
    float size = clamp(0.36 * u.view.z / s.z, 18 * ps, 144 * ps);
    return makeSprite(s, size / 2, float4(u.haze.x / 9), vid, u);
}

fragment float4 spriteFragment(Sprite in [[stage_in]]) {
    float d2 = dot(in.uv, in.uv);
    if (d2 >= 1) discard_fragment();
    float k = 1 - d2;
    float a = in.gas > 0.5 ? k * k * k : min(1.0, exp(-d2 * 12) + 0.25 * exp(-d2 * 3));
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

// smoothSample reads a low-resolution image (bloom, fog) with a cubic B-spline from four bilinear
// taps: bilinear upscaling by 4 leaves the pixel grid visible as squares and crosses.
float4 smoothSample(texture2d<float> t, float2 uv) {
    float2 size = float2(t.get_width(), t.get_height());
    float2 p = uv * size - 0.5, i = floor(p), f = p - i;
    float2 f2 = f * f, f3 = f2 * f;
    float2 w0 = (1 - 3 * f + 3 * f2 - f3) / 6, w1 = (4 - 6 * f2 + 3 * f3) / 6;
    float2 w2 = (1 + 3 * f + 3 * f2 - 3 * f3) / 6, w3 = f3 / 6;
    float2 g0 = w0 + w1, g1 = w2 + w3;
    float2 h0 = (i - 0.5 + w1 / g0) / size, h1 = (i + 1.5 + w3 / g1) / size;
    return g0.y * (g0.x * t.sample(linearSampler, h0) + g1.x * t.sample(linearSampler, float2(h1.x, h0.y)))
         + g1.y * (g0.x * t.sample(linearSampler, float2(h0.x, h1.y)) + g1.x * t.sample(linearSampler, h1));
}

// trailFragment keeps a fading history that is never brighter than the stars.
fragment float4 trailFragment(Screen in [[stage_in]], texture2d<float> stars [[texture(0)]],
                              texture2d<float> previous [[texture(1)]]) {
    float4 p = previous.sample(linearSampler, in.uv);
    float4 s = stars.sample(linearSampler, in.uv);
    return max(max(p * 0.93 - 0.002, 0.0), s);
}

// composeFragment puts the stars over the static sky, which the dust in front attenuates per
// color channel (interstellar reddening; the stars were attenuated individually).
// The dark matter fog (blurred density, 1 at the densest 10 % of the covered screen) is saturated
// and capped, so it shows where the halo is without hiding the stars.
fragment float4 composeFragment(Screen in [[stage_in]], texture2d<float> stars [[texture(0)]],
                                texture2d<float> sky [[texture(1)]], texture2d<float> dust [[texture(2)]],
                                texture2d<float> fog [[texture(3)]], constant float &fogOn [[buffer(0)]]) {
    float tau = dust.sample(linearSampler, in.uv).a * 2.5;
    float4 c = sky.sample(linearSampler, in.uv);
    float4 out = float4(c.rgb * exp(-tau * float3(0.55, 0.75, 1.0)), c.a) + stars.sample(linearSampler, in.uv);
    if (fogOn > 0) {
        float d = smoothSample(fog, in.uv).r;
        out.rgb += float3(0.55, 0.35, 1.0) * 0.4 * (1 - exp(-1.2 * d));
    }
    return out;
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

// downFragment averages the div×div source pixels of every target pixel (bilinear taps at the
// corners of 2×2 blocks), so small stars do not flicker in and out of the bloom when they move.
fragment float4 downFragment(Screen in [[stage_in]], texture2d<float> src [[texture(0)]], constant float &div [[buffer(0)]]) {
    float2 texel = 1 / float2(src.get_width(), src.get_height());
    int n = max(1, int(div / 2 + 0.5));
    float4 c = 0;
    for (int y = 0; y < n; y++) {
        for (int x = 0; x < n; x++) {
            float2 o = (float2(x, y) * 2 + 1 - float(n)) * texel;
            c += src.sample(linearSampler, in.uv + o);
        }
    }
    return c / float(n * n);
}

// blurFragment is one direction of the separable Gaussian of the bloom (σ = 3 texels, every texel
// sampled: taps spaced wider than a texel make a comb, and a bright star a plaid of squares).
fragment float4 blurFragment(Screen in [[stage_in]], texture2d<float> src [[texture(0)]], constant float2 &dir [[buffer(0)]]) {
    float2 texel = dir / float2(src.get_width(), src.get_height());
    float4 c = src.sample(linearSampler, in.uv);
    float sum = 1;
    for (int i = 1; i <= 8; i++) {
        float w = exp(-float(i * i) / 18.0);
        float2 o = texel * float(i);
        c += (src.sample(linearSampler, in.uv + o) + src.sample(linearSampler, in.uv - o)) * w;
        sum += 2 * w;
    }
    return c / sum;
}

// finalFragment adds the bloom and maps the HDR image softly to the display.
fragment float4 finalFragment(Screen in [[stage_in]], texture2d<float> scene [[texture(0)]],
                              texture2d<float> bloom [[texture(1)]], constant float &bloomStrength [[buffer(0)]]) {
    float3 c = scene.sample(linearSampler, in.uv).rgb;
    if (bloomStrength > 0) { // without glow the bloom image is not drawn: never read it (0 · NaN = NaN)
        c += bloomStrength * smoothSample(bloom, in.uv).rgb;
    }
    // Hue-preserving: the brightest channel is mapped and the others are scaled with it, so bright
    // Hα stays deep red instead of turning pink-white (T-041). A little per-channel mapping keeps
    // the cores of galaxies white.
    float m = max(max(c.r, c.g), c.b);
    float3 hue = m > 1e-5 ? c * ((1 - exp(-1.6 * m)) / m) : c * 1.6;
    c = mix(hue, 1 - exp(-1.6 * c), 0.25);
    return float4(c, 1);
}

// ---- black holes ----

struct HoleOut { float4 position [[position]]; float2 uv; float r; float rings; float spin; float3 axis; float disk; };

// Hole instances, three float4 each (engine.Hole): x, y (render pixels), drawn horizon radius R,
// ISCO ring flag | spin a*, spin axis in screen space (x, y down, z toward the camera) | disk
// brightness, unused.
vertex HoleOut holeVertex(uint vid [[vertex_id]], uint iid [[instance_id]], const device float4 *holes [[buffer(0)]],
                          constant Uniforms &u [[buffer(1)]]) {
    float4 h = holes[3 * iid], k = holes[3 * iid + 1], q = holes[3 * iid + 2];
    float rg = 1 / (1 + sqrt(max(0.0, 1 - k.x * k.x))); // r_g in units of R = r₊
    float ext = max(3.2, (q.x > 0 ? DISK_OUTER + 2 : 9.6) * rg) * h.z; // the disk or the retrograde ISCO
    float2 c = corners[vid];
    float2 p = h.xy + c * ext;
    HoleOut o;
    o.position = float4(p.x / u.view.x * 2 - 1, 1 - p.y / u.view.y * 2, 0, 1);
    o.uv = c * ext / h.z; // in units of R
    o.r = h.z;
    o.rings = h.w;
    o.spin = k.x;
    o.axis = k.yzw;
    o.disk = q.x;
    return o;
}

// kerrISCO and kerrPhoton are the equatorial radii in units of r_g (sim/kerr.go).
float kerrISCO(float a, bool prograde) {
    float z1 = 1 + pow(1 - a * a, 1.0 / 3) * (pow(1 + a, 1.0 / 3) + pow(max(1 - a, 0.0), 1.0 / 3));
    float z2 = sqrt(3 * a * a + z1 * z1);
    float s = sqrt(max(0.0, (3 - z1) * (3 + z1 + 2 * z2)));
    return 3 + z2 + (prograde ? -s : s);
}

float kerrPhoton(float a, bool prograde) { return 2 * (1 + cos(2.0 / 3 * acos(prograde ? -a : a))); }

// diskEmission returns the observed color of the thin disk at point p (units of r_g, x right, y up,
// z toward the camera) with normal n, inner edge rin, or 0 off the disk (T-065): Novikov–Thorne
// flux F ∝ (1 − √(rin/r))/r³, Keplerian speed β = √(r_g/r), shift g = √(1 − 3r_g/r)/(1 − β_los)
// (gravitational redshift and Doppler), observed intensity I ∝ g⁴ F and temperature ∝ g F^¼.
// Drawn as √I, like the compressed scales of disk images: linear, F falls as r⁻³ and everything
// outside the shadow of a fast spinning hole (ISCO near r_g) would be black.
float3 diskEmission(float3 p, float3 n, float rin) {
    float r = length(p);
    if (r < rin || r > DISK_OUTER) return 0;
    float x = r / rin;
    float f = (1 - rsqrt(x)) / (x * x * x) / 0.0567; // normalized to 1 at its maximum (x = 49/36)
    float beta = min(0.7, rsqrt(r));
    float3 phi = normalize(cross(n, p));
    float g = sqrt(max(0.05, 1 - 3 / r)) / (1 - beta * phi.z);
    float t = clamp(pow(max(f, 0.0), 0.25) * g, 0.0, 1.6); // temperature relative to the hottest ring
    float3 col = t < 0.5 ? mix(float3(0.5, 0.08, 0.02), float3(1.0, 0.45, 0.1), t * 2)
               : t < 1.0 ? mix(float3(1.0, 0.45, 0.1), float3(1.0, 0.9, 0.7), (t - 0.5) * 2)
                         : mix(float3(1.0, 0.9, 0.7), float3(0.75, 0.85, 1.0), min(1.0, (t - 1) * 1.6));
    float fade = 1 - smoothstep(0.6 * DISK_OUTER, DISK_OUTER, r); // soft outer edge (edge0 < edge1 in Metal)
    return col * sqrt(max(f, 0.0)) * g * g * fade;
}

// holeFragment: glow out to the shadow edge, ISCO ring, photon ring and the black horizon
// (premultiplied; glow has alpha 0, so it adds light). For a rotating hole the rings are drawn
// smaller on the side whose orbits run with the spin and larger on the other side, by how much the
// spin axis is tilted across the line of sight (T-071). An accreting hole shows its disk, traced
// with one deflection: rays that pass the hole bend by 4 r_g/b and show the back of the disk above
// and below the shadow; rays inside b_c = √27 r_g fall in (T-065).
fragment float4 holeFragment(HoleOut in [[stage_in]]) {
    float d = length(in.uv);
    float px = 1 / max(in.r, 1e-3); // one pixel in units of R
    float rg = 1 / (1 + sqrt(max(0.0, 1 - in.spin * in.spin)));
    float4 disk = 0;
    if (in.disk > 0 && in.rings > 0) {
        float2 th = float2(in.uv.x, -in.uv.y) / rg; // image position in r_g, y up
        float3 n = normalize(float3(in.axis.x, -in.axis.y, in.axis.z) + float3(0, 0, 1e-4));
        float rin = kerrISCO(in.spin, true);
        float b = length(th);
        float3 col = 0;
        float nz = abs(n.z) < 1e-3 ? 1e-3 : n.z;
        float zf = -dot(n.xy, th) / nz; // straight ray: where it crosses the disk plane
        bool shadow = b < 5.196;
        // A ray inside the shadow still crosses the disk on its way in if the plane lies in front
        // of the photon orbit (z > −3 r_g); otherwise the near side of the disk is in front.
        if (zf >= (shadow ? -3.0 : 0.0)) col = diskEmission(float3(th, zf), n, rin);
        if (all(col == 0) && !shadow) {
            float alpha = min(3.0, 4 / b);
            float2 dir = th / b;
            float den = alpha * dot(n.xy, dir) + nz;
            float sb = dot(n.xy, th) / (abs(den) < 1e-4 ? 1e-4 : den);
            if (sb > 0) col = diskEmission(float3(th - alpha * dir * sb, -sb), n, rin);
        }
        float ring = exp(-pow((b - 5.196) / 0.25, 2.0)) * 0.6; // photon ring at the shadow edge
        // The shadow stays dark behind the near side of the disk, which crosses it in front.
        disk = float4((col * 2.5 + ring * float3(1.0, 0.8, 0.6)) * in.disk, shadow ? in.disk : 0);
        if (d < 1) return float4(disk.rgb, 1);
        if (any(col > 0) || shadow) return disk;
    }
    if (in.disk > 0 && in.rings == 0) { // too small to resolve (real c): the disk is a bright point
        return float4(float3(1.0, 0.85, 0.65) * in.disk * exp(-d * d / 4), 0);
    }
    if (d < 1) return float4(0, 0, 0, 1);
    float sinI = min(1.0, length(in.axis.xy));
    float side = 0; // −1 prograde side … +1 retrograde side
    if (in.spin > 0 && sinI > 1e-3) {
        float2 across = float2(-in.axis.y, in.axis.x) / sinI;
        side = dot(in.uv / max(d, 1e-3), across) * sinI;
    }
    float ph = mix(kerrPhoton(in.spin, true), kerrPhoton(in.spin, false), 0.5 + 0.5 * side) * rg;
    float isco = mix(kerrISCO(in.spin, true), kerrISCO(in.spin, false), 0.5 + 0.5 * side) * rg;
    float4 c = disk;
    if (d < 5.2 * rg) c += float4(18, 7, 2, 0) / 255; // shadow edge ≈ √27 r_g
    float ring = max(1.5 * px, 0.25);
    float a = 1 - smoothstep(ring / 2, ring / 2 + px, fabs(d - ph));
    c = c * (1 - a * 0.86) + float4(1.0, 0.59, 0.2, 1) * a * 0.86;
    if (in.rings > 0) {
        float b = 1 - smoothstep(0.5 * px, 1.5 * px, fabs(d - isco));
        c = c * (1 - b * 0.47) + float4(0.35, 0.24, 0.16, 1) * b * 0.47;
    }
    return c;
}
