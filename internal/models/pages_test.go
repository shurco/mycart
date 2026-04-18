package models

import "testing"

func TestPage_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      Page
		wantErr bool
	}{
		{
			"ok",
			Page{
				Core: Core{ID: "123456789012345"},
				Name: "About us",
				Slug: "about",
			},
			false,
		},
		{"bad ID", Page{Core: Core{ID: "short"}, Name: "Abc", Slug: "abc"}, true},
		{"short name", Page{Core: Core{ID: "123456789012345"}, Name: "Hi", Slug: "abc"}, true},
		{"short slug", Page{Core: Core{ID: "123456789012345"}, Name: "Hello", Slug: "a"}, true},
		{"empty is ok (fields are optional)", Page{}, false},
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
