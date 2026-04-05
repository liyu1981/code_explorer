"use client";

import { FileCode, Loader2, Search, Code2, Database } from "lucide-react";
import { useState } from "react";
import useSWR from "swr";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { api, fetcher } from "@/lib/api";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { EmptyState } from "../_components/empty-state";
import { ErrorState } from "../_components/error-state";
import { LoadingState } from "../_components/loading-state";

type SearchEngine = "codemogger" | "zoekt" | "both";

interface Codebase {
  id: string;
  name: string;
  rootPath: string;
  type: string;
  version: string;
  createdAt: number;
}

interface ZoektLineMatch {
  line: string;
  lineNumber: number;
  lineStart: number;
  lineEnd: number;
  contentBefore: string;
  contentAfter: string;
}

interface ZoektFileMatch {
  fileName: string;
  repository: string;
  branch: string;
  content: string;
  lineMatches: ZoektLineMatch[];
  score: number;
}

interface ZoektSearchStats {
  duration: number;
  filesExamined: number;
  filesMatched: number;
  shards: number;
}

interface ZoektSearchResult {
  files: ZoektFileMatch[];
  stats: ZoektSearchStats;
}

interface CodemoggerSearchResult {
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

export default function CodeSearchPage() {
  const [query, setQuery] = useState("");
  const [searchEngine, setSearchEngine] = useState<SearchEngine>("both");
  const [selectedCodebase, setSelectedCodebase] = useState<string>("");
  const [codemoggerSearchMode, setCodemoggerSearchMode] = useState<
    "semantic" | "keyword" | "hybrid"
  >("hybrid");
  const [zoektResults, setZoektResults] = useState<ZoektSearchResult | null>(
    null,
  );
  const [codemoggerResults, setCodemoggerResults] = useState<
    CodemoggerSearchResult[]
  >([]);
  const [isSearching, setIsSearching] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);

  const {
    data: codebases,
    error,
    isLoading,
  } = useSWR<Codebase[]>("/api/codebases", fetcher);

  const handleSearch = async () => {
    if (!query.trim() || !selectedCodebase) return;

    setIsSearching(true);
    setHasSearched(true);
    setZoektResults(null);
    setCodemoggerResults([]);

    const searches: Promise<void>[] = [];

    if (searchEngine === "zoekt" || searchEngine === "both") {
      searches.push(
        (async () => {
          try {
            const res = await api.post<ZoektSearchResult>("/api/zoekt/search", {
              query: query.trim(),
              codebaseID: selectedCodebase,
              limit: 50,
            });
            setZoektResults(res.data);
          } catch (err) {
            console.error("Zoekt search failed:", err);
            setZoektResults(null);
          }
        })(),
      );
    }

    if (searchEngine === "codemogger" || searchEngine === "both") {
      searches.push(
        (async () => {
          try {
            const res = await api.post<CodemoggerSearchResult[]>(
              "/api/codemogger/search",
              {
                query: query.trim(),
                codebaseID: selectedCodebase,
                mode: codemoggerSearchMode,
                limit: 20,
              },
            );
            setCodemoggerResults(res.data);
          } catch (err) {
            console.error("Codemogger search failed:", err);
            setCodemoggerResults([]);
          }
        })(),
      );
    }

    await Promise.all(searches);
    setIsSearching(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
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

  const canSearch = query.trim() && selectedCodebase && !isSearching;

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center gap-4">
          <Search className="h-5 w-5 text-primary" />
          <h1 className="text-xl font-bold tracking-tight text-primary">
            Code Search
          </h1>
        </div>
      </AppHeader>

      <div className="flex-1 overflow-auto p-6 bg-background/50">
        <div className="max-w-4xl mx-auto w-full space-y-6">
          <div className="bg-card border border-border rounded-2xl p-6 space-y-4">
            <div className="flex flex-col gap-4">
              <div className="flex gap-2">
                <button
                  onClick={() => setSearchEngine("codemogger")}
                  className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium border transition-colors ${
                    searchEngine === "codemogger"
                      ? "bg-blue-500/10 border-blue-500/50 text-blue-500"
                      : "bg-background border-border text-muted-foreground hover:text-foreground"
                  }`}
                >
                  <Code2 className="h-4 w-4" />
                  Codemogger
                </button>
                <button
                  onClick={() => setSearchEngine("zoekt")}
                  className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium border transition-colors ${
                    searchEngine === "zoekt"
                      ? "bg-purple-500/10 border-purple-500/50 text-purple-500"
                      : "bg-background border-border text-muted-foreground hover:text-foreground"
                  }`}
                >
                  <Database className="h-4 w-4" />
                  Zoekt
                </button>
                <button
                  onClick={() => setSearchEngine("both")}
                  className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium border transition-colors ${
                    searchEngine === "both"
                      ? "bg-primary/10 border-primary/50 text-primary"
                      : "bg-background border-border text-muted-foreground hover:text-foreground"
                  }`}
                >
                  <Search className="h-4 w-4" />
                  Both
                </button>
              </div>

              <div className="flex gap-3">
                <div className="flex-1">
                  <input
                    type="text"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder={
                      searchEngine === "zoekt"
                        ? "Enter trigram search query (e.g., 'func main')"
                        : searchEngine === "codemogger"
                          ? "Enter search query (e.g., 'how is authentication implemented')"
                          : "Enter search query"
                    }
                    className="w-full px-4 py-2.5 bg-background border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
                  />
                </div>

                <select
                  value={selectedCodebase}
                  onChange={(e) => setSelectedCodebase(e.target.value)}
                  className="px-3 py-2 bg-background border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
                >
                  <option value="">Select codebase...</option>
                  {codebases?.map((cb) => (
                    <option key={cb.id} value={cb.id}>
                      {cb.name}
                    </option>
                  ))}
                </select>
                <select
                  value={codemoggerSearchMode}
                  onChange={(e) =>
                    setCodemoggerSearchMode(
                      e.target.value as "semantic" | "keyword" | "hybrid",
                    )
                  }
                  className="px-3 py-2 bg-background border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
                >
                  <option value="hybrid">Hybrid</option>
                  <option value="semantic">Semantic</option>
                  <option value="keyword">Keyword</option>
                </select>

                <Button onClick={handleSearch} disabled={!canSearch}>
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
            <Tabs key={searchEngine} defaultValue={searchEngine === "both" ? "codemogger" : searchEngine}>
              <TabsList>
                {(searchEngine === "codemogger" || searchEngine === "both") && (
                  <TabsTrigger value="codemogger">
                    <Code2 className="h-4 w-4" />
                    Codemogger
                    {!isSearching && (
                      <span className="text-xs text-muted-foreground ml-1">
                        ({codemoggerResults.length})
                      </span>
                    )}
                  </TabsTrigger>
                )}
                {(searchEngine === "zoekt" || searchEngine === "both") && (
                  <TabsTrigger value="zoekt">
                    <Database className="h-4 w-4" />
                    Zoekt
                    {!isSearching && zoektResults && (
                      <span className="text-xs text-muted-foreground ml-1">
                        ({zoektResults.files?.length ?? 0})
                      </span>
                    )}
                  </TabsTrigger>
                )}
              </TabsList>

              {(searchEngine === "codemogger" || searchEngine === "both") && (
                <TabsContent value="codemogger">
                  <div className="space-y-3 mt-4">
                    {codemoggerResults.map((result) => (
                      <div
                        key={`${result.chunkKey}-${result.filePath}-${result.startLine}`}
                        className="bg-card border border-border rounded-xl p-4 hover:bg-muted/30 transition-colors"
                      >
                        <div className="flex items-start justify-between gap-4">
                          <div className="flex items-center gap-2 min-w-0">
                            <FileCode className="h-4 w-4 text-blue-500 flex-shrink-0 mt-0.5" />
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

                    {!isSearching && codemoggerResults.length === 0 && (
                      <div className="text-center py-12 text-muted-foreground text-sm">
                        No results found
                      </div>
                    )}
                  </div>
                </TabsContent>
              )}

              {(searchEngine === "zoekt" || searchEngine === "both") && (
                <TabsContent value="zoekt">
                  <div className="space-y-3 mt-4">
                    {zoektResults &&
                      !isSearching &&
                      zoektResults.stats?.duration > 0 && (
                        <div className="text-xs text-muted-foreground">
                          {zoektResults.stats.duration.toFixed(3)}s ·{" "}
                          {zoektResults.stats.filesExamined} files examined ·{" "}
                          {zoektResults.stats.shards} shard(s)
                        </div>
                      )}

                    {zoektResults &&
                      (zoektResults.files ?? []).map((file) => {
                        const dedupedMatches = file.lineMatches.filter(
                          (lm, i, arr) =>
                            arr.findIndex(
                              (x) =>
                                x.lineStart === lm.lineStart &&
                                x.lineEnd === lm.lineEnd,
                            ) === i,
                        );

                        return (
                        <div
                          key={`file-${file.fileName}-${file.lineMatches.length}-${file.score}`}
                          className="bg-card border border-border rounded-xl overflow-hidden hover:bg-muted/30 transition-colors"
                        >
                          <div className="flex items-center justify-between p-4 pb-2">
                            <div className="flex items-center gap-2 min-w-0">
                              <FileCode className="h-4 w-4 text-purple-500 flex-shrink-0 mt-0.5" />
                              <div className="min-w-0">
                                <span className="font-mono text-sm text-foreground truncate block">
                                  {file.fileName}
                                </span>
                                {dedupedMatches.length > 0 && (
                                  <span className="text-xs text-muted-foreground">
                                    {dedupedMatches.length} match
                                    {dedupedMatches.length !== 1 ? "es" : ""}
                                  </span>
                                )}
                              </div>
                            </div>
                            <div className="flex items-center gap-2 flex-shrink-0">
                              <span className="text-sm font-medium text-primary">
                                {(file.score * 100).toFixed(1)}%
                              </span>
                            </div>
                          </div>

                          {dedupedMatches.length > 0 && (
                            <div className="border-t border-border">
                              {dedupedMatches.map((lm) => (
                                <div
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
                      );
                    })}

                    {!isSearching &&
                      zoektResults &&
                      (zoektResults.files ?? []).length === 0 && (
                        <div className="text-center py-12 text-muted-foreground text-sm">
                          No results found
                        </div>
                      )}
                  </div>
                </TabsContent>
              )}
            </Tabs>
          )}

          {!hasSearched && !isLoading && (
            <EmptyState
              icon={<Search className="h-12 w-12" />}
              title="Code Search"
              description="Search your codebases using Codemogger (semantic/keyword), Zoekt (trigram), or both engines simultaneously."
            />
          )}

          {error && <ErrorState title="Failed to load codebases" />}

          {isLoading && <LoadingState />}
        </div>
      </div>
    </AppContainer>
  );
}
