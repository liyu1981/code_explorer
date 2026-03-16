import { ChevronRight, History, Wand2, Activity, Bookmark } from "lucide-react";
import { cn } from "@/lib/utils";
import Link from "next/link";

type TabType = "system" | "llm" | "research" | "codemogger";

interface Tab {
  id: TabType;
  label: string;
  icon: any;
  description: string;
}

interface SettingsTabsProps {
  tabs: Tab[];
  activeTab: TabType;
  onTabChange: (id: TabType) => void;
}

export function SettingsTabs({
  tabs,
  activeTab,
  onTabChange,
}: SettingsTabsProps) {
  return (
    <div className="w-80 border-r border-border/40 bg-muted/10 p-4 space-y-2">
      {tabs.map((tab) => {
        const Icon = tab.icon;
        const isActive = activeTab === tab.id;
        return (
          <button
            key={tab.id}
            onClick={() => onTabChange(tab.id)}
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

      <div className="pt-8 pb-2 px-4">
        <h3 className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">
          Management
        </h3>
      </div>

      <Link
        href="/sessions"
        className="flex items-center gap-3 p-4 rounded-2xl hover:bg-muted/50 text-muted-foreground hover:text-foreground transition-all group"
      >
        <History className="h-5 w-5 text-muted-foreground/60 group-hover:text-primary transition-colors" />
        <span className="font-bold tracking-tight">Research Sessions</span>
        <ChevronRight className="h-4 w-4 ml-auto opacity-0 group-hover:opacity-100 transition-opacity" />
      </Link>

      <Link
        href="/skills"
        className="flex items-center gap-3 p-4 rounded-2xl hover:bg-muted/50 text-muted-foreground hover:text-foreground transition-all group"
      >
        <Wand2 className="h-5 w-5 text-muted-foreground/60 group-hover:text-primary transition-colors" />
        <span className="font-bold tracking-tight">Agent Skills</span>
        <ChevronRight className="h-4 w-4 ml-auto opacity-0 group-hover:opacity-100 transition-opacity" />
      </Link>

      <Link
        href="/tasks"
        className="flex items-center gap-3 p-4 rounded-2xl hover:bg-muted/50 text-muted-foreground hover:text-foreground transition-all group"
      >
        <Activity className="h-5 w-5 text-muted-foreground/60 group-hover:text-primary transition-colors" />
        <span className="font-bold tracking-tight">Tasks</span>
        <ChevronRight className="h-4 w-4 ml-auto opacity-0 group-hover:opacity-100 transition-opacity" />
      </Link>

      <Link
        href="/saved_reports"
        className="flex items-center gap-3 p-4 rounded-2xl hover:bg-muted/50 text-muted-foreground hover:text-foreground transition-all group"
      >
        <Bookmark className="h-5 w-5 text-muted-foreground/60 group-hover:text-primary transition-colors" />
        <span className="font-bold tracking-tight">Saved Reports</span>
        <ChevronRight className="h-4 w-4 ml-auto opacity-0 group-hover:opacity-100 transition-opacity" />
      </Link>
    </div>
  );
}
