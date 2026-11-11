package auth

import "testing"

func TestPasswordHash(t *testing.T) {
	password := "correct test phrase"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	other, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if hash == other || hash == password || !VerifyPassword(hash, password) || VerifyPassword(hash, "incorrect phrase") || VerifyPassword("$argon2id$v=19$m=999999,t=3,p=2$bad$bad", password) {
		t.Fatal("password verification failed")
	}
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("short password accepted")
	}
}
