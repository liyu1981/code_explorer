import { DefaultBadge } from "./default-badge";

interface Config {
  codemogger: {
    inherit_system_llm: boolean;
    embedder: {
      type: string;
      model: string;
      openai?: {
        api_base?: string;
        model?: string;
        api_key?: string;
      };
    };
    languages?: string[];
    chunk_lines?: number;
  };
}

const DEFAULTS = {
  chunk_lines: 150,
  embedder_type: "local",
  embedder_model: "all-minilm:l6-v2",
};

interface CodeMoggerSettingsProps {
  config: Config;
  setConfig: React.Dispatch<React.SetStateAction<any>>;
}

export function CodeMoggerSettings({
  config,
  setConfig,
}: CodeMoggerSettingsProps) {
  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-right-4 duration-500">
      <div className="space-y-1">
        <h2 className="text-2xl font-bold tracking-tight">
          Code Mogger & Embedder
        </h2>
        <p className="text-sm text-muted-foreground font-medium">
          Manage how code is indexed and semantically searched.
        </p>
      </div>

      <div className="space-y-8">
        <div className="flex items-center gap-4 bg-primary/5 p-6 rounded-3xl border border-primary/10">
          <div className="flex-1 space-y-1">
            <label
              htmlFor="inheritLLM"
              className="text-sm font-bold select-none cursor-pointer flex items-center"
            >
              Inherit System Provider
              <DefaultBadge
                isDefault={config?.codemogger?.inherit_system_llm !== false}
              />
            </label>
            <p className="text-xs text-muted-foreground font-medium">
              Use the global LLM settings for embedding generation.
            </p>
          </div>
          <input
            type="checkbox"
            id="inheritLLM"
            checked={config?.codemogger?.inherit_system_llm || false}
            onChange={(e) =>
              setConfig((prev: any) => ({
                ...prev!,
                codemogger: {
                  ...prev!.codemogger,
                  inherit_system_llm: e.target.checked,
                },
              }))
            }
            className="h-6 w-6 rounded-lg border-border text-primary focus:ring-primary/20 transition-all cursor-pointer"
          />
        </div>

        {!config?.codemogger?.inherit_system_llm && (
          <div className="space-y-8 animate-in fade-in duration-300">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-3">
                <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
                  Embedder Type
                  <DefaultBadge
                    isDefault={
                      (config?.codemogger?.embedder?.type ||
                        DEFAULTS.embedder_type) === DEFAULTS.embedder_type
                    }
                  />
                </label>
                <select
                  value={
                    config?.codemogger?.embedder?.type || DEFAULTS.embedder_type
                  }
                  onChange={(e) =>
                    setConfig((prev: any) => ({
                      ...prev!,
                      codemogger: {
                        ...prev!.codemogger,
                        embedder: {
                          ...prev!.codemogger.embedder,
                          type: e.target.value,
                        },
                      },
                    }))
                  }
                  className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-semibold"
                >
                  <option value="local">Local (Ollama compatible)</option>
                  <option value="openai">OpenAI</option>
                </select>
              </div>
              <div className="space-y-3">
                <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
                  Embedding Model
                  <DefaultBadge
                    isDefault={
                      (config?.codemogger?.embedder?.model ||
                        DEFAULTS.embedder_model) === DEFAULTS.embedder_model
                    }
                  />
                </label>
                <input
                  type="text"
                  value={config?.codemogger?.embedder?.model || ""}
                  placeholder={DEFAULTS.embedder_model}
                  onChange={(e) =>
                    setConfig((prev: any) => ({
                      ...prev!,
                      codemogger: {
                        ...prev!.codemogger,
                        embedder: {
                          ...prev!.codemogger.embedder,
                          model: e.target.value,
                        },
                      },
                    }))
                  }
                  className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-semibold"
                />
              </div>
            </div>

            <div className="space-y-3">
              <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest">
                API Base URL (Embedder Only)
              </label>
              <input
                type="text"
                value={config?.codemogger?.embedder?.openai?.api_base || ""}
                onChange={(e) =>
                  setConfig((prev: any) => ({
                    ...prev!,
                    codemogger: {
                      ...prev!.codemogger,
                      embedder: {
                        ...prev!.codemogger.embedder,
                        openai: {
                          ...prev!.codemogger.embedder.openai,
                          api_base: e.target.value,
                        },
                      },
                    },
                  }))
                }
                placeholder="https://api.openai.com/v1"
                className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-mono text-sm"
              />
            </div>

            <div className="space-y-3">
              <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest">
                API Key (Embedder Only)
              </label>
              <input
                type="password"
                value={config?.codemogger?.embedder?.openai?.api_key || ""}
                onChange={(e) =>
                  setConfig((prev: any) => ({
                    ...prev!,
                    codemogger: {
                      ...prev!.codemogger,
                      embedder: {
                        ...prev!.codemogger.embedder,
                        openai: {
                          ...prev!.codemogger.embedder.openai,
                          api_key: e.target.value,
                        },
                      },
                    },
                  }))
                }
                placeholder="Enter API key"
                className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-mono text-sm"
              />
            </div>
          </div>
        )}

        <div className="space-y-3">
          <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
            Max Lines per Chunk
            <DefaultBadge
              isDefault={
                (config?.codemogger?.chunk_lines || DEFAULTS.chunk_lines) ===
                DEFAULTS.chunk_lines
              }
            />
          </label>
          <input
            type="number"
            value={config?.codemogger?.chunk_lines ?? DEFAULTS.chunk_lines}
            onChange={(e) =>
              setConfig((prev: any) => ({
                ...prev!,
                codemogger: {
                  ...prev!.codemogger,
                  chunk_lines: Number.parseInt(e.target.value),
                },
              }))
            }
            className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-semibold"
          />
        </div>
      </div>
    </div>
  );
}
