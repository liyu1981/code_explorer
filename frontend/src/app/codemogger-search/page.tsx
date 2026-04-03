"use client";

import { useState } from "react";
import { Search, FileCode, Loader2, X } from "lucide-react";
import useSWR from "swr";
import { fetcher } from "@/lib/api";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { LoadingState } from "../_components/loading-state";
import { ErrorState } from "../_components/error-state";
import { EmptyState } from "../_components/empty-state";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";

interface CodebaseInfo {
  id: string;
  name: string;
  rootPath: string;
  type: string;
  indexedAt: number;
  fileCount: number;
  chunkCount: number;
}

interface SearchResult {
  chunkKey: string;
  filePath: string;
  name: string;
  kind: string;
  signature: string;
  snippet: string;
  startLine: number;
  endLine: number;
  score: number;
}

export default function CodebaseSearchPage() {
  const [query, setQuery] = useState("");
  const [selectedCodebase, setSelectedCodebase] = useState<string>("");
  const [searchMode, setSearchMode] = useState<
    "semantic" | "keyword" | "hybrid"
  >("hybrid");
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);

  const {
    data: codebases,
    error,
    isLoading,
  } = useSWR<CodebaseInfo[]>("/api/codemogger/codebases", fetcher);

  const indexedCodebases = codebases?.filter((cb) => cb.indexedAt > 0) || [];

  const handleSearch = async () => {
    if (!query.trim() || !selectedCodebase) return;

    setIsSearching(true);
    setHasSearched(true);
    try {
      const res = await api.post<SearchResult[]>("/api/codemogger/search", {
        query: query.trim(),
        codebaseID: selectedCodebase,
        mode: searchMode,
        limit: 20,
      });
      setSearchResults(res.data);
    } catch (err) {
      console.error("Search failed:", err);
      setSearchResults([]);
    } finally {
      setIsSearching(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center gap-4">
          <Search className="h-5 w-5 text-blue-500" />
          <h1 className="text-xl font-bold tracking-tight text-primary">
            Search Codebase
          </h1>
        </div>
      </AppHeader>

      <div className="flex-1 overflow-auto p-6 bg-background/50">
        <div className="max-w-4xl mx-auto w-full space-y-6">
          <div className="bg-card border border-border rounded-2xl p-6 space-y-4">
            <div className="flex flex-col gap-4">
              <div className="flex gap-3">
                <div className="flex-1">
                  <input
                    type="text"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder="Enter search query (e.g., 'how is authentication implemented')"
                    className="w-full px-4 py-2.5 bg-background border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
                  />
                </div>
                <select
                  value={selectedCodebase}
                  onChange={(e) => setSelectedCodebase(e.target.value)}
                  className="px-3 py-2 bg-background border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
                >
                  <option value="">Select codebase...</option>
                  {indexedCodebases.map((cb) => (
                    <option key={cb.id} value={cb.id}>
                      {cb.name}
                    </option>
                  ))}
                </select>
                <select
                  value={searchMode}
                  onChange={(e) =>
                    setSearchMode(
                      e.target.value as "semantic" | "keyword" | "hybrid",
                    )
                  }
                  className="px-3 py-2 bg-background border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
                >
                  <option value="hybrid">Hybrid</option>
                  <option value="semantic">Semantic</option>
                  <option value="keyword">Keyword</option>
                </select>
                <Button
                  onClick={handleSearch}
                  disabled={isSearching || !query.trim() || !selectedCodebase}
                >
                  {isSearching ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Search className="h-4 w-4" />
                  )}
                  Search
                </Button>
              </div>
            </div>
          </div>

          {hasSearched && (
            <div className="space-y-3">
              <div className="text-sm text-muted-foreground">
                {isSearching
                  ? "Searching..."
                  : searchResults.length > 0
                    ? `Found ${searchResults.length} results`
                    : "No results found"}
              </div>

              {searchResults.map((result, idx) => (
                <div
                  key={`${result.filePath}-${result.startLine}`}
                  className="bg-card border border-border rounded-xl p-4 hover:bg-muted/30 transition-colors"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex items-center gap-2 min-w-0">
                      <FileCode className="h-4 w-4 text-primary flex-shrink-0 mt-0.5" />
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm text-foreground truncate">
                            {result.filePath}
                          </span>
                          <span className="text-xs text-muted-foreground">
                            :{result.startLine}-{result.endLine}
                          </span>
                        </div>
                        {result.name && (
                          <div className="text-xs text-blue-400 truncate">
                            {result.name}
                          </div>
                        )}
                      </div>
                    </div>
                    <div className="text-sm font-medium text-primary flex-shrink-0">
                      {(result.score * 100).toFixed(1)}%
                    </div>
                  </div>

                  {result.snippet && (
                    <pre className="mt-3 p-3 bg-muted/30 rounded-lg overflow-x-auto text-xs font-mono text-muted-foreground whitespace-pre-wrap">
                      {result.snippet}
                    </pre>
                  )}
                </div>
              ))}
            </div>
          )}

          {!hasSearched && !isLoading && (
            <EmptyState
              icon={<Search className="h-12 w-12" />}
              title="Search your codebases"
              description="Enter a query and select a codebase to search. Supports semantic, keyword, or hybrid search modes."
            />
          )}

          {error && <ErrorState title="Failed to load codebases" />}

          {isLoading && <LoadingState />}
        </div>
      </div>
    </AppContainer>
  );
}
