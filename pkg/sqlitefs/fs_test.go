package sqlitefs

import (
	"path/filepath"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/libsql"
)

func TestSQLiteFSWithStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	sqlDB, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}

	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	store := db.NewStore(sqlDB, dbPath)

	fs := OpenFS(store)

	t.Run("Create and Read", func(t *testing.T) {
		err := fs.Create("/test.txt", []byte("Hello World"))
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		data, err := fs.Read("/test.txt", 0, 11)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if string(data) != "Hello World" {
			t.Errorf("Expected 'Hello World', got '%s'", string(data))
		}
	})

	t.Run("Partial Read", func(t *testing.T) {
		data, err := fs.Read("/test.txt", 0, 5)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if string(data) != "Hello" {
			t.Errorf("Expected 'Hello', got '%s'", string(data))
		}
	})

	t.Run("List", func(t *testing.T) {
		entries, err := fs.List("/")
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		found := false
		for _, e := range entries {
			if e.Name == "test.txt" && !e.IsDir {
				found = true
			}
		}
		if !found {
			t.Error("test.txt not found in listing")
		}
	})

	t.Run("Mkdir and List", func(t *testing.T) {
		err := fs.Mkdir("/subdir")
		if err != nil {
			t.Fatalf("Mkdir failed: %v", err)
		}

		err = fs.Create("/subdir/file.txt", []byte("content"))
		if err != nil {
			t.Fatalf("Create in subdir failed: %v", err)
		}

		entries, err := fs.List("/subdir")
		if err != nil {
			t.Fatalf("List subdir failed: %v", err)
		}

		if len(entries) != 1 || entries[0].Name != "file.txt" {
			t.Errorf("Expected 1 entry 'file.txt', got %v", entries)
		}
	})

	t.Run("Exists", func(t *testing.T) {
		exists, err := fs.Exists("/test.txt")
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}
		if !exists {
			t.Error("test.txt should exist")
		}

		exists, err = fs.Exists("/nonexistent")
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}
		if exists {
			t.Error("nonexistent should not exist")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := fs.Delete("/test.txt")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		exists, _ := fs.Exists("/test.txt")
		if exists {
			t.Error("test.txt should not exist after delete")
		}
	})
}

func TestChunkCache(t *testing.T) {
	cache := NewChunkCache(3)

	key1 := ChunkKey{FileID: 1, ChunkIndex: 1}
	key2 := ChunkKey{FileID: 1, ChunkIndex: 2}
	key3 := ChunkKey{FileID: 1, ChunkIndex: 3}
	key4 := ChunkKey{FileID: 1, ChunkIndex: 4}

	cache.Set(key1, []byte("data1"))
	cache.Set(key2, []byte("data2"))
	cache.Set(key3, []byte("data3"))

	if _, ok := cache.Get(key1); !ok {
		t.Error("key1 should be in cache")
	}

	cache.Set(key4, []byte("data4"))

	if _, ok := cache.Get(key1); ok {
		t.Error("key1 should have been evicted")
	}
	if _, ok := cache.Get(key4); !ok {
		t.Error("key4 should be in cache")
	}
}
