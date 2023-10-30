package internal

import (
	"math/rand"
	"os"
)

func GenRandInt(min, max int) int {
	return rand.Intn(max-min) + min
}

func GenerateRandomString(n int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num := rand.Intn(len(letters))
		ret[i] = letters[num]
	}

	return string(ret)
}

func GetHostName() string {
	val, err := os.Hostname()
	if err != nil {
		return ""
	}
	return val
}
