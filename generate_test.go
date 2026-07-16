package ghostcaptcha

import "testing"

func TestLineEndingReplacer(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no special chars", "hello world", "hello world"},
		{"crlf", "a\r\nb", "a\nb"},
		{"bare cr", "a\rb", "a\nb"},
		{"tab expands to four spaces", "a\tb", "a    b"},
		{"mixed", "a\r\nb\rc\td", "a\nb\nc    d"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lineEndingReplacer.Replace(tt.in); got != tt.want {
				t.Errorf("lineEndingReplacer.Replace(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
