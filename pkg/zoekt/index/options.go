package zoekt

type Options struct {
	IndexDir              string
	ShardPrefixOverride   string
	SizeMax               int
	Parallelism           int
	ShardMax              int
	TrigramMax            int
	LargeFiles            []string
	RepositoryDescription Repository
	DisableCTags          bool
	IndexFS               IndexFS
}

type IndexFS interface {
	Create(path string, data []byte) error
}

func (o *Options) SetDefaults() {
	if o.Parallelism == 0 {
		o.Parallelism = 4
	}
	if o.SizeMax == 0 {
		o.SizeMax = 2 << 20
	}
	if o.ShardMax == 0 {
		o.ShardMax = 100 << 20
	}
	if o.TrigramMax == 0 {
		o.TrigramMax = 20000
	}
}

type IndexState string

const (
	IndexStateMissing IndexState = "missing"
	IndexStateCorrupt IndexState = "corrupt"
	IndexStateVersion IndexState = "version-mismatch"
	IndexStateOption  IndexState = "option-mismatch"
	IndexStateMeta    IndexState = "meta-mismatch"
	IndexStateContent IndexState = "content-mismatch"
	IndexStateEqual   IndexState = "equal"
)
