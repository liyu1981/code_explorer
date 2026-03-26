"use client";

import { useAtom } from "jotai";
import { Archive, X, Search } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef } from "react";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import { useWebSocketContext } from "../_components/websocket-provider";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import {
	activeSessionIdAtom,
	researchSessionsAtom,
	type ResearchSession,
	type ResearchTurn,
} from "../_jotai/research-store";
import { FloatingThoughtProcess } from "./_components/floating-thought-process";
import { IdleResearchView } from "./_components/idle-research-view";
import { ResearchReport } from "./_components/research-report";
import { StickyResearchInput } from "./_components/sticky-research-input";
import {
	useResearchStreamProcessor,
	createStreamingCallbacks,
	createRehydrationCallbacks,
} from "./_hooks/useResearchStreamProcessor";
import { processCEText } from "@/lib/cestream";
import { nanoid } from "nanoid";

function persistSession(session: ResearchSession) {
	return api.post("/api/research/sessions", {
		id: session.id,
		codebaseId: session.codebaseId,
		title: session.title,
		state: "reported",
		createdAt: session.createdAt,
		archivedAt: session.archivedAt,
	});
}

function startSummarizeTask(session: ResearchSession) {
	return api.post(`/api/research/sessions/${session.id}/summarize`);
}

function ResearchContent() {
	const [sessions, setSessions] = useAtom(researchSessionsAtom);
	const [activeSessionId, setActiveSessionId] = useAtom(activeSessionIdAtom);
	const scrollContainerRef = useRef<HTMLDivElement>(null);
	const rehydratingRef = useRef<string | null>(null);
	const { rollbackPointsRef, clearRollbackPoints, processResearchStream } =
		useResearchStreamProcessor();
	const { subscribe, unsubscribe } = useWebSocketContext();

	const router = useRouter();
	const searchParams = useSearchParams();
	const urlId = searchParams.get("id");

	// Research Session Updates
	useEffect(() => {
		const handleResearchUpdate = (payload: any) => {
			if (payload.type === "research.session.updated") {
				const { sessionId, title } = payload;
				setSessions((prev) =>
					prev.map((s) => (s.id === sessionId ? { ...s, title } : s)),
				);
			}
		};

		subscribe("research", handleResearchUpdate);
		return () => unsubscribe("research", handleResearchUpdate);
	}, [subscribe, unsubscribe, setSessions]);

	// Rehydration Effect
	// biome-ignore lint/correctness/useExhaustiveDependencies: one time load
	useEffect(() => {
		const rehydrate = async () => {
			if (!urlId || rehydratingRef.current === urlId) return;

			const existing = sessions.find((s) => s.id === urlId);
			if (existing && existing.turns.length > 0) return;

			rehydratingRef.current = urlId;
			try {
				const sessResponse = await api.get(
					"/api/research/sessions?includeArchived=true",
				);
				const allSessions = sessResponse.data;
				const sessionData = allSessions.find((s: any) => s.id === urlId);

				if (!sessionData) {
					rehydratingRef.current = null;
					return;
				}

				const reportsResponse = await api.get(
					`/api/research/sessions/${urlId}/reports`,
				);
				const reports = reportsResponse.data || [];

				const session: ResearchSession = {
					id: sessionData.id,
					codebaseId: sessionData.codebaseId,
					codebasePath: sessionData.codebasePath,
					codebaseName: sessionData.codebaseName,
					codebaseVersion: sessionData.codebaseVersion,
					title: sessionData.title,
					state: sessionData.state as any,
					createdAt: sessionData.createdAt,
					archivedAt: sessionData.archivedAt,
					steps: [],
					thoughtProcess: "",
					turns: [],
				};

				const ceStreamCallbacks = createRehydrationCallbacks(session);
				for (const report of reports) {
					processCEText(report.turnId, report.streamData, ceStreamCallbacks);
				}

				setSessions((prev) => {
					const filtered = prev.filter((s) => s.id !== session.id);
					return [...filtered, session];
				});
			} catch (e) {
				console.error("Rehydration failed", e);
			} finally {
				rehydratingRef.current = null;
			}
		};

		rehydrate();
	}, [urlId]);

	// Sync activeSessionId with URL
	useEffect(() => {
		if (urlId && urlId !== activeSessionId) {
			setActiveSessionId(urlId);
		} else if (!urlId) {
			router.push("/");
		}
	}, [urlId, activeSessionId, setActiveSessionId, router]);

	const activeSession = sessions.find((s) => s.id === activeSessionId);
	const prevTurnsLengthRef = useRef(0);

	// biome-ignore lint/correctness/useExhaustiveDependencies: scroll management
	useEffect(() => {
		if (!scrollContainerRef.current) return;
		const turnsLength = activeSession?.turns.length ?? 0;
		const activeTurnId = activeSession?.activeTurnId;
		const isStreaming = !!activeTurnId;
		const isNewTurn = turnsLength > prevTurnsLengthRef.current;

		if (isNewTurn) {
			prevTurnsLengthRef.current = turnsLength;
		}

		if (turnsLength > 1) {
			const currentTurnId = isStreaming
				? activeTurnId
				: activeSession?.turns[turnsLength - 1]?.id;
			const turnElement = scrollContainerRef.current.querySelector(
				`[data-turn-id="${currentTurnId}"]`,
			);

			if (turnElement) {
				const offset = 16;
				const targetTop = (turnElement as HTMLElement).offsetTop - offset;
				const currentTop = scrollContainerRef.current.scrollTop;

				if (Math.abs(currentTop - targetTop) > 5) {
					scrollContainerRef.current.scrollTo({
						top: targetTop,
						behavior: isNewTurn ? "smooth" : "auto",
					});
				}
			}
		} else if (activeSession?.state === "researching") {
			setTimeout(() => {
				scrollContainerRef.current?.scrollTo({
					top: scrollContainerRef.current.scrollHeight,
					behavior: "smooth",
				});
			}, 100);
		}
	}, [
		activeSession?.turns.length,
		activeSession?.activeTurnId,
		activeSession?.state === "researching",
		activeSession?.turns[activeSession?.turns.length - 1]?.report.length,
	]);

	const updateSession = (
		sessionId: string,
		payload: Partial<ResearchSession>,
		callback?: (s: ResearchSession) => void,
	) => {
		setSessions((current) => {
			const updated = current.map((s) =>
				s.id === sessionId ? { ...s, ...payload } : s,
			);
			if (callback) {
				const updatedSession = updated.find((s) => s.id === sessionId);
				updatedSession && callback(updatedSession);
			}
			return updated;
		});
	};

	const handleResearch = async (
		sessionId: string,
		query: string,
		_deep: boolean,
	) => {
		// set state of current in research session to "researching"
		updateSession(sessionId, {
			state: "researching",
			thoughtProcess: "",
			steps: [],
		});
		try {
			const response = await api.post(
				"/api/agent/research",
				{ query, sessionId },
				{ responseType: "stream" },
			);

			const reader = response.data;
			if (!reader) throw new Error("No reader");

			const callbacks = createStreamingCallbacks(
				sessionId,
				setSessions,
				rollbackPointsRef,
			);
			const turnID = nanoid();
			await processResearchStream(turnID, reader, callbacks);
		} catch (error) {
			console.error("Research failed:", error);
		} finally {
			clearRollbackPoints(sessionId);
			updateSession(
				sessionId,
				{ state: "reported", activeTurnId: undefined },
				(updatedSession) => {
					persistSession(updatedSession).then(() => {
						if (updatedSession.turns.length === 1) {
							startSummarizeTask(updatedSession);
						}
					});
				},
			);
		}
	};

	const handleArchive = async (id: string) => {
		try {
			await api.post(`/api/research/sessions/${id}/archive`);
		} catch (e) {
			console.error("Archive failed", e);
		}

		setSessions((current) => current.filter((s) => s.id !== id));
		router.push("/new");
	};

	const handleClose = (id: string) => {
		setSessions((current) => current.filter((s) => s.id !== id));
		router.push("/new");
	};

	const handleDeleteTurn = async (turnId: string) => {
		if (!activeSessionId) return;
		try {
			await api.delete(
				`/api/research/sessions/${activeSessionId}/reports/${turnId}`,
			);
			setSessions((current) =>
				current.map((s) =>
					s.id === activeSessionId
						? { ...s, turns: s.turns.filter((t) => t.id !== turnId) }
						: s,
				),
			);
		} catch (e) {
			console.error("Delete turn failed", e);
		}
	};

	const handleRegenerate = async (turn: ResearchTurn) => {
		if (!activeSessionId) return;
		await handleDeleteTurn(turn.id);
		handleResearch(activeSessionId, turn.query, false);
	};

	const handleSaveTurn = async (turn: ResearchTurn) => {
		if (!activeSession) return;
		try {
			await api.post("/api/saved_reports", {
				sessionId: activeSession.id,
				codebaseId: activeSession.codebaseId,
				title: activeSession.title,
				query: turn.query,
				turnId: turn.id,
				codebaseName: activeSession.codebaseName,
				codebasePath: activeSession.codebasePath,
			});
			toast.success("Snapshot saved successfully!");
		} catch (e) {
			console.error("Save snapshot failed", e);
			toast.error("Failed to save snapshot.");
		}
	};

	if (!activeSession) {
		return null;
	}

	const isResearching =
		activeSession.state === "researching" ||
		activeSession.state === "reasoning";

	const isIdle =
		activeSession.state === "idle" && activeSession.turns.length === 0;

	const followUpSuggestions = [
		"Analyze performance implications",
		"How is this tested?",
		"Are there security concerns?",
	];

	return (
		<AppContainer>
			<AppHeader>
				<div className="flex items-center gap-4 w-full">
					<div className="flex items-center gap-3">
						<Search className="h-5 w-5 text-primary" />
						<h1 className="text-xl font-bold tracking-tight text-primary truncate max-w-[600px]">
							{activeSession.title}
						</h1>
					</div>
					<div className="flex items-center gap-2 ml-auto">
						<button
							type="button"
							onClick={() => handleClose(activeSession.id)}
							className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-muted rounded-md transition-colors"
							title="Close Page"
						>
							<X className="h-4 w-4" />
							Close
						</button>
						<button
							type="button"
							onClick={() => handleArchive(activeSession.id)}
							className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-md transition-colors"
							title="Archive Research"
						>
							<Archive className="h-4 w-4" />
							Archive
						</button>
					</div>
				</div>
			</AppHeader>

			<div className="flex-1 flex flex-col relative overflow-hidden">
				<div
					ref={scrollContainerRef}
					className={cn(
						"flex-1 overflow-auto transition-all duration-500",
						isIdle ? "flex items-center justify-center" : "px-10 py-6",
					)}
				>
					{isIdle ? (
						<IdleResearchView
							onResearch={(q, deep) =>
								handleResearch(activeSession.id, q, deep)
							}
						/>
					) : (
						(activeSession.turns.length > 0 || isResearching) && (
							<div className="mx-auto w-full space-y-12 pb-48">
								<ResearchReport
									turns={activeSession.turns}
									onDeleteTurn={handleDeleteTurn}
									onRegenerateTurn={handleRegenerate}
									onSaveTurn={handleSaveTurn}
									isStreaming={isResearching}
								/>
								<div className="h-[400px]" />
							</div>
						)
					)}
				</div>

				<FloatingThoughtProcess
					isVisible={isResearching}
					steps={activeSession.steps}
					thoughtProcess={activeSession.thoughtProcess}
				/>

				<StickyResearchInput
					isVisible={!isIdle}
					onSearch={(q, deep) => handleResearch(activeSession.id, q, deep)}
					suggestions={!isResearching ? followUpSuggestions : []}
				/>
			</div>
		</AppContainer>
	);
}

export default function ResearchPage() {
	return (
		<Suspense fallback={<div>Loading...</div>}>
			<ResearchContent />
		</Suspense>
	);
}
