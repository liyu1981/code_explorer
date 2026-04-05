"use client";

import { useAtom } from "jotai";
import {
  ChevronLeft,
  ChevronRight,
  Search,
  Wifi,
  WifiOff,
  Bookmark,
  Grid,
  Book,
} from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import * as React from "react";
import { ReadyState } from "react-use-websocket";
import { cn } from "@/lib/utils";
import {
  activeSessionIdAtom,
  researchSessionsAtom,
} from "../_jotai/research-store";
import {
  isSidebarExpandedAtom,
  activeSavedReportsAtom,
  activeKnowledgePagesAtom,
  type ActiveKnowledgePage,
  type ActiveSavedReport,
} from "../_jotai/ui-store";
import { navItems, navTitle } from "../nav-items";
import { useWebSocketContext } from "./websocket-provider";
import { Condiment } from "next/font/google";
import Image from "next/image";

const fontCondiment = Condiment({
  weight: "400",
  subsets: ["latin"],
});

interface NavItem {
  id: string;
  label: string;
  path: string;
  icon: React.ComponentType<{ className?: string }>;
}

interface SectionHeaderProps {
  title: string;
  navExpanded: boolean;
}

function SectionHeader({ title, navExpanded }: SectionHeaderProps) {
  return (
    <div
      className={cn(
        "flex items-center px-3 mb-2",
        navExpanded ? "justify-between" : "justify-center",
      )}
    >
      {navExpanded && (
        <span className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">
          {title}
        </span>
      )}
    </div>
  );
}

interface TopMenuItemProps {
  item: NavItem;
  navExpanded: boolean;
  isActive: boolean;
  onClick: () => void;
}

function TopMenuItem({
  item,
  navExpanded,
  isActive,
  onClick,
}: TopMenuItemProps) {
  const Icon = item.icon;
  return (
    <button
      type="button"
      key={item.id}
      onClick={onClick}
      className={cn(
        "w-full flex items-center gap-3 px-3 py-2 rounded-md transition-colors mb-1",
        isActive
          ? "bg-primary/10 text-primary"
          : "hover:bg-muted text-muted-foreground hover:text-foreground",
        !navExpanded && "justify-center",
      )}
      title={item.label}
    >
      <Icon
        className={cn("h-5 w-5 flex-shrink-0", isActive && "text-primary")}
      />
      {navExpanded && <span className="text-sm font-medium">{item.label}</span>}
    </button>
  );
}

interface KnowledgeItemProps {
  page: ActiveKnowledgePage;
  navExpanded: boolean;
  isActive: boolean;
  onClick: () => void;
}

function KnowledgeItem({
  page,
  navExpanded,
  isActive,
  onClick,
}: KnowledgeItemProps) {
  return (
    <button
      type="button"
      key={`${page.cbid}-${page.slug}`}
      onClick={onClick}
      className={cn(
        "w-full flex items-start gap-3 px-3 py-2.5 rounded-md transition-colors",
        isActive
          ? "bg-primary/10 text-primary"
          : "hover:bg-muted text-muted-foreground hover:text-foreground",
        !navExpanded && "justify-center items-center",
      )}
      title={`${page.title} (${page.codebaseName})`}
    >
      <Book
        className={cn(
          "h-5 w-5 flex-shrink-0 mt-0.5",
          isActive && "text-primary",
        )}
      />
      {navExpanded && (
        <div className="flex flex-col items-start min-w-0 flex-1 text-left">
          <span className="text-sm font-bold leading-tight line-clamp-1">
            {page.codebaseName}
          </span>
          <span className="text-[10px] opacity-60 truncate w-full">
            {page.title}
          </span>
        </div>
      )}
    </button>
  );
}

interface ResearchItemProps {
  session: {
    id: string;
    title: string;
    codebaseName: string;
    codebaseVersion: string;
  };
  navExpanded: boolean;
  isActive: boolean;
  onClick: () => void;
}

function ResearchItem({
  session,
  navExpanded,
  isActive,
  onClick,
}: ResearchItemProps) {
  return (
    <button
      type="button"
      key={session.id}
      onClick={onClick}
      className={cn(
        "w-full flex items-start gap-3 px-3 py-2.5 rounded-md transition-colors",
        isActive
          ? "bg-primary/10 text-primary"
          : "hover:bg-muted text-muted-foreground hover:text-foreground",
        !navExpanded && "justify-center items-center",
      )}
      title={`${session.title} (${session.codebaseName})`}
    >
      <Search
        className={cn(
          "h-5 w-5 flex-shrink-0 mt-0.5",
          isActive && "text-primary",
        )}
      />
      {navExpanded && (
        <div className="flex flex-col items-start min-w-0 flex-1">
          <span className="text-sm font-bold leading-tight break-words text-left w-full">
            {session.title}
          </span>
          <div className="flex items-center gap-1.5 mt-1 opacity-60">
            <span className="text-[10px] font-mono truncate max-w-[100px]">
              {session.codebaseName}
            </span>
            <span className="text-[10px]">•</span>
            <span className="text-[10px] font-mono truncate">
              {session.codebaseVersion}
            </span>
          </div>
        </div>
      )}
    </button>
  );
}

interface SavedReportItemProps {
  report: ActiveSavedReport;
  navExpanded: boolean;
  isActive: boolean;
  onClick: () => void;
}

function SavedReportItem({
  report,
  navExpanded,
  isActive,
  onClick,
}: SavedReportItemProps) {
  return (
    <button
      type="button"
      key={report.id}
      onClick={onClick}
      className={cn(
        "w-full flex items-start gap-3 px-3 py-2.5 rounded-md transition-colors",
        isActive
          ? "bg-primary/10 text-primary"
          : "hover:bg-muted text-muted-foreground hover:text-foreground",
        !navExpanded && "justify-center items-center",
      )}
      title={report.query}
    >
      <Bookmark
        className={cn(
          "h-5 w-5 flex-shrink-0 mt-0.5",
          isActive && "text-primary",
        )}
      />
      {navExpanded && (
        <div className="flex flex-col items-start min-w-0 flex-1 text-left">
          <span className="text-sm font-bold leading-tight line-clamp-2">
            {report.query}
          </span>
          <span className="text-[10px] mt-1 opacity-60 truncate w-full">
            {report.title}
          </span>
        </div>
      )}
    </button>
  );
}

interface KnowledgeSectionProps {
  pages: ActiveKnowledgePage[];
  navExpanded: boolean;
  pathname: string;
  searchParams: URLSearchParams;
  onItemClick: (cbid: string, slug?: string) => void;
}

function KnowledgeSection({
  pages,
  navExpanded,
  pathname,
  searchParams,
  onItemClick,
}: KnowledgeSectionProps) {
  if (pages.length === 0) return null;

  const isActive = (page: ActiveKnowledgePage) =>
    searchParams.get("cbid") === page.cbid && pathname === "/knowledge";

  return (
    <div className="space-y-1">
      <SectionHeader title="Knowledge" navExpanded={navExpanded} />
      <div className="space-y-1">
        {pages.map((page) => (
          <KnowledgeItem
            key={`${page.cbid}-${page.slug}`}
            page={page}
            navExpanded={navExpanded}
            isActive={isActive(page)}
            onClick={() => onItemClick(page.cbid, page.slug)}
          />
        ))}
      </div>
    </div>
  );
}

interface ResearchSectionProps {
  sessions: Array<{
    id: string;
    title: string;
    codebaseName: string;
    codebaseVersion: string;
  }>;
  navExpanded: boolean;
  pathname: string;
  searchParams: URLSearchParams;
  onItemClick: (id: string) => void;
}

function ResearchSection({
  sessions,
  navExpanded,
  pathname,
  searchParams,
  onItemClick,
}: ResearchSectionProps) {
  if (sessions.length === 0) return null;

  const isActive = (session: { id: string }) =>
    searchParams.get("id") === session.id && pathname === "/research";

  return (
    <div className="space-y-1">
      <SectionHeader title="Research" navExpanded={navExpanded} />
      <div className="space-y-1">
        {sessions.map((session) => (
          <ResearchItem
            key={session.id}
            session={session}
            navExpanded={navExpanded}
            isActive={isActive(session)}
            onClick={() => onItemClick(session.id)}
          />
        ))}
      </div>
    </div>
  );
}

interface SavedReportsSectionProps {
  reports: ActiveSavedReport[];
  navExpanded: boolean;
  pathname: string;
  searchParams: URLSearchParams;
  onItemClick: (id: string) => void;
}

function SavedReportsSection({
  reports,
  navExpanded,
  pathname,
  searchParams,
  onItemClick,
}: SavedReportsSectionProps) {
  if (reports.length === 0) return null;

  const isActive = (report: ActiveSavedReport) =>
    searchParams.get("id") === report.id && pathname === "/saved_report";

  return (
    <div className="space-y-1">
      <SectionHeader title="Saved Report" navExpanded={navExpanded} />
      <div className="space-y-1">
        {reports.map((report) => (
          <SavedReportItem
            key={report.id}
            report={report}
            navExpanded={navExpanded}
            isActive={isActive(report)}
            onClick={() => onItemClick(report.id)}
          />
        ))}
      </div>
    </div>
  );
}

interface NavHeaderProps {
  navExpanded: boolean;
  onToggle: () => void;
}

function NavHeader({ navExpanded, onToggle }: NavHeaderProps) {
  return (
    <div className="h-[60px] border-b px-4 flex items-center justify-between">
      {navExpanded && (
        <div className="flex items-center gap-2">
          <Image
            src="/favicon-32x32.png"
            alt="Logo"
            width="32"
            height="32"
            className="h-5 w-5"
          />
          <span
            className={cn(
              "text-xl text-primary ml-1 mt-[10px]",
              fontCondiment.className,
            )}
          >
            {navTitle}
          </span>
        </div>
      )}
      <button
        type="button"
        className="h-8 w-8 flex items-center gap-3 px-1 py-2 rounded-md transition-colors mb-1 hover:bg-muted hover:text-foreground"
        onClick={onToggle}
      >
        {navExpanded ? (
          <ChevronLeft className="h-4 w-4" />
        ) : (
          <Image
            src="/favicon-32x32.png"
            alt="Logo"
            width="32"
            height="32"
            className="h-5 w-5"
          />
        )}
      </button>
    </div>
  );
}

interface ManageMenuProps {
  isOpen: boolean;
  navExpanded: boolean;
  pathname: string;
  manageMenuRef: React.RefObject<HTMLDivElement | null>;
  manageItems: NavItem[];
  settingsItem: NavItem | undefined;
  readyState: ReadyState;
  onToggle: () => void;
  onItemClick: (itemId: string) => void;
}

function ManageMenu({
  isOpen,
  navExpanded,
  pathname,
  manageMenuRef,
  manageItems,
  settingsItem,
  readyState,
  onToggle,
  onItemClick,
}: ManageMenuProps) {
  const isConnected = readyState === ReadyState.OPEN;
  const StatusIcon = isConnected ? Wifi : WifiOff;

  const connectionStatusMap: Record<ReadyState, string> = {
    [ReadyState.CONNECTING]: "Connecting",
    [ReadyState.OPEN]: "Connected",
    [ReadyState.CLOSING]: "Closing",
    [ReadyState.CLOSED]: "Closed",
    [ReadyState.UNINSTANTIATED]: "Uninstantiated",
  };

  return (
    <div className="p-2 border-t space-y-1 relative" ref={manageMenuRef}>
      {isOpen && (
        <div
          className={cn(
            "absolute bottom-2 left-full ml-2 bg-card border border-border rounded-xl shadow-2xl p-1 z-50 animate-in slide-in-from-left-2 duration-200 min-w-[160px]",
          )}
        >
          <div className="space-y-1">
            {manageItems.map((item) => {
              const Icon = item.icon;
              const isActive = pathname.startsWith(item.path);
              return (
                <button
                  type="button"
                  key={item.id}
                  onClick={() => onItemClick(item.id)}
                  className={cn(
                    "w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors",
                    isActive
                      ? "bg-primary/10 text-primary"
                      : "hover:bg-muted text-muted-foreground hover:text-foreground",
                  )}
                  title={item.label}
                >
                  <Icon className="h-4 w-4 flex-shrink-0" />
                  <span className="text-xs font-semibold">{item.label}</span>
                </button>
              );
            })}
          </div>
        </div>
      )}

      <button
        type="button"
        onClick={onToggle}
        className={cn(
          "w-full flex items-center gap-3 px-3 py-2 rounded-md transition-colors mb-1",
          isOpen
            ? "bg-muted text-foreground"
            : "text-muted-foreground hover:bg-muted hover:text-foreground",
          !navExpanded && "justify-center",
        )}
        title="Manage"
      >
        <Grid className="h-5 w-5 flex-shrink-0" />
        {navExpanded && (
          <div className="flex items-center justify-between flex-1">
            <span className="text-sm font-medium">Manage</span>
            <ChevronRight
              className={cn(
                "h-3 w-3 transition-transform",
                isOpen && "rotate-90",
              )}
            />
          </div>
        )}
      </button>

      {settingsItem && (
        <button
          type="button"
          key={settingsItem.id}
          onClick={() => onItemClick(settingsItem.id)}
          className={cn(
            "w-full flex items-center gap-3 px-3 py-2 rounded-md transition-colors mb-1",
            pathname.startsWith(settingsItem.path)
              ? "bg-primary/10 text-primary"
              : "hover:bg-muted text-muted-foreground hover:text-foreground",
            !navExpanded && "justify-center",
          )}
          title={settingsItem.label}
        >
          <settingsItem.icon
            className={cn(
              "h-5 w-5 flex-shrink-0",
              pathname.startsWith(settingsItem.path) && "text-primary",
            )}
          />
          {navExpanded && (
            <span className="text-sm font-medium">{settingsItem.label}</span>
          )}
        </button>
      )}

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
    </div>
  );
}

function SidebarContent() {
  const [navExpanded, setNavExpanded] = useAtom(isSidebarExpandedAtom);
  const [allSessions] = useAtom(researchSessionsAtom);
  const [activeReports] = useAtom(activeSavedReportsAtom);
  const [activeKnowledgePages] = useAtom(activeKnowledgePagesAtom);
  const [isManageOpen, setIsManageOpen] = React.useState(false);
  const manageMenuRef = React.useRef<HTMLDivElement>(null);
  const sessions = allSessions.filter((s) => !s.archivedAt);

  React.useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        manageMenuRef.current &&
        !manageMenuRef.current.contains(event.target as Node)
      ) {
        setIsManageOpen(false);
      }
    }

    if (isManageOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    } else {
      document.removeEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isManageOpen]);

  const [, setActiveSessionId] = useAtom(activeSessionIdAtom);

  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { readyState } = useWebSocketContext();

  const getActiveMenuFromPath = React.useCallback(() => {
    const activeItem = navItems.find((item) => {
      if (item.path === "/") return pathname === "/" && !searchParams.get("id");
      return pathname.startsWith(item.path);
    });
    return activeItem ? activeItem.id : "";
  }, [pathname, searchParams]);

  const [activeMenu, setActiveMenu] = React.useState(getActiveMenuFromPath());

  const topMenuItems = navItems.filter(
    (item) => (item as any).position !== "bottom",
  );
  const manageItems = navItems.filter((item) =>
    ["agent_prompts", "tasks", "saved_reports", "sessions"].includes(item.id),
  );
  const settingsItem = navItems.find((item) => item.id === "settings");

  React.useEffect(() => {
    setActiveMenu(getActiveMenuFromPath());
  }, [getActiveMenuFromPath]);

  const handleMenuClick = (itemId: string) => {
    const item = navItems.find((m) => m.id === itemId);
    if (item) {
      setActiveMenu(itemId);
      setActiveSessionId(null);
      router.push(item.path);
      setIsManageOpen(false);
    }
  };

  const handleSessionClick = (id: string) => {
    setActiveSessionId(id);
    setActiveMenu("");
    router.push(`/research?id=${id}`);
  };

  const handleReportClick = (id: string) => {
    setActiveSessionId(null);
    setActiveMenu("");
    router.push(`/saved_report?id=${id}`);
  };

  const handleKnowledgeClick = (cbid: string, slug?: string) => {
    setActiveSessionId(null);
    setActiveMenu("");
    router.push(`/knowledge?cbid=${cbid}${slug ? `&slug=${slug}` : ""}`);
  };

  const isTopMenuActive = (itemId: string) =>
    activeMenu === itemId &&
    pathname === navItems.find((n) => n.id === itemId)?.path &&
    !searchParams.get("id");

  return (
    <div
      className={cn(
        "border-r bg-muted/20 flex flex-col transition-all duration-300 h-full",
        navExpanded ? "w-56" : "w-16",
      )}
    >
      <NavHeader
        navExpanded={navExpanded}
        onToggle={() => setNavExpanded(!navExpanded)}
      />

      <nav className="flex-1 p-2 space-y-4 overflow-y-auto">
        <div className="space-y-1">
          {topMenuItems.map((item) => (
            <TopMenuItem
              key={item.id}
              item={item}
              navExpanded={navExpanded}
              isActive={isTopMenuActive(item.id)}
              onClick={() => handleMenuClick(item.id)}
            />
          ))}
        </div>

        <div className="px-2">
          <div className="h-px bg-border/60 w-full" />
        </div>

        <KnowledgeSection
          pages={activeKnowledgePages}
          navExpanded={navExpanded}
          pathname={pathname}
          searchParams={searchParams}
          onItemClick={handleKnowledgeClick}
        />

        <ResearchSection
          sessions={sessions}
          navExpanded={navExpanded}
          pathname={pathname}
          searchParams={searchParams}
          onItemClick={handleSessionClick}
        />

        <SavedReportsSection
          reports={activeReports}
          navExpanded={navExpanded}
          pathname={pathname}
          searchParams={searchParams}
          onItemClick={handleReportClick}
        />
      </nav>

      <ManageMenu
        isOpen={isManageOpen}
        navExpanded={navExpanded}
        pathname={pathname}
        manageMenuRef={manageMenuRef}
        manageItems={manageItems}
        settingsItem={settingsItem}
        readyState={readyState}
        onToggle={() => setIsManageOpen(!isManageOpen)}
        onItemClick={handleMenuClick}
      />
    </div>
  );
}

export function AppNavSidebar() {
  return (
    <React.Suspense fallback={<div className="w-16 border-r bg-muted/20" />}>
      <SidebarContent />
    </React.Suspense>
  );
}
