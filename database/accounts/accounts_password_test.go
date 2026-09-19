package accounts

import "testing"

func TestPasswordHashUsesBcryptAndVerifies(t *testing.T) {
	hashed, err := hashPasswd("correct horse battery staple")
	if err != nil {
		t.Fatalf("hashPasswd failed: %v", err)
	}
	if ok, legacy := verifyPasswordHash(hashed, "correct horse battery staple"); !ok || legacy {
		t.Fatalf("bcrypt hash did not verify as modern hash: ok=%v legacy=%v", ok, legacy)
	}
	if ok, _ := verifyPasswordHash(hashed, "wrong"); ok {
		t.Fatal("bcrypt hash accepted wrong password")
	}
}

func TestLegacyPasswordHashStillVerifiesForMigration(t *testing.T) {
	legacy := legacyHashPasswd("old-password")
	if ok, needsUpgrade := verifyPasswordHash(legacy, "old-password"); !ok || !needsUpgrade {
		t.Fatalf("legacy hash migration detection failed: ok=%v needsUpgrade=%v", ok, needsUpgrade)
	}
	if ok, _ := verifyPasswordHash(legacy, "wrong"); ok {
		t.Fatal("legacy hash accepted wrong password")
	}
}
