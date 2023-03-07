package toolkit

import (
	"crypto/rand"
)

const ALPHABET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is a toolkit for general purpose
type Tools struct{}

// RandomString generates a random string with given length
func (t *Tools) RandomString(length int) string {
	s, r := make([]rune, length), []rune(ALPHABET)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]

	}
	return string(s)
}
