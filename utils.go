package main

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	MaxURLRuneCount = 2083
	MinURLRuneCount = 11

	IP           string = `(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`
	URLSchema    string = `((https?):\/\/)`
	URLPath      string = `((\/|\?|#)[^\s]*)`
	URLPort      string = `(:(\d{1,5}))`
	URLIP        string = `([1-9]\d?|1\d\d|2[01]\d|22[0-3])(\.(1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.([0-9]\d?|1\d\d|2[0-4]\d|25[0-4]))`
	URLSubdomain string = `((www\.)|([a-zA-Z0-9]([-\.][a-zA-Z0-9]+)*))`
	URL          string = `^` + URLSchema + `((` + URLIP + `|(\[` + IP + `\])|(([a-zA-Z0-9]([a-zA-Z0-9-]+)?[a-zA-Z0-9]([-\.][a-zA-Z0-9]+)*)|(` + URLSubdomain + `?))?(([a-zA-Z\x{00a1}-\x{ffff}0-9]+-?-?)*[a-zA-Z\x{00a1}-\x{ffff}0-9]+)(?:\.([a-zA-Z\x{00a1}-\x{ffff}]{1,}))?))` + URLPort + `?` + URLPath + `?$`
)

var rxURL = regexp.MustCompile(URL)

// IsValidURL checks if the string is a valid URL.
func IsValidURL(str string) bool {
	if str == "" || utf8.RuneCountInString(str) >= MaxURLRuneCount || len(str) <= MinURLRuneCount || strings.HasPrefix(str, ".") {
		return false
	}
	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	if strings.HasPrefix(u.Host, ".") {
		return false
	}
	if u.Host == "" && (u.Path != "" && !strings.Contains(u.Path, ".")) {
		return false
	}
	return rxURL.MatchString(str)

}

// MergeAndDedup merges input arrays and removes duplicates. Input data should be sorted
func MergeAndDedup(arrs ...[]int) []int {
	positionMap := make([]int, len(arrs))
	resultCap := 0
	min := 0
	isMinSet := false
	for i, arr := range arrs {
		positionMap[i] = 0
		resultCap += len(arr)
		if !isMinSet && len(arr) > 0 {
			min = arr[0]
			isMinSet = true
		}
		if len(arr) > 0 && arr[0] < min {
			min = arr[0]
		}
	}

	if resultCap == 0 {
		return []int{}
	}

	result := make([]int, 0, resultCap)
	result = append(result, min)
	haveItemsToMerge := true
	lastInsertedFrom := 0
	for haveItemsToMerge {
		haveItemsToMerge = false

		nextItem := 0
		isNextItemSet := false
		for c, array := range arrs {
			// check array for empty size
			if len(array) == 0 {
				continue
			}
			position := positionMap[c]
			// check complete iteration on array
			if position == len(array) {
				continue
			}

			haveItemsToMerge = true

			// skip duplicates
			if result[len(result)-1] == array[position] {
				positionMap[c]++
				continue
			}

			// select next item from array comparision
			if !isNextItemSet {
				lastInsertedFrom = c
				nextItem = array[position]
				positionMap[c]++
				isNextItemSet = true
				continue
			}

			// skip duplicates
			if array[position] == nextItem {
				positionMap[c]++
				continue
			}

			// select min value
			if array[position] < nextItem {
				nextItem = array[position]
				positionMap[c]++
				//decrease previous
				positionMap[lastInsertedFrom]--

				lastInsertedFrom = c

			}
		}

		if isNextItemSet {
			result = append(result, nextItem)
		}
	}
	return result
}
