package zoekt

import (
	"testing"

	"github.com/grafana/regexp"

	zkq "github.com/liyu1981/code_explorer/pkg/zoekt/query"
)

func TestQueryString(t *testing.T) {
	tests := []struct {
		q    zkq.Q
		want string
	}{
		{
			q:    &zkq.Substring{Pattern: "hello"},
			want: "substr:\"hello\"",
		},
		{
			q:    &zkq.Substring{Pattern: "main", FileName: true},
			want: "file_substr:\"main\"",
		},
		{
			q:    &zkq.Substring{Pattern: "main", CaseSensitive: true},
			want: "case_substr:\"main\"",
		},
		{
			q:    &zkq.Substring{Pattern: "main", FileName: true, CaseSensitive: true},
			want: "case_file_substr:\"main\"",
		},
		{
			q:    &zkq.And{Children: []zkq.Q{&zkq.Substring{Pattern: "a"}, &zkq.Substring{Pattern: "b"}}},
			want: "(and substr:\"a\" substr:\"b\")",
		},
		{
			q:    &zkq.Or{Children: []zkq.Q{&zkq.Substring{Pattern: "a"}, &zkq.Substring{Pattern: "b"}}},
			want: "(or substr:\"a\" substr:\"b\")",
		},
		{
			q:    &zkq.Not{Child: &zkq.Substring{Pattern: "a"}},
			want: "(not substr:\"a\")",
		},
		{
			q:    &zkq.Branch{Pattern: "main"},
			want: "branch:\"main\"",
		},
		{
			q:    &zkq.Repo{Regexp: regexp.MustCompile("repo")},
			want: "repo:repo",
		},
		{
			q:    &zkq.Language{Language: "Go"},
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
