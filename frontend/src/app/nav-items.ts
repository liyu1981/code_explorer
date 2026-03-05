import { PlusCircle, Settings } from "lucide-react";

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
    id: "new",
    label: "New Research",
    path: "/new",
    icon: PlusCircle,
  },
  {
    id: "settings",
    label: "Settings",
    path: "/settings",
    icon: Settings,
    position: "bottom",
  },
];
