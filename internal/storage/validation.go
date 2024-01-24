package storage

import "errors"

var ErrKeyFormatInvalid = errors.New("key format invalid")

func validateKeyFormat(key string) bool {
	for _, ch := range []rune(key) {
		if (ch >= '0' && ch <= '9') ||
			(ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch == '_') {
			continue
		} else {
			return false
		}
	}
	return true
}
