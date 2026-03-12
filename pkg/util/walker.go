package util

import (
	"github.com/boyter/gocodewalker"
	"github.com/liyu1981/code_explorer/pkg/constant"
)

type File struct {
	Location string
}

func StartFileWalker(rootDir string, includeHidden bool) <-chan *File {
	fileListQueue := make(chan *gocodewalker.File, 1000)
	walker := gocodewalker.NewFileWalker(rootDir, fileListQueue)
	walker.ExcludeDirectory = constant.VcsIgnoreDirectories
	walker.IgnoreBinaryFiles = false
	walker.IncludeHidden = includeHidden

	out := make(chan *File, 1000)
	go func() {
		defer close(out)
		go walker.Start()
		for f := range fileListQueue {
			out <- &File{Location: f.Location}
		}
	}()
	return out
}
