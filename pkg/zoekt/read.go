package zoekt

import (
	"encoding/binary"
	"fmt"
	"io"
)

type IndexFile interface {
	Read(off uint32, sz uint32) ([]byte, error)
	Size() (uint32, error)
	Close() error
	Name() string
}

type reader struct {
	r   IndexFile
	off uint32
}

func (r *reader) seek(off uint32) {
	r.off = off
}

func (r *reader) readBool() (bool, error) {
	b, err := r.readByte()
	if err != nil {
		return false, err
	}
	return b != 0, nil
}

func (r *reader) readByte() (byte, error) {
	data, err := r.r.Read(r.off, 1)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, io.EOF
	}
	r.off++
	return data[0], nil
}

func (r *reader) readU32() (uint32, error) {
	data, err := r.r.Read(r.off, 4)
	if err != nil {
		return 0, err
	}
	if len(data) < 4 {
		return 0, io.EOF
	}
	r.off += 4
	return binary.BigEndian.Uint32(data), nil
}

func (r *reader) readU64() (uint64, error) {
	data, err := r.r.Read(r.off, 8)
	if err != nil {
		return 0, err
	}
	if len(data) < 8 {
		return 0, io.EOF
	}
	r.off += 8
	return binary.BigEndian.Uint64(data), nil
}

func (r *reader) readString() (string, error) {
	n, err := r.readVarint()
	if err != nil {
		return "", err
	}
	data, err := r.r.Read(r.off, uint32(n))
	if err != nil {
		return "", err
	}
	r.off += uint32(n)
	return string(data), nil
}

func (r *reader) readVarintBytes() ([]byte, error) {
	var buf [binary.MaxVarintLen64]byte
	var n int
	for {
		b, err := r.readByte()
		if err != nil {
			if err == io.EOF {
				if n > 0 {
					return buf[:n], nil
				}
				return nil, io.EOF
			}
			return nil, err
		}
		buf[n] = b
		n++
		if b < 0x80 {
			break
		}
	}
	return buf[:n], nil
}

func (r *reader) readVarint() (uint64, error) {
	buf, err := r.readVarintBytes()
	if err != nil {
		return 0, err
	}
	n, size := binary.Uvarint(buf)
	if size <= 0 {
		return 0, fmt.Errorf("varint decode error")
	}
	return n, nil
}

func (r *reader) readSectionU32() ([]uint32, error) {
	size, err := r.readVarint()
	if err != nil {
		return nil, err
	}

	result := make([]uint32, 0, size)
	var last uint32
	for i := uint64(0); i < size; i++ {
		delta, err := r.readVarint()
		if err != nil {
			return nil, err
		}
		last += uint32(delta)
		result = append(result, last)
	}

	return result, nil
}

func (r *reader) readSectionBlob(s simpleSection) ([]byte, error) {
	if s.sz == 0 {
		return nil, nil
	}
	return r.r.Read(s.off, s.sz)
}

func OpenIndexFile(path string) (IndexFile, error) {
	return nil, fmt.Errorf("not implemented: use sqlitefs-backed index")
}

type ShardSearcher struct {
	path string
	data *indexData
}

func OpenShard(file IndexFile) (*ShardSearcher, error) {
	data, err := loadIndexData(file)
	if err != nil {
		return nil, fmt.Errorf("failed to load index data: %w", err)
	}
	return &ShardSearcher{
		data: data,
	}, nil
}

func (s *ShardSearcher) Search(query Query, opts *SearchOptions) (*SearchResult, error) {
	if s.data == nil {
		return nil, fmt.Errorf("shard not loaded")
	}

	return s.data.Search(query, opts)
}

func (s *ShardSearcher) Close() error {
	if s.data != nil {
		s.data.Close()
	}
	return nil
}

type IndexFileReader struct {
	name string
	data []byte
}

func NewIndexFile(data []byte, name string) IndexFile {
	return &IndexFileReader{name: name, data: data}
}

func (f *IndexFileReader) Read(off uint32, sz uint32) ([]byte, error) {
	if off+sz > uint32(len(f.data)) {
		return nil, io.EOF
	}
	return f.data[off : off+sz], nil
}

func (f *IndexFileReader) Size() (uint32, error) {
	return uint32(len(f.data)), nil
}

func (f *IndexFileReader) Close() error {
	return nil
}

func (f *IndexFileReader) Name() string {
	return f.name
}
