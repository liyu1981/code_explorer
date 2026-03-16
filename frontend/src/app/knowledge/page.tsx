"use client";

import { useAtom } from "jotai";
import {
  Book,
  ChevronRight,
  Hash,
  Loader2,
  X,
  Copy,
  RefreshCw,
  Plus,
} from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";
import { toast } from "sonner";
import useSWR from "swr";
import { api, fetcher } from "@/lib/api";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { LoadingState } from "../_components/loading-state";
import { ErrorState } from "../_components/error-state";
import { KnowledgeViewer } from "../knowledge/_components/knowledge-viewer";
import { cn } from "@/lib/utils";
import { activeKnowledgePagesAtom } from "../_jotai/ui-store";
import { Button } from "@/components/ui/button";

interface KnowledgePage {
  id: string;
  codebase_id: string;
  slug: string;
  title: string;
  content: string;
  build_instructions: string;
  created_at: string;
  updated_at: string;
}

interface Codebase {
  id: string;
  name: string;
  root_path: string;
}

function KnowledgeContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const cbid = searchParams.get("cbid");
  const slug = searchParams.get("slug");

  const [, setActiveKnowledgePages] = useAtom(activeKnowledgePagesAtom);
  const [activeSlug, setActiveSlug] = useState<string | null>(null);
  const [isBuilding, setIsBuilding] = useState(false);

  const {
    data: page,
    error,
    isLoading,
    mutate,
  } = useSWR<KnowledgePage>(
    cbid
      ? `/api/knowledge/get?codebase_id=${cbid}&slug=${slug || "overview"}`
      : null,
    fetcher,
  );

  const { data: codebases } = useSWR<Codebase[]>(
    "/api/codemogger/codebases",
    fetcher,
  );

  const { data: pages } = useSWR<KnowledgePage[]>(
    cbid ? `/api/knowledge?codebase_id=${cbid}` : null,
    fetcher,
  );

  const codebase = codebases?.find((cb) => cb.id === cbid);

  useEffect(() => {
    if (page?.slug && activeSlug !== page.slug) {
      setActiveSlug(page.slug);
    }
  }, [page?.slug, activeSlug]);

  useEffect(() => {
    if (cbid) {
      const currentSlug = slug || "overview";
      setActiveKnowledgePages((prev) => {
        const exists = prev.some(
          (p) => p.cbid === cbid && p.slug === currentSlug,
        );
        if (exists) return prev;
        const pageTitle = page?.title || "Overview";
        return [
          ...prev,
          {
            cbid,
            slug: currentSlug,
            title: pageTitle,
            codebaseName: codebase?.name || "Unknown",
          },
        ];
      });
    }
  }, [cbid, slug, page, codebase, setActiveKnowledgePages]);

  const handleBuild = async () => {
    if (!cbid) return;

    setIsBuilding(true);
    try {
      const response = await api.post("/api/knowledge/build", {
        codebaseId: cbid,
      });
      const data = response.data;
      toast.info(`Build started. Task ID: ${data.taskId}`);

      setTimeout(() => {
        mutate();
        setIsBuilding(false);
        toast.success("Knowledge page build completed!");
      }, 5000);
    } catch (err) {
      console.error("Build failed", err);
      toast.error("Failed to start build");
      setIsBuilding(false);
    }
  };

  const handleCopyMarkdown = () => {
    if (!page?.content) return;

    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(page.content).then(() => {
        toast.success("Copied to clipboard!");
      });
    }
  };

  const handleClose = () => {
    if (cbid) {
      const currentSlug = slug || "overview";
      setActiveKnowledgePages((prev) =>
        prev.filter((p) => !(p.cbid === cbid && p.slug === currentSlug)),
      );
    }
    router.push("/codebase");
  };

  const handlePageSelect = (slug: string) => {
    router.push(`/knowledge?cbid=${cbid}&slug=${slug}`);
  };

  if (!cbid) {
    return (
      <AppContainer>
        <AppHeader>
          <div className="flex items-center gap-4">
            <Book className="h-5 w-5 text-primary" />
            <h1 className="text-xl font-bold tracking-tight text-primary">
              Knowledge Base
            </h1>
          </div>
        </AppHeader>
        <div className="flex-1 overflow-auto p-6">
          <ErrorState
            title="No Codebase Selected"
            message="Please select a codebase from the codebases page."
          />
        </div>
      </AppContainer>
    );
  }

  if (isLoading) {
    return (
      <AppContainer>
        <LoadingState className="flex-1" />
      </AppContainer>
    );
  }

  const isEmpty = !page || !page.content || page.content.trim() === "";

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center gap-4">
            <Book className="h-5 w-5 text-primary" />
            <div className="flex items-center gap-2">
              <span className="text-xl font-bold tracking-tight text-primary">
                Knowledge
              </span>
              {codebase && (
                <span className="px-2 py-0.5 rounded-md bg-muted text-[10px] font-bold text-muted-foreground uppercase tracking-widest">
                  {codebase.name}
                </span>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            {page && page.content && (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleBuild}
                  disabled={isBuilding}
                >
                  {isBuilding ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <RefreshCw className="h-4 w-4" />
                  )}
                  Rebuild
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleCopyMarkdown}
                >
                  <Copy className="h-4 w-4" />
                  Copy
                </Button>
              </>
            )}
            <Button variant="ghost" size="sm" onClick={handleClose}>
              <X className="h-4 w-4" />
              Close
            </Button>
          </div>
        </div>
      </AppHeader>

      <div className="flex-1 flex overflow-hidden">
        {/* Navigation Sidebar */}
        <div className="w-72 border-r border-border/40 bg-muted/5 flex flex-col">
          <div className="p-4 border-b border-border/20">
            <h3 className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground px-2 mb-4">
              Pages
            </h3>
            <div className="space-y-1">
              {pages?.map((p) => (
                <button
                  key={p.id}
                  onClick={() => handlePageSelect(p.slug)}
                  className={cn(
                    "w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-semibold transition-all group",
                    activeSlug === p.slug || page?.id === p.id
                      ? "bg-primary/10 text-primary shadow-sm"
                      : "hover:bg-muted text-muted-foreground hover:text-foreground",
                  )}
                >
                  <Hash
                    className={cn(
                      "h-3.5 w-3.5",
                      activeSlug === p.slug || page?.id === p.id
                        ? "text-primary"
                        : "text-muted-foreground/40 group-hover:text-muted-foreground",
                    )}
                  />
                  {p.title}
                  {(activeSlug === p.slug || page?.id === p.id) && (
                    <ChevronRight className="h-3.5 w-3.5 ml-auto opacity-50" />
                  )}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Content Area */}
        <div className="flex-1 overflow-auto bg-background/50">
          {isEmpty ? (
            <div className="flex-1 flex items-center justify-center h-full">
              <div className="text-center max-w-md">
                <div className="w-16 h-16 rounded-full bg-muted flex items-center justify-center mx-auto mb-4">
                  <Book className="h-8 w-8 text-muted-foreground" />
                </div>
                <h3 className="text-lg font-bold text-foreground mb-2">
                  No Content Available
                </h3>
                <p className="text-muted-foreground mb-6">
                  This knowledge page doesn't exist yet. Build it to generate
                  documentation from your codebase.
                </p>
                <Button onClick={handleBuild} disabled={isBuilding}>
                  {isBuilding ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Plus className="h-4 w-4" />
                  )}
                  Build This Page
                </Button>
              </div>
            </div>
          ) : (
            <div className="max-w-5xl mx-auto p-12 lg:p-16">
              <KnowledgeViewer content={page?.content || ""} />

              <div className="mt-20 pt-12 border-t border-border/20 flex items-center justify-between">
                <div className="text-muted-foreground">
                  <p className="text-[10px] font-mono opacity-50">
                    PAGE_ID: {page?.id?.toUpperCase() || "UNKNOWN"}
                  </p>
                  {page?.updated_at && (
                    <p className="text-[10px] font-mono opacity-50 mt-1">
                      Last updated: {new Date(page.updated_at).toLocaleString()}
                    </p>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleBuild}
                    disabled={isBuilding}
                  >
                    {isBuilding ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <RefreshCw className="h-3.5 w-3.5" />
                    )}
                    Rebuild
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleCopyMarkdown}
                  >
                    <Copy className="h-3.5 w-3.5" />
                    Copy as Markdown
                  </Button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </AppContainer>
  );
}

export default function KnowledgePage() {
  return (
    <Suspense
      fallback={
        <AppContainer>
          <LoadingState className="flex-1" />
        </AppContainer>
      }
    >
      <KnowledgeContent />
    </Suspense>
  );
}
