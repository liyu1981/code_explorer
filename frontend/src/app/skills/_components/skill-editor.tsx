import { Save, Loader2, RotateCcw } from "lucide-react";
import { cn } from "@/lib/utils";

interface Skill {
  id: string;
  name: string;
  description: string;
  system_prompt: string;
  tags: string;
  is_builtin: boolean;
  updated_at: string;
}

interface SkillEditorProps {
  selectedSkill: Skill;
  saving: boolean;
  isDirty: boolean | null | undefined;
  message: { type: "success" | "error"; text: string } | null;
  onSave: () => void;
  onReset: () => void;
  onChange: (updates: Partial<Skill>) => void;
}

export function SkillEditor({
  selectedSkill,
  saving,
  isDirty,
  message,
  onSave,
  onReset,
  onChange,
}: SkillEditorProps) {
  return (
    <div className="flex-1 flex flex-col bg-background/50 overflow-hidden">
      <div className="flex-1 flex flex-col p-8 overflow-hidden">
        <div className="flex items-center justify-between mb-6">
          <div className="flex flex-col">
            <h2 className="text-lg font-bold tracking-tight">
              {selectedSkill.name}
            </h2>
            <p className="text-xs text-muted-foreground font-medium">
              Configure how the agent behaves when using this skill.
            </p>
          </div>
          <button
            onClick={onSave}
            disabled={saving || !isDirty}
            className="flex items-center gap-2 bg-primary text-primary-foreground px-6 py-2 rounded-xl text-sm font-bold hover:opacity-90 transition-all disabled:opacity-50 shadow-lg shadow-primary/20"
          >
            {saving ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            Save
          </button>
        </div>

        {message && (
          <div
            className={cn(
              "mb-6 p-4 rounded-2xl text-sm font-bold flex items-center gap-3 animate-in slide-in-from-top-2",
              message.type === "success"
                ? "bg-green-500/10 text-green-500 border border-green-500/20"
                : "bg-destructive/10 text-destructive border border-destructive/20",
            )}
          >
            {message.text}
          </div>
        )}

        <div className="space-y-6 flex-1 flex flex-col overflow-hidden">
          <div className="grid grid-cols-2 gap-6">
            <div className="space-y-2">
              <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest">
                Description
              </label>
              <input
                type="text"
                value={selectedSkill.description}
                onChange={(e) => onChange({ description: e.target.value })}
                className="w-full bg-card border border-border rounded-xl px-4 py-3 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-medium"
                placeholder="Skill description..."
              />
            </div>

            <div className="space-y-2">
              <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest">
                Tags (space separated)
              </label>
              <input
                type="text"
                value={selectedSkill.tags}
                onChange={(e) => onChange({ tags: e.target.value })}
                className="w-full bg-card border border-border rounded-xl px-4 py-3 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-medium"
                placeholder="e.g. go backend analysis"
              />
            </div>
          </div>

          <div className="space-y-2 flex-1 flex flex-col overflow-hidden">
            <div className="flex items-center justify-between">
              <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest">
                System Prompt
              </label>
              {selectedSkill.is_builtin && (
                <button
                  onClick={onReset}
                  className="text-[10px] font-bold text-muted-foreground hover:text-primary flex items-center gap-1 transition-colors"
                >
                  <RotateCcw className="h-3 w-3" />
                  Reset to Default
                </button>
              )}
            </div>
            <textarea
              value={selectedSkill.system_prompt}
              onChange={(e) => onChange({ system_prompt: e.target.value })}
              className="flex-1 w-full bg-card border border-border rounded-xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-mono text-sm resize-none"
              placeholder="System instructions for the agent..."
            />
          </div>
        </div>
      </div>
    </div>
  );
}
