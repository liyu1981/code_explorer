import { DefaultBadge } from "./default-badge";

interface Config {
  system: {
    llm?: Record<string, any>;
    context_length?: number;
  };
}

const DEFAULTS = {
  llm_type: "openai",
  llm_model: "gpt-4o",
  llm_base_url: "https://api.openai.com/v1",
  context_length: 262144,
};

interface LLMSettingsProps {
  config: Config;
  setConfig: React.Dispatch<React.SetStateAction<any>>;
}

export function LLMSettings({ config, setConfig }: LLMSettingsProps) {
  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-right-4 duration-500">
      <div className="space-y-1">
        <h2 className="text-2xl font-bold tracking-tight">LLM Configuration</h2>
        <p className="text-sm text-muted-foreground font-medium">
          Configure your primary reasoning provider and model details.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-8">
        <div className="space-y-3">
          <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
            Provider Type
            <DefaultBadge
              isDefault={
                (config?.system?.llm?.type || DEFAULTS.llm_type) ===
                DEFAULTS.llm_type
              }
            />
          </label>
          <select
            value={config?.system?.llm?.type || DEFAULTS.llm_type}
            onChange={(e) =>
              setConfig((prev: any) => ({
                ...prev!,
                system: {
                  ...prev!.system,
                  llm: {
                    ...prev!.system.llm,
                    type: e.target.value,
                  },
                },
              }))
            }
            className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-semibold"
          >
            <option value="openai">OpenAI (or Compatible)</option>
            <option value="ollama">Ollama</option>
          </select>
        </div>

        <div className="space-y-3">
          <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
            Model Name
            <DefaultBadge
              isDefault={
                (config?.system?.llm?.model || DEFAULTS.llm_model) ===
                DEFAULTS.llm_model
              }
            />
          </label>
          <input
            type="text"
            value={config?.system?.llm?.model || ""}
            placeholder={DEFAULTS.llm_model}
            onChange={(e) =>
              setConfig((prev: any) => ({
                ...prev!,
                system: {
                  ...prev!.system,
                  llm: {
                    ...prev!.system.llm,
                    model: e.target.value,
                  },
                },
              }))
            }
            className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-semibold"
          />
        </div>

        <div className="space-y-3">
          <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
            LLM Context Window Length
            <DefaultBadge
              isDefault={
                (config?.system?.context_length || DEFAULTS.context_length) ===
                DEFAULTS.context_length
              }
            />
          </label>
          <input
            type="number"
            value={config?.system?.context_length ?? DEFAULTS.context_length}
            onChange={(e) =>
              setConfig((prev: any) => ({
                ...prev!,
                system: {
                  ...prev!.system,
                  context_length: Number.parseInt(e.target.value),
                },
              }))
            }
            className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-semibold"
          />
          <p className="text-[11px] text-muted-foreground font-medium px-1">
            The maximum character count allowed in a single conversation history
            before the agent stops to prevent context overflow.
          </p>
        </div>

        <div className="space-y-3">
          <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
            API Base URL
            <DefaultBadge
              isDefault={
                (config?.system?.llm?.base_url || DEFAULTS.llm_base_url) ===
                DEFAULTS.llm_base_url
              }
            />
          </label>
          <input
            type="text"
            value={config?.system?.llm?.base_url || ""}
            placeholder={DEFAULTS.llm_base_url}
            onChange={(e) =>
              setConfig((prev: any) => ({
                ...prev!,
                system: {
                  ...prev!.system,
                  llm: {
                    ...prev!.system.llm,
                    base_url: e.target.value,
                  },
                },
              }))
            }
            className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-mono text-sm"
          />
        </div>

        <div className="space-y-3">
          <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
            API Key
          </label>
          <input
            type="password"
            value={(config?.system?.llm?.api_key as string) || ""}
            onChange={(e) =>
              setConfig((prev: any) => ({
                ...prev!,
                system: {
                  ...prev!.system,
                  llm: {
                    ...prev!.system.llm,
                    api_key: e.target.value,
                  },
                },
              }))
            }
            placeholder="Enter API key"
            className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-mono text-sm"
          />
        </div>
      </div>
    </div>
  );
}
