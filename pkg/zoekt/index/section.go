package index

import (
	"encoding/binary"
	"io"
)

type writer struct {
	err error
	w   io.Writer
	off uint32
}

func (w *writer) Write(b []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}

	var n int
	n, w.err = w.w.Write(b)
	w.off += uint32(n)
	return n, w.err
}

func (w *writer) Off() uint32 { return w.off }

func (w *writer) B(b byte) {
	s := []byte{b}
	w.Write(s)
}

func (w *writer) U32(n uint32) {
	var enc [4]byte
	binary.BigEndian.PutUint32(enc[:], n)
	w.Write(enc[:])
}

func (w *writer) U64(n uint64) {
	var enc [8]byte
	binary.BigEndian.PutUint64(enc[:], n)
	w.Write(enc[:])
}

func (w *writer) Varint(n uint32) {
	var enc [8]byte
	m := binary.PutUvarint(enc[:], uint64(n))
	w.Write(enc[:m])
}

func (w *writer) String(s string) {
	b := []byte(s)
	w.Varint(uint32(len(b)))
	w.Write(b)
}

type simpleSection struct {
	off uint32
	sz  uint32
}

func (s *simpleSection) start(w *writer) {
	s.off = w.Off()
}

func (s *simpleSection) end(w *writer) {
	s.sz = w.Off() - s.off
}

func (s *simpleSection) write(w *writer) {
	w.U32(s.off)
	w.U32(s.sz)
}

func (s *simpleSection) read(r *reader) error {
	var err error
	s.off, err = r.readU32()
	if err != nil {
		return err
	}
	s.sz, err = r.readU32()
	return err
}

func (s *simpleSection) skip(r *reader) error {
	_, err := r.readU32()
	if err != nil {
		return err
	}
	_, err = r.readU32()
	return err
}

type compoundSection struct {
	data    simpleSection
	offsets []uint32
	index   simpleSection
}

func (s *compoundSection) start(w *writer) {
	s.data.start(w)
}

func (s *compoundSection) end(w *writer) {
	s.data.end(w)
	s.index.start(w)
	for _, o := range s.offsets {
		w.U32(o)
	}
	s.index.end(w)
}

func (s *compoundSection) addItem(w *writer, item []byte) {
	s.offsets = append(s.offsets, w.Off())
	w.Write(item)
}

func (s *compoundSection) write(w *writer) {
	s.data.write(w)
	s.index.write(w)
}

func (s *compoundSection) read(r *reader) error {
	if err := s.data.read(r); err != nil {
		return err
	}
	if err := s.index.read(r); err != nil {
		return err
	}
	offsets, err := readSectionU32(r.r, s.index)
	if err != nil {
		return err
	}
	s.offsets = offsets
	return nil
}

func (s *compoundSection) skip(r *reader) error {
	if _, err := r.readU32(); err != nil {
		return err
	}
	if _, err := r.readU32(); err != nil {
		return err
	}
	return nil
}

func (s *compoundSection) relativeIndex() []uint32 {
	ri := make([]uint32, 0, len(s.offsets)+1)
	for _, o := range s.offsets {
		ri = append(ri, o-s.offsets[0])
	}
	if len(s.offsets) > 0 {
		ri = append(ri, s.data.sz)
	}
	return ri
}

type lazyCompoundSection struct {
	compoundSection
}

type section interface {
	read(*reader) error
	skip(*reader) error
	write(*writer)
	kind() sectionKind
}

type sectionKind int

const (
	sectionKindSimple       sectionKind = 0
	sectionKindCompound     sectionKind = 1
	sectionKindCompoundLazy sectionKind = 2
)

func (s *simpleSection) kind() sectionKind {
	return sectionKindSimple
}

func (s *compoundSection) kind() sectionKind {
	return sectionKindCompound
}

func (s *lazyCompoundSection) kind() sectionKind {
	return sectionKindCompoundLazy
}

type taggedSection struct {
	tag string
	sec section
}
