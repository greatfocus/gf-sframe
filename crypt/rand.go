package crypt

import (
	mathRand "math/rand"
	"strconv"
	"strings"
	"time"
)

// RandomNumber generates numbers
func RandomNumber(size int) int64 {
	var num = "0123456789"

	buf := make([]byte, size)
	for i := 0; i < size; i++ {
		buf[i] = num[mathRand.Intn(len(num))]
	}
	strVal := string(buf)
	intVal, err := strconv.ParseInt(strVal, 10, 0)
	if err != nil {
		panic(err)
	}
	return intVal
}

// RandomString generates string
func RandomString(leng int64) string {
	mathRand.Seed(time.Now().Unix())
	var output strings.Builder
	var charSet = "abcdedfghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var length = 20
	for i := 0; i < length; i++ {
		random := mathRand.Intn(len(charSet))
		randomChar := charSet[random]
		output.WriteString(string(randomChar))
	}

	return output.String()
}
