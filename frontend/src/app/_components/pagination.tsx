import { cn } from "@/lib/utils";

interface PaginationProps {
  page: number;
  totalPages: number;
  totalItems: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  itemName?: string;
}

export function Pagination({
  page,
  totalPages,
  totalItems,
  pageSize,
  onPageChange,
  itemName = "items",
}: PaginationProps) {
  if (totalPages <= 1) return null;

  return (
    <div className="px-6 py-4 border-t border-border bg-muted/20 flex items-center justify-between">
      <span className="text-sm text-muted-foreground font-medium">
        Showing {(page - 1) * pageSize + 1} to{" "}
        {Math.min(page * pageSize, totalItems)} of {totalItems} {itemName}
      </span>
      <div className="flex items-center gap-2">
        <button
          type="button"
          disabled={page === 1}
          onClick={() => onPageChange(page - 1)}
          className="px-4 py-1.5 rounded-lg border border-border bg-background text-sm font-bold disabled:opacity-50 hover:bg-muted transition-all active:scale-95"
        >
          Previous
        </button>
        <button
          type="button"
          disabled={page === totalPages}
          onClick={() => onPageChange(page + 1)}
          className="px-4 py-1.5 rounded-lg border border-border bg-background text-sm font-bold disabled:opacity-50 hover:bg-muted transition-all active:scale-95"
        >
          Next
        </button>
      </div>
    </div>
  );
}
