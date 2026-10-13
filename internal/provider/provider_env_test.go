package provider

import "testing"

func TestBaseURLFromEnv(t *testing.T) {
	cases := []struct {
		name, apiURL, baseURL, want string
	}{
		{"canonical UW_API_URL", "https://a.example", "", "https://a.example"},
		{"alias UW_BASE_URL", "", "https://b.example", "https://b.example"},
		{"UW_API_URL wins over the alias", "https://a.example", "https://b.example", "https://a.example"},
		{"neither set", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("UW_API_URL", tc.apiURL)
			t.Setenv("UW_BASE_URL", tc.baseURL)
			if got := baseURLFromEnv(); got != tc.want {
				t.Fatalf("baseURLFromEnv() = %q, want %q", got, tc.want)
			}
		})
	}
}
