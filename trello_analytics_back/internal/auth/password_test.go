package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("secret-pass-12")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" || hash == "secret-pass-12" {
		t.Fatalf("expected bcrypt hash, got %q", hash)
	}
	if !CheckPassword(hash, "secret-pass-12") {
		t.Fatal("expected password match")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("expected mismatch")
	}
}

func TestValidatePassword_MinLength(t *testing.T) {
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected error for short password")
	}
	if err := ValidatePassword("123456789012"); err != nil {
		t.Fatalf("12 chars should pass: %v", err)
	}
}

func TestDummyPasswordHash_Usable(t *testing.T) {
	h := DummyPasswordHash()
	if h == "" {
		t.Fatal("empty dummy hash")
	}
	if CheckPassword(h, "timing-dummy-password-not-used") != true {
		t.Fatal("dummy hash should verify its seed password")
	}
}

func TestRoleAtLeast(t *testing.T) {
	if !RoleAtLeast(RoleAdmin, RoleMember) {
		t.Fatal("admin should satisfy member")
	}
	if RoleAtLeast(RoleMember, RoleAdmin) {
		t.Fatal("member must not satisfy admin")
	}
	if !RoleAtLeast(RoleTeamLead, RoleTeamLead) {
		t.Fatal("team_lead should satisfy team_lead")
	}
}
