package game

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
)

func newID(prefix string) string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return prefix + "_" + hex.EncodeToString(buf)
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
