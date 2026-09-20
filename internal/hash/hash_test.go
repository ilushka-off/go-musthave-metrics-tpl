package hash

import "testing"

func TestSign_ProducesHexOfExpectedLength(t *testing.T) {
	sig := Sign([]byte("payload"), "secret")

	// HMAC-SHA256 is 32 bytes -> 64 hex characters.
	if len(sig) != 64 {
		t.Fatalf("len(Sign(...)) = %d; want 64", len(sig))
	}
}

func TestSign_IsDeterministic(t *testing.T) {
	a := Sign([]byte("payload"), "secret")
	b := Sign([]byte("payload"), "secret")

	if a != b {
		t.Fatalf("Sign() is not deterministic: %q != %q", a, b)
	}
}

func TestValid_AcceptsOwnSignature(t *testing.T) {
	data := []byte("payload")
	sig := Sign(data, "secret")

	if !Valid(data, "secret", sig) {
		t.Fatal("Valid() = false; want true for a signature produced by Sign()")
	}
}

func TestValid_RejectsTamperedData(t *testing.T) {
	sig := Sign([]byte("payload"), "secret")

	if Valid([]byte("tampered"), "secret", sig) {
		t.Fatal("Valid() = true; want false for tampered data")
	}
}

func TestValid_RejectsWrongKey(t *testing.T) {
	data := []byte("payload")
	sig := Sign(data, "secret")

	if Valid(data, "wrong-secret", sig) {
		t.Fatal("Valid() = true; want false for the wrong key")
	}
}

func TestValid_RejectsMalformedHex(t *testing.T) {
	if Valid([]byte("payload"), "secret", "not-hex-at-all") {
		t.Fatal("Valid() = true; want false for a malformed hex signature")
	}
}
