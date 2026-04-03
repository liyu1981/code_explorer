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
  indexedAt: number;
  fileCount: number;
}

interface LineMatch {
  line: string;
  lineNumber: number;
  lineStart: number;
  lineEnd: number;
  contentBefore: string;
  contentAfter: string;
}

interface FileMatch {
  fileName: string;
  repository: string;
  branch: string;
  content: string;
  lineMatches: LineMatch[];
  score: number;
}

interface SearchStats {
  duration: number;
  filesExamined: number;
  filesMatched: number;
  shards: number;
}

interface SearchResult {
  files: FileMatch[];
  stats: SearchStats;
}

export default function ZoektQueryPage() {
  const [query, setQuery] = useState("");
  const [selectedCodebase, setSelectedCodebase] = useState<string>("");
  const [searchResults, setSearchResults] = useState<SearchResult | null>(null);
  const [isSearching, setIsSearching] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);
  const [expandedFiles, setExpandedFiles] = useState<Set<number>>(new Set());

  const {
    data: codebases,
    error,
    isLoading,
  } = useSWR<CodebaseInfo[]>("/api/zoekt/codebases", fetcher);

  const indexedCodebases = codebases?.filter((cb) => cb.indexedAt > 0) || [];

  const handleSearch = async () => {
    if (!query.trim() || !selectedCodebase) return;

    setIsSearching(true);
    setHasSearched(true);
    try {
      const res = await api.post<SearchResult>("/api/zoekt/search", {
        query: query.trim(),
        codebaseID: selectedCodebase,
        limit: 50,
      });
      setSearchResults(res.data);
      setExpandedFiles(new Set());
    } catch (err) {
      console.error("Zoekt search failed:", err);
      setSearchResults(null);
    } finally {
      setIsSearching(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  const toggleFile = (idx: number) => {
    const next = new Set(expandedFiles);
    if (next.has(idx)) {
      next.delete(idx);
    } else {
      next.add(idx);
    }
    setExpandedFiles(next);
  };

  const highlightLine = (line: string, pattern: string) => {
    if (!pattern) return line;
    const lowerLine = line.toLowerCase();
    const lowerPattern = pattern.toLowerCase();
    const idx = lowerLine.indexOf(lowerPattern);
    if (idx === -1) return line;
    const before = line.slice(0, idx);
    const match = line.slice(idx, idx + pattern.length);
    const after = line.slice(idx + pattern.length);
    return (
      <>
        {before}
        <mark className="bg-yellow-200 dark:bg-yellow-800 rounded px-0.5">
          {match}
        </mark>
        {after}
      </>
    );
  };

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center gap-4">
          <Search className="h-5 w-5 text-purple-500" />
          <h1 className="text-xl font-bold tracking-tight text-primary">
            Zoekt Query
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
                    placeholder="Enter trigram search query (e.g., 'func main')"
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

          {hasSearched && searchResults && (
            <div className="space-y-3">
              <div className="flex items-center justify-between text-sm text-muted-foreground">
                <span>
                  {isSearching
                    ? "Searching..."
                    : (searchResults.files?.length ?? 0) > 0
                      ? `Found ${searchResults.files.length} file(s) in ${searchResults.stats?.shards ?? 0} shard(s)`
                      : "No results found"}
                </span>
                {!isSearching && (searchResults.stats?.duration ?? 0) > 0 && (
                  <span className="text-xs">
                    {(searchResults.stats?.duration ?? 0).toFixed(3)}s ·{" "}
                    {searchResults.stats?.filesExamined ?? 0} files examined
                  </span>
                )}
              </div>

              {(searchResults.files ?? []).map((file, idx) => (
                <div
                  // biome-ignore lint/suspicious/noArrayIndexKey: stable unique key from file data
                  key={`file-${file.fileName}-${file.lineMatches.length}-${file.score}`}
                  className="bg-card border border-border rounded-xl overflow-hidden hover:bg-muted/30 transition-colors"
                >
                  <button
                    onClick={() => toggleFile(idx)}
                    className="w-full flex items-center justify-between p-4 text-left"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <FileCode className="h-4 w-4 text-purple-500 flex-shrink-0 mt-0.5" />
                      <div className="min-w-0">
                        <span className="font-mono text-sm text-foreground truncate block">
                          {file.fileName}
                        </span>
                        {file.lineMatches.length > 0 && (
                          <span className="text-xs text-muted-foreground">
                            {file.lineMatches.length} match
                            {file.lineMatches.length !== 1 ? "es" : ""}
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      <span className="text-sm font-medium text-primary">
                        {(file.score * 100).toFixed(1)}%
                      </span>
                      <X
                        className={`h-4 w-4 text-muted-foreground transition-transform ${expandedFiles.has(idx) ? "rotate-45" : ""}`}
                      />
                    </div>
                  </button>

                  {expandedFiles.has(idx) && file.lineMatches.length > 0 && (
                    <div className="border-t border-border">
                      {file.lineMatches.map((lm, lIdx) => (
                        <div
                          // biome-ignore lint/suspicious/noArrayIndexKey: stable unique key from line data
                          key={`${file.fileName}-line-${lm.lineNumber}`}
                          className="px-4 py-2 bg-muted/20 border-b border-border last:border-b-0"
                        >
                          <div className="flex items-start gap-3">
                            <span className="text-xs text-muted-foreground font-mono pt-0.5 flex-shrink-0 w-8 text-right">
                              {lm.lineNumber}
                            </span>
                            <pre className="text-xs font-mono text-muted-foreground whitespace-pre-wrap overflow-x-auto flex-1">
                              {highlightLine(lm.line, query)}
                            </pre>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {hasSearched && !searchResults && (
            <ErrorState title="Search failed" />
          )}

          {!hasSearched && !isLoading && (
            <EmptyState
              icon={<Search className="h-12 w-12" />}
              title="Zoekt Trigram Search"
              description="Fast trigram-based code search. Enter a query and select a codebase to search. Click on results to expand line matches."
            />
          )}

          {error && <ErrorState title="Failed to load codebases" />}

          {isLoading && <LoadingState />}
        </div>
      </div>
    </AppContainer>
  );
}
