package search

import (
	"strings"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func RRFMerge(ftsResults []db.SearchResult, vecResults []db.SearchResult, limit int, k float64, ftsWeight float64, vecWeight float64) []db.SearchResult {
	scores := make(map[string]float64)
	data := make(map[string]db.SearchResult)

	for i, r := range ftsResults {
		scores[r.ChunkKey] = scores[r.ChunkKey] + ftsWeight/(k+float64(i+1))
		data[r.ChunkKey] = r
	}

	for i, r := range vecResults {
		scores[r.ChunkKey] = scores[r.ChunkKey] + vecWeight/(k+float64(i+1))
		if _, exists := data[r.ChunkKey]; !exists {
			data[r.ChunkKey] = r
		}
	}

	type scoredResult struct {
		key   string
		score float64
	}

	var ranked []scoredResult
	for key, score := range scores {
		ranked = append(ranked, scoredResult{key: key, score: score})
	}

	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].score > ranked[i].score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	if limit > len(ranked) {
		limit = len(ranked)
	}

	results := make([]db.SearchResult, limit)
	for i := 0; i < limit; i++ {
		row := data[ranked[i].key]
		row.Score = ranked[i].score
		results[i] = row
	}

	return results
}

func PreprocessQuery(query string) string {
	stopwords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"as": true, "is": true, "was": true, "are": true, "were": true,
		"be": true, "been": true, "being": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true,
		"would": true, "should": true, "could": true, "may": true, "might": true,
	}

	words := strings.Fields(strings.ToLower(query))
	var filtered []string
	for _, word := range words {
		if !stopwords[word] {
			filtered = append(filtered, word)
		}
	}
	return strings.Join(filtered, " ")
}
