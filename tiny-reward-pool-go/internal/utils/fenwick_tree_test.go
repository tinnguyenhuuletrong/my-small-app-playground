package utils

import (
	"testing"
)

func TestFenwickTree_AddAndQuery(t *testing.T) {
	size := 10
	ft := NewFenwickTree(size)

	ft.Add(0, 5)
	ft.Add(1, 3)
	ft.Add(2, 7)
	ft.Add(3, 2)
	ft.Add(4, 8)

	if ft.Query(0) != 5 {
		t.Errorf("Query(0) = %d, want 5", ft.Query(0))
	}
	if ft.Query(1) != 8 {
		t.Errorf("Query(1) = %d, want 8", ft.Query(1))
	}
	if ft.Query(2) != 15 {
		t.Errorf("Query(2) = %d, want 15", ft.Query(2))
	}
	if ft.Query(3) != 17 {
		t.Errorf("Query(3) = %d, want 17", ft.Query(3))
	}
	if ft.Query(4) != 25 {
		t.Errorf("Query(4) = %d, want 25", ft.Query(4))
	}
	if ft.Query(9) != 25 {
		t.Errorf("Query(9) = %d, want 25", ft.Query(9))
	}

	ft.Add(0, 10)
	if ft.Query(0) != 15 {
		t.Errorf("Query(0) after update = %d, want 15", ft.Query(0))
	}
	if ft.Query(4) != 35 {
		t.Errorf("Query(4) after update = %d, want 35", ft.Query(4))
	}
}

func TestFenwickTree_Find(t *testing.T) {
	size := 5
	ft := NewFenwickTree(size)

	ft.Add(0, 10)
	ft.Add(1, 20)
	ft.Add(2, 30)
	ft.Add(3, 40)
	ft.Add(4, 50)

	tests := []struct {
		value    int64
		expected int
	}{
		{1, 0},
		{10, 0},
		{11, 1},
		{30, 1},
		{31, 2},
		{60, 2},
		{100, 3},
		{150, 4},
		{151, -1}, // Should not find if value is greater than total sum
		{0, 0},
	}

	for _, tt := range tests {
		result := ft.Find(tt.value)
		if result != tt.expected {
			t.Errorf("Find(%d) = %d, want %d", tt.value, result, tt.expected)
		}
	}

	// Test with some zeros
	ftZero := NewFenwickTree(5)
	ftZero.Add(0, 0)
	ftZero.Add(1, 10)
	ftZero.Add(2, 0)
	ftZero.Add(3, 20)
	ftZero.Add(4, 0)

	zeroTests := []struct {
		value    int64
		expected int
	}{
		{1, 1},
		{10, 1},
		{11, 3},
		{30, 3},
		{31, -1},
	}

	for _, tt := range zeroTests {
		result := ftZero.Find(tt.value)
		if result != tt.expected {
			t.Errorf("Find(%d) with zeros = %d, want %d", tt.value, result, tt.expected)
		}
	}
}

func TestFenwickTree_Empty(t *testing.T) {
	ft := NewFenwickTree(0)

	ft.Add(0, 10) // Should not panic

	if ft.Query(0) != 0 {
		t.Errorf("Query(0) on empty tree = %d, want 0", ft.Query(0))
	}

	if ft.Find(1) != -1 {
		t.Errorf("Find(1) on empty tree = %d, want -1", ft.Find(1))
	}
}

func TestFenwickTree_SingleElement(t *testing.T) {
	ft := NewFenwickTree(1)

	ft.Add(0, 100)

	if ft.Query(0) != 100 {
		t.Errorf("Query(0) on single element tree = %d, want 100", ft.Query(0))
	}

	if ft.Find(50) != 0 {
		t.Errorf("Find(50) on single element tree = %d, want 0", ft.Find(50))
	}

	if ft.Find(100) != 0 {
		t.Errorf("Find(100) on single element tree = %d, want 0", ft.Find(100))
	}

	if ft.Find(101) != -1 {
		t.Errorf("Find(101) on single element tree = %d, want -1", ft.Find(101))
	}
}
