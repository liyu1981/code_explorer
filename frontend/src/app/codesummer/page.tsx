"use client";

import { useState } from "react";
import {
  Sun,
  FileCode,
  Folder,
  Loader2,
  X,
  ChevronRight,
  Hash,
  Code,
  GitBranch,
  Database,
  ArrowRightLeft,
} from "lucide-react";
import { cn } from "@/lib/utils";
import useSWR from "swr";
import { fetcher } from "@/lib/api";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { LoadingState } from "../_components/loading-state";
import { ErrorState } from "../_components/error-state";
import { EmptyState } from "../_components/empty-state";
import { Button } from "@/components/ui/button";
import * as Dialog from "@radix-ui/react-dialog";
import ReactMarkdown from "react-markdown";

interface Codebase {
  id: string;
  name: string;
  root_path: string;
}

interface CodesummerSummary {
  id: string;
  codesummerId: string;
  nodePath: string;
  nodeType: string;
  language: string;
  summary: string;
  definitions: any;
  dependencies: any;
  dataManipulated: any;
  dataFlow: any;
  indexedAt: number;
}

interface CodesummerResponse {
  summaries: CodesummerSummary[];
  total: number;
  indexedAt: number;
}

export default function CodesummerPage() {
  const [selectedCodebase, setSelectedCodebase] = useState<Codebase | null>(
    null,
  );
  const [selectedSummary, setSelectedSummary] =
    useState<CodesummerSummary | null>(null);
  const [filterType, setFilterType] = useState<string>("all");

  const { data: codebases } = useSWR<Codebase[]>(
    "/api/codemogger/codebases",
    fetcher,
  );

  const {
    data: summaryData,
    error: summaryError,
    isLoading: summaryLoading,
  } = useSWR<CodesummerResponse>(
    selectedCodebase
      ? `/api/codesummer/summaries?codebase_id=${selectedCodebase.id}`
      : null,
    fetcher,
    { refreshInterval: 10000 },
  );

  const summaries = summaryData?.summaries || [];
  const filteredSummaries =
    filterType === "all"
      ? summaries
      : summaries.filter((s) => s.nodeType === filterType);

  const getNodeIcon = (nodeType: string) => {
    switch (nodeType) {
      case "file":
        return <FileCode className="h-4 w-4 text-primary" />;
      case "directory":
        return <Folder className="h-4 w-4 text-amber-500" />;
      default:
        return <FileCode className="h-4 w-4 text-muted-foreground" />;
    }
  };

  const getNodeTypeBadge = (nodeType: string) => {
    const classes =
      nodeType === "directory"
        ? "bg-amber-500/10 text-amber-500 border-amber-500/20"
        : "bg-primary/10 text-primary border-primary/20";
    return cn(
      "inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider border",
      classes,
    );
  };

  const formatDate = (timestamp: number) => {
    if (!timestamp) return "Never";
    return new Date(timestamp * 1000).toLocaleString();
  };

  const isSimpleArray = (data: any): data is string[] => {
    return (
      Array.isArray(data) && data.length > 0 && typeof data[0] === "string"
    );
  };

  const renderArray = (data: any[]) => {
    if (isSimpleArray(data)) {
      return data.join(", ");
    }
    return (
      <div className="space-y-2">
        {data.map((item, idx) => (
          <div
            key={idx}
            className="bg-background/50 rounded-lg p-3 border border-border/30"
          >
            {typeof item === "object" && item !== null ? (
              <div className="space-y-1">
                {Object.entries(item).map(([key, value]) => (
                  <div key={key} className="flex items-start gap-2">
                    <span className="text-xs font-bold text-primary uppercase min-w-[80px]">
                      {key}:
                    </span>
                    <span className="text-xs font-mono text-foreground break-all">
                      {typeof value === "object"
                        ? JSON.stringify(value)
                        : String(value)}
                    </span>
                  </div>
                ))}
              </div>
            ) : (
              <span className="text-sm font-mono text-foreground">
                {String(item)}
              </span>
            )}
          </div>
        ))}
      </div>
    );
  };

  const renderData = (data: any) => {
    if (!data) return "None";
    if (Array.isArray(data)) {
      return data.length > 0 ? renderArray(data) : "None";
    }
    if (typeof data === "object") {
      return (
        <pre className="text-xs font-mono text-foreground whitespace-pre-wrap">
          {JSON.stringify(data, null, 2)}
        </pre>
      );
    }
    return String(data);
  };

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center gap-4">
            <Sun className="h-5 w-5 text-amber-500" />
            <h1 className="text-xl font-bold tracking-tight text-primary">
              Code Summer
            </h1>
          </div>
          {selectedCodebase && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setSelectedCodebase(null);
                setSelectedSummary(null);
              }}
            >
              <X className="h-4 w-4" />
              Close
            </Button>
          )}
        </div>
      </AppHeader>

      <div className="flex-1 overflow-auto p-6 bg-background/50">
        {!selectedCodebase ? (
          <div className="max-w-4xl mx-auto w-full">
            <h2 className="text-sm font-semibold text-muted-foreground mb-4">
              Select a codebase to view Code Summer summaries
            </h2>
            {!codebases ? (
              <LoadingState />
            ) : codebases.length === 0 ? (
              <EmptyState
                icon={<Sun className="h-12 w-12" />}
                title="No codebases available"
                description="Add a codebase first to view its Code Summer summaries."
              />
            ) : (
              <div className="grid gap-3">
                {codebases.map((cb) => (
                  <button
                    key={cb.id}
                    onClick={() => setSelectedCodebase(cb)}
                    className="flex items-center justify-between p-4 bg-card border border-border rounded-xl hover:bg-muted/30 transition-colors text-left"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
                        <Sun className="h-5 w-5 text-primary" />
                      </div>
                      <div>
                        <h3 className="font-bold text-foreground">{cb.name}</h3>
                        <p className="text-xs text-muted-foreground font-mono truncate max-w-md">
                          {cb.root_path}
                        </p>
                      </div>
                    </div>
                    <ChevronRight className="h-5 w-5 text-muted-foreground" />
                  </button>
                ))}
              </div>
            )}
          </div>
        ) : (
          <div className="max-w-7xl mx-auto w-full">
            <div className="flex items-center justify-between mb-6">
              <div className="flex items-center gap-3">
                <span className="px-2 py-0.5 rounded-md bg-muted text-[10px] font-bold text-muted-foreground uppercase tracking-widest">
                  {selectedCodebase.name}
                </span>
                <span className="text-muted-foreground text-sm">
                  {summaryData?.total || 0} summaries • Indexed:{" "}
                  {formatDate(summaryData?.indexedAt || 0)}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-muted-foreground mr-2">
                  Filter:
                </span>
                {["all", "file", "directory"].map((type) => (
                  <button
                    key={type}
                    onClick={() => setFilterType(type)}
                    className={cn(
                      "px-3 py-1 rounded-lg text-xs font-bold uppercase tracking-wider transition-colors",
                      filterType === type
                        ? "bg-primary text-primary-foreground"
                        : "bg-muted text-muted-foreground hover:bg-muted/80",
                    )}
                  >
                    {type}
                  </button>
                ))}
              </div>
            </div>

            {summaryLoading ? (
              <LoadingState />
            ) : summaryError ? (
              <ErrorState title="Failed to load summaries" />
            ) : filteredSummaries.length === 0 ? (
              <EmptyState
                icon={<Sun className="h-12 w-12" />}
                title="No summaries found"
                description="Run the Code Summer build to generate summaries for this codebase."
              />
            ) : (
              <div className="bg-card border border-border rounded-2xl overflow-hidden shadow-sm">
                <div className="overflow-x-auto">
                  <table className="w-full text-left border-collapse">
                    <thead>
                      <tr className="bg-muted/30 border-b border-border">
                        <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                          Path
                        </th>
                        <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                          Type
                        </th>
                        <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                          Language
                        </th>
                        <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                          Summary
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border/50">
                      {filteredSummaries.map((summary) => (
                        <tr
                          key={summary.id}
                          onClick={() => setSelectedSummary(summary)}
                          className="hover:bg-muted/10 transition-colors cursor-pointer"
                        >
                          <td className="px-6 py-4">
                            <div className="flex items-center gap-2">
                              {getNodeIcon(summary.nodeType)}
                              <span className="font-mono text-sm text-foreground truncate max-w-md">
                                {summary.nodePath}
                              </span>
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            {getNodeTypeBadge(summary.nodeType)}
                          </td>
                          <td className="px-6 py-4">
                            <span className="text-sm text-muted-foreground uppercase font-mono">
                              {summary.language || "-"}
                            </span>
                          </td>
                          <td className="px-6 py-4">
                            <p className="text-sm text-muted-foreground line-clamp-2 max-w-xl">
                              {summary.summary || "No summary available"}
                            </p>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      <Dialog.Root
        open={!!selectedSummary}
        onOpenChange={() => setSelectedSummary(null)}
      >
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50" />
          <Dialog.Content className="fixed inset-4 md:inset-10 lg:inset-20 bg-card border border-border shadow-2xl rounded-3xl z-50 flex flex-col overflow-hidden">
            <div className="flex items-center justify-between p-6 border-b border-border">
              <div className="flex items-center gap-3">
                {selectedSummary && getNodeIcon(selectedSummary.nodeType)}
                <div>
                  <Dialog.Title className="text-lg font-bold text-foreground">
                    {selectedSummary?.nodePath.split("/").pop()}
                  </Dialog.Title>
                  <p className="text-xs text-muted-foreground font-mono">
                    {selectedSummary?.nodePath}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                {selectedSummary && getNodeTypeBadge(selectedSummary.nodeType)}
                <Dialog.Close asChild>
                  <Button variant="ghost" size="icon">
                    <X className="h-5 w-5" />
                  </Button>
                </Dialog.Close>
              </div>
            </div>

            <div className="flex-1 overflow-auto p-6">
              {selectedSummary && (
                <div className="space-y-8">
                  <div>
                    <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground mb-3 flex items-center gap-2">
                      <Code className="h-4 w-4" />
                      Summary
                    </h3>
                    <div className="bg-muted/30 rounded-xl p-4 border border-border/50">
                      <div className="prose prose-sm dark:prose-invert max-w-none">
                        <ReactMarkdown>
                          {selectedSummary.summary || "No summary available"}
                        </ReactMarkdown>
                      </div>
                    </div>
                  </div>

                  {selectedSummary.definitions &&
                    selectedSummary.definitions.length > 0 && (
                      <div>
                        <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground mb-3 flex items-center gap-2">
                          <Hash className="h-4 w-4" />
                          Definitions (
                          {Array.isArray(selectedSummary.definitions)
                            ? selectedSummary.definitions.length
                            : 0}
                          )
                        </h3>
                        <div className="bg-muted/30 rounded-xl p-4 border border-border/50">
                          {renderData(selectedSummary.definitions)}
                        </div>
                      </div>
                    )}

                  {selectedSummary.dependencies &&
                    selectedSummary.dependencies.length > 0 && (
                      <div>
                        <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground mb-3 flex items-center gap-2">
                          <GitBranch className="h-4 w-4" />
                          Dependencies (
                          {Array.isArray(selectedSummary.dependencies)
                            ? selectedSummary.dependencies.length
                            : 0}
                          )
                        </h3>
                        <div className="bg-muted/30 rounded-xl p-4 border border-border/50">
                          {renderData(selectedSummary.dependencies)}
                        </div>
                      </div>
                    )}

                  {selectedSummary.dataManipulated &&
                    selectedSummary.dataManipulated.length > 0 && (
                      <div>
                        <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground mb-3 flex items-center gap-2">
                          <Database className="h-4 w-4" />
                          Data Manipulated (
                          {Array.isArray(selectedSummary.dataManipulated)
                            ? selectedSummary.dataManipulated.length
                            : 0}
                          )
                        </h3>
                        <div className="bg-muted/30 rounded-xl p-4 border border-border/50">
                          {renderData(selectedSummary.dataManipulated)}
                        </div>
                      </div>
                    )}

                  {selectedSummary.dataFlow &&
                    selectedSummary.dataFlow.length > 0 && (
                      <div>
                        <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground mb-3 flex items-center gap-2">
                          <ArrowRightLeft className="h-4 w-4" />
                          Data Flow (
                          {Array.isArray(selectedSummary.dataFlow)
                            ? selectedSummary.dataFlow.length
                            : 0}
                          )
                        </h3>
                        <div className="bg-muted/30 rounded-xl p-4 border border-border/50">
                          {renderData(selectedSummary.dataFlow)}
                        </div>
                      </div>
                    )}
                </div>
              )}
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </AppContainer>
  );
}
