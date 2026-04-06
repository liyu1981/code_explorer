package zoekt

import (
	"bytes"
	"testing"
)

func TestAllMatchTree(t *testing.T) {
	m := &allMatchTree{}
	doc := m.nextDoc()
	if doc != 0 {
		t.Errorf("allMatchTree.nextDoc() = %d, want 0", doc)
	}
}

func TestNoMatchTree(t *testing.T) {
	m := &noMatchTree{}
	doc := m.nextDoc()
	if doc != ^uint32(0) {
		t.Errorf("noMatchTree.nextDoc() = %d, want max uint32", doc)
	}
}

func TestAndMatchTree(t *testing.T) {
	children := []matchTree{
		&simpleMatchTree{docs: []uint32{1, 3, 5}},
		&simpleMatchTree{docs: []uint32{2, 3, 4}},
	}
	m := &andMatchTree{children: children}

	doc := m.nextDoc()
	if doc != 3 {
		t.Errorf("andMatchTree.nextDoc() first = %d, want 3 (common to both)", doc)
	}
}

func TestOrMatchTree(t *testing.T) {
	children := []matchTree{
		&simpleMatchTree{docs: []uint32{1, 3, 5}},
		&simpleMatchTree{docs: []uint32{2, 3, 4}},
	}
	m := &orMatchTree{children: children}

	doc := m.nextDoc()
	if doc != 1 {
		t.Errorf("orMatchTree.nextDoc() first = %d, want 1", doc)
	}
	doc = m.nextDoc()
	if doc != 2 {
		t.Errorf("orMatchTree.nextDoc() second = %d, want 2", doc)
	}
}

func TestBranchMaskMatchTree(t *testing.T) {
	masks := []uint64{0b001, 0b010, 0b100, 0b011}
	m := &branchMaskMatchTree{
		masks: masks,
		mask:  0b001,
	}

	doc := m.nextDoc()
	if doc != 0 {
		t.Errorf("branchMaskMatchTree.nextDoc() first = %d, want 0", doc)
	}
	doc = m.nextDoc()
	if doc != 3 {
		t.Errorf("branchMaskMatchTree.nextDoc() second = %d, want 3", doc)
	}
	doc = m.nextDoc()
	if doc != ^uint32(0) {
		t.Errorf("branchMaskMatchTree.nextDoc() third = %d, want max uint32", doc)
	}
}

func TestSubstringMatchTree(t *testing.T) {
	m := &substringMatchTree{
		ids:           []uint32{0, 1, 2},
		ends:          []uint32{100, 200, 300},
		pattern:       []byte("test"),
		caseSensitive: false,
		fileName:      false,
		lastDoc:       -1,
	}

	doc := m.nextDoc()
	if doc != 0 {
		t.Errorf("substringMatchTree.nextDoc() first = %d, want 0", doc)
	}
	doc = m.nextDoc()
	if doc != 1 {
		t.Errorf("substringMatchTree.nextDoc() second = %d, want 1", doc)
	}
}

func TestConstQueryString(t *testing.T) {
	c := &Const{Value: true}
	if c.String() != "TRUE" {
		t.Errorf("Const{true}.String() = %q, want \"TRUE\"", c.String())
	}

	c = &Const{Value: false}
	if c.String() != "FALSE" {
		t.Errorf("Const{false}.String() = %q, want \"FALSE\"", c.String())
	}
}

func TestSplitNGrams(t *testing.T) {
	tests := []struct {
		input    string
		wantLen  int
		wantNums []int
	}{
		{"ab", 0, nil},
		{"abc", 1, []int{0}},
		{"abcd", 2, []int{0, 1}},
		{"abcdef", 4, []int{0, 1, 2, 3}},
	}

	for _, tt := range tests {
		offs := splitNGrams([]byte(tt.input))
		if len(offs) != tt.wantLen {
			t.Errorf("splitNGrams(%q) len = %d, want %d", tt.input, len(offs), tt.wantLen)
		}
	}
}

func TestFindSelectiveNgrams(t *testing.T) {
	offs := []ngramOff{
		{ngram: 1, index: 0},
		{ngram: 2, index: 1},
		{ngram: 3, index: 2},
	}
	indexMap := []uint32{0, 1, 2}
	frequencies := []uint32{100, 50, 200}

	first, last := findSelectiveNgrams(offs, indexMap, frequencies)

	if first.index != 1 {
		t.Errorf("first.index = %d, want 1", first.index)
	}
	if last.index != 2 {
		t.Errorf("last.index = %d, want 2", last.index)
	}
}

func TestIntersectUint32(t *testing.T) {
	a := []uint32{1, 2, 3, 4, 5}
	b := []uint32{3, 4, 5, 6, 7}

	res := intersectUint32(a, b)

	if len(res) != 3 {
		t.Errorf("intersectUint32 len = %d, want 3", len(res))
	}
	if res[0] != 3 || res[1] != 4 || res[2] != 5 {
		t.Errorf("intersectUint32 = %v, want [3 4 5]", res)
	}
}

func TestDecodePostingList(t *testing.T) {
	data := []byte{0x03, 0x02, 0x03}
	postings := decodePostingList(data)

	if len(postings) != 3 {
		t.Errorf("decodePostingList len = %d, want 3", len(postings))
	}
	if postings[0] != 3 || postings[1] != 5 || postings[2] != 8 {
		t.Errorf("decodePostingList = %v, want [3 5 8]", postings)
	}
}

func TestToLower(t *testing.T) {
	input := []byte("Hello World")
	result := toLower(input)

	expected := []byte("hello world")
	if !bytes.Equal(result, expected) {
		t.Errorf("toLower = %q, want %q", string(result), string(expected))
	}
}

type simpleMatchTree struct {
	docs []uint32
	idx  int
}

func (m *simpleMatchTree) nextDoc() uint32 {
	if m.idx >= len(m.docs) {
		return ^uint32(0)
	}
	doc := m.docs[m.idx]
	m.idx++
	return doc
}
