import { ExternalLink, Folder, Trash2 } from "lucide-react";

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

interface ReportsTableProps {
  reports: SavedReport[];
  onOpen: (id: string) => void;
  onDelete: (id: string) => void;
}

export function ReportsTable({ reports, onOpen, onDelete }: ReportsTableProps) {
  return (
    <div className="bg-card border border-border rounded-2xl overflow-hidden shadow-sm">
      <table className="w-full text-left border-collapse">
        <thead>
          <tr className="bg-muted/30 border-b border-border">
            <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
              Report Snapshot
            </th>
            <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
              Codebase
            </th>
            <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
              Saved At
            </th>
            <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest text-right">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border/50">
          {reports.map((report) => (
            <tr key={report.id} className="hover:bg-muted/10 transition-colors">
              <td className="px-6 py-4">
                <div className="flex flex-col gap-1 max-w-md">
                  <span className="font-bold text-foreground truncate">
                    {report.query}
                  </span>
                  <span className="text-xs text-muted-foreground truncate italic">
                    Part of: {report.title}
                  </span>
                </div>
              </td>
              <td className="px-6 py-4">
                <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
                  <Folder className="h-3.5 w-3.5" />
                  <span>{report.codebaseName}</span>
                </div>
              </td>
              <td className="px-6 py-4 text-sm text-muted-foreground font-mono">
                {new Date(report.createdAt).toLocaleDateString()}
              </td>
              <td className="px-6 py-4 text-right">
                <div className="flex items-center justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => onOpen(report.id)}
                    className="p-2 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-lg transition-all"
                    title="Open Snapshot"
                  >
                    <ExternalLink className="h-4 w-4" />
                  </button>
                  <button
                    type="button"
                    onClick={() => onDelete(report.id)}
                    className="p-2 text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-lg transition-all"
                    title="Delete"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
