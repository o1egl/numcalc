package main

import (
	"math/rand"
	"sort"
	"sync"
	"testing"
)

func BenchmarkA(b *testing.B) {
	a1 := createSlice(1000)
	a2 := createSlice(5000)
	a3 := createSlice(10000)
	b.ResetTimer()
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
	b.ResetTimer()
	wg := sync.WaitGroup{}
	wg.Add(3)
	go sortInts(a1, &wg)
	go sortInts(a2, &wg)
	go sortInts(a3, &wg)
	wg.Wait()
	for i := 0; i < b.N; i++ {
		MergeAndDedup(a1, a2, a3)
	}
}

func sortInts(arr []int, wg *sync.WaitGroup) {
	sort.Ints(arr)
	wg.Done()
}

func createSlice(l int) []int {
	res := make([]int, l)
	for i := 0; i < l; i++ {
		res[i] = rand.Int()
	}
	return res
}
