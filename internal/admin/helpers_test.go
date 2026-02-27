package admin

import "testing"

func TestFormatIDR(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{100, "100"},
		{999, "999"},
		{1000, "1.000"},
		{10000, "10.000"},
		{100000, "100.000"},
		{1000000, "1.000.000"},
		{1234567, "1.234.567"},
		{50000, "50.000"},
		{150000, "150.000"},
	}

	for _, tt := range tests {
		got := formatIDR(tt.input)
		if got != tt.want {
			t.Errorf("formatIDR(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestContainsCI(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "xyz", false},
		{"", "", true},
		{"foo", "", true},
		{"", "bar", false},
		{"Admin", "admin", true},
		{"admin", "Admin", true},
		{"nunutech4.0@gmail.com", "nunutech", true},
		{"nunutech4.0@gmail.com", "GMAIL", true},
	}

	for _, tt := range tests {
		got := containsCI(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("containsCI(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestUuidStr(t *testing.T) {
	// Test invalid UUID returns empty string
	got := uuidStr(parseUUID(""))
	if got != "" {
		t.Errorf("uuidStr(invalid) = %q, want empty", got)
	}

	// Test valid UUID roundtrip
	input := "550e8400-e29b-41d4-a716-446655440000"
	u := parseUUID(input)
	got = uuidStr(u)
	if got != input {
		t.Errorf("uuidStr(parseUUID(%q)) = %q, want %q", input, got, input)
	}
}

func TestGenerateGuestCode(t *testing.T) {
	code := generateGuestCode()

	// Must start with "ATOM-"
	if len(code) < 5 || code[:5] != "ATOM-" {
		t.Errorf("generateGuestCode() = %q, want prefix 'ATOM-'", code)
	}

	// Must be exactly 9 chars ("ATOM-" + 4 chars)
	if len(code) != 9 {
		t.Errorf("generateGuestCode() = %q, length = %d, want 9", code, len(code))
	}

	// Generate two codes — they should (almost always) be different
	code2 := generateGuestCode()
	if code == code2 {
		t.Logf("warning: two generated codes are identical: %q (unlikely but possible)", code)
	}
}

func TestResolveIPLocation_Localhost(t *testing.T) {
	// Localhost should return "Localhost" without making HTTP calls
	tests := []string{"", "127.0.0.1", "::1"}
	for _, ip := range tests {
		got := resolveIPLocation(ip)
		expected := "Localhost"
		if ip == "" {
			expected = "Localhost"
		}
		if got != expected {
			t.Errorf("resolveIPLocation(%q) = %q, want %q", ip, got, expected)
		}
	}
}
