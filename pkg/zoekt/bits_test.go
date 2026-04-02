package zoekt

import (
	"testing"
)

func TestNgramBasic(t *testing.T) {
	runes := [3]rune{'a', 'b', 'c'}
	ng := runesToNGram(runes)
	if ng == 0 {
		t.Error("ngram should not be zero")
	}

	decoded := ngramToRunes(ng)
	if decoded[0] != 'a' || decoded[1] != 'b' || decoded[2] != 'c' {
		t.Errorf("ngramToRunes = %v, want [a b c]", decoded)
	}
}

func TestNgramToBytes(t *testing.T) {
	ng := runesToNGram([3]rune{'a', 'b', 'c'})
	b := ngramToBytes(ng)
	if len(b) != 3 || b[0] != 'a' || b[1] != 'b' || b[2] != 'c' {
		t.Errorf("ngramToBytes = %v, want [a b c]", b)
	}
}

func TestStringToNGram(t *testing.T) {
	ng := stringToNGram("abc")
	if ng == 0 {
		t.Error("ngram should not be zero")
	}

	ngEmpty := stringToNGram("")
	if ngEmpty != 0 {
		t.Error("empty string should produce zero ngram")
	}
}

func TestToSizedDeltas(t *testing.T) {
	tests := []struct {
		input    []uint32
		expected string // we'll just verify it doesn't panic
	}{
		{[]uint32{}, ""},
		{[]uint32{0}, ""},
		{[]uint32{1, 2, 3}, ""},
		{[]uint32{100, 200, 300}, ""},
	}

	for _, tc := range tests {
		result := toSizedDeltas(tc.input)
		if result == nil {
			t.Errorf("toSizedDeltas(%v) returned nil", tc.input)
		}
	}
}

func TestFromSizedDeltas(t *testing.T) {
	original := []uint32{0, 5, 10, 15}
	encoded := toSizedDeltas(original)
	decoded := fromSizedDeltas(encoded, nil)

	if len(decoded) != len(original) {
		t.Errorf("decoded len = %d, want %d", len(decoded), len(original))
	}

	for i, v := range original {
		if decoded[i] != v {
			t.Errorf("decoded[%d] = %d, want %d", i, decoded[i], v)
		}
	}
}

func TestToSizedDeltas16(t *testing.T) {
	original := []uint16{0, 5, 10, 15}
	encoded := toSizedDeltas16(original)
	if encoded == nil {
		t.Error("toSizedDeltas16 returned nil")
	}
}

func TestFromSizedDeltas16(t *testing.T) {
	original := []uint16{0, 5, 10, 15}
	encoded := toSizedDeltas16(original)
	decoded := fromSizedDeltas16(encoded, nil)

	if len(decoded) != len(original) {
		t.Errorf("decoded len = %d, want %d", len(decoded), len(original))
	}
}

func TestMarshalDocSections(t *testing.T) {
	secs := []DocumentSection{
		{Start: 0, End: 10},
		{Start: 20, End: 30},
	}

	data := marshalDocSections(secs)
	if data == nil {
		t.Error("marshalDocSections returned nil")
	}

	// Verify roundtrip
	unmarshaled := unmarshalDocSections(data, nil)
	if len(unmarshaled) != len(secs) {
		t.Errorf("unmarshaled len = %d, want %d", len(unmarshaled), len(secs))
	}
	for i, sec := range secs {
		if unmarshaled[i].Start != sec.Start || unmarshaled[i].End != sec.End {
			t.Errorf("unmarshaled[%d] = %v, want %v", i, unmarshaled[i], sec)
		}
	}
}

func TestNewLinesIndices(t *testing.T) {
	tests := []struct {
		input    string
		expected []uint32
	}{
		{"", nil},
		{"no newlines", nil},
		{"line1\nline2", []uint32{5}},
		{"a\nb\nc", []uint32{1, 3}},
		{"\n\n", []uint32{0, 1}},
	}

	for _, tc := range tests {
		result := newLinesIndices([]byte(tc.input))
		if len(result) != len(tc.expected) {
			t.Errorf("newLinesIndices(%q) len = %d, want %d", tc.input, len(result), len(tc.expected))
			continue
		}
		for i, v := range tc.expected {
			if result[i] != v {
				t.Errorf("newLinesIndices(%q)[%d] = %d, want %d", tc.input, i, result[i], v)
			}
		}
	}
}

func TestCountNewlines(t *testing.T) {
	if countNewlines([]byte("")) != 0 {
		t.Error("countNewlines empty string")
	}
	if countNewlines([]byte("no newline")) != 0 {
		t.Error("countNewlines no newline")
	}
	if countNewlines([]byte("a\nb\nc")) != 2 {
		t.Error("countNewlines two newlines")
	}
}
