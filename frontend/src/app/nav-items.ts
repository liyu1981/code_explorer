import { Plus, Settings } from "lucide-react";

export const navTitle = "code_explorer";

export const defaultNavItem = "home";

export const navItems = [
  {
    id: "home" as const,
    icon: Plus,
    label: "New",
    path: "/",
  },
  {
    id: "settings" as const,
    icon: Settings,
    label: "Settings",
    path: "/settings",
    position: "bottom",
  },
];
