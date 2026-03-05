# Frontend Research UI Plan

This plan outlines the refactoring of the default Home page into a "Research" page with a UI inspired by `deepwiki.com`. The goal is to create a sophisticated, AI-centric search and exploration interface.

## UI Analysis: DeepWiki-style Interface

A typical "Research" or "Deep Search" interface (like DeepWiki) focuses on:
1.  **Immersive Search Input**: A prominent, centered search area that transitions to the top upon query execution.
2.  **Reasoning/Thinking Process**: A dedicated area showing the "steps" the AI is taking (e.g., "Searching codebase...", "Analyzing chunks...", "Synthesizing answer...").
3.  **Structured Results**: Results are not just a list of links but an integrated report with source citations, code snippets, and structured explanations.
4.  **Interactive Exploration**: Options to "Deepen Research" or ask follow-up questions.

## Proposed Plan

### 1. Navigation Refactoring
- Update `frontend/src/app/nav-items.ts`:
    - Change "Home" label to "Research".
    - Update icon to `Search` or `Microscope`.
    - Change path to `/research` or keep `/` but update the semantic meaning.

### 2. New Component Architecture
- **`ResearchInput`**: A high-end search component with:
    - Multi-line support.
    - Mode toggles (e.g., "Standard Search", "Deep Analysis").
    - Visual feedback during input.
- **`ReasoningTrace`**: A component to display mock or real-time progress steps with "thinking" animations.
- **`SourceCard`**: A component to display code snippets or file references as "sources" used in the research.
- **`ResearchReport`**: The main display area for the synthesized answer or search results.

### 3. Page Structure (`src/app/page.tsx` or `src/app/research/page.tsx`)
- **Initial State**: Large centered search bar with minimal branding.
- **Active State**: 
    - Search bar moves to the top (sticky).
    - Left/Center area for the main "Answer/Report".
    - Right sidebar for "Sources" and "References" (optional, for desktop).

### 4. Mock Data & Interaction
- Implement a mock "Reasoning" phase that simulates a multi-step process.
- Create dummy data for search results to test the `SourceCard` and `ResearchReport` layouts.

## Visual Aesthetics
- **Color Palette**: Minimalist, high contrast (deep blacks/whites).
- **Typography**: Clean sans-serif (Inter) with monospace for code snippets.
- **Spacing**: Generous padding and "breathable" layout.
- **Animations**: Smooth transitions for the search bar and staggered appearance of reasoning steps.

## Folder Structure Update
```
frontend/src/app/_components/research/
├── research-input.tsx
├── reasoning-trace.tsx
├── source-card.tsx
└── research-report.tsx
```
