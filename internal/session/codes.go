package session

import (
	"crypto/rand"
	"encoding/hex"
)

// codeAlphabet excludes visually ambiguous chars (0/O, 1/I/L) so a code read
// aloud across the table can't be mistyped. 31 symbols.
const (
	codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	codeLen      = 4
)

// newCode returns a random share code like "K7QF". crypto/rand, not math/rand:
// codes shouldn't be guessable from one another. Modulo bias over 31 symbols is
// negligible for a 4-char code.
func newCode() (string, error) {
	b := make([]byte, codeLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i, c := range b {
		b[i] = codeAlphabet[int(c)%len(codeAlphabet)]
	}
	return string(b), nil
}

// newPlayerID returns a 128-bit random hex token. It's the player's identity in
// SQLite AND the value the signed cookie carries, so it must be
// unguessable, not pretty.
func newPlayerID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
