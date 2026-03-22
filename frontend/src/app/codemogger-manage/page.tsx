"use client";

import { useState } from "react";
import {
  Trash2,
  Folder,
  Loader2,
  X,
  ChevronRight,
  AlertTriangle,
  Database,
  FileCode,
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

export default function CodebaseManagementPage() {
  const [selectedCodebase, setSelectedCodebase] = useState<CodebaseInfo | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [deleteSuccess, setDeleteSuccess] = useState(false);

  const { data: codebases, error, isLoading, mutate } = useSWR<CodebaseInfo[]>(
    "/api/codemogger/codebases",
    fetcher,
  );

  const formatDate = (timestamp: number) => {
    if (!timestamp) return "Never";
    return new Date(timestamp * 1000).toLocaleString();
  };

  const formatNumber = (n: number) => {
    return n.toLocaleString();
  };

  const handleDelete = async () => {
    if (!selectedCodebase) return;

    setIsDeleting(true);
    try {
      await api.delete(`/api/codemogger/codebases?codebase_id=${selectedCodebase.id}`);
      setDeleteSuccess(true);
      mutate();
      setTimeout(() => {
        setSelectedCodebase(null);
        setDeleteSuccess(false);
      }, 1500);
    } catch (err) {
      console.error("Failed to delete codemogger entries:", err);
    } finally {
      setIsDeleting(false);
    }
  };

  const indexedCodebases = codebases?.filter(cb => cb.indexedAt > 0) || [];

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center gap-4">
            <Database className="h-5 w-5 text-red-500" />
            <h1 className="text-xl font-bold tracking-tight text-primary">
              Codebase Management
            </h1>
          </div>
          {selectedCodebase && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setSelectedCodebase(null);
                setDeleteSuccess(false);
              }}
            >
              <X className="h-4 w-4" />
              Close
            </Button>
          )}
        </div>
      </AppHeader>

      <div className="flex-1 overflow-auto p-6 bg-background/50">
        <div className="max-w-4xl mx-auto w-full">
          {!selectedCodebase ? (
            <>
              <h2 className="text-sm font-semibold text-muted-foreground mb-4">
                Manage indexed codebases. Delete entries to reindex fresh.
              </h2>
              {isLoading ? (
                <LoadingState />
              ) : error ? (
                <ErrorState title="Failed to load codebases" />
              ) : indexedCodebases.length === 0 ? (
                <EmptyState
                  icon={<Database className="h-12 w-12" />}
                  title="No indexed codebases"
                  description="Index a codebase first to manage its entries here."
                />
              ) : (
                <div className="grid gap-3">
                  {indexedCodebases.map((cb) => (
                    <div
                      key={cb.id}
                      className="flex items-center justify-between p-4 bg-card border border-border rounded-xl hover:bg-muted/30 transition-colors"
                    >
                      <button
                        onClick={() => setSelectedCodebase(cb)}
                        className="flex items-center gap-3 flex-1 text-left"
                      >
                        <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
                          <FileCode className="h-5 w-5 text-primary" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <h3 className="font-bold text-foreground">{cb.name}</h3>
                          <p className="text-xs text-muted-foreground font-mono truncate max-w-md">
                            {cb.rootPath}
                          </p>
                          <div className="flex items-center gap-4 mt-1 text-xs text-muted-foreground">
                            <span>{formatNumber(cb.fileCount)} files</span>
                            <span>{formatNumber(cb.chunkCount)} chunks</span>
                            <span>Indexed: {formatDate(cb.indexedAt)}</span>
                          </div>
                        </div>
                      </button>
                      <ChevronRight className="h-5 w-5 text-muted-foreground" />
                    </div>
                  ))}
                </div>
              )}
            </>
          ) : (
            <div className="space-y-6">
              <div className="bg-card border border-border rounded-2xl p-6 space-y-4">
                <div className="flex items-center gap-3">
                  <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center">
                    <FileCode className="h-6 w-6 text-primary" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-foreground">{selectedCodebase.name}</h3>
                    <p className="text-sm text-muted-foreground font-mono">
                      {selectedCodebase.rootPath}
                    </p>
                  </div>
                </div>

                <div className="grid grid-cols-3 gap-4 pt-4 border-t border-border">
                  <div className="text-center p-3 bg-muted/30 rounded-xl">
                    <div className="text-2xl font-bold text-primary">
                      {formatNumber(selectedCodebase.fileCount)}
                    </div>
                    <div className="text-xs text-muted-foreground uppercase tracking-wider">
                      Files
                    </div>
                  </div>
                  <div className="text-center p-3 bg-muted/30 rounded-xl">
                    <div className="text-2xl font-bold text-primary">
                      {formatNumber(selectedCodebase.chunkCount)}
                    </div>
                    <div className="text-xs text-muted-foreground uppercase tracking-wider">
                      Chunks
                    </div>
                  </div>
                  <div className="text-center p-3 bg-muted/30 rounded-xl">
                    <div className="text-sm font-bold text-muted-foreground">
                      {formatDate(selectedCodebase.indexedAt)}
                    </div>
                    <div className="text-xs text-muted-foreground uppercase tracking-wider">
                      Last Indexed
                    </div>
                  </div>
                </div>
              </div>

              <Dialog.Root>
                <Dialog.Trigger asChild>
                  <Button
                    variant="destructive"
                    className="w-full"
                    disabled={deleteSuccess}
                  >
                    {deleteSuccess ? (
                      <>
                        <X className="h-4 w-4 mr-2" />
                        Deleted
                      </>
                    ) : (
                      <>
                        <Trash2 className="h-4 w-4 mr-2" />
                        Delete Codemogger Entries
                      </>
                    )}
                  </Button>
                </Dialog.Trigger>
                <Dialog.Portal>
                  <Dialog.Overlay className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50" />
                  <Dialog.Content className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-full max-w-md bg-card border border-border shadow-2xl rounded-2xl z-50 p-6">
                    <Dialog.Title className="text-lg font-bold text-foreground flex items-center gap-2">
                      <AlertTriangle className="h-5 w-5 text-destructive" />
                      Delete Codemogger Entries
                    </Dialog.Title>
                    <Dialog.Description className="text-sm text-muted-foreground mt-2">
                      This will delete all indexed files, chunks, and embeddings for this codebase.
                      The codebase registry entry will be kept. You can reindex later.
                    </Dialog.Description>

                    <div className="bg-muted/30 rounded-xl p-4 mt-4 border border-border">
                      <div className="flex items-center gap-2 text-sm">
                        <Folder className="h-4 w-4 text-muted-foreground" />
                        <span className="font-mono truncate">{selectedCodebase.rootPath}</span>
                      </div>
                      <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
                        <span>{selectedCodebase.fileCount} files</span>
                        <span>{selectedCodebase.chunkCount} chunks</span>
                      </div>
                    </div>

                    <div className="flex gap-3 mt-6">
                      <Dialog.Close asChild>
                        <Button variant="outline" className="flex-1">
                          Cancel
                        </Button>
                      </Dialog.Close>
                      <Button
                        variant="destructive"
                        className="flex-1"
                        onClick={handleDelete}
                        disabled={isDeleting}
                      >
                        {isDeleting ? (
                          <>
                            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                            Deleting...
                          </>
                        ) : (
                          <>
                            <Trash2 className="h-4 w-4 mr-2" />
                            Delete
                          </>
                        )}
                      </Button>
                    </div>
                  </Dialog.Content>
                </Dialog.Portal>
              </Dialog.Root>
            </div>
          )}
        </div>
      </div>
    </AppContainer>
  );
}
