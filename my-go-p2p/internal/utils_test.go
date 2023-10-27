package internal_test

import (
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/internal"
)

func Test_Name(t *testing.T) {
	val := internal.GenerateRandomString(5)
	t.Log(val)
}
