package animation

import (
	"testing"
	"time"
)

func TestEasings(t *testing.T) {
	easings := map[string]EasingFunc{
		"EaseLinear":       EaseLinear,
		"EaseSmoothstep":   EaseSmoothstep,
		"EaseSmootherstep": EaseSmootherstep,
		"EaseInQuad":       EaseInQuad,
		"EaseOutQuad":      EaseOutQuad,
		"EaseInOutQuad":    EaseInOutQuad,
		"EaseInCubic":      EaseInCubic,
		"EaseOutCubic":     EaseOutCubic,
		"EaseInOutCubic":   EaseInOutCubic,
	}

	for name, fn := range easings {
		t.Run(name, func(t *testing.T) {
			v0 := fn(0)
			if v0 != 0 {
				t.Errorf("easing(%s)(0) = %f, want 0", name, v0)
			}
			v1 := fn(1)
			if v1 != 1 {
				t.Errorf("easing(%s)(1) = %f, want 1", name, v1)
			}
		})
	}
}

func TestTimeline_AnimateTo(t *testing.T) {
	tl := NewTimeline(0.0)
	duration := 100 * time.Millisecond
	start := time.Now()

	// Start animation from 0 to 1
	v := tl.AnimateTo("key", 1.0, duration, start)
	if v != 0.0 {
		t.Errorf("initial value = %f, want 0.0", v)
	}

	// At midpoint, value should be between 0 and 1
	mid := start.Add(50 * time.Millisecond)
	vMid := tl.Get("key", mid)
	if vMid <= 0.0 || vMid >= 1.0 {
		t.Errorf("mid value = %f, want between 0 and 1", vMid)
	}

	// After duration, value should be 1
	end := start.Add(duration + time.Millisecond)
	vEnd := tl.Get("key", end)
	if vEnd != 1.0 {
		t.Errorf("end value = %f, want 1.0", vEnd)
	}
}

func TestTimeline_IsAnimating(t *testing.T) {
	tl := NewTimeline(0.0)
	duration := 100 * time.Millisecond
	start := time.Now()

	tl.AnimateTo("key", 1.0, duration, start)

	// During animation
	mid := start.Add(50 * time.Millisecond)
	if !tl.IsAnimating("key", mid) {
		t.Error("expected IsAnimating=true during animation")
	}

	// After animation completes
	after := start.Add(duration + time.Millisecond)
	// Must call Get to update current value so IsAnimating sees current == target
	tl.Get("key", after)
	if tl.IsAnimating("key", after) {
		t.Error("expected IsAnimating=false after animation completes")
	}
}
