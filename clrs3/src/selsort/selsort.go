package selsort

func Sort(a []int64) {
	for j := 0; j < len(a); j++ {
		key := a[j]
		min := key
		minIdx := j
		for i := j + 1; i < len(a); i++ {
			if a[i] < min {
				min = a[i]
				minIdx = i
			}
		}
		a[j] = min
		a[minIdx] = key
	}
}
