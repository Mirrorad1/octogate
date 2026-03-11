package testutil

import (
	"strings"
	"testing"
)

// AssertNonceFormat verifies that a nonce matches the expected format
// Expected format: "0x" + exactly 64 hexadecimal characters (66 chars total)
func AssertNonceFormat(t *testing.T, nonce string) {
	t.Helper()

	if len(nonce) != 66 {
		t.Errorf("nonce length: got %d, want 66. nonce=%q", len(nonce), nonce)
	}

	if !strings.HasPrefix(nonce, "0x") {
		t.Errorf("nonce: must start with '0x', got %q", nonce[:2])
	}

	hexPart := nonce[2:]
	if len(hexPart) != 64 {
		t.Errorf("nonce hex part: got %d chars, want 64", len(hexPart))
	}

	for i, ch := range hexPart {
		if !isHexChar(rune(ch)) {
			t.Errorf("nonce: invalid hex character '%c' at position %d", ch, i+2)
		}
	}
}

// isHexChar checks if a rune is a valid hexadecimal character
func isHexChar(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

// AssertStatusCode verifies HTTP status code
func AssertStatusCode(t *testing.T, got, want int) {
	t.Helper()

	if got != want {
		t.Errorf("status code: got %d, want %d", got, want)
	}
}

// AssertContentType verifies Content-Type header
func AssertContentType(t *testing.T, got, want string) {
	t.Helper()

	if !strings.Contains(got, want) {
		t.Errorf("content-type: got %q, want %q", got, want)
	}
}
