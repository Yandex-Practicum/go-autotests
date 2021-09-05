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

	s := make([]byte, slen)
	for i := 0; i < slen; i++ {
		num, _ := rand.Int(rand.Reader, lettersLen)
		s[i] = letters[num.Int64()]
	}

	return string(s)
}
