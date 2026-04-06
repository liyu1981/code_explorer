package index

import (
	"encoding/binary"
	"fmt"
	"sort"
)

const ngramEncoding = 8
const btreeBucketSize = (4096 * 2) / ngramEncoding

const (
	interfaceBytes uint64 = 16
	pointerSize    uint64 = 8
)

type btree struct {
	root node
	opts btreeOpts

	lastBucketIndex int
}

type btreeOpts struct {
	bucketSize int
	v          int
}

func newBtree(opts btreeOpts) *btree {
	return &btree{
		root: &leaf{},
		opts: opts,
	}
}

func (bt *btree) insert(ng ngram) {
	if leftNode, rightNode, newKey, ok := bt.root.maybeSplit(bt.opts); ok {
		bt.root = &innerNode{keys: []ngram{newKey}, children: []node{leftNode, rightNode}}
	}
	bt.root.insert(ng, bt.opts)
}

func (bt *btree) find(ng ngram) (int, int) {
	if bt.root == nil {
		return -1, -1
	}
	return bt.root.find(ng)
}

func (bt *btree) visit(f func(n node)) {
	bt.root.visit(f)
}

func (bt *btree) freeze() {
	offset, bucketIndex := 0, 0
	bt.visit(func(no node) {
		switch n := no.(type) {
		case *leaf:
			n.bucketIndex = bucketIndex
			bucketIndex++

			n.postingIndexOffset = offset
			offset += n.bucketSize
		case *innerNode:
			return
		}
	})

	bt.lastBucketIndex = bucketIndex - 1
}

func (bt *btree) sizeBytes() int {
	sz := 2 * 8
	sz += int(interfaceBytes)

	bt.visit(func(n node) {
		sz += n.sizeBytes()
	})

	return sz
}

type node interface {
	insert(ng ngram, opts btreeOpts)
	maybeSplit(opts btreeOpts) (left node, right node, newKey ngram, ok bool)
	find(ng ngram) (int, int)
	visit(func(n node))
	sizeBytes() int
}

type innerNode struct {
	keys     []ngram
	children []node
}

type leaf struct {
	bucketIndex        int
	postingIndexOffset int
	bucketSize         int
	splitKey           ngram
}

func (n *innerNode) sizeBytes() int {
	return len(n.keys)*ngramEncoding + len(n.children)*int(interfaceBytes)
}

func (n *leaf) sizeBytes() int {
	return 4 * 8
}

func (n *leaf) insert(ng ngram, opts btreeOpts) {
	n.bucketSize++

	if n.bucketSize == (opts.bucketSize/2)+1 {
		n.splitKey = ng
	}
}

func (n *innerNode) insert(ng ngram, opts btreeOpts) {
	insertAt := func(i int) {
		if leftNode, rightNode, newKey, ok := n.children[i].maybeSplit(opts); ok {
			n.keys = append(n.keys[0:i], append([]ngram{newKey}, n.keys[i:]...)...)
			n.children = append(n.children[0:i], append([]node{leftNode, rightNode}, n.children[i+1:]...)...)

			if ng >= n.keys[i] {
				i++
			}
		}
		n.children[i].insert(ng, opts)
	}

	for i, k := range n.keys {
		if ng < k {
			insertAt(i)
			return
		}
	}
	insertAt(len(n.children) - 1)
}

func (n *innerNode) find(ng ngram) (int, int) {
	for i, k := range n.keys {
		if ng < k {
			return n.children[i].find(ng)
		}
	}
	return n.children[len(n.children)-1].find(ng)
}

func (n *leaf) find(ng ngram) (int, int) {
	return int(n.bucketIndex), int(n.postingIndexOffset)
}

func (n *leaf) maybeSplit(opts btreeOpts) (left node, right node, newKey ngram, ok bool) {
	if n.bucketSize < opts.bucketSize {
		return
	}
	return &leaf{bucketSize: opts.bucketSize / 2},
		&leaf{bucketSize: opts.bucketSize / 2},
		n.splitKey,
		true
}

func (n *innerNode) maybeSplit(opts btreeOpts) (left node, right node, newKey ngram, ok bool) {
	if len(n.children) < 2*opts.v {
		return
	}
	return &innerNode{
			keys:     append(make([]ngram, 0, opts.v-1), n.keys[0:opts.v-1]...),
			children: append(make([]node, 0, opts.v), n.children[:opts.v]...),
		},
		&innerNode{
			keys:     append(make([]ngram, 0, (2*opts.v)-1), n.keys[opts.v:]...),
			children: append(make([]node, 0, 2*opts.v), n.children[opts.v:]...),
		},
		n.keys[opts.v-1],
		true
}

func (n *leaf) visit(f func(n node)) {
	f(n)
}

func (n *innerNode) visit(f func(n node)) {
	f(n)
	for _, child := range n.children {
		child.visit(f)
	}
}

func (bt *btree) String() string {
	s := ""
	s += fmt.Sprintf("%+v", bt.opts)
	bt.root.visit(func(n node) {
		switch nd := n.(type) {
		case *leaf:
			return
		case *innerNode:
			s += fmt.Sprintf("[")
			for _, key := range nd.keys {
				s += fmt.Sprintf("%d,", key)
			}
			s = s[:len(s)-1]
			s += fmt.Sprintf("]")
		}
	})
	return s
}

type btreeIndex struct {
	bt *btree

	file IndexFile

	ngramSec simpleSection

	postingIndex simpleSection
}

func (b btreeIndex) SizeBytes() (sz int) {
	if b.bt != nil {
		sz += int(pointerSize) + b.bt.sizeBytes()
	}
	sz += 8
	sz += 8
	sz += 4
	return
}

func (b btreeIndex) Get(ng ngram) simpleSection {
	if b.bt == nil {
		return simpleSection{}
	}

	bucketIndex, postingIndexOffset := b.bt.find(ng)

	off, sz := b.getBucket(bucketIndex)
	bucket, err := b.file.Read(off, sz)
	if err != nil {
		return simpleSection{}
	}

	getNGram := func(i int) ngram {
		i *= ngramEncoding
		return ngram(binary.BigEndian.Uint64(bucket[i : i+ngramEncoding]))
	}

	bucketSize := len(bucket) / ngramEncoding
	x := sort.Search(bucketSize, func(i int) bool {
		return ng <= getNGram(i)
	})

	if x >= bucketSize || getNGram(x) != ng {
		return simpleSection{}
	}

	return b.getPostingList(postingIndexOffset + x)
}

func (b btreeIndex) getPostingList(ngramIndex int) simpleSection {
	relativeOffsetBytes := uint32(ngramIndex) * 4

	if relativeOffsetBytes+8 <= b.postingIndex.sz {
		o, err := b.file.Read(b.postingIndex.off+relativeOffsetBytes, 8)
		if err != nil {
			return simpleSection{}
		}

		start := binary.BigEndian.Uint32(o[0:4])
		end := binary.BigEndian.Uint32(o[4:8])
		return simpleSection{
			off: start,
			sz:  end - start,
		}
	} else {
		o, err := b.file.Read(b.postingIndex.off+relativeOffsetBytes, 4)
		if err != nil {
			return simpleSection{}
		}

		start := binary.BigEndian.Uint32(o[0:4])
		return simpleSection{
			off: start,
			sz:  b.postingIndex.off - start,
		}
	}
}

func (b btreeIndex) getBucket(bucketIndex int) (off uint32, sz uint32) {
	sz = uint32(b.bt.opts.bucketSize / 2 * ngramEncoding)
	off = b.ngramSec.off + uint32(bucketIndex)*sz

	if bucketIndex == b.bt.lastBucketIndex {
		sz = b.ngramSec.off + b.ngramSec.sz - off
	}

	return
}
