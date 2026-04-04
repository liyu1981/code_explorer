package zoekt

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"slices"
	"sort"
	"time"
)

const IndexFormatVersion = 16
const NextIndexFormatVersion = 17
const FeatureVersion = 12
const WriteMinFeatureVersion = 10

type indexTOC struct {
	fileContents compoundSection
	fileNames    compoundSection
	fileSections compoundSection
	postings     compoundSection
	newlines     compoundSection
	ngramText    simpleSection
	runeOffsets  simpleSection
	fileEndRunes simpleSection
	languages    simpleSection
	categories   simpleSection

	fileEndSymbol  simpleSection
	symbolMap      lazyCompoundSection
	symbolKindMap  compoundSection
	symbolMetaData simpleSection

	branchMasks simpleSection
	subRepos    simpleSection

	nameNgramText    simpleSection
	namePostings     compoundSection
	nameRuneOffsets  simpleSection
	metaData         simpleSection
	repoMetaData     simpleSection
	nameEndRunes     simpleSection
	contentChecksums simpleSection
	runeDocSections  simpleSection

	repos          simpleSection
	reposIDsBitmap simpleSection
}

func (b *ShardBuilder) Write(out io.Writer) error {
	buffered := bufio.NewWriterSize(out, 1<<20)
	defer buffered.Flush()

	w := &writer{w: buffered}
	toc := indexTOC{}

	toc.fileContents.writeStrings(w, b.contentStrings)
	toc.newlines.start(w)
	for _, f := range b.contentStrings {
		toc.newlines.addItem(w, toSizedDeltas(newLinesIndices(f.data)))
	}
	toc.newlines.end(w)

	toc.fileEndSymbol.start(w)
	for _, m := range b.fileEndSymbol {
		w.U32(m)
	}
	toc.fileEndSymbol.end(w)

	toc.symbolMap.writeMap(w, b.symIndex)
	toc.symbolKindMap.writeMap(w, b.symKindIndex)
	toc.symbolMetaData.start(w)
	for _, m := range b.symMetaData {
		w.U32(m)
	}
	toc.symbolMetaData.end(w)

	toc.branchMasks.start(w)
	for _, m := range b.branchMasks {
		w.U64(m)
	}
	toc.branchMasks.end(w)

	toc.fileSections.start(w)
	for _, s := range b.docSections {
		toc.fileSections.addItem(w, marshalDocSections(s))
	}
	toc.fileSections.end(w)

	writePostings(w, b.contentPostings, &toc.ngramText, &toc.runeOffsets, &toc.postings, &toc.fileEndRunes)

	toc.fileNames.writeStrings(w, b.nameStrings)

	writePostings(w, b.namePostings, &toc.nameNgramText, &toc.nameRuneOffsets, &toc.namePostings, &toc.nameEndRunes)

	toc.subRepos.start(w)
	w.Write(toSizedDeltas(b.subRepos))
	toc.subRepos.end(w)

	toc.contentChecksums.start(w)
	w.Write(b.checksums)
	toc.contentChecksums.end(w)

	toc.languages.start(w)
	w.Write(b.languages)
	toc.languages.end(w)

	toc.categories.start(w)
	w.Write(b.categories)
	toc.categories.end(w)

	toc.runeDocSections.start(w)
	w.Write(marshalDocSections(b.runeDocSections))
	toc.runeDocSections.end(w)

	indexTime := b.IndexTime
	if indexTime.IsZero() {
		indexTime = time.Now().UTC()
	}

	if err := b.writeJSON(&IndexMetadata{
		IndexFormatVersion:    b.indexFormatVersion,
		IndexTime:             indexTime,
		IndexFeatureVersion:   b.featureVersion,
		IndexMinReaderVersion: WriteMinFeatureVersion,
		PlainASCII:            b.contentPostings.isPlainASCII && b.namePostings.isPlainASCII,
		LanguageMap:           b.languageMap,
		ID:                    b.ID,
	}, &toc.metaData, w); err != nil {
		return err
	}

	if len(b.repoList) != 1 {
		return nil
	}
	if err := b.writeJSON(b.repoList[0], &toc.repoMetaData, w); err != nil {
		return err
	}

	var tocSection simpleSection
	tocSection.start(w)
	w.writeTOC(&toc)
	tocSection.end(w)
	tocSection.write(w)
	return w.err
}

func (b *ShardBuilder) writeJSON(data any, sec *simpleSection, w *writer) error {
	blob, err := json.Marshal(data)
	if err != nil {
		return err
	}
	sec.start(w)
	w.Write(blob)
	sec.end(w)
	return nil
}

func (w *writer) writeTOC(toc *indexTOC) {
	w.U32(0)
	secs := toc.sectionsTaggedList()
	for _, s := range secs {
		if cs, ok := s.sec.(*compoundSection); ok && cs.data.sz == 0 {
			continue
		}
		if ss, ok := s.sec.(*simpleSection); ok && ss.sz == 0 {
			continue
		}
		w.String(s.tag)
		w.Varint(uint32(s.sec.kind()))
		s.sec.write(w)
	}
}

func (s *compoundSection) writeStrings(w *writer, strs []*searchableString) {
	s.start(w)
	for _, f := range strs {
		s.addItem(w, f.data)
	}
	s.end(w)
}

func (s *compoundSection) writeMap(w *writer, m map[string]uint32) {
	keys := make([]*searchableString, 0, len(m))
	for k := range m {
		keys = append(keys, &searchableString{data: []byte(k)})
	}
	sort.Slice(keys, func(i, j int) bool {
		return m[string(keys[i].data)] < m[string(keys[j].data)]
	})
	s.writeStrings(w, keys)
}

func writePostings(w *writer, s *postingsBuilder, ngramText *simpleSection,
	charOffsets *simpleSection, postings *compoundSection, endRunes *simpleSection,
) {
	type ngramPosting struct {
		ng ngram
		pl *postingList
	}
	all := make([]ngramPosting, 0, len(s.asciiPopulated)+len(s.postings))
	for _, idx := range s.asciiPopulated {
		pl := s.asciiPostings[idx]
		if len(pl.data) > 0 {
			all = append(all, ngramPosting{asciiIndexToNgram(idx), pl})
		}
	}
	for k, pl := range s.postings {
		if len(pl.data) > 0 {
			all = append(all, ngramPosting{k, pl})
		}
	}
	slices.SortFunc(all, func(a, b ngramPosting) int {
		if a.ng < b.ng {
			return -1
		}
		if a.ng > b.ng {
			return 1
		}
		return 0
	})

	ngramText.start(w)
	for _, np := range all {
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(np.ng))
		w.Write(buf[:])
	}
	ngramText.end(w)

	postings.start(w)
	for _, np := range all {
		postings.addItem(w, np.pl.data)
	}
	postings.end(w)

	charOffsets.start(w)
	w.Write(toSizedDeltas(s.runeOffsets))
	charOffsets.end(w)

	endRunes.start(w)
	w.Write(toSizedDeltas(s.endRunes))
	endRunes.end(w)
}

func (t *indexTOC) sectionsTaggedList() []taggedSection {
	var unusedSimple simpleSection
	return []taggedSection{
		{"metaData", &t.metaData},
		{"repoMetaData", &t.repoMetaData},
		{"fileContents", &t.fileContents},
		{"fileNames", &t.fileNames},
		{"fileSections", &t.fileSections},
		{"fileEndSymbol", &t.fileEndSymbol},
		{"symbolMap", &t.symbolMap},
		{"symbolKindMap", &t.symbolKindMap},
		{"symbolMetaData", &t.symbolMetaData},
		{"newlines", &t.newlines},
		{"ngramText", &t.ngramText},
		{"postings", &t.postings},
		{"nameNgramText", &t.nameNgramText},
		{"namePostings", &t.namePostings},
		{"branchMasks", &t.branchMasks},
		{"subRepos", &t.subRepos},
		{"runeOffsets", &t.runeOffsets},
		{"nameRuneOffsets", &t.nameRuneOffsets},
		{"fileEndRunes", &t.fileEndRunes},
		{"nameEndRunes", &t.nameEndRunes},
		{"contentChecksums", &t.contentChecksums},
		{"languages", &t.languages},
		{"categories", &t.categories},
		{"runeDocSections", &t.runeDocSections},
		{"nameBloom", &unusedSimple},
		{"contentBloom", &unusedSimple},
		{"ranks", &unusedSimple},
	}
}
