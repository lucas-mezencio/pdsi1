package user

import (
	"errors"
	"strconv"
	"strings"
)

// ErrInvalidCPF is returned when a CPF fails validation.
var ErrInvalidCPF = errors.New("invalid CPF")

// ValidateCPF validates a Brazilian CPF (Cadastro de Pessoa Física).
// It accepts formatted input (e.g., "529.982.247-25") or digits-only,
// strips non-digit characters, and verifies both check digits using the
// official modulo-11 algorithm. Returns the normalized digits-only CPF on success.
//
// Rejects:
//   - empty input
//   - non-11-digit inputs
//   - inputs where all digits are equal (e.g., "00000000000")
//   - inputs whose first or second check digit is incorrect
func ValidateCPF(raw string) (string, error) {
	digits := stripNonDigits(raw)
	if len(digits) != 11 {
		return "", ErrInvalidCPF
	}
	if allSameDigit(digits) {
		return "", ErrInvalidCPF
	}

	first, err := computeCheckDigit(digits[:9])
	if err != nil {
		return "", ErrInvalidCPF
	}
	if string(first) != string(digits[9]) {
		return "", ErrInvalidCPF
	}

	second, err := computeCheckDigit(digits[:10])
	if err != nil {
		return "", ErrInvalidCPF
	}
	if string(second) != string(digits[10]) {
		return "", ErrInvalidCPF
	}

	return digits, nil
}

// computeCheckDigit computes the CPF check digit at position 10 (or 11) of
// the given digit slice using the modulo-11 algorithm:
//   sum = sum(digit[i] * (len - i)) for i in 0..len-1
//   if sum*10 % 11 == 10 -> check = 0, else check = (sum*10) % 11
func computeCheckDigit(prefix string) (byte, error) {
	n := len(prefix)
	sum := 0
	for i, r := range prefix {
		d, err := strconv.Atoi(string(r))
		if err != nil {
			return 0, err
		}
		// weight starts at (n+1) for i=0 and decreases: e.g. for n=9, weights are 10,9,...,2.
		sum += d * (n + 1 - i)
	}
	mod := (sum * 10) % 11
	if mod == 10 {
		mod = 0
	}
	return byte('0' + mod), nil
}

func stripNonDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func allSameDigit(s string) bool {
	if s == "" {
		return false
	}
	first := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != first {
			return false
		}
	}
	return true
}