package passwords

import "testing"

func TestHashAndCompare(t *testing.T) {
	password := "super-secret"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == password {
		t.Fatal("hash must not match plaintext")
	}
	if err := ComparePassword(hash, password); err != nil {
		t.Fatalf("expected valid password, got error: %v", err)
	}
	if err := ComparePassword(hash, "wrong"); err == nil {
		t.Fatal("expected error for wrong password")
	}
}
