package sqlitefs

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/rs/zerolog/log"
)

const DefaultChunkSize = 4096
const DefaultCacheSize = 1000

var (
	instance *SQLiteFS
	once     sync.Once
)

type SQLiteFS struct {
	store       *db.Store
	chunkSize   int
	cacheSize   int
	enableCache bool
	cache       *ChunkCache
	mu          sync.RWMutex
}

type FileInfo struct {
	Name     string
	Path     string
	IsDir    bool
	Size     int64
	Modified int64
}

type ChunkKey struct {
	FileID     int64
	ChunkIndex int64
}

func GetFS() *SQLiteFS {
	once.Do(func() {
		instance = &SQLiteFS{
			chunkSize:   DefaultChunkSize,
			cacheSize:   DefaultCacheSize,
			enableCache: true,
			cache:       NewChunkCache(DefaultCacheSize),
		}
	})
	return instance
}

func OpenFS(store *db.Store) *SQLiteFS {
	fs := &SQLiteFS{
		store:       store,
		chunkSize:   DefaultChunkSize,
		cacheSize:   DefaultCacheSize,
		enableCache: true,
		cache:       NewChunkCache(DefaultCacheSize),
	}

	once.Do(func() {
		instance = fs
	})

	log.Info().Int("chunkSize", fs.chunkSize).Bool("cache", fs.enableCache).Msg("SQLiteFS opened with db.Store")
	return fs
}

func (fs *SQLiteFS) resolvePath(path string) (int64, error) {
	if path == "/" || path == "" {
		return 1, nil
	}

	parts := splitPath(path)
	parentID := int64(1)

	for _, part := range parts {
		var id int64
		err := fs.store.DB().QueryRowContext(context.Background(),
			`SELECT id FROM fs_nodes WHERE parent_id = ? AND name = ?`,
			parentID, part,
		).Scan(&id)
		if err == sql.ErrNoRows {
			return 0, ErrNotFound
		}
		if err != nil {
			return 0, err
		}
		parentID = id
	}

	return parentID, nil
}

func (fs *SQLiteFS) resolveOrCreate(path string, isDir bool) (int64, error) {
	if path == "/" || path == "" {
		return 1, nil
	}

	parts := splitPath(path)
	parentID := int64(1)

	for i, part := range parts {
		var id int64
		err := fs.store.DB().QueryRowContext(context.Background(),
			`SELECT id FROM fs_nodes WHERE parent_id = ? AND name = ?`,
			parentID, part,
		).Scan(&id)

		if err == sql.ErrNoRows {
			nodeType := "file"
			if isDir || i < len(parts)-1 {
				nodeType = "dir"
			}
			result, err := fs.store.DB().ExecContext(context.Background(),
				`INSERT INTO fs_nodes (name, parent_id, type) VALUES (?, ?, ?)`,
				part, parentID, nodeType,
			)
			if err != nil {
				return 0, err
			}
			id, err = result.LastInsertId()
			if err != nil {
				return 0, err
			}
		} else if err != nil {
			return 0, err
		}
		parentID = id
	}

	return parentID, nil
}

func (fs *SQLiteFS) Read(path string, offset int64, size int) ([]byte, error) {
	fileID, err := fs.resolvePath(path)
	if err != nil {
		return nil, err
	}

	startChunk := offset / int64(fs.chunkSize)
	endChunk := (offset + int64(size) - 1) / int64(fs.chunkSize)

	chunks, err := fs.readChunks(fileID, startChunk, endChunk)
	if err != nil {
		return nil, err
	}

	return fs.combineAndSlice(chunks, offset, size), nil
}

func (fs *SQLiteFS) readChunks(fileID int64, startChunk, endChunk int64) (map[int64][]byte, error) {
	chunks := make(map[int64][]byte)

	rows, err := fs.store.DB().QueryContext(context.Background(),
		`SELECT chunk_index, data FROM fs_file_chunks 
		 WHERE file_id = ? AND chunk_index BETWEEN ? AND ?
		 ORDER BY chunk_index`,
		fileID, startChunk, endChunk,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var index int64
		var data []byte
		if err := rows.Scan(&index, &data); err != nil {
			return nil, err
		}
		chunks[index] = data

		if fs.enableCache {
			fs.cache.Set(ChunkKey{FileID: fileID, ChunkIndex: index}, data)
		}
	}

	return chunks, rows.Err()
}

func (fs *SQLiteFS) combineAndSlice(chunks map[int64][]byte, offset int64, size int) []byte {
	startChunk := offset / int64(fs.chunkSize)
	chunkOffset := offset % int64(fs.chunkSize)

	var result []byte
	remaining := size

	idx := startChunk
	for remaining > 0 {
		chunk, ok := chunks[idx]
		if !ok {
			break
		}

		start := 0
		if idx == startChunk && chunkOffset > 0 {
			start = int(chunkOffset)
		}

		end := len(chunk)
		if remaining < end-start {
			end = start + remaining
		}

		result = append(result, chunk[start:end]...)
		remaining -= (end - start)
		idx++
	}

	return result
}

func (fs *SQLiteFS) Write(path string, offset int64, data []byte) error {
	fileID, err := fs.resolveOrCreate(path, false)
	if err != nil {
		return err
	}

	return fs.writeChunks(fileID, offset, data)
}

func (fs *SQLiteFS) writeChunks(fileID int64, offset int64, data []byte) error {
	startChunk := offset / int64(fs.chunkSize)
	offsetInChunk := offset % int64(fs.chunkSize)

	var chunks []struct {
		index int64
		data  []byte
	}

	chunkData := make([]byte, fs.chunkSize)

	if offsetInChunk > 0 {
		existing, err := fs.readChunks(fileID, startChunk, startChunk)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if existingChunk, ok := existing[startChunk]; ok {
			copy(chunkData, existingChunk)
		}
	}

	copy(chunkData[offsetInChunk:], data)
	currentOffset := int64(len(data))

	chunks = append(chunks, struct {
		index int64
		data  []byte
	}{startChunk, make([]byte, fs.chunkSize)})
	copy(chunks[0].data, chunkData)

	for currentOffset < int64(len(data)) {
		chunkIdx := startChunk + int64(len(chunks))
		chunkStart := int(currentOffset)
		chunkEnd := chunkStart + fs.chunkSize
		if chunkEnd > len(data) {
			chunkEnd = len(data)
		}

		chunkData := make([]byte, chunkEnd-chunkStart)
		copy(chunkData, data[chunkStart:chunkEnd])

		chunks = append(chunks, struct {
			index int64
			data  []byte
		}{chunkIdx, chunkData})
		currentOffset = int64(chunkEnd)
	}

	_, err := fs.store.DB().Exec("BEGIN", nil)
	if err != nil {
		return err
	}

	for _, c := range chunks {
		_, err := fs.store.DB().Exec(
			`INSERT OR REPLACE INTO fs_file_chunks (file_id, chunk_index, data) VALUES (?, ?, ?)`,
			fileID, c.index, c.data,
		)
		if err != nil {
			fs.store.DB().Exec("ROLLBACK", nil)
			return err
		}
	}

	totalSize := int64(offset) + int64(len(data))
	_, err = fs.store.DB().Exec(
		`UPDATE fs_nodes SET size = ?, updated_at = strftime('%s', 'now') WHERE id = ?`,
		totalSize, fileID,
	)
	if err != nil {
		fs.store.DB().Exec("ROLLBACK", nil)
		return err
	}

	_, err = fs.store.DB().Exec("COMMIT", nil)
	return err
}

func (fs *SQLiteFS) Create(path string, data []byte) error {
	fileID, err := fs.resolveOrCreate(path, false)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	return fs.writeChunks(fileID, 0, data)
}

func (fs *SQLiteFS) Delete(path string) error {
	nodeID, err := fs.resolvePath(path)
	if err != nil {
		return err
	}

	_, err = fs.store.DB().ExecContext(context.Background(),
		`DELETE FROM fs_nodes WHERE id = ?`,
		nodeID,
	)
	return err
}

func (fs *SQLiteFS) Mkdir(path string) error {
	_, err := fs.resolveOrCreate(path, true)
	return err
}

func (fs *SQLiteFS) List(path string) ([]FileInfo, error) {
	parentID, err := fs.resolvePath(path)
	if err != nil {
		return nil, err
	}

	rows, err := fs.store.DB().QueryContext(context.Background(),
		`SELECT name, type, size, updated_at FROM fs_nodes WHERE parent_id = ?`,
		parentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []FileInfo
	for rows.Next() {
		var info FileInfo
		var nodeType string
		if err := rows.Scan(&info.Name, &nodeType, &info.Size, &info.Modified); err != nil {
			return nil, err
		}
		info.IsDir = nodeType == "dir"
		info.Path = filepath.Join(path, info.Name)
		results = append(results, info)
	}

	return results, rows.Err()
}

func (fs *SQLiteFS) Exists(path string) (bool, error) {
	_, err := fs.resolvePath(path)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func splitPath(path string) []string {
	if path == "/" || path == "" {
		return nil
	}
	path = filepath.Clean(path)
	return strings.Split(path, "/")
}

var ErrNotFound = errors.New("path not found")
