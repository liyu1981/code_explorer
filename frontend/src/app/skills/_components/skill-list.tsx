import { Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";

interface Skill {
	id: string;
	name: string;
	system_prompt: string;
	tags: string;
	tools: string;
	updated_at: string;
	is_builtin: boolean;
}

interface SkillListProps {
	skills: Skill[];
	selectedSkillId: string | undefined;
	onSkillSelect: (skill: Skill) => void;
	onDelete: (skillId: string) => void;
}

export function SkillList({
	skills,
	selectedSkillId,
	onSkillSelect,
	onDelete,
}: SkillListProps) {
	return (
		<div className="w-80 border-r border-border/40 bg-muted/5 flex flex-col">
			<div className="p-4 border-b border-border/20 overflow-auto">
				<h3 className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground px-2 mb-4">
					Available Skills
				</h3>
				<div className="space-y-1">
					{skills.map((skill) => (
						<div
							key={skill.id}
							className={cn(
								"w-full text-left px-3 py-3 rounded-xl transition-all group flex items-center justify-between",
								selectedSkillId === skill.id
									? "bg-primary/10 text-primary"
									: "hover:bg-muted text-muted-foreground hover:text-foreground",
							)}
						>
							<button
								type="button"
								onClick={() => onSkillSelect(skill)}
								className="flex-1 text-left"
							>
								<div className="font-bold text-sm">{skill.name}</div>
								<p className="text-[10px] opacity-70 line-clamp-1 mt-0.5 font-medium">
									{skill.tags || "No tags"}
								</p>
							</button>
							{!skill.is_builtin && (
								<button
									type="button"
									onClick={(e) => {
										e.stopPropagation();
										onDelete(skill.id);
									}}
									className="p-1.5 rounded-lg opacity-0 group-hover:opacity-100 hover:bg-destructive/10 hover:text-destructive transition-all ml-2"
									title="Delete skill"
								>
									<Trash2 className="w-4 h-4" />
								</button>
							)}
						</div>
					))}
				</div>
			</div>
		</div>
	);
}
