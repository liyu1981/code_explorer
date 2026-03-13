import { DefaultBadge } from "./default-badge";

interface Config {
  system: {
    db_path?: string;
    is_default_db?: boolean;
    llm?: Record<string, any>;
    context_length?: number;
    max_task_retention_days?: number;
  };
}

const DEFAULTS = {
  max_task_retention_days: 180,
};

interface SystemSettingsProps {
  config: Config;
  setConfig: React.Dispatch<React.SetStateAction<any>>;
}

export function SystemSettings({ config, setConfig }: SystemSettingsProps) {
  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-right-4 duration-500">
      <div className="space-y-1">
        <h2 className="text-2xl font-bold tracking-tight">System Settings</h2>
        <p className="text-sm text-muted-foreground font-medium">
          Configure global paths and general system behavior.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-8">
        <div className="space-y-3">
          <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
            Global Database Path
            <DefaultBadge isDefault={!!config?.system?.is_default_db} />
          </label>
          <input
            type="text"
            value={config?.system?.db_path || ""}
            readOnly
            className="w-full bg-muted/50 border border-border/60 rounded-2xl px-4 py-4 outline-none text-muted-foreground cursor-not-allowed font-mono text-sm"
          />
        </div>

        <div className="space-y-3">
          <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
            Task Retention (Days)
            <DefaultBadge
              isDefault={
                (config?.system?.max_task_retention_days ||
                  DEFAULTS.max_task_retention_days) ===
                DEFAULTS.max_task_retention_days
              }
            />
          </label>
          <input
            type="number"
            value={
              config?.system?.max_task_retention_days ??
              DEFAULTS.max_task_retention_days
            }
            onChange={(e) =>
              setConfig((prev: any) => ({
                ...prev!,
                system: {
                  ...prev!.system,
                  max_task_retention_days: Number.parseInt(e.target.value),
                },
              }))
            }
            className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-semibold"
          />
          <p className="text-[11px] text-muted-foreground font-medium px-1">
            Number of days to keep background task history. Older tasks will be
            automatically purged.
          </p>
        </div>
      </div>
    </div>
  );
}
