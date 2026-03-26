"use client";

import { useAtom } from "jotai";
import { Archive } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef } from "react";
import { cn } from "@/lib/utils";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import {
	activeSessionIdAtom,
	createSession,
	researchSessionsAtom,
} from "../_jotai/research-store";
import { FloatingThoughtProcess } from "../research/_components/floating-thought-process";
import { IdleResearchView } from "../research/_components/idle-research-view";
import { ResearchReport } from "../research/_components/research-report";
import { StickyResearchInput } from "../research/_components/sticky-research-input";
import { createStreamingCallbacks } from "../research/_hooks/useResearchStreamProcessor";
import { getMockStream } from "./_mock/ce";
function ResearchMockContent() {
	const [sessions, setSessions] = useAtom(researchSessionsAtom);
	const [activeSessionId, setActiveSessionId] = useAtom(activeSessionIdAtom);
	const scrollContainerRef = useRef<HTMLDivElement>(null);
	const searchParams = useSearchParams();
	const keepThoughtOpen = searchParams.get("keepThoughtOpen") === "true";

	const prevTurnsLengthRef = useRef(0);

	// Initialize a default mock session if none exists
	useEffect(() => {
		if (sessions.length === 0) {
			setSessions((current) => {
				if (current.length === 0) {
					const mockSession = createSession(
						"mock-id",
						"/mock/path",
						"mock-project",
						"v1.0.0",
					);
					mockSession.title = "Mock Research Session";
					return [mockSession];
				}
				return current;
			});
		}
	}, [sessions.length, setSessions]);

	useEffect(() => {
		if (sessions.length > 0 && !activeSessionId) {
			setActiveSessionId(sessions[0].id);
		}
	}, [sessions, activeSessionId, setActiveSessionId]);

	const activeSession = sessions.find((s) => s.id === activeSessionId);

	// biome-ignore lint/correctness/useExhaustiveDependencies: scroll management
	useEffect(() => {
		if (!scrollContainerRef.current) return;
		const turnsLength = activeSession?.turns.length ?? 0;
		const activeTurnId = activeSession?.activeTurnId;
		const isStreaming = !!activeTurnId;
		const isNewTurn = turnsLength > prevTurnsLengthRef.current;

		// Update ref for next run
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

				// If we are significantly off target (> 5px), and we are either in a "new turn" event
				// OR we are currently streaming that turn, retry the scroll.
				if (Math.abs(currentTop - targetTop) > 5) {
					scrollContainerRef.current.scrollTo({
						top: targetTop,
						behavior: isNewTurn ? "smooth" : "auto", // Smooth for first jump, auto for micro-adjustments
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

	const handleResearch = async (
		sessionId: string,
		query: string,
		_deep: boolean,
	) => {
		setSessions((current) =>
			current.map((s) =>
				s.id === sessionId
					? {
							...s,
							state: "researching",
							thoughtProcess: "",
							steps: [],
						}
					: s,
			),
		);

		const stream = getMockStream(query);
		const callbacks = createStreamingCallbacks(sessionId, setSessions, { current: new Map() });

		for (const line of stream) {
			await new Promise((resolve) => setTimeout(resolve, 50));

			if (line.startsWith("data: ")) {
				try {
					callbacks.onOpenaiChunk(crypto.randomUUID(), JSON.parse(line.slice(6)));
				} catch (e) {
					console.error("Failed to parse mock data chunk", e, line);
				}
			} else if (line.startsWith("ce: ")) {
				try {
					const event = JSON.parse(line.slice(4));
					switch (event.object) {
						case "llm.try.run.start":
							callbacks.onLLMTryRunStart(crypto.randomUUID(), event);
							break;
						case "llm.try.run.end":
							callbacks.onLLMTryRunEnd(crypto.randomUUID(), event);
							break;
						case "llm.try.run.failed":
							callbacks.onLLMTryRunFailed(crypto.randomUUID(), event);
							break;
						case "research.turn.started":
							callbacks.onResearchTurnStarted(crypto.randomUUID(), event);
							break;
						case "research.step.update":
							callbacks.onResearchStepUpdate(crypto.randomUUID(), event);
							break;
						case "research.reasoning.delta":
							callbacks.onResearchReasoningDelta(crypto.randomUUID(), event);
							break;
						case "research.source.added":
							callbacks.onResearchSourceAdded(crypto.randomUUID(), event);
							break;
						case "resource.material":
							callbacks.onResourceMaterial(crypto.randomUUID(), event);
							break;
					}
				} catch (e) {
					console.error("Failed to parse mock CE event", e, line);
				}
			}
		}

		setSessions((current) =>
			current.map((s) =>
				s.id === sessionId
					? { ...s, state: "reported", activeTurnId: undefined }
					: s,
			),
		);
	};

	const handleArchive = (id: string) => {
		setSessions((current) => current.filter((s) => s.id !== id));
	};

	if (!activeSession) {
		return (
			<div className="flex items-center justify-center h-full">
				No active session. Select one from sidebar.
			</div>
		);
	}

	const isResearching =
		keepThoughtOpen || activeSession.state === "researching";
	const isIdle = activeSession.state === "idle";

	return (
		<AppContainer>
			<AppHeader>
				<div className="flex items-center gap-4 w-full">
					<h1 className="text-xl font-bold tracking-tight text-primary">
						Research Mock UI
					</h1>
					<button
						type="button"
						onClick={() => handleArchive(activeSession.id)}
						className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-md transition-colors ml-auto"
						title="Archive Research"
					>
						<Archive className="h-4 w-4" />
						Archive
					</button>
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
							onResearch={(q, deep) => handleResearch(activeSession.id, q, deep)}
						/>
					) : (
						(activeSession.turns.length > 0 || isResearching) && (
							<div className="mx-auto w-full space-y-12 pb-48">
								<ResearchReport
									turns={activeSession.turns}
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
				/>
			</div>
		</AppContainer>
	);
}

export default function ResearchMockPage() {
	return (
		<Suspense fallback={<div>Loading...</div>}>
			<ResearchMockContent />
		</Suspense>
	);
}
