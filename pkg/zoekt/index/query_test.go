package zoekt

import (
	"testing"

	"github.com/grafana/regexp"
)

func TestQueryString(t *testing.T) {
	tests := []struct {
		q    Query
		want string
	}{
		{
			q:    &Substring{Pattern: "hello"},
			want: "substr:\"hello\"",
		},
		{
			q:    &Substring{Pattern: "main", FileName: true},
			want: "file_substr:\"main\"",
		},
		{
			q:    &Substring{Pattern: "main", CaseSensitive: true},
			want: "case_substr:\"main\"",
		},
		{
			q:    &Substring{Pattern: "main", FileName: true, CaseSensitive: true},
			want: "case_file_substr:\"main\"",
		},
		{
			q:    &And{Children: []Query{&Substring{Pattern: "a"}, &Substring{Pattern: "b"}}},
			want: "(and substr:\"a\" substr:\"b\")",
		},
		{
			q:    &Or{Children: []Query{&Substring{Pattern: "a"}, &Substring{Pattern: "b"}}},
			want: "(or substr:\"a\" substr:\"b\")",
		},
		{
			q:    &Not{Child: &Substring{Pattern: "a"}},
			want: "(not substr:\"a\")",
		},
		{
			q:    &Branch{Pattern: "main"},
			want: "branch:\"main\"",
		},
		{
			q:    &Repo{Regexp: regexp.MustCompile("repo")},
			want: "repo:repo",
		},
		{
			q:    &Language{Language: "Go"},
			want: "lang:Go",
		},
	}

	for _, tt := range tests {
		if got := tt.q.String(); got != tt.want {
			t.Errorf("%T.String() = %q, want %q", tt.q, got, tt.want)
		}
	}
}

func TestParseQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "TRUE"},
		{"  ", "TRUE"},
		{"hello", "substr:\"hello\""},
		{"  world  ", "substr:\"world\""},
	}

	for _, tt := range tests {
		q, err := ParseQuery(tt.input)
		if err != nil {
			t.Errorf("ParseQuery(%q) error: %v", tt.input, err)
			continue
		}
		if got := q.String(); got != tt.want {
			t.Errorf("ParseQuery(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
