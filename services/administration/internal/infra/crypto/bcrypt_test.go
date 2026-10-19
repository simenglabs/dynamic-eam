package crypto

import "testing"

func TestBcryptRoundTrip(t *testing.T) {
	h := Bcrypt{Cost: 4} // cost rendah supaya test cepat
	hash, err := h.Hash("password1")
	if err != nil {
		t.Fatal(err)
	}
	if string(hash) == "password1" || !h.Compare(hash, "password1") || h.Compare(hash, "salah") {
		t.Error("hash/compare salah")
	}
}

func TestTokenIssuerSignsVerifiableJWT(t *testing.T) {
	tok, ttl, err := (JWT{TTL: 3600e9}).Issue(&fakeUser)
	if err != nil || tok == "" || ttl.Hours() != 1 {
		t.Fatalf("tok=%q ttl=%v err=%v", tok, ttl, err)
	}
}
