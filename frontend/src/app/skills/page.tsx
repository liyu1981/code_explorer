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

interface Skill {
  id: string;
  name: string;
  system_prompt: string;
  tags: string;
  tools: string;
  updated_at: string;
}

export default function SkillsSettingsPage() {
  const [skills, setSkills] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedSkill, setSelectedSkill] = useState<Skill | null>(null);
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
      const resp = await api.get("/api/agent_skills");
      const data = resp.data;
      setSkills(data || []);
      if (data.length > 0 && !selectedSkill) {
        setSelectedSkill(data[0]);
      }
    } catch (e) {
      console.error("Failed to fetch skills", e);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!selectedSkill) return;
    setSaving(true);
    setMessage(null);
    try {
      const resp = await api.put("/api/agent_skills", selectedSkill);
      if (resp.status === 200) {
        setMessage({ type: "success", text: "Skill updated successfully" });
        setSkills((current) =>
          current.map((s) => (s.id === selectedSkill.id ? selectedSkill : s)),
        );
      } else {
        setMessage({ type: "error", text: "Failed to update skill" });
      }
    } catch (e) {
      setMessage({ type: "error", text: "An error occurred while saving" });
    } finally {
      setSaving(false);
    }
  };

  const originalSkill = skills.find((s) => s.id === selectedSkill?.id);
  const isDirty =
    selectedSkill &&
    originalSkill &&
    (selectedSkill.system_prompt !== originalSkill.system_prompt ||
      selectedSkill.tags !== originalSkill.tags ||
      selectedSkill.tools !== originalSkill.tools);

  const handleSkillChange = (updates: Partial<Skill>) => {
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
                Agent Skills
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
            Select a skill to edit
          </div>
        )}
      </div>
    </AppContainer>
  );
}
