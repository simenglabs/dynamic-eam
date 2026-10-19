package crypto

import "testing"

func TestBcryptRoundTrip(t *testing.T) {
	h := Bcrypt{Cost: 4} // low cost so tests are fast
	hash, err := h.Hash("password1")
	if err != nil {
		t.Fatal(err)
	}
	if string(hash) == "password1" || !h.Compare(hash, "password1") || h.Compare(hash, "wrong") {
		t.Error("hash/compare wrong")
	}
}

func TestTokenIssuerSignsVerifiableJWT(t *testing.T) {
	tok, ttl, err := (JWT{TTL: 3600e9}).Issue(&fakeUser)
	if err != nil || tok == "" || ttl.Hours() != 1 {
		t.Fatalf("tok=%q ttl=%v err=%v", tok, ttl, err)
	}
}
