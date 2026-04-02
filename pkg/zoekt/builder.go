package zoekt

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Builder struct {
	opts     Options
	throttle chan int

	nextShardNum int
	todo         []*Document
	docChecker   DocChecker
	size         int

	building sync.WaitGroup

	errMu      sync.Mutex
	buildError error

	finishedShards map[string]string

	indexTime time.Time

	id string

	finishCalled bool

	postingsPool sync.Pool
}

type finishedShard struct {
	temp, final string
}

func NewBuilder(opts Options) (*Builder, error) {
	opts.SetDefaults()
	if opts.RepositoryDescription.Name == "" {
		return nil, fmt.Errorf("builder: must set Name")
	}

	b := &Builder{
		opts:           opts,
		throttle:       make(chan int, opts.Parallelism),
		finishedShards: map[string]string{},
	}

	if _, err := b.newShardBuilder(); err != nil {
		return nil, err
	}

	now := time.Now()
	b.indexTime = now
	b.id = fmt.Sprintf("%d%02d%02d%02d%02d%02d%04d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second(),
		now.Nanosecond()/100000)

	return b, nil
}

func (b *Builder) AddFile(name string, content []byte) error {
	return b.Add(Document{Name: name, Content: content})
}

func (b *Builder) Add(doc Document) error {
	if b.finishCalled {
		return nil
	}

	allowLargeFile := b.opts.IgnoreSizeMax(doc.Name)
	if len(doc.Content) > b.opts.SizeMax && !allowLargeFile {
		doc.SkipReason = SkipReasonTooLarge
	} else if skip := b.docChecker.Check(doc.Content, b.opts.TrigramMax, allowLargeFile); skip != SkipReasonNone {
		doc.SkipReason = skip
	}

	b.todo = append(b.todo, &doc)

	if doc.SkipReason == SkipReasonNone {
		b.size += len(doc.Name) + len(doc.Content)
	} else {
		b.size += len(doc.Name)
		doc.Content = nil
	}

	if b.size > b.opts.ShardMax {
		return b.flush()
	}

	return nil
}

func (b *Builder) Finish() error {
	if b.finishCalled {
		return b.buildError
	}

	b.finishCalled = true

	b.flush()
	b.building.Wait()

	if b.buildError != nil {
		b.finishedShards = map[string]string{}
		return b.buildError
	}

	return b.buildError
}

func (b *Builder) flush() error {
	todo := b.todo
	b.todo = nil
	b.size = 0
	b.errMu.Lock()
	defer b.errMu.Unlock()
	if b.buildError != nil {
		return b.buildError
	}

	hasShard := b.nextShardNum > 0
	if len(todo) == 0 && hasShard {
		return nil
	}

	shard := b.nextShardNum
	b.nextShardNum++

	if b.opts.Parallelism > 1 {
		b.building.Add(1)
		b.throttle <- 1
		go func() {
			done, err := b.buildShard(todo, shard)
			<-b.throttle

			b.errMu.Lock()
			defer b.errMu.Unlock()
			if err != nil && b.buildError == nil {
				b.buildError = err
			}
			if err == nil && done != nil {
				b.finishedShards[done.temp] = done.final
			}
			b.building.Done()
		}()
	} else {
		done, err := b.buildShard(todo, shard)
		b.buildError = err
		if err == nil && done != nil {
			b.finishedShards[done.temp] = done.final
		}

		return b.buildError
	}

	return nil
}

func sortDocuments(todo []*Document) {
	rs := make([]rankedDoc, 0, len(todo))
	for i, t := range todo {
		rd := rankedDoc{t, rank(t, i)}
		rs = append(rs, rd)
	}
	sort.Slice(rs, func(i, j int) bool {
		r1 := rs[i].rank
		r2 := rs[j].rank
		for i := range r1 {
			if r1[i] < r2[i] {
				return true
			}
			if r1[i] > r2[i] {
				return false
			}
		}

		return false
	})
	for i := range todo {
		todo[i] = rs[i].Document
	}
}

type rankedDoc struct {
	*Document
	rank []float64
}

func squashRange(j int) float64 {
	x := float64(j)
	return x / (1 + x)
}

func rank(d *Document, origIdx int) []float64 {
	skipped := 0.0
	if d.SkipReason != SkipReasonNone {
		skipped = 1.0
	}

	return []float64{
		skipped,
		squashRange(len(d.Name)),
		squashRange(len(d.Content)),
		squashRange(origIdx),
	}
}

func (b *Builder) buildShard(todo []*Document, nextShardNum int) (*finishedShard, error) {
	sortDocuments(todo)

	shardBuilder, err := b.newShardBuilder()
	if err != nil {
		return nil, err
	}

	for _, t := range todo {
		if err := shardBuilder.Add(*t); err != nil {
			return nil, err
		}
	}

	result, err := b.writeShard(nextShardNum, shardBuilder)
	b.returnPostingsBuilders(shardBuilder)
	return result, err
}

func (b *Builder) getPostingsBuilder() *postingsBuilder {
	if pb, ok := b.postingsPool.Get().(*postingsBuilder); ok {
		pb.reset()
		return pb
	}
	return newPostingsBuilder(b.opts.ShardMax)
}

func (b *Builder) returnPostingsBuilders(sb *ShardBuilder) {
	if sb.contentPostings != nil {
		b.postingsPool.Put(sb.contentPostings)
		sb.contentPostings = nil
	}
	if sb.namePostings != nil {
		b.postingsPool.Put(sb.namePostings)
		sb.namePostings = nil
	}
}

func (b *Builder) newShardBuilder() (*ShardBuilder, error) {
	desc := b.opts.RepositoryDescription
	desc.HasSymbols = !b.opts.DisableCTags

	content := b.getPostingsBuilder()
	name := b.getPostingsBuilder()
	shardBuilder := newShardBuilderWithPostings(content, name)
	if err := shardBuilder.setRepository(&desc); err != nil {
		return nil, err
	}
	shardBuilder.IndexTime = b.indexTime
	shardBuilder.ID = b.id
	return shardBuilder, nil
}

func (b *Builder) writeShard(n int, ib *ShardBuilder) (*finishedShard, error) {
	if b.opts.IndexFS == nil {
		return nil, nil
	}

	repoID := b.opts.RepositoryDescription.ID
	if repoID == 0 {
		repoID = uint32(n)
	}

	path := shardPath(repoID, n)

	var buf bytes.Buffer
	if err := ib.Write(&buf); err != nil {
		return nil, err
	}

	if err := b.opts.IndexFS.Create("/"+path, buf.Bytes()); err != nil {
		return nil, err
	}

	return &finishedShard{temp: path, final: path}, nil
}

func shardPath(repoID uint32, shardNum int) string {
	return ShardFileName(repoID, shardNum, IndexFormatVersion)
}

func ShardFileName(repoID uint32, shardNum int, version int) string {
	return ShardPrefix(repoID) + fmt.Sprintf("_v%d.%05d.zoekt", version, shardNum)
}

func ShardPrefix(repoID uint32) string {
	return fmt.Sprintf("repo_%08d", repoID)
}

func (o *Options) IgnoreSizeMax(name string) bool {
	// Simple prefix matching for now - can be enhanced later
	for _, pattern := range o.LargeFiles {
		negated, validatedPattern := checkIsNegatePattern(pattern)

		// Simple path matching - check if name ends with pattern or matches pattern
		matched := name == validatedPattern || strings.HasSuffix(name, "/"+validatedPattern) || strings.HasPrefix(name, validatedPattern)
		if matched {
			if negated {
				return false
			} else {
				return true
			}
		}
	}

	return false
}

func checkIsNegatePattern(pattern string) (bool, string) {
	negate := "!"

	if strings.HasPrefix(pattern, negate) {
		return true, pattern[len(negate):]
	}

	return false, pattern
}
