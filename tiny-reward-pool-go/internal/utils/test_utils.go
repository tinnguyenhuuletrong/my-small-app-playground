package utils

import (
	"math/rand"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// MockRandSource is a mock implementation of rand.Source for predictable testing.
type MockRandSource struct {
	Values []int64
	index  int
}

func (m *MockRandSource) Int63() int64 {
	if m.index >= len(m.Values) {
		panic("not enough mock random values")
	}
	val := m.Values[m.index]
	m.index++
	return val
}

func (m *MockRandSource) Seed(seed int64) {
	// No-op for mock
}

var _ rand.Source = (*MockRandSource)(nil)

type MockWAL struct{}

func (m *MockWAL) LogDraw(item types.WalLogDrawItem) error { return nil }
func (m *MockWAL) Close() error                            { return nil }
func (m *MockWAL) Flush() error                            { return nil }
func (m *MockWAL) SetSnapshotPath(path string)             {}
func (m *MockWAL) Rotate(path string) error                { return nil }
