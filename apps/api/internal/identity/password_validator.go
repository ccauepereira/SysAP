package identity

import "unicode"

func isValidPassword(p string) bool {
	if len(p) < 15 {
		return false
	}
	var digits, specials int
	for _, r := range p {
		if r >= '0' && r <= '9' {
			digits++
		} else if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			specials++
		}
	}
	return digits >= 4 && specials >= 2
}
