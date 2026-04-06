package index

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc64"
	"sort"
	"time"
	"unicode/utf8"
)

type searchableString struct {
	data []byte
}

const runeOffsetFrequency = 100

type postingList struct {
	data    []byte
	lastOff uint32
}

const asciiNgramBits = 21

func asciiNgramIndex(a, b, c byte) uint32 {
	return uint32(a)<<14 | uint32(b)<<7 | uint32(c)
}

func asciiIndexToNgram(idx uint32) ngram {
	r0 := uint64(idx >> 14)
	r1 := uint64((idx >> 7) & 0x7f)
	r2 := uint64(idx & 0x7f)
	return ngram(r0<<42 | r1<<21 | r2)
}

type postingsBuilder struct {
	asciiPostings [1 << asciiNgramBits]*postingList
	postings      map[ngram]*postingList

	asciiPopulated []uint32

	runeOffsets []uint32
	runeCount   uint32

	isPlainASCII bool

	endRunes []uint32
	endByte  uint32
}

const initialPostingCap = 64

func estimateNgrams(shardMaxBytes int) int {
	n := shardMaxBytes / 600
	if n < 1024 {
		n = 1024
	}
	return n
}

func newPostingsBuilder(shardMaxBytes int) *postingsBuilder {
	return &postingsBuilder{
		postings:     make(map[ngram]*postingList, estimateNgrams(shardMaxBytes)),
		isPlainASCII: true,
	}
}

func (s *postingsBuilder) reset() {
	for _, idx := range s.asciiPopulated {
		pl := s.asciiPostings[idx]
		pl.data = pl.data[:0]
		pl.lastOff = 0
	}
	s.asciiPopulated = s.asciiPopulated[:0]
	for _, pl := range s.postings {
		pl.data = pl.data[:0]
		pl.lastOff = 0
	}
	s.runeOffsets = s.runeOffsets[:0]
	s.runeCount = 0
	s.isPlainASCII = true
	s.endRunes = s.endRunes[:0]
	s.endByte = 0
}

func (s *postingsBuilder) newSearchableString(data []byte, byteSections []DocumentSection) (*searchableString, []DocumentSection, error) {
	dest := searchableString{
		data: data,
	}
	var buf [8]byte
	var runeGram [3]rune

	var runeIndex uint32
	byteCount := 0
	dataSz := uint32(len(data))

	byteSectionBoundaries := make([]uint32, 0, 2*len(byteSections))
	for _, sec := range byteSections {
		byteSectionBoundaries = append(byteSectionBoundaries, sec.Start, sec.End)
	}
	var runeSectionBoundaries []uint32

	endRune := s.runeCount
	for ; len(data) > 0; runeIndex++ {
		var c rune
		sz := 1
		if data[0] < utf8.RuneSelf {
			c = rune(data[0])
		} else {
			c, sz = utf8.DecodeRune(data)
			s.isPlainASCII = false
		}
		data = data[sz:]

		runeGram[0], runeGram[1], runeGram[2] = runeGram[1], runeGram[2], c

		if idx := s.runeCount + runeIndex; idx%runeOffsetFrequency == 0 {
			s.runeOffsets = append(s.runeOffsets, s.endByte+uint32(byteCount))
		}
		for len(byteSectionBoundaries) > 0 && byteSectionBoundaries[0] == uint32(byteCount) {
			runeSectionBoundaries = append(runeSectionBoundaries,
				endRune+uint32(runeIndex))
			byteSectionBoundaries = byteSectionBoundaries[1:]
		}

		byteCount += sz

		if runeIndex < 2 {
			continue
		}

		newOff := endRune + uint32(runeIndex) - 2

		var pl *postingList
		if runeGram[0] < utf8.RuneSelf && runeGram[1] < utf8.RuneSelf && runeGram[2] < utf8.RuneSelf {
			idx := asciiNgramIndex(byte(runeGram[0]), byte(runeGram[1]), byte(runeGram[2]))
			pl = s.asciiPostings[idx]
			if pl == nil {
				pl = &postingList{data: make([]byte, 0, initialPostingCap)}
				s.asciiPostings[idx] = pl
				s.asciiPopulated = append(s.asciiPopulated, idx)
			} else if len(pl.data) == 0 {
				s.asciiPopulated = append(s.asciiPopulated, idx)
			}
		} else {
			ng := runesToNGram(runeGram)
			pl = s.postings[ng]
			if pl == nil {
				pl = &postingList{data: make([]byte, 0, initialPostingCap)}
				s.postings[ng] = pl
			}
		}
		m := binary.PutUvarint(buf[:], uint64(newOff-pl.lastOff))
		pl.data = append(pl.data, buf[:m]...)
		pl.lastOff = newOff
	}
	s.runeCount += runeIndex

	for len(byteSectionBoundaries) > 0 && byteSectionBoundaries[0] < uint32(byteCount) {
		return nil, nil, fmt.Errorf("no rune for section boundary at byte %d", byteSectionBoundaries[0])
	}

	for len(byteSectionBoundaries) > 0 && byteSectionBoundaries[0] == uint32(byteCount) {
		runeSectionBoundaries = append(runeSectionBoundaries,
			endRune+runeIndex)
		byteSectionBoundaries = byteSectionBoundaries[1:]
	}
	runeSecs := make([]DocumentSection, 0, len(byteSections))
	for i := 0; i < len(runeSectionBoundaries); i += 2 {
		runeSecs = append(runeSecs, DocumentSection{
			Start: runeSectionBoundaries[i],
			End:   runeSectionBoundaries[i+1],
		})
	}

	s.endRunes = append(s.endRunes, s.runeCount)
	s.endByte += dataSz
	return &dest, runeSecs, nil
}

type ShardBuilder struct {
	indexFormatVersion int
	featureVersion     int

	contentStrings  []*searchableString
	nameStrings     []*searchableString
	docSections     [][]DocumentSection
	runeDocSections []DocumentSection

	symID        uint32
	symIndex     map[string]uint32
	symKindID    uint32
	symKindIndex map[string]uint32
	symMetaData  []uint32

	checksums []byte

	branchMasks []uint64
	subRepos    []uint32

	repos []uint16

	contentPostings *postingsBuilder
	namePostings    *postingsBuilder

	repoList []Repository

	subRepoIndices []map[string]uint32

	languageMap map[string]uint16

	languages []uint8

	categories []byte

	fileEndSymbol []uint32

	IndexTime time.Time
	ID        string
}

const defaultShardMax = 100 << 20

func newShardBuilder(shardMax int) *ShardBuilder {
	if shardMax <= 0 {
		shardMax = defaultShardMax
	}
	return newShardBuilderWithPostings(
		newPostingsBuilder(shardMax),
		newPostingsBuilder(shardMax),
	)
}

func newShardBuilderWithPostings(content, name *postingsBuilder) *ShardBuilder {
	return &ShardBuilder{
		indexFormatVersion: IndexFormatVersion,
		featureVersion:     FeatureVersion,
		contentPostings:    content,
		namePostings:       name,
		fileEndSymbol:      []uint32{0},
		symIndex:           make(map[string]uint32),
		symKindIndex:       make(map[string]uint32),
		languageMap:        make(map[string]uint16),
	}
}

func (b *ShardBuilder) ContentSize() uint32 {
	return b.contentPostings.endByte + b.namePostings.endByte
}

func (b *ShardBuilder) NumFiles() int {
	return len(b.contentStrings)
}

func (b *ShardBuilder) Add(doc Document) error {
	if index := bytes.IndexByte(doc.Content, 0); index > 0 {
		doc.SkipReason = SkipReasonBinary
	}

	if doc.SkipReason != SkipReasonNone {
		doc.Content = []byte("NOT-INDEXED: " + doc.SkipReason.explanation())
		doc.Symbols = nil
		doc.SymbolsMetaData = nil
	}

	sort.Sort(symbolSlice{doc.Symbols, doc.SymbolsMetaData})
	var last DocumentSection
	for i, s := range doc.Symbols {
		if i > 0 {
			if last.End > s.Start {
				return fmt.Errorf("sections overlap")
			}
		}
		last = s
	}
	if last.End > uint32(len(doc.Content)) {
		return fmt.Errorf("section goes past end of content")
	}

	docStr, runeSecs, err := b.contentPostings.newSearchableString(doc.Content, doc.Symbols)
	if err != nil {
		return err
	}
	nameStr, _, err := b.namePostings.newSearchableString([]byte(doc.Name), nil)
	if err != nil {
		return err
	}
	b.addSymbols(doc.SymbolsMetaData)

	repoIdx := 0
	if len(b.repoList) > 0 {
		subRepoIdx, ok := b.subRepoIndices[repoIdx][doc.SubRepositoryPath]
		if !ok {
			return fmt.Errorf("unknown subrepo path %q", doc.SubRepositoryPath)
		}
		b.subRepos = append(b.subRepos, subRepoIdx)
	} else {
		b.subRepos = append(b.subRepos, 0)
	}

	var mask uint64
	for _, br := range doc.Branches {
		m := b.branchMask(br)
		if m == 0 {
			return fmt.Errorf("no branch found for %s", br)
		}
		mask |= m
	}

	b.repos = append(b.repos, uint16(repoIdx))

	hasher := crc64.New(crc64.MakeTable(crc64.ISO))
	hasher.Write(doc.Content)

	b.contentStrings = append(b.contentStrings, docStr)
	b.runeDocSections = append(b.runeDocSections, runeSecs...)

	b.nameStrings = append(b.nameStrings, nameStr)
	b.docSections = append(b.docSections, doc.Symbols)
	b.fileEndSymbol = append(b.fileEndSymbol, uint32(len(b.runeDocSections)))
	b.branchMasks = append(b.branchMasks, mask)
	b.checksums = append(b.checksums, hasher.Sum(nil)...)

	langCode, ok := b.languageMap[doc.Language]
	if !ok {
		if len(b.languageMap) >= 65535 {
			return fmt.Errorf("too many languages")
		}
		langCode = uint16(len(b.languageMap))
		b.languageMap[doc.Language] = langCode
	}
	b.languages = append(b.languages, uint8(langCode), uint8(langCode>>8))

	b.categories = append(b.categories, byte(len(doc.Language)%256))

	return nil
}

func (b *ShardBuilder) branchMask(br string) uint64 {
	if len(b.repoList) == 0 {
		return 1
	}
	for i, brData := range b.repoList[len(b.repoList)-1].Branches {
		if brData.Name == br {
			return uint64(1) << uint(i)
		}
	}
	return 0
}

func (b *ShardBuilder) addSymbols(symbols []*Symbol) {
	for _, sym := range symbols {
		b.symMetaData = append(b.symMetaData,
			0,
			b.symbolKindID(sym.Kind),
			b.symbolID(sym.Parent),
			b.symbolKindID(sym.ParentKind))
	}
}

func (b *ShardBuilder) symbolID(sym string) uint32 {
	if _, ok := b.symIndex[sym]; !ok {
		b.symIndex[sym] = b.symID
		b.symID++
	}
	return b.symIndex[sym]
}

func (b *ShardBuilder) symbolKindID(t string) uint32 {
	if _, ok := b.symKindIndex[t]; !ok {
		b.symKindIndex[t] = b.symKindID
		b.symKindID++
	}
	return b.symKindIndex[t]
}

func (b *ShardBuilder) setRepository(desc *Repository) error {
	if len(desc.Branches) > 64 {
		return fmt.Errorf("too many branches")
	}

	repo := *desc

	repo.SubRepoMap = map[string]*Repository{}
	for k, v := range desc.SubRepoMap {
		if k != "" {
			repo.SubRepoMap[k] = v
		}
	}

	b.repoList = append(b.repoList, repo)

	return b.populateSubRepoIndices()
}

func (b *ShardBuilder) populateSubRepoIndices() error {
	if len(b.subRepoIndices) == len(b.repoList) {
		return nil
	}
	if len(b.subRepoIndices) != len(b.repoList)-1 {
		return fmt.Errorf("populateSubRepoIndices not called for a repo: %d != %d - 1", len(b.subRepoIndices), len(b.repoList))
	}
	repo := b.repoList[len(b.repoList)-1]
	b.subRepoIndices = append(b.subRepoIndices, mkSubRepoIndices(repo))
	return nil
}

func mkSubRepoIndices(repo Repository) map[string]uint32 {
	paths := []string{""}
	for k := range repo.SubRepoMap {
		paths = append(paths, k)
	}
	sort.Strings(paths)
	subRepoIndices := make(map[string]uint32, len(paths))
	for i, p := range paths {
		subRepoIndices[p] = uint32(i)
	}
	return subRepoIndices
}

type symbolSlice struct {
	symbols  []DocumentSection
	metaData []*Symbol
}

func (s symbolSlice) Len() int { return len(s.symbols) }

func (s symbolSlice) Swap(i, j int) {
	s.symbols[i], s.symbols[j] = s.symbols[j], s.symbols[i]
	s.metaData[i], s.metaData[j] = s.metaData[j], s.metaData[i]
}

func (s symbolSlice) Less(i, j int) bool {
	return s.symbols[i].Start < s.symbols[j].Start
}

type DocChecker struct {
	trigrams map[ngram]struct{}
}

func (t *DocChecker) Check(content []byte, maxTrigramCount int, allowLargeFile bool) SkipReason {
	if len(content) == 0 {
		return SkipReasonNone
	}

	if len(content) < ngramSize {
		return SkipReasonTooSmall
	}

	if index := bytes.IndexByte(content, 0); index > 0 {
		return SkipReasonBinary
	}

	if trigramsUpperBound := len(content) - ngramSize + 1; trigramsUpperBound <= maxTrigramCount || allowLargeFile {
		return SkipReasonNone
	}

	var cur [3]rune
	byteCount := 0
	t.clearTrigrams(maxTrigramCount)

	for len(content) > 0 {
		r, sz := utf8.DecodeRune(content)
		content = content[sz:]
		byteCount += sz

		cur[0], cur[1], cur[2] = cur[1], cur[2], r
		if cur[0] == 0 {
			continue
		}

		t.trigrams[runesToNGram(cur)] = struct{}{}
		if len(t.trigrams) > maxTrigramCount {
			return SkipReasonTooManyTrigrams
		}
	}
	return SkipReasonNone
}

func (t *DocChecker) clearTrigrams(maxTrigramCount int) {
	if t.trigrams == nil {
		t.trigrams = make(map[ngram]struct{}, maxTrigramCount)
	}
	for key := range t.trigrams {
		delete(t.trigrams, key)
	}
}
