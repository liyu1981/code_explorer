"use client";

import { Save, Loader2, Settings, Monitor, Search, Database } from "lucide-react";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { LoadingState } from "../_components/loading-state";
import { SettingsTabs } from "./_components/settings-tabs";
import { SystemSettings } from "./_components/system-settings";
import { ResearchSettings } from "./_components/research-settings";
import { CodeMoggerSettings } from "./_components/codemogger-settings";

interface Config {
  system: {
    db_path?: string;
    is_default_db?: boolean;
    llm?: Record<string, any>;
    max_task_retention_days?: number;
  };
  research: {
    max_reports_per_codebase: number;
    max_reports_per_session: number;
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
          <div className="flex items-center gap-4">
            <Settings className="h-5 w-5 text-primary" />
            <h1 className="text-xl font-bold tracking-tight text-primary">
              Settings
            </h1>
          </div>
        </AppHeader>
        <LoadingState className="h-full" />
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

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center gap-4">
            <Settings className="h-5 w-5 text-primary" />
            <h1 className="text-xl font-bold tracking-tight text-primary">
              Settings
            </h1>
          </div>
          <button
            type="submit"
            form="settings-form"
            disabled={saving}
            className="flex items-center gap-2 px-6 py-2 bg-primary text-primary-foreground rounded-xl font-bold shadow-lg shadow-primary/20 hover:scale-105 active:scale-95 transition-all disabled:opacity-50 disabled:scale-100"
          >
            {saving ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            Save Changes
          </button>
        </div>
      </AppHeader>

      <div className="flex-1 flex overflow-hidden">
        <SettingsTabs
          tabs={tabs}
          activeTab={activeTab}
          onTabChange={setActiveTab}
        />

        <div className="flex-1 overflow-auto bg-background/50">
          <form
            id="settings-form"
            onSubmit={handleSave}
            className="max-w-3xl mx-auto p-8 lg:p-12 space-y-12 pb-32"
          >
            {message && (
              <div
                className={`p-4 rounded-2xl text-sm font-bold border animate-in fade-in slide-in-from-top-2 ${
                  message.type === "success"
                    ? "bg-green-500/10 text-green-500 border-green-500/20"
                    : "bg-destructive/10 text-destructive border border-destructive/20"
                }`}
              >
                {message.text}
              </div>
            )}

            {config && (
              <>
                {activeTab === "system" && (
                  <SystemSettings config={config} setConfig={setConfig} />
                )}
                {activeTab === "research" && (
                  <ResearchSettings config={config} setConfig={setConfig} />
                )}
                {activeTab === "codemogger" && (
                  <CodeMoggerSettings config={config} setConfig={setConfig} />
                )}
              </>
            )}
          </form>
        </div>
      </div>
    </AppContainer>
  );
}
