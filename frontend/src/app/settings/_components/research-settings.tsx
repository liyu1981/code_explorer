import { DefaultBadge } from "./default-badge";

interface Config {
  research: {
    max_reports_per_codebase: number;
    max_reports_per_session: number;
  };
}

const DEFAULTS = {
  max_reports: 10,
  max_reports_per_session: 50,
};

interface ResearchSettingsProps {
  config: Config;
  setConfig: React.Dispatch<React.SetStateAction<any>>;
}

export function ResearchSettings({ config, setConfig }: ResearchSettingsProps) {
  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-right-4 duration-500">
      <div className="space-y-1">
        <h2 className="text-2xl font-bold tracking-tight">Research Settings</h2>
        <p className="text-sm text-muted-foreground font-medium">
          Control persistence and session retention policies.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-8">
        <div className="space-y-3">
          <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
            Max Sessions per Codebase
            <DefaultBadge
              isDefault={
                (config?.research?.max_reports_per_codebase ||
                  DEFAULTS.max_reports) === DEFAULTS.max_reports
              }
            />
          </label>
          <input
            type="number"
            value={
              config?.research?.max_reports_per_codebase ?? DEFAULTS.max_reports
            }
            onChange={(e) =>
              setConfig((prev: any) => ({
                ...prev!,
                research: {
                  ...prev!.research,
                  max_reports_per_codebase: Number.parseInt(e.target.value),
                },
              }))
            }
            className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-semibold"
          />
          <p className="text-[11px] text-muted-foreground font-medium px-1">
            The total number of research sessions (active and archived) kept per
            codebase. Oldest sessions (prioritizing archived) will be
            automatically pruned when this limit is reached.
          </p>
        </div>

        <div className="space-y-3">
          <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
            Max Reports per Session
            <DefaultBadge
              isDefault={
                (config?.research?.max_reports_per_session ||
                  DEFAULTS.max_reports_per_session) ===
                DEFAULTS.max_reports_per_session
              }
            />
          </label>
          <input
            type="number"
            value={
              config?.research?.max_reports_per_session ??
              DEFAULTS.max_reports_per_session
            }
            onChange={(e) =>
              setConfig((prev: any) => ({
                ...prev!,
                research: {
                  ...prev!.research,
                  max_reports_per_session: Number.parseInt(e.target.value),
                },
              }))
            }
            className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-semibold"
          />
          <p className="text-[11px] text-muted-foreground font-medium px-1">
            The maximum number of reports (turns) kept within a single research
            session. Oldest reports will be automatically pruned when this limit
            is reached.
          </p>
        </div>
      </div>
    </div>
  );
}
