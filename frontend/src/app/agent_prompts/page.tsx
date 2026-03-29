"use client";

import { ArrowLeft, Wand2 } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { LoadingState } from "../_components/loading-state";
import { SkillEditor } from "./_components/skill-editor";
import { SkillList } from "./_components/skill-list";

interface Prompt {
  id: string;
  name: string;
  system_prompt: string;
  user_prompt_tpl: string;
  tags: string;
  tools: string;
  updated_at: string;
  is_builtin: boolean;
}

export default function SkillsSettingsPage() {
  const [skills, setSkills] = useState<Prompt[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedSkill, setSelectedSkill] = useState<Prompt | null>(null);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{
    type: "success" | "error";
    text: string;
  } | null>(null);

  useEffect(() => {
    fetchSkills();
  }, []);

  const fetchSkills = async () => {
    try {
      const resp = await api.get("/api/agent_prompts");
      const data = resp.data;
      setSkills(data || []);
      if (data.length > 0 && !selectedSkill) {
        setSelectedSkill(data[0]);
      }
    } catch (e) {
      console.error("Failed to fetch prompts", e);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!selectedSkill) return;
    setSaving(true);
    setMessage(null);
    try {
      const resp = await api.put("/api/agent_prompts", selectedSkill);
      if (resp.status === 200) {
        setMessage({ type: "success", text: "Prompt updated successfully" });
        setSkills((current) =>
          current.map((s) => (s.id === selectedSkill.id ? selectedSkill : s)),
        );
      } else {
        setMessage({ type: "error", text: "Failed to update prompt" });
      }
    } catch (e) {
      setMessage({ type: "error", text: "An error occurred while saving" });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (skillId: string) => {
    if (!confirm("Are you sure you want to delete this prompt?")) return;
    try {
      const resp = await api.delete(`/api/agent_prompts?id=${skillId}`);
      if (resp.status === 200) {
        setSkills((current) => current.filter((s) => s.id !== skillId));
        if (selectedSkill?.id === skillId) {
          setSelectedSkill(null);
        }
        setMessage({ type: "success", text: "Prompt deleted successfully" });
      } else {
        setMessage({ type: "error", text: "Failed to delete prompt" });
      }
    } catch (e) {
      setMessage({ type: "error", text: "An error occurred while deleting" });
    }
  };

  const originalSkill = skills.find((s) => s.id === selectedSkill?.id);
  const isDirty =
    selectedSkill &&
    originalSkill &&
    (selectedSkill.system_prompt !== originalSkill.system_prompt ||
      selectedSkill.tags !== originalSkill.tags ||
      selectedSkill.tools !== originalSkill.tools);

  const handleSkillChange = (updates: Partial<Prompt>) => {
    if (!selectedSkill) return;
    setSelectedSkill({ ...selectedSkill, ...updates });
  };

  if (loading) {
    return (
      <AppContainer>
        <LoadingState className="h-full" />
      </AppContainer>
    );
  }

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center gap-4">
            <Link
              href="/settings"
              className="p-2 hover:bg-muted rounded-xl transition-colors"
            >
              <ArrowLeft className="h-5 w-5 text-muted-foreground" />
            </Link>
            <div className="flex items-center gap-3">
              <Wand2 className="h-5 w-5 text-primary" />
              <span className="text-xl font-bold tracking-tight text-primary">
                Agent Prompts
              </span>
            </div>
          </div>
        </div>
      </AppHeader>

      <div className="flex-1 flex overflow-hidden">
        <SkillList
          skills={skills}
          selectedSkillId={selectedSkill?.id}
          onSkillSelect={(skill) => {
            setSelectedSkill(skill);
            setMessage(null);
          }}
          onDelete={handleDelete}
        />

        {selectedSkill ? (
          <SkillEditor
            selectedSkill={selectedSkill}
            saving={saving}
            isDirty={isDirty}
            message={message}
            onSave={handleSave}
            onChange={handleSkillChange}
          />
        ) : (
          <div className="flex-1 flex items-center justify-center bg-background/50 text-muted-foreground font-medium">
            Select a prompt to edit
          </div>
        )}
      </div>
    </AppContainer>
  );
}
