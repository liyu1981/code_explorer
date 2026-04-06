package index

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	zkq "github.com/liyu1981/code_explorer/pkg/zoekt/query"
)

type indexData struct {
	file IndexFile

	metaData     IndexMetadata
	repoMetaData []Repository

	contentNgrams btreeIndex
	nameNgrams    btreeIndex

	boundaries      []uint32
	boundariesStart uint32

	newlinesStart uint32
	newlinesIndex []uint32

	docSectionsStart uint32
	docSectionsIndex []uint32

	fileNameContent []byte
	fileNameIndex   []uint32

	fileBranchMasks []uint64
	branchNames     []map[uint]string
	branchIDs       []map[string]uint

	checksums []byte
	languages []byte

	subRepos []uint32
	repos    []uint16

	runeDocSections     []byte
	rawFileEndRunes     []uint32
	rawFileNameEndRunes []uint32
	fileEndSymbol       []uint32
}

func (d *indexData) readSectionBlob(sec simpleSection) ([]byte, error) {
	if sec.sz == 0 {
		return nil, nil
	}
	return d.file.Read(sec.off, sec.sz)
}

func (d *indexData) readContentSlice(off uint32, sz uint32) ([]byte, error) {
	return d.file.Read(off, sz)
}

func (d *indexData) readContents(i uint32) ([]byte, error) {
	if i >= uint32(len(d.boundaries))-1 {
		return nil, fmt.Errorf("document index %d out of range", i)
	}
	off := d.boundaries[i]
	sz := d.boundaries[i+1] - off
	return d.readContentSlice(d.boundariesStart+off, sz)
}

func (d *indexData) readNewlines(i uint32) ([]uint32, error) {
	if i >= uint32(len(d.newlinesIndex))-1 {
		return nil, fmt.Errorf("newlines index %d out of range", i)
	}
	off := d.newlinesIndex[i]
	sz := d.newlinesIndex[i+1] - off
	data, err := d.readContentSlice(d.newlinesStart+off, sz)
	if err != nil {
		return nil, err
	}
	return fromSizedDeltas(data, nil), nil
}

func (d *indexData) readDocSections(i uint32) ([]DocumentSection, error) {
	if i >= uint32(len(d.docSectionsIndex))-1 {
		return nil, fmt.Errorf("doc sections index %d out of range", i)
	}
	off := d.docSectionsIndex[i]
	sz := d.docSectionsIndex[i+1] - off
	data, err := d.readContentSlice(d.docSectionsStart+off, sz)
	if err != nil {
		return nil, err
	}
	return unmarshalDocSections(data, nil), nil
}

func (d *indexData) fileName(i uint32) []byte {
	if i >= uint32(len(d.fileNameIndex))-1 {
		return nil
	}
	return d.fileNameContent[d.fileNameIndex[i]:d.fileNameIndex[i+1]]
}

func (d *indexData) numDocs() uint32 {
	return uint32(len(d.fileBranchMasks))
}

func (d *indexData) branchIndex(docID uint32) int {
	mask := d.fileBranchMasks[docID]
	i := 0
	for mask > 0 {
		if mask&1 == 1 {
			return i
		}
		mask >>= 1
		i++
	}
	return -1
}

func (d *indexData) getChecksum(idx uint32) []byte {
	if idx*8+8 > uint32(len(d.checksums)) {
		return nil
	}
	return d.checksums[idx*8 : idx*8+8]
}

func (d *indexData) getLanguage(idx uint32) uint16 {
	if d.metaData.IndexFeatureVersion < 12 {
		if idx < uint32(len(d.languages)) {
			return uint16(d.languages[idx])
		}
		return 0
	}
	if idx*2+1 < uint32(len(d.languages)) {
		return uint16(d.languages[idx*2]) | uint16(d.languages[idx*2+1])<<8
	}
	return 0
}

func (d *indexData) ngrams(filename bool) btreeIndex {
	if filename {
		return d.nameNgrams
	}
	return d.contentNgrams
}

func (d *indexData) fileEndRunes() []uint32 {
	return d.rawFileEndRunes
}

func (d *indexData) fileNameEndRunes() []uint32 {
	return d.rawFileNameEndRunes
}

func (d *indexData) Search(query zkq.Q, opts *zkq.SearchOptions) (*zkq.SearchResult, error) {
	res := &zkq.SearchResult{}

	if opts == nil {
		opts = &zkq.SearchOptions{}
	}
	opts.SetDefaults()

	if d.numDocs() == 0 {
		return res, nil
	}

	q := d.simplify(query)
	if c, ok := q.(*zkq.Const); ok && !c.Value {
		return res, nil
	}

	mt, err := d.newMatchTree(q)
	if err != nil {
		return nil, err
	}

	if mt == nil {
		res.Stats.FilesExamined = int(d.numDocs())
		return res, nil
	}

	res.Stats.FilesExamined = int(d.numDocs())

	cp := &contentProvider{
		id:    d,
		stats: &res.Stats,
	}

	for {
		docID := mt.nextDoc()
		if docID == ^uint32(0) {
			break
		}

		fileName := string(d.fileName(docID))
		fileMatch := cp.scoreFile(docID, mt, opts)
		if fileMatch == nil {
			continue
		}

		fileMatch.FileName = fileName
		res.Files = append(res.Files, *fileMatch)
		res.Stats.FilesMatched++

		if opts.MaxMatchCount > 0 && res.Stats.FilesMatched >= opts.MaxMatchCount {
			return res, nil
		}
	}

	return res, nil
}

func (d *indexData) simplify(q zkq.Q) zkq.Q {
	if and, ok := q.(*zkq.And); ok {
		var children []zkq.Q
		for _, c := range and.Children {
			simplified := d.simplify(c)
			if simplified != nil {
				children = append(children, simplified)
			}
		}
		if len(children) == 0 {
			return &zkq.Const{Value: false}
		}
		if len(children) == 1 {
			return children[0]
		}
		return &zkq.And{Children: children}
	}
	return q
}

func (d *indexData) newMatchTree(q zkq.Q) (matchTree, error) {
	switch q := q.(type) {
	case *zkq.Substring:
		return d.newSubstringMatchTree(q)
	case *zkq.And:
		children := make([]matchTree, 0, len(q.Children))
		for _, c := range q.Children {
			mt, err := d.newMatchTree(c)
			if err != nil {
				return nil, err
			}
			children = append(children, mt)
		}
		return &andMatchTree{children: children}, nil
	case *zkq.Or:
		children := make([]matchTree, 0, len(q.Children))
		for _, c := range q.Children {
			mt, err := d.newMatchTree(c)
			if err != nil {
				return nil, err
			}
			children = append(children, mt)
		}
		return &orMatchTree{children: children}, nil
	case *zkq.Not:
		mt, err := d.newMatchTree(q.Child)
		if err != nil {
			return nil, err
		}
		return &notMatchTree{child: mt}, nil
	case *zkq.Const:
		if q.Value {
			return &allMatchTree{}, nil
		}
		return &noMatchTree{}, nil
	case *zkq.Branch:
		return d.newBranchMatchTree(q.Pattern)
	case *zkq.Repo:
		return &noMatchTree{}, nil
	case *zkq.Language:
		return &noMatchTree{}, nil
	default:
		return nil, fmt.Errorf("unsupported query type: %T", q)
	}
}

func (d *indexData) newSubstringMatchTree(s *zkq.Substring) (matchTree, error) {
	str := s.Pattern
	if len(str) == 0 {
		return &noMatchTree{}, nil
	}

	ngrams := d.ngrams(s.FileName)
	searchNgrams := splitNGrams([]byte(str))
	if len(searchNgrams) == 0 {
		return &noMatchTree{}, nil
	}

	var ends []uint32
	if s.FileName {
		ends = d.fileNameEndRunes()
	} else {
		ends = d.fileEndRunes()
	}

	var candidates []uint32
	first := true
	for _, o := range searchNgrams {
		sec := ngrams.Get(o.ngram)
		if sec.sz == 0 {
			return &noMatchTree{}, nil
		}
		postingData, err := d.file.Read(sec.off, sec.sz)
		if err != nil {
			return nil, err
		}
		postings := decodePostingList(postingData)

		docIDs := make([]uint32, 0, len(postings))
		lastDocID := ^uint32(0)
		for _, p := range postings {
			docID := uint32(sort.Search(len(ends), func(i int) bool {
				return ends[i] > p
			}))
			if docID < uint32(len(ends)) && docID != lastDocID {
				docIDs = append(docIDs, docID)
				lastDocID = docID
			}
		}

		if first {
			candidates = docIDs
			first = false
		} else {
			candidates = intersectUint32(candidates, docIDs)
			if len(candidates) == 0 {
				return &noMatchTree{}, nil
			}
		}
	}

	return &substringMatchTree{
		ids:           candidates,
		ends:          ends,
		pattern:       []byte(s.Pattern),
		lowerPattern:  toLower([]byte(s.Pattern)),
		caseSensitive: s.CaseSensitive,
		fileName:      s.FileName,
		d:             d,
		lastDoc:       -1,
	}, nil
}

func (d *indexData) newBranchMatchTree(branch string) (matchTree, error) {
	mask := uint64(0)
	for _, names := range d.branchNames {
		for id, name := range names {
			if name == branch {
				mask = mask | uint64(id)
			}
		}
	}
	if mask == 0 {
		return &noMatchTree{}, nil
	}
	return &branchMaskMatchTree{
		masks:     d.fileBranchMasks,
		mask:      mask,
		branch:    branch,
		branchIDs: d.branchIDs,
	}, nil
}

func (s *indexData) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

func loadIndexData(r IndexFile) (*indexData, error) {
	rd := &reader{r: r}

	toc, err := rd.readTOC()
	if err != nil {
		return nil, err
	}
	return rd.readIndexData(toc)
}

func (r *reader) readTOC() (*indexTOC, error) {
	toc := &indexTOC{}

	sz, err := r.r.Size()
	if err != nil {
		return nil, err
	}

	r.off = sz - 8

	var tocSection simpleSection
	if err := tocSection.read(r); err != nil {
		return nil, fmt.Errorf("read tocSection: %w", err)
	}

	r.seek(tocSection.off)

	if _, err := r.readU32(); err != nil {
		return nil, fmt.Errorf("read sectionCount: %w", err)
	}

	secs := sectionsTagged()
	for r.off < tocSection.off+tocSection.sz {
		tag, err := r.readString()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read tag at offset %d: %w", r.off, err)
		}
		kind, err := r.readVarint()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read kind for %s: %w", tag, err)
		}

		sec, ok := secs[tag]
		if !ok || sec == nil {
			// Create a dummy section to skip
			var dummy section
			switch sectionKind(kind) {
			case sectionKindSimple:
				dummy = &simpleSection{}
			case sectionKindCompound:
				dummy = &compoundSection{}
			case sectionKindCompoundLazy:
				dummy = &lazyCompoundSection{}
			default:
				break
			}
			if err := dummy.skip(r); err != nil {
				break
			}
			continue
		}
		if sec.kind() != sectionKind(kind) {
			// Try to skip using sec's skip method
			if err := sec.skip(r); err != nil {
				break
			}
			continue
		}

		if err := sec.read(r); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read section %s: %w", tag, err)
		}

		switch tag {
		case "metaData":
			if ss, ok := sec.(*simpleSection); ok {
				toc.metaData = *ss
			}
		case "repoMetaData":
			if ss, ok := sec.(*simpleSection); ok {
				toc.repoMetaData = *ss
			}
		case "fileContents":
			if cs, ok := sec.(*compoundSection); ok {
				toc.fileContents = *cs
			}
		case "fileNames":
			if cs, ok := sec.(*compoundSection); ok {
				toc.fileNames = *cs
			}
		case "fileSections":
			if cs, ok := sec.(*compoundSection); ok {
				toc.fileSections = *cs
			}
		case "newlines":
			if cs, ok := sec.(*compoundSection); ok {
				toc.newlines = *cs
			}
		case "ngramText":
			if ss, ok := sec.(*simpleSection); ok {
				toc.ngramText = *ss
			}
		case "postings":
			if cs, ok := sec.(*compoundSection); ok {
				toc.postings = *cs
			}
		case "nameNgramText":
			if ss, ok := sec.(*simpleSection); ok {
				toc.nameNgramText = *ss
			}
		case "namePostings":
			if cs, ok := sec.(*compoundSection); ok {
				toc.namePostings = *cs
			}
		case "branchMasks":
			if ss, ok := sec.(*simpleSection); ok {
				toc.branchMasks = *ss
			}
		case "subRepos":
			if ss, ok := sec.(*simpleSection); ok {
				toc.subRepos = *ss
			}
		case "runeOffsets":
			if ss, ok := sec.(*simpleSection); ok {
				toc.runeOffsets = *ss
			}
		case "nameRuneOffsets":
			if ss, ok := sec.(*simpleSection); ok {
				toc.nameRuneOffsets = *ss
			}
		case "fileEndRunes":
			if ss, ok := sec.(*simpleSection); ok {
				toc.fileEndRunes = *ss
			}
		case "nameEndRunes":
			if ss, ok := sec.(*simpleSection); ok {
				toc.nameEndRunes = *ss
			}
		case "contentChecksums":
			if ss, ok := sec.(*simpleSection); ok {
				toc.contentChecksums = *ss
			}
		case "languages":
			if ss, ok := sec.(*simpleSection); ok {
				toc.languages = *ss
			}
		case "categories":
			if ss, ok := sec.(*simpleSection); ok {
				toc.categories = *ss
			}
		case "fileEndSymbol":
			if ss, ok := sec.(*simpleSection); ok {
				toc.fileEndSymbol = *ss
			}
		case "symbolMap":
			if lcs, ok := sec.(*lazyCompoundSection); ok {
				toc.symbolMap = *lcs
			}
		case "symbolKindMap":
			if cs, ok := sec.(*compoundSection); ok {
				toc.symbolKindMap = *cs
			}
		case "symbolMetaData":
			if ss, ok := sec.(*simpleSection); ok {
				toc.symbolMetaData = *ss
			}
		case "repos":
			if ss, ok := sec.(*simpleSection); ok {
				toc.repos = *ss
			}
		case "reposIDsBitmap":
			if ss, ok := sec.(*simpleSection); ok {
				toc.reposIDsBitmap = *ss
			}
		}
	}

	return toc, nil
}

func sectionsTagged() map[string]section {
	secs := make(map[string]section)
	secs["metaData"] = &simpleSection{}
	secs["repoMetaData"] = &simpleSection{}
	secs["fileContents"] = &compoundSection{}
	secs["fileNames"] = &compoundSection{}
	secs["fileSections"] = &compoundSection{}
	secs["newlines"] = &compoundSection{}
	secs["ngramText"] = &simpleSection{}
	secs["postings"] = &compoundSection{}
	secs["nameNgramText"] = &simpleSection{}
	secs["namePostings"] = &compoundSection{}
	secs["branchMasks"] = &simpleSection{}
	secs["subRepos"] = &simpleSection{}
	secs["runeOffsets"] = &simpleSection{}
	secs["nameRuneOffsets"] = &simpleSection{}
	secs["fileEndRunes"] = &simpleSection{}
	secs["nameEndRunes"] = &simpleSection{}
	secs["contentChecksums"] = &simpleSection{}
	secs["languages"] = &simpleSection{}
	secs["categories"] = &simpleSection{}
	secs["fileEndSymbol"] = &simpleSection{}
	secs["symbolMap"] = &lazyCompoundSection{}
	secs["symbolKindMap"] = &compoundSection{}
	secs["symbolMetaData"] = &simpleSection{}
	secs["repos"] = &simpleSection{}
	secs["reposIDsBitmap"] = &simpleSection{}
	return secs
}

func (r *reader) readIndexData(toc *indexTOC) (*indexData, error) {
	d := indexData{
		file: r.r,
	}

	metaDataBytes, err := r.readSectionBlob(toc.metaData)
	if err != nil {
		return nil, err
	}
	if len(metaDataBytes) > 0 {
		if err := json.Unmarshal(metaDataBytes, &d.metaData); err != nil {
			return nil, fmt.Errorf("decode metaData: %w", err)
		}
	}

	if d.metaData.IndexFeatureVersion < WriteMinFeatureVersion {
		return nil, fmt.Errorf("file feature version %d < min %d", d.metaData.IndexFeatureVersion, WriteMinFeatureVersion)
	}

	d.boundariesStart = toc.fileContents.data.off
	d.boundaries = toc.fileContents.relativeIndex()
	d.newlinesStart = toc.newlines.data.off
	d.newlinesIndex = toc.newlines.relativeIndex()
	d.docSectionsStart = toc.fileSections.data.off
	d.docSectionsIndex = toc.fileSections.relativeIndex()

	d.contentNgrams, err = d.newBtreeIndex(toc.ngramText, toc.postings)
	if err != nil {
		return nil, err
	}

	branchMasksData, err := r.readSectionBlob(toc.branchMasks)
	if err != nil {
		return nil, err
	}
	d.fileBranchMasks = make([]uint64, len(branchMasksData)/8)
	for i := range d.fileBranchMasks {
		d.fileBranchMasks[i] = binary.BigEndian.Uint64(branchMasksData[i*8:])
	}

	d.fileNameContent, err = r.readSectionBlob(toc.fileNames.data)
	if err != nil {
		return nil, err
	}
	d.fileNameIndex = toc.fileNames.relativeIndex()

	d.nameNgrams, err = d.newBtreeIndex(toc.nameNgramText, toc.namePostings)
	if err != nil {
		return nil, err
	}

	d.checksums, err = r.readSectionBlob(toc.contentChecksums)
	if err != nil {
		return nil, err
	}

	d.languages, err = r.readSectionBlob(toc.languages)
	if err != nil {
		return nil, err
	}

	subReposData, err := r.readSectionBlob(toc.subRepos)
	if err != nil {
		return nil, err
	}
	if len(subReposData) > 0 {
		d.subRepos = fromSizedDeltas(subReposData, nil)
	}

	fileEndRunesData, err := r.readSectionBlob(toc.fileEndRunes)
	if err != nil {
		return nil, err
	}
	if len(fileEndRunesData) > 0 {
		d.rawFileEndRunes = fromSizedDeltas(fileEndRunesData, nil)
	}

	nameEndRunesData, err := r.readSectionBlob(toc.nameEndRunes)
	if err != nil {
		return nil, err
	}
	if len(nameEndRunesData) > 0 {
		d.rawFileNameEndRunes = fromSizedDeltas(nameEndRunesData, nil)
	}

	d.fileEndSymbol, err = readSectionU32(d.file, toc.fileEndSymbol)
	if err != nil {
		return nil, err
	}

	d.runeDocSections, err = r.readSectionBlob(toc.runeDocSections)
	if err != nil {
		return nil, err
	}

	repoMetaDataBytes, err := r.readSectionBlob(toc.repoMetaData)
	if err != nil {
		return nil, err
	}
	if len(repoMetaDataBytes) > 0 {
		var repo Repository
		if err := json.Unmarshal(repoMetaDataBytes, &repo); err != nil {
			// Try as a slice if single Repository fails
			if err2 := json.Unmarshal(repoMetaDataBytes, &d.repoMetaData); err2 != nil {
				return nil, fmt.Errorf("decode repoMetaData: %w (single) or %v (slice)", err, err2)
			}
		} else {
			d.repoMetaData = []Repository{repo}
		}
	}

	d.branchIDs = []map[string]uint{}
	d.branchNames = []map[uint]string{}

	for i := range d.repoMetaData {
		repoBranchIDs := make(map[string]uint)
		repoBranchNames := make(map[uint]string)
		for j, br := range d.repoMetaData[i].Branches {
			id := uint(1) << uint(j)
			repoBranchIDs[br.Name] = id
			repoBranchNames[id] = br.Name
		}
		d.branchIDs = append(d.branchIDs, repoBranchIDs)
		d.branchNames = append(d.branchNames, repoBranchNames)
	}

	if toc.repos.sz > 0 {
		reposData, err := r.readSectionBlob(toc.repos)
		if err != nil {
			return nil, err
		}
		d.repos = make([]uint16, len(reposData)/2)
		for i := range d.repos {
			d.repos[i] = binary.BigEndian.Uint16(reposData[i*2:])
		}
	}

	return &d, nil
}

func (d *indexData) newBtreeIndex(ngramSec simpleSection, postings compoundSection) (btreeIndex, error) {
	bi := btreeIndex{file: d.file}

	textContent, err := d.readSectionBlob(ngramSec)
	if err != nil {
		return btreeIndex{}, err
	}

	bt := newBtree(btreeOpts{bucketSize: btreeBucketSize, v: 50})
	for i := 0; i < len(textContent); i += ngramEncoding {
		ng := ngram(binary.BigEndian.Uint64(textContent[i : i+ngramEncoding]))
		bt.insert(ng)
	}
	bt.freeze()

	bi.bt = bt

	bi.ngramSec = ngramSec
	bi.postingIndex = postings.index

	return bi, nil
}

func readSectionU64(file IndexFile, sec simpleSection) ([]uint64, error) {
	if sec.sz == 0 {
		return nil, nil
	}
	data, err := file.Read(sec.off, sec.sz)
	if err != nil {
		return nil, err
	}
	res := make([]uint64, len(data)/8)
	for i := range res {
		res[i] = binary.BigEndian.Uint64(data[i*8:])
	}
	return res, nil
}

func readSectionU32(file IndexFile, sec simpleSection) ([]uint32, error) {
	if sec.sz == 0 {
		return nil, nil
	}
	data, err := file.Read(sec.off, sec.sz)
	if err != nil {
		return nil, err
	}
	res := make([]uint32, len(data)/4)
	for i := range res {
		res[i] = binary.BigEndian.Uint32(data[i*4:])
	}
	return res, nil
}
