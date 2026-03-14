import { cn } from "@/lib/utils";

interface Skill {
  id: string;
  name: string;
  system_prompt: string;
  tags: string;
  tools: string;
  updated_at: string;
}

interface SkillListProps {
  skills: Skill[];
  selectedSkillId: string | undefined;
  onSkillSelect: (skill: Skill) => void;
}

export function SkillList({
  skills,
  selectedSkillId,
  onSkillSelect,
}: SkillListProps) {
  return (
    <div className="w-80 border-r border-border/40 bg-muted/5 flex flex-col">
      <div className="p-4 border-b border-border/20">
        <h3 className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground px-2 mb-4">
          Available Skills
        </h3>
        <div className="space-y-1">
          {skills.map((skill) => (
            <button
              key={skill.id}
              onClick={() => onSkillSelect(skill)}
              className={cn(
                "w-full text-left px-3 py-3 rounded-xl transition-all group",
                selectedSkillId === skill.id
                  ? "bg-primary/10 text-primary"
                  : "hover:bg-muted text-muted-foreground hover:text-foreground",
              )}
            >
              <div className="font-bold text-sm">{skill.name}</div>
              <p className="text-[10px] opacity-70 line-clamp-1 mt-0.5 font-medium">
                {skill.tags || "No tags"}
              </p>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
