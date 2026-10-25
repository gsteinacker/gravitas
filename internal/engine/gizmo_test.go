package engine

import (
	"math"
	"testing"
	"time"

	"galaxy/internal/app"
	"galaxy/internal/sim"
)

// padScene plans a launch in the spiral galaxy seen from above, right of the center.
func padScene(t *testing.T) *Engine {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	e := New(0, 5000)
	t.Cleanup(e.Close)
	e.Runner.SetPaused(true)
	e.SetViewport(1440, 900)
	time.Sleep(50 * time.Millisecond)
	e.Cam.Pitch = math.Pi / 2
	e.BeginLaunch(720+300, 450)
	return e
}

func (e *Engine) handleScreen(t *testing.T, h Handle) (float64, float64) {
	t.Helper()
	p, ok := e.padGeometry()
	if !ok {
		t.Fatal("no pad")
	}
	x, y, ok := e.screen(e.anchor(p, h))
	if !ok {
		t.Fatalf("handle %d behind the camera", h)
	}
	return x, y
}

func TestHitLaunchFindsHandles(t *testing.T) {
	e := padScene(t)
	for _, h := range []Handle{HandleGhost, HandlePrograde, HandleRetrograde, HandleOut, HandleIn} {
		x, y := e.handleScreen(t, h)
		if got := e.HitLaunch(x+2, y-2); got != h {
			t.Errorf("hit at handle %d gives %d", h, got)
		}
	}
	gx, gy := e.handleScreen(t, HandleGhost)
	if got := e.HitLaunch(gx+ringPoints, gy); got != HandleRing {
		t.Errorf("ring: %d", got)
	}
	if got := e.HitLaunch(10, 10); got != HandleNone {
		t.Errorf("far away: %d", got)
	}
	// Seen from above, up, down and height point at the camera: hidden, they must not cover the
	// ghost. From the side, they are there.
	if got := e.HitLaunch(gx, gy); got != HandleGhost {
		t.Errorf("ghost from above: %d", got)
	}
	e.Cam.Pitch = 0.3
	for _, h := range []Handle{HandleUp, HandleHeight} {
		x, y := e.handleScreen(t, h)
		if got := e.HitLaunch(x, y); got != h {
			t.Errorf("from the side, handle %d gives %d", h, got)
		}
	}
}

func TestDragAxisHandleAddsVelocity(t *testing.T) {
	e := padScene(t)
	p, _ := e.padGeometry()
	x0, y0 := e.handleScreen(t, HandlePrograde)
	gx, gy := e.handleScreen(t, HandleGhost)
	e.BeginDrag(HandlePrograde, x0, y0)
	// One axis length further out: Δv = a/k = v_c / arrowPerCirc, prograde.
	e.DragTo(x0+(x0-gx), y0+(y0-gy), false)
	l := e.LaunchPlan()
	want := p.a / p.k
	if math.Abs(l.Speed-want) > 1e-6*want || math.Abs(l.Azimuth) > 1e-6 || math.Abs(l.Elevation) > 1e-6 {
		t.Errorf("speed %v (want %v), azimuth %v, elevation %v", l.Speed, want, l.Azimuth, l.Elevation)
	}
	if e.EndDrag() {
		t.Error("an axis drag is no slingshot")
	}
}

func TestSnapToCircularSpeedAndAxes(t *testing.T) {
	e := padScene(t)
	l := e.LaunchPlan()
	radial, prograde := l.basis()
	tilted := prograde.Scale(math.Cos(0.05)).Add(radial.Scale(math.Sin(0.05))) // 3° off prograde
	e.setVelocity(tilted.Scale(1.02*l.Circular), true)
	if got := e.LaunchPlan(); got.Speed != got.Circular || got.Azimuth != 0 {
		t.Errorf("snapped speed %v (circular %v), azimuth %v", got.Speed, got.Circular, got.Azimuth)
	}
	e.setVelocity(tilted.Scale(1.02*l.Circular), false)
	if got := e.LaunchPlan(); got.Speed == got.Circular || got.Azimuth == 0 {
		t.Error("without snapping the values must stay")
	}
}

func TestSlingshotLaunchesAgainstThePull(t *testing.T) {
	e := padScene(t)
	gx, gy := e.handleScreen(t, HandleGhost)
	px, py := e.handleScreen(t, HandlePrograde)
	e.BeginDrag(HandleGhost, gx, gy)
	e.DragTo(gx-(px-gx), gy-(py-gy), false) // pull backwards (retrograde)
	l := e.LaunchPlan()
	if l.Speed <= 0 || math.Abs(l.Azimuth) > 0.05 {
		t.Errorf("slingshot: speed %v, azimuth %v (want prograde)", l.Speed, l.Azimuth)
	}
	if !e.EndDrag() {
		t.Error("releasing the slingshot must launch")
	}
	// A tiny pull is a click, not a launch.
	e.BeginDrag(HandleGhost, gx, gy)
	e.DragTo(gx+1, gy, false)
	if e.EndDrag() {
		t.Error("a click on the ghost launched it")
	}
}

func TestBinaryConservesMomentum(t *testing.T) {
	e := padScene(t)
	e.Tool, e.Mass = app.ToolBlackHole, 5
	e.LaunchPreset(PresetBinary)
	l := e.LaunchPlan()
	if !l.Binary || l.HoleMass <= 0 {
		t.Fatalf("binary needs a hole as attractor: %+v", l)
	}
	// The new hole starts with the center's velocity plus its share of the relative velocity.
	before := l.CenterVel.Scale(l.HoleMass + e.Mass)
	e.GoLaunch()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var holes []int
		var p [3]float64
		e.Runner.View(func(sn *sim.Snapshot) {
			holes = holes[:0]
			p = [3]float64{}
			for i, k := range sn.Kind {
				if k == sim.BlackHole {
					holes = append(holes, i)
					m := float64(sn.Mass[i])
					p[0] += m * float64(sn.VX[i])
					p[1] += m * float64(sn.VY[i])
					p[2] += m * float64(sn.VZ[i])
				}
			}
		})
		if len(holes) == 2 {
			d := math.Hypot(math.Hypot(p[0]-before.X, p[1]-before.Y), p[2]-before.Z)
			if scale := l.HoleMass * l.Speed; d > 1e-3*scale {
				t.Errorf("momentum of the pair changed by %v (scale %v)", d, scale)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("second hole not added")
}
