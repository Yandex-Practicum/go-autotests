package random

// ASCIIString generates random ASCII string
func ASCIIString(minLen, maxLen int) string {
	var letters = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFJHIJKLMNOPQRSTUVWXYZ"

	slen := rnd.Intn(maxLen-minLen) + minLen

	s := make([]byte, 0, slen)
	i := 0
	for len(s) < slen {
		idx := rnd.Intn(len(letters) - 1)
		char := letters[idx]
		if i == 0 && '0' <= char && char <= '9' {
			continue
		}
		s = append(s, char)
		i++
	}

	return string(s)
}

// DigitString generates random string consists of digits
func DigitString(minLen, maxLen int) string {
	var letters = "0123456789"

	slen := rnd.Intn(maxLen-minLen) + minLen

	s := make([]byte, 0, slen)
	i := 0
	for len(s) < slen {
		idx := rnd.Intn(len(letters) - 1)
		char := letters[idx]
		if i == 0 && '0' == char {
			continue
		}
		s = append(s, char)
		i++
	}

	return string(s)
}
