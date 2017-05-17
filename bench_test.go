package main

import (
	"math/rand"
	"sort"
	"testing"
)

func BenchmarkA(b *testing.B) {
	a1 := createSlice(1000)
	a2 := createSlice(5000)
	a3 := createSlice(10000)
	arr := [][]int{a1, a2, a3}
	numbersSet := make(map[int]struct{})
	for i := 0; i < b.N; i++ {
		result := make([]int, 0)
		for _, a := range arr {
			for _, n := range a {
				if _, ok := numbersSet[n]; !ok {
					numbersSet[n] = struct{}{}
					result = append(result, n)
				}
			}
		}
		sort.Ints(result)
	}
}

func BenchmarkB(b *testing.B) {
	a1 := createSlice(1000)
	a2 := createSlice(5000)
	a3 := createSlice(10000)
	for i := 0; i < b.N; i++ {
		MergeAndDedup(a1, a2, a3)
	}
}

func createSlice(l int) []int {
	res := make([]int, l)
	for i := 0; i < l; i++ {
		res[i] = rand.Int()
	}
	return res
}
