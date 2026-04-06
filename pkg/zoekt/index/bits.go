package index

import (
	"encoding/binary"
)

const ngramSize = 3

type ngram uint64

func runesToNGram(b [ngramSize]rune) ngram {
	return ngram(uint64(b[0])<<42 | uint64(b[1])<<21 | uint64(b[2]))
}

func bytesToNGram(b []byte) ngram {
	return runesToNGram([ngramSize]rune{rune(b[0]), rune(b[1]), rune(b[2])})
}

func stringToNGram(s string) ngram {
	if len(s) < 3 {
		return 0
	}
	return bytesToNGram([]byte(s))
}

func ngramToBytes(n ngram) []byte {
	rs := ngramToRunes(n)
	return []byte{byte(rs[0]), byte(rs[1]), byte(rs[2])}
}

const runeMask = 1<<21 - 1

func ngramToRunes(n ngram) [ngramSize]rune {
	return [ngramSize]rune{rune((n >> 42) & runeMask), rune((n >> 21) & runeMask), rune(n & runeMask)}
}

func (n ngram) String() string {
	rs := ngramToRunes(n)
	return string(rs[:])
}

func toSizedDeltas(offsets []uint32) []byte {
	var enc [8]byte

	deltas := make([]byte, 0, len(offsets)*2)

	m := binary.PutUvarint(enc[:], uint64(len(offsets)))
	deltas = append(deltas, enc[:m]...)

	var last uint32
	for _, p := range offsets {
		delta := p - last
		last = p

		m := binary.PutUvarint(enc[:], uint64(delta))
		deltas = append(deltas, enc[:m]...)
	}
	return deltas
}

func fromSizedDeltas(data []byte, ps []uint32) []uint32 {
	sz, m := binary.Uvarint(data)
	data = data[m:]

	if cap(ps) < int(sz) {
		ps = make([]uint32, 0, sz)
	} else {
		ps = ps[:0]
	}

	var last uint32
	for len(data) > 0 {
		delta, m := binary.Uvarint(data)
		offset := last + uint32(delta)
		last = offset
		data = data[m:]
		ps = append(ps, offset)
	}
	return ps
}

func toSizedDeltas16(offsets []uint16) []byte {
	var enc [8]byte

	deltas := make([]byte, 0, len(offsets)*2)

	m := binary.PutUvarint(enc[:], uint64(len(offsets)))
	deltas = append(deltas, enc[:m]...)

	var last uint16
	for _, p := range offsets {
		delta := p - last
		last = p

		m := binary.PutUvarint(enc[:], uint64(delta))
		deltas = append(deltas, enc[:m]...)
	}
	return deltas
}

func fromSizedDeltas16(data []byte, ps []uint16) []uint16 {
	sz, m := binary.Uvarint(data)
	data = data[m:]

	if cap(ps) < int(sz) {
		ps = make([]uint16, 0, sz)
	} else {
		ps = ps[:0]
	}

	var last uint16
	for len(data) > 0 {
		delta, m := binary.Uvarint(data)
		offset := last + uint16(delta)
		last = offset
		data = data[m:]
		ps = append(ps, offset)
	}
	return ps
}

func marshalDocSections(secs []DocumentSection) []byte {
	ints := make([]uint32, 0, len(secs)*2)
	for _, s := range secs {
		ints = append(ints, uint32(s.Start), uint32(s.End))
	}

	return toSizedDeltas(ints)
}

func unmarshalDocSections(data []byte, ds []DocumentSection) []DocumentSection {
	sz, m := binary.Uvarint(data)
	data = data[m:]

	if cap(ds) < int(sz)/2 {
		ds = make([]DocumentSection, 0, sz/2)
	} else {
		ds = ds[:0]
	}

	var last uint32
	for len(data) > 0 {
		var d DocumentSection

		delta, m := binary.Uvarint(data)
		last += uint32(delta)
		data = data[m:]
		d.Start = last

		delta, m = binary.Uvarint(data)
		last += uint32(delta)
		data = data[m:]
		d.End = last

		ds = append(ds, d)
	}
	return ds
}

func newLinesIndices(in []byte) []uint32 {
	out := make([]uint32, 0, countNewlines(in))
	for i, c := range in {
		if c == '\n' {
			out = append(out, uint32(i))
		}
	}
	return out
}

func countNewlines(in []byte) int {
	n := 0
	for _, c := range in {
		if c == '\n' {
			n++
		}
	}
	return n
}
