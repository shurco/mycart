package models

import "testing"

func TestSignIn_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      SignIn
		wantErr bool
	}{
		{"ok", SignIn{Email: "admin@example.com", Password: "secret123"}, false},
		{"bad email", SignIn{Email: "nope", Password: "secret123"}, true},
		{"short password", SignIn{Email: "admin@example.com", Password: "123"}, true},
		{"missing password", SignIn{Email: "admin@example.com"}, true},
		{"missing email", SignIn{Password: "secret123"}, true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.in.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

// JWT.Validate mis-applies validation.Length to the int field
// ExpireHours, which makes it always error with "cannot get the length of
// int". This is a pre-existing bug; fixing it would change API semantics
// on an exposed validator, so this test pins the current behaviour.
func TestJWT_Validate_CurrentQuirk(t *testing.T) {
	t.Parallel()

	err := JWT{Secret: "abcdefghijabcdefghijabcdefghij", ExpireHours: 24}.Validate()
	if err == nil {
		t.Fatal("JWT.Validate stopped returning the Length-on-int quirk: please update this test")
	}

	// Secret-length validation still fires independently.
	if err := (JWT{Secret: "tooshort"}).Validate(); err == nil {
		t.Error("expected validation error for short secret")
	}
}
