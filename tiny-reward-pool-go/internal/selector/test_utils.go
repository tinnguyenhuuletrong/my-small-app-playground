package selector

// MockRandSource is a mock for rand.Source to control random number generation.
type MockRandSource struct {
	values []int64
	idx    int
}

func (m *MockRandSource) Int63() int64 {
	if m.idx >= len(m.values) {
		panic("not enough mock random values")
	}
	val := m.values[m.idx]
	m.idx++
	return val
}

func (m *MockRandSource) Seed(seed int64) {
	// Do nothing for mock
}
