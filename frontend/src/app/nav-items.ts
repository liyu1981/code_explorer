import {
  GitGraph,
  Settings,
  History,
  Bookmark,
  Activity,
  Bot,
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
    id: "skills",
    label: "Agent Skills",
    path: "/skills",
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
