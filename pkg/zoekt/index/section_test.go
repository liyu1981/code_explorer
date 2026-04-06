package index

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestWriterBasic(t *testing.T) {
	buf := &bytes.Buffer{}
	w := &writer{w: buf}

	n, err := w.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != 5 {
		t.Errorf("Write returned %d, want 5", n)
	}
	if w.Off() != 5 {
		t.Errorf("Off = %d, want 5", w.Off())
	}

	w.err = errors.New("test error")
	_, err = w.Write([]byte("test"))
	if err == nil {
		t.Error("Write should fail with error")
	}
}

func TestWriterU32(t *testing.T) {
	buf := &bytes.Buffer{}
	w := &writer{w: buf}

	w.U32(0x12345678)
	if !bytes.Equal(buf.Bytes(), []byte{0x12, 0x34, 0x56, 0x78}) {
		t.Errorf("U32 wrote %v, want [12 34 56 78]", buf.Bytes())
	}
}

func TestWriterU64(t *testing.T) {
	buf := &bytes.Buffer{}
	w := &writer{w: buf}

	w.U64(0x123456789ABCDEF0)
	expected := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("U64 wrote %v, want %v", buf.Bytes(), expected)
	}
}

func TestWriterVarint(t *testing.T) {
	buf := &bytes.Buffer{}
	w := &writer{w: buf}

	w.Varint(300)
	if buf.Len() == 0 {
		t.Error("Varint wrote nothing")
	}
}

func TestWriterString(t *testing.T) {
	buf := &bytes.Buffer{}
	w := &writer{w: buf}

	w.String("hello")
	result := buf.Bytes()
	if result[0] != 5 {
		t.Errorf("String length prefix = %d, want 5", result[0])
	}
	if !bytes.Equal(result[1:], []byte("hello")) {
		t.Errorf("String content = %v, want 'hello'", result[1:])
	}
}

func TestWriterB(t *testing.T) {
	buf := &bytes.Buffer{}
	w := &writer{w: buf}

	w.B(0x42)
	if buf.Bytes()[0] != 0x42 {
		t.Errorf("B wrote %v, want 0x42", buf.Bytes()[0])
	}
}

func TestWriterErrorPropagation(t *testing.T) {
	w := &writer{w: &errorWriter{}}

	w.Write([]byte("test"))
	if w.err == nil {
		t.Error("Write should propagate error")
	}
}

type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (int, error) {
	return 0, io.EOF
}

func TestSimpleSectionStartEnd(t *testing.T) {
	buf := &bytes.Buffer{}
	w := &writer{w: buf}

	sec := simpleSection{}
	sec.start(w)
	initialOff := sec.off

	w.Write([]byte("test"))

	sec.end(w)

	if sec.off != initialOff {
		t.Errorf("off = %d, want %d", sec.off, initialOff)
	}
	if sec.sz != 4 {
		t.Errorf("sz = %d, want 4", sec.sz)
	}
}

func TestCompoundSection(t *testing.T) {
	buf := &bytes.Buffer{}
	w := &writer{w: buf}

	sec := compoundSection{}
	sec.start(w)

	sec.addItem(w, []byte("item1"))
	sec.addItem(w, []byte("item2"))

	sec.end(w)

	if len(sec.offsets) != 2 {
		t.Errorf("offsets len = %d, want 2", len(sec.offsets))
	}
	if sec.data.sz == 0 {
		t.Error("data.sz should not be 0")
	}
}

func TestCompoundSectionRelativeIndex(t *testing.T) {
	sec := compoundSection{
		offsets: []uint32{100, 150, 220},
		data:    simpleSection{sz: 200},
	}

	ri := sec.relativeIndex()

	if len(ri) != 4 {
		t.Errorf("relativeIndex len = %d, want 4", len(ri))
	}
	if ri[0] != 0 {
		t.Errorf("ri[0] = %d, want 0", ri[0])
	}
	if ri[1] != 50 {
		t.Errorf("ri[1] = %d, want 50", ri[1])
	}
	if ri[3] != 200 {
		t.Errorf("ri[3] = %d, want 200", ri[3])
	}
}
