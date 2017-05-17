package main

import (
	"reflect"
	"testing"
)

func TestMergeAndDedup(t *testing.T) {
	a := []int{1, 3, 5, 8, 12}
	b := []int{1, 3, 7, 8, 10, 16}

	expected := []int{1, 3, 5, 7, 8, 10, 12, 16}

	obtained := MergeAndDedup(a, b)
	if !reflect.DeepEqual(obtained, expected) {
		t.Errorf("Expected: %v, obtained %v", expected, obtained)
	}
}
