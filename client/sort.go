package client

import "sort"

//SortMap sort map
func SortMap(m map[int32]int64) []int64 {
	keys := make([]int, 0)
	for k := range m {
		keys = append(keys, int(k))
	}

	result := make([]int64, 0)
	sort.Ints(keys)

	for _, v := range keys {
		result = append(result, m[int32(v)])
	}

	return result
}
