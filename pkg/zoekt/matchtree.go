package zoekt

import (
	"sort"
)

type matchTree interface {
	nextDoc() uint32
}

type allMatchTree struct{}

func (m *allMatchTree) nextDoc() uint32 {
	return 0
}

type noMatchTree struct{}

func (m *noMatchTree) nextDoc() uint32 {
	return ^uint32(0)
}

type andMatchTree struct {
	children  []matchTree
	childDocs []uint32
	lastDoc   uint32
	started   bool
}

func (m *andMatchTree) nextDoc() uint32 {
	if m.started && m.lastDoc == ^uint32(0) {
		return ^uint32(0)
	}

	if !m.started {
		m.childDocs = make([]uint32, len(m.children))
		for i := range m.childDocs {
			m.childDocs[i] = m.children[i].nextDoc()
		}
		m.started = true
	}

	for {
		var maxDoc uint32 = 0
		allMatch := true
		for _, doc := range m.childDocs {
			if doc == ^uint32(0) {
				m.lastDoc = ^uint32(0)
				return ^uint32(0)
			}
			if doc > maxDoc {
				maxDoc = doc
				allMatch = false
			}
		}

		if allMatch {
			m.lastDoc = maxDoc
			// Advance all children for next call
			for i := range m.childDocs {
				m.childDocs[i] = m.children[i].nextDoc()
			}
			return maxDoc
		}

		// Not all match.
		// Advance all children that are behind maxDoc.
		allMatch = true
		for i, doc := range m.childDocs {
			for doc != ^uint32(0) && doc < maxDoc {
				doc = m.children[i].nextDoc()
				m.childDocs[i] = doc
			}
			if doc == ^uint32(0) {
				m.lastDoc = ^uint32(0)
				return ^uint32(0)
			}
			if doc > maxDoc {
				maxDoc = doc
				allMatch = false
			}
		}

		if allMatch {
			m.lastDoc = maxDoc
			// Advance all children for next call
			for i := range m.childDocs {
				m.childDocs[i] = m.children[i].nextDoc()
			}
			return maxDoc
		}
	}
}

type orMatchTree struct {
	children []matchTree
	docs     []uint32
	idx      int
}

func (m *orMatchTree) nextDoc() uint32 {
	if m.docs == nil {
		// Collect all docs and sort them
		docMap := make(map[uint32]struct{})
		for _, c := range m.children {
			for {
				doc := c.nextDoc()
				if doc == ^uint32(0) {
					break
				}
				docMap[doc] = struct{}{}
			}
		}
		m.docs = make([]uint32, 0, len(docMap))
		for doc := range docMap {
			m.docs = append(m.docs, doc)
		}
		sort.Slice(m.docs, func(i, j int) bool { return m.docs[i] < m.docs[j] })
	}

	if m.idx < len(m.docs) {
		doc := m.docs[m.idx]
		m.idx++
		return doc
	}
	return ^uint32(0)
}

type notMatchTree struct {
	child matchTree
}

func (m *notMatchTree) nextDoc() uint32 {
	return ^uint32(0)
}

type branchMaskMatchTree struct {
	masks     []uint64
	mask      uint64
	branch    string
	branchIDs []map[string]uint
	idx       uint32
}

func (m *branchMaskMatchTree) nextDoc() uint32 {
	for ; m.idx < uint32(len(m.masks)); m.idx++ {
		if m.masks[m.idx]&m.mask != 0 {
			doc := m.idx
			m.idx++
			return doc
		}
	}
	return ^uint32(0)
}

type substringMatchTree struct {
	ids           []uint32
	ends          []uint32
	pattern       []byte
	lowerPattern  []byte
	caseSensitive bool
	fileName      bool
	d             *indexData
	idx           int
	lastDoc       int
}

func (m *substringMatchTree) nextDoc() uint32 {
	for ; m.idx < len(m.ids); m.idx++ {
		docID := m.ids[m.idx]
		if int(docID) > m.lastDoc {
			m.lastDoc = int(docID)
			m.idx++
			return docID
		}
	}
	return ^uint32(0)
}

type contentProvider struct {
	id    *indexData
	stats *SearchStats
}

func (cp *contentProvider) scoreFile(doc uint32, mt matchTree, opts *SearchOptions) *FileMatch {
	content, err := cp.id.readContents(doc)
	if err != nil {
		return nil
	}

	var matches []LineMatch

	// If it's a substring match tree, we can find the exact line matches.
	// For other match trees (like branch or repo), we might not have specific line matches
	// unless we search for the original query in the content.
	if smt, ok := mt.(*substringMatchTree); ok {
		searchIn := content
		if smt.fileName {
			searchIn = cp.id.fileName(doc)
		}
		matches = cp.searchContent(searchIn, smt.pattern, smt.lowerPattern, smt.caseSensitive)

		if len(matches) == 0 && !smt.fileName {
			return nil
		}
	}

	return &FileMatch{
		Content:     string(content),
		LineMatches: matches,
		Score:       1.0,
	}
}

func (cp *contentProvider) searchContent(content, pattern, lowerPattern []byte, caseSensitive bool) []LineMatch {
	var matches []LineMatch

	searchIn := content
	if !caseSensitive {
		searchIn = toLower(content)
		pattern = lowerPattern
	}

	for i := 0; i <= len(searchIn)-len(pattern); i++ {
		found := true
		for j := range pattern {
			if searchIn[i+j] != pattern[j] {
				found = false
				break
			}
		}
		if found {
			lineStart := i
			for lineStart > 0 && content[lineStart-1] != '\n' {
				lineStart--
			}
			lineEnd := i
			for lineEnd < len(content) && content[lineEnd] != '\n' {
				lineEnd++
			}

			line := string(content[lineStart:lineEnd])
			lineNumber := 1
			for _, b := range content[:lineStart] {
				if b == '\n' {
					lineNumber++
				}
			}

			matches = append(matches, LineMatch{
				Line:       line,
				LineNumber: lineNumber,
				LineStart:  lineStart,
				LineEnd:    lineEnd,
			})
			i += len(pattern) - 1
		}
	}

	return matches
}

type Const struct {
	Value bool
}

func (c *Const) String() string {
	if c.Value {
		return "true"
	}
	return "false"
}

type ngramOff struct {
	ngram ngram
	index int
}

func splitNGrams(data []byte) []ngramOff {
	if len(data) < 3 {
		return nil
	}

	var res []ngramOff
	for i := 0; i <= len(data)-3; i++ {
		// Use the same encoding as runesToNGram: r0<<42 | r1<<21 | r2
		ng := ngram(data[i])<<42 | ngram(data[i+1])<<21 | ngram(data[i+2])
		res = append(res, ngramOff{ngram: ng, index: i})
	}
	return res
}

func generateCaseNgrams(ng ngram) []ngram {
	var res []ngram
	for i := 0; i < 3; i++ {
		r := rune((ng >> (21 * (2 - i))) & 0x7F)
		if r >= 'A' && r <= 'Z' {
			lower := ngram(rune(r + ('a' - 'A')))
			variant := ng&^(ngram(0x7F)<<(21*(2-i))) | (lower << (21 * (2 - i)))
			res = append(res, variant)
		}
	}
	if len(res) == 0 {
		res = append(res, ng)
	}
	return res
}

func findSelectiveNgrams(offs []ngramOff, indexMap, frequencies []uint32) (first, last ngramOff) {
	if len(offs) == 0 {
		return
	}
	first = offs[0]
	last = offs[len(offs)-1]
	minIdx := 0
	maxIdx := 0
	for i, f := range frequencies {
		if f < frequencies[minIdx] {
			minIdx = i
		}
		if f > frequencies[maxIdx] {
			maxIdx = i
		}
	}
	first = offs[indexMap[minIdx]]
	last = offs[indexMap[maxIdx]]
	if first.index > last.index {
		first, last = last, first
	}
	return
}

type noMatchIterator struct{}

func (i *noMatchIterator) nextDoc() uint32 {
	return ^uint32(0)
}

func (i *noMatchIterator) nextMatch() []byte {
	return nil
}

func intersectUint32(a, b []uint32) []uint32 {
	m := make(map[uint32]struct{})
	for _, v := range a {
		m[v] = struct{}{}
	}
	var res []uint32
	for _, v := range b {
		if _, ok := m[v]; ok {
			res = append(res, v)
		}
	}
	return res
}

func decodePostingList(data []byte) []uint32 {
	if len(data) == 0 {
		return nil
	}

	var res []uint32
	var last uint32
	i := 0
	for i < len(data) {
		var delta uint64
		n := 0
		for {
			if i+n >= len(data) {
				break
			}
			b := data[i+n]
			delta = delta | uint64(b&0x7F)<<(7*n)
			n++
			if b&0x80 == 0 {
				break
			}
		}
		i += n
		last += uint32(delta)
		res = append(res, last)
	}
	return res
}

func toLower(b []byte) []byte {
	res := make([]byte, len(b))
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			res[i] = c + ('a' - 'A')
		} else {
			res[i] = c
		}
	}
	return res
}
