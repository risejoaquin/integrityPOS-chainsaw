package crypto

import "testing"

func TestSignVerify(t *testing.T) {
	secret := "secret"
	message := "receipt123|session123|15050|2026-03-25T12:00:00Z|cajero1|terminal1"
	sig := Sign(secret, message)
	if ok, err := Verify(secret, message, sig); !ok || err != nil {
		t.Fatalf("expected valid signature got %v %v", ok, err)
	}
}
