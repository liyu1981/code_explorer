// Copyright 2016 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package zoekt provides source code indexing functionality.
package index

import "time"

type RepositoryBranch struct {
	Name    string
	Version string
}

type Repository struct {
	ID       string
	Name     string
	Branches []RepositoryBranch

	SubRepoMap map[string]*Repository

	HasSymbols       bool
	IndexOptions     string
	FileTombstones   map[string]struct{}
	LatestCommitDate time.Time
}

type IndexMetadata struct {
	IndexFormatVersion    int
	IndexFeatureVersion   int
	IndexMinReaderVersion int
	IndexTime             time.Time
	PlainASCII            bool
	LanguageMap           map[string]uint16
	ZoektVersion          string
	ID                    string
}

type Symbol struct {
	Sym        string
	Kind       string
	Parent     string
	ParentKind string
}
