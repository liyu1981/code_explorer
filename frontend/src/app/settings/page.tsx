"use client";

import { useAtom } from "jotai";
import {
  Save,
  Loader2,
  Settings2,
  Monitor,
  Search,
  Database,
  ChevronRight,
} from "lucide-react";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { cn } from "@/lib/utils";

interface Config {
  system: {
    db_path?: string;
    is_default_db?: boolean;
    llm?: Record<string, any>;
  };
  research: {
    max_reports_per_codebase: number;
  };
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
  llm_type: "openai",
  llm_model: "gpt-4o",
  llm_endpoint: "https://api.openai.com/v1/chat/completions",
  max_reports: 10,
  chunk_lines: 150,
  embedder_type: "local",
  embedder_model: "all-minilm:l6-v2",
};

type TabType = "system" | "research" | "codemogger";

export default function SettingsPage() {
  const [config, setConfig] = useState<Config | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [activeTab, setActiveTab] = useState<TabType>("system");
  const [message, setMessage] = useState<{
    text: string;
    type: "success" | "error";
  } | null>(null);

  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const res = await api.get("/api/config");
        setConfig(res.data);
      } catch (e) {
        console.error("Failed to fetch config", e);
        setMessage({ text: "Failed to load configuration", type: "error" });
      } finally {
        setLoading(false);
      }
    };
    fetchConfig();
  }, []);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!config) return;

    setSaving(true);
    setMessage(null);
    try {
      const res = await api.post("/api/config", config);
      setConfig(res.data);
      setMessage({ text: "Configuration saved successfully", type: "success" });
    } catch (e) {
      console.error("Failed to save config", e);
      setMessage({ text: "Failed to save configuration", type: "error" });
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <AppContainer>
        <AppHeader>
          <h1 className="text-xl font-bold tracking-tight text-primary">
            Settings
          </h1>
        </AppHeader>
        <div className="flex-1 flex items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </AppContainer>
    );
  }

  const tabs: { id: TabType; label: string; icon: any; description: string }[] =
    [
      {
        id: "system",
        label: "System & LLM",
        icon: Monitor,
        description: "Global paths and general reasoning provider settings.",
      },
      {
        id: "research",
        label: "Research Agent",
        icon: Search,
        description: "Retention and persistence for research sessions.",
      },
      {
        id: "codemogger",
        label: "Code Mogger",
        icon: Database,
        description: "Embeddings, chunking, and semantic search configuration.",
      },
    ];

  const DefaultBadge = ({ isDefault }: { isDefault: boolean }) => {
    if (!isDefault) return null;
    return (
      <span className="ml-2 px-1.5 py-0.5 rounded bg-primary/10 text-primary text-[9px] font-bold uppercase tracking-tighter">
        Default
      </span>
    );
  };

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center gap-4">
          <Settings2 className="h-5 w-5 text-primary" />
          <h1 className="text-xl font-bold tracking-tight text-primary">
            Settings
          </h1>
        </div>
      </AppHeader>

      <div className="flex-1 flex overflow-hidden">
        {/* Sidebar Tabs */}
        <div className="w-80 border-r border-border/40 bg-muted/10 p-4 space-y-2">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={cn(
                  "w-full text-left p-4 rounded-2xl transition-all group relative",
                  isActive
                    ? "bg-primary/10 text-primary shadow-sm"
                    : "hover:bg-muted/50 text-muted-foreground hover:text-foreground",
                )}
              >
                <div className="flex items-center gap-3">
                  <Icon
                    className={cn(
                      "h-5 w-5",
                      isActive ? "text-primary" : "text-muted-foreground/60",
                    )}
                  />
                  <span className="font-bold tracking-tight">{tab.label}</span>
                  {isActive && <ChevronRight className="h-4 w-4 ml-auto" />}
                </div>
                <p
                  className={cn(
                    "text-[11px] mt-1 font-medium leading-relaxed line-clamp-2",
                    isActive ? "text-primary/60" : "text-muted-foreground/40",
                  )}
                >
                  {tab.description}
                </p>
              </button>
            );
          })}
        </div>

        {/* Content Area */}
        <div className="flex-1 overflow-auto bg-background/50">
          <form
            onSubmit={handleSave}
            className="max-w-3xl mx-auto p-8 lg:p-12 space-y-12 pb-32"
          >
            {message && (
              <div
                className={cn(
                  "p-4 rounded-2xl text-sm font-bold border animate-in fade-in slide-in-from-top-2",
                  message.type === "success"
                    ? "bg-green-500/10 text-green-500 border-green-500/20"
                    : "bg-destructive/10 text-destructive border-destructive/20",
                )}
              >
                {message.text}
              </div>
            )}

            {activeTab === "system" && (
              <div className="space-y-8 animate-in fade-in slide-in-from-right-4 duration-500">
                <div className="space-y-1">
                  <h2 className="text-2xl font-bold tracking-tight">
                    System & General LLM
                  </h2>
                  <p className="text-sm text-muted-foreground font-medium">
                    Configure global paths and your primary reasoning provider.
                  </p>
                </div>

                <div className="grid grid-cols-1 gap-8">
                  <div className="space-y-3">
                    <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
                      Global Database Path
                      <DefaultBadge
                        isDefault={!!config?.system?.is_default_db}
                      />
                    </label>
                    <input
                      type="text"
                      value={config?.system?.db_path || ""}
                      readOnly
                      className="w-full bg-muted/50 border border-border/60 rounded-2xl px-4 py-4 outline-none text-muted-foreground cursor-not-allowed font-mono text-sm"
                    />
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
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
                          setConfig((prev) => ({
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
                            (config?.system?.llm?.model ||
                              DEFAULTS.llm_model) === DEFAULTS.llm_model
                          }
                        />
                      </label>
                      <input
                        type="text"
                        value={config?.system?.llm?.model || ""}
                        placeholder={DEFAULTS.llm_model}
                        onChange={(e) =>
                          setConfig((prev) => ({
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
                  </div>

                  <div className="space-y-3">
                    <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
                      API Endpoint / Base URL
                      <DefaultBadge
                        isDefault={
                          (config?.system?.llm?.endpoint ||
                            DEFAULTS.llm_endpoint) === DEFAULTS.llm_endpoint
                        }
                      />
                    </label>
                    <input
                      type="text"
                      value={config?.system?.llm?.endpoint || ""}
                      placeholder={DEFAULTS.llm_endpoint}
                      onChange={(e) =>
                        setConfig((prev) => ({
                          ...prev!,
                          system: {
                            ...prev!.system,
                            llm: {
                              ...prev!.system.llm,
                              endpoint: e.target.value,
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
                        setConfig((prev) => ({
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
            )}

            {activeTab === "research" && (
              <div className="space-y-8 animate-in fade-in slide-in-from-right-4 duration-500">
                <div className="space-y-1">
                  <h2 className="text-2xl font-bold tracking-tight">
                    Research Settings
                  </h2>
                  <p className="text-sm text-muted-foreground font-medium">
                    Control persistence and session retention policies.
                  </p>
                </div>

                <div className="grid grid-cols-1 gap-8">
                  <div className="space-y-3">
                    <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
                      Max Archived Sessions per Codebase
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
                        config?.research?.max_reports_per_codebase ??
                        DEFAULTS.max_reports
                      }
                      onChange={(e) =>
                        setConfig((prev) => ({
                          ...prev!,
                          research: {
                            ...prev!.research,
                            max_reports_per_codebase: Number.parseInt(
                              e.target.value,
                            ),
                          },
                        }))
                      }
                      className="w-full bg-card border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-semibold"
                    />
                    <p className="text-[11px] text-muted-foreground font-medium px-1">
                      Oldest sessions will be automatically pruned when this
                      limit is reached.
                    </p>
                  </div>
                </div>
              </div>
            )}

            {activeTab === "codemogger" && (
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
                          isDefault={
                            config?.codemogger?.inherit_system_llm !== false
                          }
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
                        setConfig((prev) => ({
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
                                  DEFAULTS.embedder_type) ===
                                DEFAULTS.embedder_type
                              }
                            />
                          </label>
                          <select
                            value={
                              config?.codemogger?.embedder?.type ||
                              DEFAULTS.embedder_type
                            }
                            onChange={(e) =>
                              setConfig((prev) => ({
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
                            <option value="local">
                              Local (Ollama compatible)
                            </option>
                            <option value="openai">OpenAI</option>
                          </select>
                        </div>
                        <div className="space-y-3">
                          <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center">
                            Embedding Model
                            <DefaultBadge
                              isDefault={
                                (config?.codemogger?.embedder?.model ||
                                  DEFAULTS.embedder_model) ===
                                DEFAULTS.embedder_model
                              }
                            />
                          </label>
                          <input
                            type="text"
                            value={config?.codemogger?.embedder?.model || ""}
                            placeholder={DEFAULTS.embedder_model}
                            onChange={(e) =>
                              setConfig((prev) => ({
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
                          value={
                            config?.codemogger?.embedder?.openai?.api_base || ""
                          }
                          onChange={(e) =>
                            setConfig((prev) => ({
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
                          value={
                            config?.codemogger?.embedder?.openai?.api_key || ""
                          }
                          onChange={(e) =>
                            setConfig((prev) => ({
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
                          (config?.codemogger?.chunk_lines ||
                            DEFAULTS.chunk_lines) === DEFAULTS.chunk_lines
                        }
                      />
                    </label>
                    <input
                      type="number"
                      value={
                        config?.codemogger?.chunk_lines ?? DEFAULTS.chunk_lines
                      }
                      onChange={(e) =>
                        setConfig((prev) => ({
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
            )}

            <div className="fixed bottom-8 right-8 z-50">
              <button
                type="submit"
                disabled={saving}
                className="flex items-center gap-3 px-10 py-5 bg-primary text-primary-foreground rounded-full font-bold shadow-2xl shadow-primary/40 hover:scale-105 active:scale-95 transition-all disabled:opacity-50 disabled:scale-100"
              >
                {saving ? (
                  <Loader2 className="h-6 w-6 animate-spin" />
                ) : (
                  <Save className="h-6 w-6" />
                )}
                Save Configuration
              </button>
            </div>
          </form>
        </div>
      </div>
    </AppContainer>
  );
}
