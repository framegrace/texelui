// Copyright © 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: animation/easing.go
// Summary: Easing functions for animation timelines.

package animation

// EasingFunc defines an easing function that maps progress [0,1] to eased value [0,1]
type EasingFunc func(progress float32) float32

// Common easing functions
var (
	// EaseLinear - No easing, constant speed
	EaseLinear EasingFunc = func(t float32) float32 { return t }

	// EaseSmoothstep - Smooth S-curve (default, recommended for most animations)
	// Accelerates at start, decelerates at end
	EaseSmoothstep EasingFunc = func(t float32) float32 {
		return t * t * (3.0 - 2.0*t)
	}

	// EaseSmootherstep - Even smoother S-curve with zero derivatives at 0 and 1
	EaseSmootherstep EasingFunc = func(t float32) float32 {
		return t * t * t * (t*(t*6.0-15.0) + 10.0)
	}

	// EaseInQuad - Quadratic ease-in (slow start, accelerating)
	EaseInQuad EasingFunc = func(t float32) float32 {
		return t * t
	}

	// EaseOutQuad - Quadratic ease-out (fast start, decelerating)
	EaseOutQuad EasingFunc = func(t float32) float32 {
		return t * (2.0 - t)
	}

	// EaseInOutQuad - Quadratic ease-in-out
	EaseInOutQuad EasingFunc = func(t float32) float32 {
		if t < 0.5 {
			return 2.0 * t * t
		}
		return -1.0 + (4.0-2.0*t)*t
	}

	// EaseInCubic - Cubic ease-in (slower start)
	EaseInCubic EasingFunc = func(t float32) float32 {
		return t * t * t
	}

	// EaseOutCubic - Cubic ease-out
	EaseOutCubic EasingFunc = func(t float32) float32 {
		t1 := t - 1.0
		return t1*t1*t1 + 1.0
	}

	// EaseInOutCubic - Cubic ease-in-out
	EaseInOutCubic EasingFunc = func(t float32) float32 {
		if t < 0.5 {
			return 4.0 * t * t * t
		}
		t1 := 2.0*t - 2.0
		return 1.0 + t1*t1*t1*0.5
	}
)
