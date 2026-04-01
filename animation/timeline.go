// Copyright © 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: animation/timeline.go
// Summary: Thread-safe animation timeline with configurable easing functions.
// Usage: Simplifies effect implementation by handling all animation state automatically.
// Notes: Supports per-key timelines with linear, smoothstep, and custom easing.

package animation

import (
	"sync"
	"time"
)

// AnimateOptions configures an animation transition
type AnimateOptions struct {
	Duration time.Duration // Animation duration (default: 0 = instant)
	Easing   EasingFunc    // Easing function (default: EaseSmoothstep)
}

// DefaultAnimateOptions returns options with smoothstep easing
func DefaultAnimateOptions(duration time.Duration) AnimateOptions {
	return AnimateOptions{
		Duration: duration,
		Easing:   EaseSmoothstep,
	}
}

// keyState tracks animation state for a single key
type keyState struct {
	current   float32
	start     float32
	target    float32
	startTime time.Time
	duration  time.Duration
	easing    EasingFunc
}

// Timeline provides thread-safe, per-key animation timelines with automatic state management
type Timeline struct {
	states         map[any]*keyState
	mu             sync.RWMutex
	defaultEasing  EasingFunc
	defaultInitial float32
}

// NewTimeline creates a new timeline manager
// defaultInitial: initial value for uninitialized keys (typically 0.0)
func NewTimeline(defaultInitial float32) *Timeline {
	return &Timeline{
		states:         make(map[any]*keyState),
		defaultEasing:  EaseSmoothstep,
		defaultInitial: defaultInitial,
	}
}

// AnimateTo starts or updates an animation for the given key
// Returns the current animated value at this moment
//
// Minimal usage:
//
//	value := timeline.AnimateTo(key, target, duration, now)
//
// With custom easing:
//
//	value := timeline.AnimateToWithOptions(key, target, AnimateOptions{
//	    Duration: 300*time.Millisecond,
//	    Easing: EaseInOutCubic,
//	}, now)
func (tl *Timeline) AnimateTo(key any, target float32, duration time.Duration, now time.Time) float32 {
	return tl.AnimateToWithOptions(key, target, DefaultAnimateOptions(duration), now)
}

// AnimateToWithOptions starts an animation with custom easing function
func (tl *Timeline) AnimateToWithOptions(key any, target float32, opts AnimateOptions, now time.Time) float32 {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	state := tl.states[key]

	if state == nil {
		// Initialize new key
		state = &keyState{
			current:  tl.defaultInitial,
			start:    tl.defaultInitial,
			target:   target,
			duration: opts.Duration,
			easing:   opts.Easing,
		}
		if opts.Easing == nil {
			state.easing = tl.defaultEasing
		}
		tl.states[key] = state

		// If duration is zero, jump to target immediately
		if opts.Duration <= 0 {
			state.current = target
			return target
		}

		state.startTime = now
		return state.current
	}

	// Update existing animation
	// First, compute current value to use as new start
	current := tl.computeValue(state, now)

	// Start new animation from current position
	state.current = current
	state.start = current
	state.target = target
	state.startTime = now
	state.duration = opts.Duration
	if opts.Easing != nil {
		state.easing = opts.Easing
	}

	// If duration is zero or already at target, finish immediately
	if opts.Duration <= 0 || current == target {
		state.current = target
		return target
	}

	return current
}

// Get returns the current animated value for a key
// If the key hasn't been initialized, returns the default initial value
func (tl *Timeline) Get(key any, now time.Time) float32 {
	tl.mu.RLock()
	state := tl.states[key]
	tl.mu.RUnlock()

	if state == nil {
		return tl.defaultInitial
	}

	tl.mu.Lock()
	value := tl.computeValue(state, now)
	state.current = value
	tl.mu.Unlock()

	return value
}

// GetCached returns the last computed value for a key without recomputing.
// This is useful when called after Update() in the same frame to avoid redundant calculations.
// If the key hasn't been initialized, returns the default initial value.
func (tl *Timeline) GetCached(key any) float32 {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	state := tl.states[key]
	if state == nil {
		return tl.defaultInitial
	}

	return state.current
}

// IsAnimating returns true if the key is currently animating
func (tl *Timeline) IsAnimating(key any, now time.Time) bool {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	state := tl.states[key]
	if state == nil || state.duration <= 0 {
		return false
	}

	elapsed := now.Sub(state.startTime)
	return elapsed < state.duration && state.current != state.target
}

// HasActiveAnimations returns true if any key is currently animating
// This method still exists for convenience but should be used sparingly.
// Prefer checking specific keys with IsAnimating when possible for better performance.
func (tl *Timeline) HasActiveAnimations(now time.Time) bool {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	for _, state := range tl.states {
		if state.duration > 0 && now.Sub(state.startTime) < state.duration {
			if state.current != state.target {
				return true
			}
		}
	}
	return false
}

// Update advances all animations to the given time
// This is called by the effect manager on each frame
func (tl *Timeline) Update(now time.Time) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	for _, state := range tl.states {
		state.current = tl.computeValue(state, now)
	}
}

// Reset removes the timeline state for a key
func (tl *Timeline) Reset(key any) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	delete(tl.states, key)
}

// Clear removes all timeline states
func (tl *Timeline) Clear() {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.states = make(map[any]*keyState)
}

// computeValue calculates the current value for a state at the given time
// Must be called with lock held
func (tl *Timeline) computeValue(state *keyState, now time.Time) float32 {
	if state.duration <= 0 {
		return state.target
	}

	if now.Before(state.startTime) {
		return state.start
	}

	elapsed := now.Sub(state.startTime)
	if elapsed >= state.duration {
		return state.target
	}

	// Calculate progress [0, 1]
	progress := float32(elapsed) / float32(state.duration)
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}

	// Apply easing function
	easing := state.easing
	if easing == nil {
		easing = tl.defaultEasing
	}
	easedProgress := easing(progress)

	// Interpolate
	return state.start + (state.target-state.start)*easedProgress
}
