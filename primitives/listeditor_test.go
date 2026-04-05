// Copyright 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package primitives

import (
	"testing"
)

func newTestEditor() *ListEditor {
	return NewListEditor(ListEditorConfig{
		LabelKey:  "id",
		ToggleKey: "enabled",
	})
}

func item(id string, enabled bool) map[string]interface{} {
	return map[string]interface{}{
		"id":      id,
		"enabled": enabled,
	}
}

// TestListEditor_AddRemove verifies adding and removing items.
func TestListEditor_AddRemove(t *testing.T) {
	le := newTestEditor()
	le.SetItems([]map[string]interface{}{
		item("alpha", true),
		item("beta", false),
	})

	if got := len(le.Items()); got != 2 {
		t.Fatalf("expected 2 items after SetItems, got %d", got)
	}

	le.addItem()
	if got := len(le.Items()); got != 3 {
		t.Fatalf("expected 3 items after addItem, got %d", got)
	}

	le.removeItem(1)
	items := le.Items()
	if got := len(items); got != 2 {
		t.Fatalf("expected 2 items after removeItem(1), got %d", got)
	}
	if got := le.itemLabel(items[0]); got != "alpha" {
		t.Errorf("items[0] label: want %q, got %q", "alpha", got)
	}
	// index 1 was "beta" which was removed; index 1 is now the blank item added earlier
	if got := le.itemLabel(items[1]); got != "(unnamed)" {
		t.Errorf("items[1] label: want %q (blank added item), got %q", "(unnamed)", got)
	}
}

// TestListEditor_Reorder verifies moveDown and moveUp.
func TestListEditor_Reorder(t *testing.T) {
	le := newTestEditor()
	le.SetItems([]map[string]interface{}{
		item("A", true),
		item("B", true),
		item("C", true),
	})

	// moveDown(0): A/B/C → B/A/C
	le.moveDown(0)
	items := le.Items()
	wantOrder := []string{"B", "A", "C"}
	for i, want := range wantOrder {
		if got := le.itemLabel(items[i]); got != want {
			t.Errorf("after moveDown(0): items[%d] = %q, want %q", i, got, want)
		}
	}

	// moveUp(2): B/A/C → B/C/A
	le.moveUp(2)
	items = le.Items()
	wantOrder = []string{"B", "C", "A"}
	for i, want := range wantOrder {
		if got := le.itemLabel(items[i]); got != want {
			t.Errorf("after moveUp(2): items[%d] = %q, want %q", i, got, want)
		}
	}
}

// TestListEditor_Toggle verifies toggling an item's enabled field.
func TestListEditor_Toggle(t *testing.T) {
	le := newTestEditor()
	le.SetItems([]map[string]interface{}{
		item("x", true),
	})

	if !le.itemToggle(le.Items()[0]) {
		t.Fatal("expected initial toggle value to be true")
	}

	le.toggleItem(0)

	if le.itemToggle(le.Items()[0]) {
		t.Error("expected toggle value to be false after toggleItem(0)")
	}
}

// TestListEditor_ExpandCollapse verifies expand/collapse logic via expandedIdx.
func TestListEditor_ExpandCollapse(t *testing.T) {
	le := newTestEditor()
	le.SetItems([]map[string]interface{}{
		item("y", false),
	})

	if le.expandedIdx != -1 {
		t.Fatalf("expandedIdx should start at -1, got %d", le.expandedIdx)
	}

	le.toggleExpand(0)
	if le.expandedIdx != 0 {
		t.Errorf("after first toggleExpand(0): expandedIdx = %d, want 0", le.expandedIdx)
	}

	le.toggleExpand(0)
	if le.expandedIdx != -1 {
		t.Errorf("after second toggleExpand(0): expandedIdx = %d, want -1", le.expandedIdx)
	}
}

// TestListEditor_OnChange verifies that OnChange fires on mutations.
func TestListEditor_OnChange(t *testing.T) {
	le := newTestEditor()
	le.SetItems([]map[string]interface{}{
		item("a", true),
		item("b", false),
		item("c", true),
	})

	var calls int
	le.OnChange = func(_ []map[string]interface{}) { calls++ }

	le.addItem()
	if calls != 1 {
		t.Errorf("addItem: OnChange called %d times, want 1", calls)
	}

	le.removeItem(0)
	if calls != 2 {
		t.Errorf("removeItem: OnChange called %d times, want 2", calls)
	}

	le.moveUp(1)
	if calls != 3 {
		t.Errorf("moveUp: OnChange called %d times, want 3", calls)
	}

	le.toggleItem(0)
	if calls != 4 {
		t.Errorf("toggleItem: OnChange called %d times, want 4", calls)
	}
}

// TestListEditor_DetailKeys verifies that detailKeys returns sorted keys
// excluding LabelKey and ToggleKey.
func TestListEditor_DetailKeys(t *testing.T) {
	le := newTestEditor() // LabelKey="id", ToggleKey="enabled"
	it := map[string]interface{}{
		"id":      "foo",
		"enabled": true,
		"style":   "bold",
		"extra":   42,
	}

	keys := le.detailKeys(it)
	want := []string{"extra", "style"}
	if len(keys) != len(want) {
		t.Fatalf("detailKeys: got %v, want %v", keys, want)
	}
	for i, k := range want {
		if keys[i] != k {
			t.Errorf("detailKeys[%d]: got %q, want %q", i, keys[i], k)
		}
	}
}
