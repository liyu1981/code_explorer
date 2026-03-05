import { Home, Settings } from "lucide-react";

export const navTitle = "code_explorer";

export const defaultNavItem = "home";

export const navItems = [
  {
    id: "home" as const,
    icon: Home,
    label: "Home",
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
