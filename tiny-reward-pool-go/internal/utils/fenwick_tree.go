package utils

// https://www.geeksforgeeks.org/dsa/binary-indexed-tree-or-fenwick-tree-2/
// The idea is based on the fact that all positive integers can be represented as the sum of powers of 2
// Example 12 = 8 + 4
// -> sum(1..12) = sum(1..8) + sum(9..12)

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

func (ft *FenwickTree) Size() int {
	return ft.size
}

func (ft *FenwickTree) Add(index int, value int64) {
	if index < 0 || index >= ft.size {
		return // Or handle error
	}
	if ft.size == 0 {
		return
	}
	index++ // 1-based index

	// Traverse all ancestors and add 'val'
	for index <= ft.size {

		// Add 'val' to current node of BI Tree
		ft.tree[index] += value

		// Update index to that of parent in update View
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

	// Traverse ancestors of BITree[index]
	for index > 0 {
		// Add current element of BITree to sum
		sum += ft.tree[index]

		// Move index to parent node in getSum View
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
