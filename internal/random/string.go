package random

import (
	"crypto/rand"
	"math/big"
	mathrand "math/rand"
)

// ASCIIString generates random ASCII string
func ASCIIString(minLen, maxLen int) string {
	var letters = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFJHIJKLMNOPQRSTUVWXYZ"

	slen := mathrand.Intn(maxLen-minLen) + minLen
	lettersLen := big.NewInt(int64(len(letters)))

	s := make([]byte, 0, slen)
	i := 0
	for len(s) < slen {
		num, _ := rand.Int(rand.Reader, lettersLen)
		char := letters[num.Int64()]
		if i == 0 && '0' <= char && char <= '9' {
			continue
		}
		s = append(s, char)
		i++
	}

	return string(s)
}
