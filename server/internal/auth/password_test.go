package auth

import (
	"testing"
	"time"
)

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

func TestLoginLimiter(t *testing.T) {
	limiter := loginLimiter{}
	now := time.Now()
	for i := 0; i < 20; i++ {
		if !limiter.allow(now) {
			t.Fatal("limited too early")
		}
	}
	if limiter.allow(now) || !limiter.allow(now.Add(time.Minute)) {
		t.Fatal("rate limit did not reset")
	}
}
