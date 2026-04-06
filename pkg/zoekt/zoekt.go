package zoekt

import (
	"context"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/sqlitefs"
	zkindex "github.com/liyu1981/code_explorer/pkg/zoekt/index"
	zkq "github.com/liyu1981/code_explorer/pkg/zoekt/query"
)

type ZoektIndex struct {
	store    *db.Store
	fs       *sqlitefs.SQLiteFS
	indexer  *ZkIndexer
	searcher *ZkSearcher
}

func NewZoektIndex(store *db.Store, fs *sqlitefs.SQLiteFS) *ZoektIndex {
	return &ZoektIndex{
		store:    store,
		fs:       fs,
		indexer:  NewZkIndexer(store, fs),
		searcher: NewZkSearcher(store, fs),
	}
}

func (z *ZoektIndex) GetStore() *db.Store {
	return z.store
}

func (z *ZoektIndex) Index(ctx context.Context, dir string, opts *zkindex.IndexOptions) (*zkindex.IndexResult, error) {
	return z.indexer.Index(ctx, dir, opts)
}

func (z *ZoektIndex) Search(ctx context.Context, codebaseID string, query string, opts *zkq.SearchOptions) (*zkq.SearchResult, error) {
	return z.searcher.Search(ctx, codebaseID, query, opts)
}

func (z *ZoektIndex) ListFiles(ctx context.Context, codebaseID string) ([]db.FileInfo, error) {
	metadata, err := z.store.ZoektGetMetadataByCodebase(ctx, codebaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get zoekt metadata for codebase %v: %w", codebaseID, err)
	}
	if metadata == nil {
		return []db.FileInfo{}, nil
	}

	return z.store.ZoektListFiles(ctx, metadata.ID)
}
