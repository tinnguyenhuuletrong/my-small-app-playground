package selector

import "math/rand"

// MockRandSource is a mock implementation of rand.Source for predictable testing.
type MockRandSource struct {
	values []int64
	index  int
}

func (m *MockRandSource) Int63() int64 {
	if m.index >= len(m.values) {
		panic("not enough mock random values")
	}
	val := m.values[m.index]
	m.index++
	return val
}

func (m *MockRandSource) Seed(seed int64) {
	// No-op for mock
}

var _ rand.Source = (*MockRandSource)(nil)