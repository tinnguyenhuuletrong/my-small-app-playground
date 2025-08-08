package utils

type FenwickTree struct {
	size int
	tree []int64
}

func NewFenwickTree(size int) *FenwickTree {
	return &FenwickTree{
		size: size,
		tree: make([]int64, size+1),
	}
}

func (ft *FenwickTree) Add(index int, value int64) {
	if index < 0 || index >= ft.size {
		return // Or handle error
	}
	if ft.size == 0 {
		return
	}
	index++ // 1-based index
	for index <= ft.size {
		ft.tree[index] += value
		index += index & -index
	}
}

func (ft *FenwickTree) Query(index int) int64 {
	if index < 0 || index >= ft.size {
		return 0 // Or handle error, depending on desired behavior
	}
	if ft.size == 0 {
		return 0
	}
	index++ // 1-based index
	var sum int64
	for index > 0 {
		sum += ft.tree[index]
		index -= index & -index
	}
	return sum
}

func (ft *FenwickTree) Find(value int64) int {
	low := 0
	high := ft.size - 1
	result := -1

	for low <= high {
		mid := (low + high) / 2
		if ft.Query(mid) >= value {
			result = mid
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	return result
}
