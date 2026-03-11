"use client";

import { useAtom } from "jotai";
import {
	ChevronLeft,
	ChevronRight,
	Globe,
	Search,
	Wifi,
	WifiOff,
	Bookmark,
	Grid,
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
} from "../_jotai/ui-store";
import { navItems, navTitle } from "../nav-items";
import { useWebSocketContext } from "./websocket-provider";
import { Condiment } from "next/font/google";
import Image from "next/image";

const fontCondiment = Condiment({
	weight: "400",
	subsets: ["latin"],
});

function SidebarContent() {
	const [navExpanded, setNavExpanded] = useAtom(isSidebarExpandedAtom);
	const [allSessions] = useAtom(researchSessionsAtom);
	const [activeReports] = useAtom(activeSavedReportsAtom);
	const [isManageOpen, setIsManageOpen] = React.useState(false);
	const manageMenuRef = React.useRef<HTMLDivElement>(null);
	const sessions = allSessions.filter((s) => !s.archivedAt);

	// Close manage menu when clicking outside
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
		["skills", "tasks", "saved_reports", "sessions"].includes(item.id),
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

	const connectionStatusMap: Record<ReadyState, string> = {
		[ReadyState.CONNECTING]: "Connecting",
		[ReadyState.OPEN]: "Connected",
		[ReadyState.CLOSING]: "Closing",
		[ReadyState.CLOSED]: "Closed",
		[ReadyState.UNINSTANTIATED]: "Uninstantiated",
	};

	const isConnected = readyState === ReadyState.OPEN;
	const StatusIcon = isConnected ? Wifi : WifiOff;

	return (
		<div
			className={cn(
				"border-r bg-muted/20 flex flex-col transition-all duration-300 h-full",
				navExpanded ? "w-56" : "w-16",
			)}
		>
			{/* Nav Header */}
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
					onClick={() => setNavExpanded(!navExpanded)}
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
						// <ChevronRight className="h-4 w-4" />
					)}
				</button>
			</div>

			<nav className="flex-1 p-2 space-y-4 overflow-y-auto">
				<div className="space-y-1">
					{topMenuItems.map((item) => {
						const Icon = item.icon;
						const isActive =
							activeMenu === item.id &&
							pathname === item.path &&
							!searchParams.get("id");
						return (
							<button
								type="button"
								key={item.id}
								onClick={() => handleMenuClick(item.id)}
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
									className={cn(
										"h-5 w-5 flex-shrink-0",
										isActive && "text-primary",
									)}
								/>
								{navExpanded && (
									<span className="text-sm font-medium">{item.label}</span>
								)}
							</button>
						);
					})}
				</div>

				<div className="px-2">
					<div className="h-px bg-border/60 w-full" />
				</div>

				<div className="space-y-1">
					<div
						className={cn(
							"flex items-center px-3 mb-2",
							navExpanded ? "justify-between" : "justify-center",
						)}
					>
						{navExpanded && (
							<span className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">
								Research
							</span>
						)}
					</div>

					<div className="space-y-1">
						{sessions.map((session) => (
							<button
								type="button"
								key={session.id}
								onClick={() => handleSessionClick(session.id)}
								className={cn(
									"w-full flex items-start gap-3 px-3 py-2.5 rounded-md transition-colors",
									searchParams.get("id") === session.id &&
										pathname === "/research"
										? "bg-primary/10 text-primary"
										: "hover:bg-muted text-muted-foreground hover:text-foreground",
									!navExpanded && "justify-center items-center",
								)}
								title={`${session.title} (${session.codebaseName})`}
							>
								<Search
									className={cn(
										"h-5 w-5 flex-shrink-0 mt-0.5",
										searchParams.get("id") === session.id &&
											pathname === "/research" &&
											"text-primary",
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
						))}
					</div>
				</div>

				{activeReports.length > 0 && (
					<div className="space-y-1">
						<div
							className={cn(
								"flex items-center px-3 mb-2 mt-4",
								navExpanded ? "justify-between" : "justify-center",
							)}
						>
							{navExpanded && (
								<span className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">
									Saved Report
								</span>
							)}
						</div>

						<div className="space-y-1">
							{activeReports.map((report) => (
								<button
									type="button"
									key={report.id}
									onClick={() => handleReportClick(report.id)}
									className={cn(
										"w-full flex items-start gap-3 px-3 py-2.5 rounded-md transition-colors",
										searchParams.get("id") === report.id &&
											pathname === "/saved_report"
											? "bg-primary/10 text-primary"
											: "hover:bg-muted text-muted-foreground hover:text-foreground",
										!navExpanded && "justify-center items-center",
									)}
									title={report.query}
								>
									<Bookmark
										className={cn(
											"h-5 w-5 flex-shrink-0 mt-0.5",
											searchParams.get("id") === report.id &&
												pathname === "/saved_report" &&
												"text-primary",
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
							))}
						</div>
					</div>
				)}
			</nav>

			<div className="p-2 border-t space-y-1 relative" ref={manageMenuRef}>
				{/* Manage Menu Popout to the right */}
				{isManageOpen && (
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
										onClick={() => handleMenuClick(item.id)}
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
					onClick={() => setIsManageOpen(!isManageOpen)}
					className={cn(
						"w-full flex items-center gap-3 px-3 py-2 rounded-md transition-colors mb-1",
						isManageOpen
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
									isManageOpen && "rotate-90",
								)}
							/>
						</div>
					)}
				</button>

				{settingsItem && (
					<button
						type="button"
						key={settingsItem.id}
						onClick={() => handleMenuClick(settingsItem.id)}
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
