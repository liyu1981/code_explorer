import {
  GitGraph,
  Settings,
  History,
  Bookmark,
  Activity,
  Bot,
  Sun,
  Database,
} from "lucide-react";

export const navTitle = "Code Explorer";

export interface NavItem {
  id: string;
  label: string;
  path: string;
  icon: any;
  position?: "top" | "bottom";
}

export const navItems: NavItem[] = [
  {
    id: "codebase",
    label: "Codebases",
    path: "/codebase",
    icon: GitGraph,
  },
  {
    id: "codesummer",
    label: "Code Summer",
    path: "/codesummer",
    icon: Sun,
  },
  {
    id: "codemogger-manage",
    label: "Manage Indexes",
    path: "/codemogger-manage",
    icon: Database,
  },
  {
    id: "agent_prompts",
    label: "Agent Prompts",
    path: "/agent_prompts",
    icon: Bot,
    position: "bottom",
  },
  {
    id: "tasks",
    label: "Tasks",
    path: "/tasks",
    icon: Activity,
    position: "bottom",
  },
  {
    id: "saved_reports",
    label: "Saved Reports",
    path: "/saved_reports",
    icon: Bookmark,
    position: "bottom",
  },
  {
    id: "sessions",
    label: "Sessions",
    path: "/sessions",
    icon: History,
    position: "bottom",
  },
  {
    id: "settings",
    label: "Settings",
    path: "/settings",
    icon: Settings,
    position: "bottom",
  },
];
