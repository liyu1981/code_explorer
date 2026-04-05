"use client";

import { useAtom } from "jotai";
import { Archive, X, Search } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useRef, useMemo } from "react";
import { toast } from "sonner";
import { api, apiStream } from "@/lib/api";
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
import { nanoid } from "nanoid";
import type { Source } from "@/app/research/_components/source-card";
import {
	type CEStreamCallbacks,
	processCEStream,
	type CEEvent,
	processCEText,
} from "@/lib/cestream";

// --- Types ---

interface RollbackPoint {
	turnID: string;
	report: string;
	sources: Source[];
}

// --- Callback Logic Helpers ---

function getSourceKey(source: Source): string {
	return `${source.path}::${source.snippet}`;
}

function createTurnFromEvent(e: CEEvent): ResearchTurn {
	return {
		id: e.turnid!,
		query: e.query!,
		report: "",
		sources: [],
		timestamp: e.timestamp!,
		updatedAt: Date.now(),
	};
}

function withSourceDedup(callbacks: CEStreamCallbacks): CEStreamCallbacks {
	const sourceKeys = new Set<string>();
	return {
		...callbacks,
		onResearchSourceAdded: (id, e) => {
			if (e.source) {
				const key = getSourceKey(e.source);
				if (!sourceKeys.has(key)) {
					sourceKeys.add(key);
					callbacks.onResearchSourceAdded(id, e);
				}
			}
		},
		onResourceMaterial: (id, e) => {
			if (e.resource) {
				const key = getSourceKey(e.resource);
				if (!sourceKeys.has(key)) {
					sourceKeys.add(key);
					callbacks.onResourceMaterial(id, e);
				}
			}
		},
	};
}

// --- API Helpers ---

const persistSession = (session: ResearchSession) =>
	api.post("/api/research/sessions", {
		id: session.id,
		codebaseId: session.codebaseId,
		title: session.title,
		state: "reported",
		createdAt: session.createdAt,
		archivedAt: session.archivedAt,
	});

const startSummarizeTask = (session: ResearchSession) =>
	api.post(`/api/research/sessions/${session.id}/summarize`);

// --- Hooks ---

function useResearchSessionState() {
	const [sessions, setSessions] = useAtom(researchSessionsAtom);
	const [activeSessionId, setActiveSessionId] = useAtom(activeSessionIdAtom);
	const activeSession = useMemo(
		() => sessions.find((s) => s.id === activeSessionId),
		[sessions, activeSessionId],
	);

	const updateSession = useCallback(
		(
			sessionId: string,
			payload: Partial<ResearchSession>,
			callback?: (s: ResearchSession) => void,
		) => {
			setSessions((current) => {
				const updated = current.map((s) =>
					s.id === sessionId ? { ...s, ...payload } : s,
				);
				if (callback) {
					const s = updated.find((sess) => sess.id === sessionId);
					if (s) callback(s);
				}
				return updated;
			});
		},
		[setSessions],
	);

	const deleteTurn = useCallback(
		async (sessionId: string, turnId: string) => {
			try {
				await api.delete(
					`/api/research/sessions/${sessionId}/reports/${turnId}`,
				);
				setSessions((current) =>
					current.map((s) =>
						s.id === sessionId
							? { ...s, turns: s.turns.filter((t) => t.id !== turnId) }
							: s,
					),
				);
			} catch (e) {
				console.error("Delete turn failed", e);
			}
		},
		[setSessions],
	);

	return {
		sessions,
		setSessions,
		activeSessionId,
		setActiveSessionId,
		activeSession,
		updateSession,
		deleteTurn,
	};
}

function useResearchAutoScroll(
	containerRef: React.RefObject<HTMLDivElement | null>,
	activeSession?: ResearchSession,
) {
	const prevTurnsLengthRef = useRef(0);

	// biome-ignore lint/correctness/useExhaustiveDependencies: scroll management
	useEffect(() => {
		if (!containerRef.current || !activeSession) return;

		const turnsLength = activeSession.turns.length;
		const activeTurnId = activeSession.activeTurnId;
		const isStreaming = !!activeTurnId;
		const isNewTurn = turnsLength > prevTurnsLengthRef.current;

		if (isNewTurn) {
			prevTurnsLengthRef.current = turnsLength;
		}

		if (turnsLength > 1) {
			const currentTurnId = isStreaming
				? activeTurnId
				: activeSession.turns[turnsLength - 1]?.id;

			const turnElement = containerRef.current.querySelector(
				`[data-turn-id="${currentTurnId}"]`,
			);

			if (turnElement) {
				const offset = 16;
				const targetTop = (turnElement as HTMLElement).offsetTop - offset;
				const currentTop = containerRef.current.scrollTop;

				if (Math.abs(currentTop - targetTop) > 5) {
					containerRef.current.scrollTo({
						top: targetTop,
						behavior: isNewTurn ? "smooth" : "auto",
					});
				}
			}
		} else if (activeSession.state === "researching") {
			setTimeout(() => {
				containerRef.current?.scrollTo({
					top: containerRef.current.scrollHeight,
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
}

function useResearchRehydration(
	urlId: string | null,
	setSessions: (
		updater: (prev: ResearchSession[]) => ResearchSession[],
	) => void,
) {
	const rehydratingRef = useRef<string | null>(null);
	const reportsDataRef = useRef<Map<string, string>>(new Map());

	useEffect(() => {
		const rehydrate = async () => {
			if (!urlId || rehydratingRef.current === urlId) return;
			rehydratingRef.current = urlId;

			try {
				const [sessRes, reportsRes] = await Promise.all([
					api.get("/api/research/sessions?includeArchived=true"),
					api.get(`/api/research/sessions/${urlId}/reports`),
				]);

				const sessionData = sessRes.data.find((s: any) => s.id === urlId);
				if (!sessionData) return;

				const reports = reportsRes.data || [];
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

				const handlers: CEStreamCallbacks = {
					onOpenaiChunk: (turnID, chunk) => {
						const content = chunk.choices[0]?.delta?.content;
						if (content) {
							const turn = session.turns.find((t) => t.id === turnID);
							if (turn) {
								turn.report += content;
							}
						}
					},
					onResearchTurnStarted(turnID, e) {
						const turn = createTurnFromEvent(e);
						session.turns.push(turn);
					},
					onResearchStepUpdate(turnID, e) {
						if (e.stepid && e.label && e.status) {
							const step = { id: e.stepid, label: e.label, status: e.status };
							const idx = session.steps.findIndex((s) => s.id === step.id);
							if (idx > -1) {
								session.steps[idx] = {
									...session.steps[idx],
									...step,
									status: step.status as any,
								};
							} else {
								session.steps.push({ ...step, status: step.status as any });
							}
						}
					},
					onResearchReasoningDelta(turnID, e) {
						if (e.content && e.content !== "") {
							const delta = e.content ?? "";
							session.thoughtProcess += delta;
						}
					},
					onResearchSourceAdded(turnID, e) {
						if (e.resource) {
							const source = e.resource;
							const turn = session.turns.find((t) => t.id === turnID);
							if (turn) {
								turn.sources.push(source);
							}
						}
					},
					onResourceMaterial(turnID, e) {
						if (e.resource) {
							const source = e.resource;
							const turn = session.turns.find((t) => t.id === turnID);
							if (turn) {
								turn.sources.push(source);
							}
						}
					},
					onLLMTryRunStart: (turnID) => {},
					onLLMTryRunEnd: () => {},
					onLLMTryRunFailed: (turnID) => {},
					onToolCallRequest: (turnID, e) => {},
					onToolCallResponse: (turnID, e) => {},
					onStreamEnd: (turnID) => {
						console.log("Stream ended for turn", turnID);
					},
				};

				const callbacks = withSourceDedup(handlers);

				for (const report of reports) {
					reportsDataRef.current.set(report.turnId, report.streamData);
					processCEText(report.turnId, report.streamData, callbacks);
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
	}, [urlId, setSessions]);

	return { reportsDataRef };
}

function useResearchStreaming(
	sessionId: string | undefined,
	setSessions: (
		updater: (prev: ResearchSession[]) => ResearchSession[],
	) => void,
	updateSession: (
		id: string,
		payload: Partial<ResearchSession>,
		cb?: (s: ResearchSession) => void,
	) => void,
) {
	const rollbackPointsRef = useRef<Map<string, RollbackPoint[]>>(new Map());

	const handleResearch = useCallback(
		async (query: string, _deep?: boolean) => {
			if (!sessionId) return;

			updateSession(sessionId, {
				state: "researching",
				thoughtProcess: "",
				steps: [],
			});

			const nextTurnID = nanoid();

			try {
				const response = await apiStream("/api/agent/research", {
					query,
					sessionId,
					turnId: nextTurnID,
				});

				// const response = await apiStream("/api/mock/research", {
				//   query,
				//   sessionId,
				//   turnId: nextTurnID,
				// });

				const reader = response.body?.getReader();
				if (!reader) throw new Error("No reader");

				const handlers: CEStreamCallbacks = {
					onOpenaiChunk: (turnID, chunk) => {
						const content = chunk.choices[0]?.delta?.content;
						if (content) {
							setSessions((current) => {
								const session = current.find((s) => s.id === sessionId);
								if (!session) {
									console.warn(
										"Session not found for appending report",
										sessionId,
									);
									return current;
								}
								const turn = session.turns.find((t) => t.id === turnID);
								if (!turn) {
									console.warn("Turn not found for appending report", turnID);
									return current;
								}
								const newTurn = {
									...turn,
									report: turn.report + content,
									updatedAt: Date.now(),
								};
								const newSession = {
									...session,
									turns: session.turns.map((t) =>
										t.id === turnID ? newTurn : t,
									),
								};
								const updated = current.map((s) =>
									s.id === sessionId ? newSession : s,
								);
								return updated;
							});
						}
					},
					onResearchTurnStarted: (_turnID, e) => {
						const turn = createTurnFromEvent(e);
						setSessions((current) => {
							const sesssion = current.find((s) => s.id === sessionId);
							if (!sesssion) {
								console.warn("Session not found for adding turn", sessionId);
								return current;
							}
							const newSession = {
								...sesssion,
								activeTurnId: turn.id,
								turns: [...sesssion.turns, turn],
							};
							const updated = current.map((s) =>
								s.id === sessionId ? newSession : s,
							);
							return updated;
						});
					},
					onResearchStepUpdate(_turnID, e) {
						if (e.stepid && e.label && e.status) {
							const step = {
								id: e.stepid,
								label: e.label,
								status: e.status,
							};
							setSessions((current) =>
								current.map((s) => {
									if (s.id !== sessionId) return s;
									const updatedSteps = [...s.steps];
									const existingIdx = updatedSteps.findIndex(
										(st) => st.id === step.id,
									);
									if (existingIdx > -1) {
										updatedSteps[existingIdx] = {
											...updatedSteps[existingIdx],
											...step,
											status: step.status as any,
										};
									} else {
										updatedSteps.push({ ...step, status: step.status as any });
									}
									return { ...s, steps: updatedSteps };
								}),
							);
						}
					},
					onResearchReasoningDelta: (_turnID, e) => {
						if (e.content && e.content !== "") {
							const delta = e.content;
							setSessions((current) => {
								const session = current.find((s) => s.id === sessionId);
								if (!session) {
									console.warn(
										"Session not found for appending thought process",
										sessionId,
									);
									return current;
								}
								const newSession = {
									...session,
									thoughtProcess: session.thoughtProcess + delta,
								};
								const updated = current.map((s) =>
									s.id === sessionId ? newSession : s,
								);
								return updated;
							});
						}
					},
					onResearchSourceAdded(turnID, e) {
						if (e.resource) {
							const source = e.resource;
							setSessions((current) => {
								const s = current.find((s) => s.id === sessionId);
								if (!s) {
									console.warn(
										"Session not found for adding source",
										sessionId,
									);
									return current;
								}
								const turn = s.turns.find((t) => t.id === turnID);
								if (!turn) {
									console.warn("Turn not found for adding source", turnID);
									return current;
								}
								const newTurn = { ...turn, sources: [...turn.sources, source] };
								console.log("Adding source to turn", turnID, turn, newTurn);
								const newSession = {
									...s,
									turns: s.turns.map((t) => (t.id === turnID ? newTurn : t)),
								};
								return current.map((s) =>
									s.id === sessionId ? newSession : s,
								);
							});
						}
					},
					onResourceMaterial(turnID, e) {
						if (e.resource) {
							const source = e.resource;
							setSessions((current) => {
								const s = current.find((s) => s.id === sessionId);
								if (!s) {
									console.warn(
										"Session not found for adding source",
										sessionId,
									);
									return current;
								}
								const turn = s.turns.find((t) => t.id === turnID);
								if (!turn) {
									console.warn("Turn not found for adding source", turnID);
									return current;
								}
								const newTurn = { ...turn, sources: [...turn.sources, source] };
								console.log("Adding source to turn", turnID, turn, newTurn);
								const newSession = {
									...s,
									turns: s.turns.map((t) => (t.id === turnID ? newTurn : t)),
								};
								return current.map((s) =>
									s.id === sessionId ? newSession : s,
								);
							});
						}
					},
					onLLMTryRunStart: (turnID) => {
						const points = rollbackPointsRef.current.get(sessionId) ?? [];
						points.push({ turnID, report: "", sources: [] });
						rollbackPointsRef.current.set(sessionId, points);
					},
					onLLMTryRunEnd: () => {
						const points = rollbackPointsRef.current.get(sessionId);
						if (points && points.length > 0) points.pop();
					},
					onLLMTryRunFailed: (turnID) => {
						const points = rollbackPointsRef.current.get(sessionId);
						if (points && points.length > 0) {
							const lastPoint = points.pop()!;
							setSessions((current) => {
								const session = current.find((s) => s.id === sessionId);
								if (!session) {
									console.warn(
										"Session not found for LLM try run failure",
										sessionId,
									);
									return current;
								}
								const turn = session.turns.find((t) => t.id === turnID);
								if (!turn) {
									console.warn(
										"Turn not found for LLM try run failure",
										turnID,
									);
									return current;
								}
								const newTurn = {
									...turn,
									report: lastPoint.report,
									sources: lastPoint.sources,
								};
								const newSession = {
									...session,
									turns: session.turns.map((t) =>
										t.id === turnID ? newTurn : t,
									),
								};
								const updated = current.map((s) =>
									s.id === sessionId ? newSession : s,
								);
								return updated;
							});
						}
					},
					onToolCallRequest: (turnID, e) => {},
					onToolCallResponse: (turnID, e) => {},
					onStreamEnd: (turnID) => {
						console.log("Stream ended for turn", turnID);
					},
				};

				await processCEStream(nextTurnID, reader, withSourceDedup(handlers));
				updateSession(sessionId, {}, (updatedSession) => {
					console.log("Session updated after stream end:", updatedSession);
				});
			} catch (error) {
				console.error("Research failed:", error);
			} finally {
				rollbackPointsRef.current.delete(sessionId);
				updateSession(
					sessionId,
					{ state: "reported", activeTurnId: undefined },
					(updatedSession) => {
						console.log("Session updated to reported:", updatedSession);
						persistSession(updatedSession).then(() => {
							if (updatedSession.turns.length === 1) {
								startSummarizeTask(updatedSession);
							}
						});
					},
				);
			}
		},
		[sessionId, setSessions, updateSession],
	);

	return { handleResearch };
}

// --- Main Component ---

function ResearchContent() {
	const router = useRouter();
	const searchParams = useSearchParams();
	const urlId = searchParams.get("id");

	const {
		sessions,
		setSessions,
		activeSessionId,
		setActiveSessionId,
		activeSession,
		updateSession,
		deleteTurn,
	} = useResearchSessionState();

	const scrollContainerRef = useRef<HTMLDivElement>(null);
	const { subscribe, unsubscribe } = useWebSocketContext();

	// Custom hooks for logic encapsulation
	const { reportsDataRef } = useResearchRehydration(urlId, setSessions);
	const { handleResearch } = useResearchStreaming(
		activeSessionId || undefined,
		setSessions,
		updateSession,
	);
	useResearchAutoScroll(scrollContainerRef, activeSession);

	// Sync activeSessionId with URL
	useEffect(() => {
		if (urlId && urlId !== activeSessionId) {
			setActiveSessionId(urlId);
		} else if (!urlId) {
			router.push("/");
		}
	}, [urlId, activeSessionId, setActiveSessionId, router]);

	// WebSocket Updates
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

	// Handlers
	const handleArchive = async (id: string) => {
		try {
			await api.post(`/api/research/sessions/${id}/archive`);
			setSessions((current) => current.filter((s) => s.id !== id));
			router.push("/new");
		} catch (e) {
			console.error("Archive failed", e);
		}
	};

	const handleClose = (id: string) => {
		setSessions((current) => current.filter((s) => s.id !== id));
		router.push("/new");
	};

	const handleRegenerate = async (turn: ResearchTurn) => {
		if (!activeSessionId) return;
		await deleteTurn(activeSessionId, turn.id);
		handleResearch(turn.query);
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

	const handleFetchRawStream = async (turnId: string) => {
		const rawData = reportsDataRef.current.get(turnId);
		if (rawData) return rawData;
		if (!activeSessionId) return null;

		try {
			const { data: reports } = await api.get(
				`/api/research/sessions/${activeSessionId}/reports`,
			);
			for (const r of reports) {
				reportsDataRef.current.set(r.turnId, r.streamData);
			}
			return reportsDataRef.current.get(turnId) || null;
		} catch (e) {
			console.error("Failed to fetch raw stream data", e);
			return null;
		}
	};

	if (!activeSession) return null;

	const isResearching =
		activeSession.state === "researching" ||
		activeSession.state === "reasoning";

	const isIdle =
		activeSession.state === "idle" && activeSession.turns.length === 0;

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
						>
							<X className="h-4 w-4" />
							Close
						</button>
						<button
							type="button"
							onClick={() => handleArchive(activeSession.id)}
							className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-md transition-colors"
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
							onResearch={(q, deep) => handleResearch(q, deep)}
						/>
					) : (
						(activeSession.turns.length > 0 || isResearching) && (
							<div className="mx-auto w-full space-y-12 pb-48">
								<ResearchReport
									turns={activeSession.turns}
									onDeleteTurn={(tid) => deleteTurn(activeSession.id, tid)}
									onRegenerateTurn={handleRegenerate}
									onSaveTurn={handleSaveTurn}
									onFetchRawStream={handleFetchRawStream}
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
					onSearch={(q, deep) => handleResearch(q, deep)}
					suggestions={
						!isResearching
							? [
									"Analyze performance implications",
									"How is this tested?",
									"Are there security concerns?",
								]
							: []
					}
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
