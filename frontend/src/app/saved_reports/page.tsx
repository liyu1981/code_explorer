"use client";

import { Bookmark, Search } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";
import useSWR from "swr";
import { API_URL, api, fetcher } from "@/lib/api";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { LoadingState } from "../_components/loading-state";
import { ErrorState } from "../_components/error-state";
import { EmptyState } from "../_components/empty-state";
import { Pagination } from "../_components/pagination";
import { ReportsTable } from "./_components/reports-table";

interface SavedReport {
  id: string;
  sessionId: string;
  codebaseId: string;
  title: string;
  query: string;
  content: string;
  codebaseName: string;
  codebasePath: string;
  createdAt: number;
}

export default function SavedReportsPage() {
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [searchQuery, setSearchQuery] = useState("");
  const router = useRouter();

  const { data, error, isLoading, mutate } = useSWR(
    `${API_URL}/api/saved_reports?page=${page}&pageSize=${pageSize}&q=${encodeURIComponent(searchQuery)}`,
    fetcher,
  );

  const reports = data?.reports as SavedReport[];
  const total = data?.total || 0;
  const totalPages = Math.ceil(total / pageSize);

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this saved report?")) return;
    try {
      await api.delete(`/api/saved_reports/${id}`);
      toast.success("Snapshot deleted successfully");
      mutate();
    } catch (e) {
      console.error("Delete failed", e);
      toast.error("Failed to delete snapshot");
    }
  };

  const handleOpen = (id: string) => {
    router.push(`/saved_report?id=${id}`);
  };

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center gap-4 w-full">
          <Bookmark className="h-5 w-5 text-primary" />
          <h1 className="text-xl font-bold tracking-tight text-primary">
            Saved Reports
          </h1>
        </div>
      </AppHeader>

      <div className="flex-1 overflow-auto p-6 bg-background/50">
        <div className="max-w-6xl mx-auto w-full">
          <div className="flex items-center justify-between mb-6">
            <p className="text-muted-foreground text-sm">
              Search and browse through your saved research snapshots.
            </p>
            <div className="relative w-80">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground/50" />
              <input
                type="text"
                placeholder="Search snapshots..."
                className="w-full bg-card border border-border/60 rounded-xl py-2 pl-9 pr-4 text-sm focus:ring-4 focus:ring-primary/10 transition-all outline-none"
                value={searchQuery}
                onChange={(e) => {
                  setSearchQuery(e.target.value);
                  setPage(1);
                }}
              />
            </div>
          </div>

          {isLoading ? (
            <LoadingState />
          ) : error ? (
            <ErrorState title="Failed to load saved reports" />
          ) : !reports || reports.length === 0 ? (
            <EmptyState
              icon={<Bookmark className="h-12 w-12" />}
              title="No saved reports found."
            />
          ) : (
            <>
              <ReportsTable
                reports={reports}
                onOpen={handleOpen}
                onDelete={handleDelete}
              />
              <Pagination
                page={page}
                totalPages={totalPages}
                totalItems={total}
                pageSize={pageSize}
                onPageChange={setPage}
                itemName="snapshots"
              />
            </>
          )}
        </div>
      </div>
    </AppContainer>
  );
}
