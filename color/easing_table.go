// Copyright © 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: color/easing_table.go
// Summary: Shared easing index/name/function mapping for DynamicColor serialization.

package color

import "github.com/framegrace/texelui/animation"

// Easing indices for protocol serialization.
const (
	EasingLinear       uint8 = 0
	EasingSmoothstep   uint8 = 1
	EasingSmootherstep uint8 = 2
	EasingInQuad       uint8 = 3
	EasingOutQuad      uint8 = 4
	EasingInOutQuad    uint8 = 5
	EasingInCubic      uint8 = 6
	EasingOutCubic     uint8 = 7
	EasingInOutCubic   uint8 = 8
)

// EasingByIndex maps indices to easing functions.
var EasingByIndex = []animation.EasingFunc{
	animation.EaseLinear,
	animation.EaseSmoothstep,
	animation.EaseSmootherstep,
	animation.EaseInQuad,
	animation.EaseOutQuad,
	animation.EaseInOutQuad,
	animation.EaseInCubic,
	animation.EaseOutCubic,
	animation.EaseInOutCubic,
}

// EasingByName maps easing names to indices.
var EasingByName = map[string]uint8{
	"linear":          EasingLinear,
	"smoothstep":      EasingSmoothstep,
	"smootherstep":    EasingSmootherstep,
	"ease-in-quad":    EasingInQuad,
	"ease-out-quad":   EasingOutQuad,
	"ease-in-out-quad": EasingInOutQuad,
	"ease-in-cubic":    EasingInCubic,
	"ease-out-cubic":   EasingOutCubic,
	"ease-in-out-cubic": EasingInOutCubic,
}

// LookupEasing returns the easing function for an index, defaulting to linear.
func LookupEasing(idx uint8) animation.EasingFunc {
	if int(idx) < len(EasingByIndex) {
		return EasingByIndex[idx]
	}
	return animation.EaseLinear
}

// LookupEasingByName returns the index for a named easing, defaulting to linear.
func LookupEasingByName(name string) uint8 {
	if idx, ok := EasingByName[name]; ok {
		return idx
	}
	return EasingLinear
}
