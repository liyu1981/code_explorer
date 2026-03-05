"use client";

import { useAtom } from "jotai";
import {
  ChevronLeft,
  ChevronRight,
  Globe,
  Wifi,
  WifiOff,
} from "lucide-react";
import { usePathname, useRouter } from "next/navigation";
import * as React from "react";
import { ReadyState } from "react-use-websocket";
import { cn } from "@/lib/utils";
import { isSidebarExpandedAtom } from "../_jotai/ui-store";
import { navItems, navTitle } from "../nav-items";
import { useWebSocketContext } from "./websocket-provider";

export function AppNavSidebar() {
  const [navExpanded, setNavExpanded] = useAtom(isSidebarExpandedAtom);
  const router = useRouter();
  const pathname = usePathname();
  const { readyState } = useWebSocketContext();

  const getActiveMenuFromPath = React.useCallback(() => {
    const activeItem = navItems.find((item) => {
        if (item.path === "/") return pathname === "/";
        return pathname.startsWith(item.path);
    });
    return activeItem ? activeItem.id : "";
  }, [pathname]);

  const [activeMenu, setActiveMenu] = React.useState(getActiveMenuFromPath());

  const topMenuItems = navItems.filter((item) => (item as any).position !== "bottom");
  const bottomMenuItems = navItems.filter((item) => (item as any).position === "bottom");

  React.useEffect(() => {
    setActiveMenu(getActiveMenuFromPath());
  }, [getActiveMenuFromPath]);

  const handleMenuClick = (itemId: string) => {
    const item = navItems.find((m) => m.id === itemId);
    if (item) {
      setActiveMenu(itemId);
      router.push(item.path);
    }
  };

  const connectionStatusMap: Record<ReadyState, string> = {
    [ReadyState.CONNECTING]: "Connecting",
    [ReadyState.OPEN]: "Connected",
    [ReadyState.CLOSING]: "Closing",
    [ReadyState.CLOSED]: "Closed",
    [ReadyState.UNINSTANTIATED]: "Uninstantiated",
  };

  const isConnected = readyState === ReadyState.OPEN;
  const StatusIcon = isConnected ? Wifi : WifiOff;

  const statusButton = (
    <div
      className={cn(
        "w-full flex items-center gap-3 px-3 py-2 rounded-md transition-colors cursor-default",
        isConnected ? "text-green-500" : "text-yellow-500",
        !navExpanded && "justify-center",
      )}
    >
      <StatusIcon className="h-5 w-5 flex-shrink-0" />
      {navExpanded && (
        <span className="text-sm font-medium">
          {connectionStatusMap[readyState]}
        </span>
      )}
    </div>
  );

  return (
    <div
      className={cn(
        "border-r bg-muted/20 flex flex-col transition-all duration-300 h-full",
        navExpanded ? "w-56" : "w-16",
      )}
    >
      <div className="h-[60px] border-b px-4 flex items-center justify-between">
        {navExpanded && (
          <div className="flex items-center gap-2">
            <Globe className="h-5 w-5 text-primary" />
            <span className="text-lg font-bold text-primary">
              {navTitle}
            </span>
          </div>
        )}
        <button
          onClick={() => setNavExpanded(!navExpanded)}
          className="h-8 w-8 flex items-center justify-center hover:bg-muted rounded-md transition-colors"
        >
          {navExpanded ? (
            <ChevronLeft className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
        </button>
      </div>

      <nav className="flex-1 p-2">
        {topMenuItems.map((item) => {
          const Icon = item.icon;
          return (
            <button
              type="button"
              key={item.id}
              onClick={() => handleMenuClick(item.id)}
              className={cn(
                "w-full flex items-center gap-3 px-3 py-2 rounded-md transition-colors mb-1",
                activeMenu === item.id
                  ? "bg-primary/10 text-primary"
                  : "hover:bg-muted text-muted-foreground hover:text-foreground",
                !navExpanded && "justify-center",
              )}
              title={item.label}
            >
              <Icon
                className={cn(
                  "h-5 w-5 flex-shrink-0",
                  activeMenu === item.id && "text-primary",
                )}
              />
              {navExpanded && (
                <span className="text-sm font-medium">{item.label}</span>
              )}
            </button>
          );
        })}
      </nav>

      <div className="p-2 border-t space-y-1">
        {bottomMenuItems.map((item) => {
          const Icon = item.icon;
          return (
            <button
              type="button"
              key={item.id}
              onClick={() => handleMenuClick(item.id)}
              className={cn(
                "w-full flex items-center gap-3 px-3 py-2 rounded-md transition-colors mb-1",
                activeMenu === item.id
                  ? "bg-primary/10 text-primary"
                  : "hover:bg-muted text-muted-foreground hover:text-foreground",
                !navExpanded && "justify-center",
              )}
              title={item.label}
            >
              <Icon
                className={cn(
                  "h-5 w-5 flex-shrink-0",
                  activeMenu === item.id && "text-primary",
                )}
              />
              {navExpanded && (
                <span className="text-sm font-medium">{item.label}</span>
              )}
            </button>
          );
        })}
        {statusButton}
      </div>
    </div>
  );
}
