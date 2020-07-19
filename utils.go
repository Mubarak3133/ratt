package ratt

import (
	"math/rand"
	"time"
)

func FatalCheck(e error) {
	if e != nil {
		panic(e)
	}
}

func CreateInlineJSFileName() string {
	seededRand := rand.New(
		rand.NewSource(time.Now().UnixNano()))
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b) + ".js"
}
