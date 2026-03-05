# Frontend Initialization Plan

This plan outlines the steps to initialize the `frontend` directory for the `code_explorer` project, based on the reference architecture found in `vendor/frontend`.

## Findings from Reference Frontend (`vendor/frontend`)

The reference frontend is a **Next.js 16** (React 19) application using **Tailwind CSS 4** and **Lucide React** for icons. It employs **Jotai** for state management and **react-use-websocket** for WebSocket connectivity.

### Key Architecture & Directory Structure
- **`src/app/`**: Next.js App Router structure.
  - **`_components/`**: Core UI components used across the app.
    - `websocket-provider.tsx`: Manages a shared WebSocket connection and provides subscription logic.
    - `app-nav-sidebar.tsx`: The main navigation sidebar with expansion/collapsing and connection status.
    - `base-layout.tsx`: Orchestrates the sidebar, main content area, and bottom panels.
    - `app-container.tsx` & `app-header.tsx`: Standard page layout wrappers.
  - **`_jotai/`**: Atomic state definitions (e.g., sidebar expansion, panel states).
  - **`settings/`**: System configuration page.
- **`src/lib/`**: Utility functions and API clients (`api.ts`).
- **`package.json`**: Uses `pnpm`, `next`, `jotai`, `axios`, and `lucide-react`.

### Specific Logic Identified
- **WebSocket**: Uses a `WebSocketProvider` with a topic-based subscription system (`subscribe(topic, callback)`). It maps the connection state to UI icons in the sidebar.
- **Navigation**: Defined in `nav-items.ts`. Supports top and bottom positions.
- **Theme/Styling**: Uses `geist` fonts and a specific font "Fontdiner Swanky" for the logo.

## Proposed Plan for `frontend/`

We will scaffold a new Next.js application in the `frontend` directory that mirrors this architecture but stripped down to the requested core features.

### 1. Project Scaffolding
- Initialize a new Next.js project with TypeScript, Tailwind CSS, and App Router.
- Configure `package.json` with dependencies identified (Jotai, react-use-websocket, axios, lucide-react, radix-ui components).
- Set up `pnpm` workspace or standalone pnpm configuration.

### 2. Core Library & State
- **`src/lib/api.ts`**: Implement `API_URL` and `GET_WS_URL` helpers.
- **`src/app/_jotai/`**: Define basic UI atoms (e.g., `isSidebarExpandedAtom`).

### 3. Essential Components (Replicated from vendor)
- **`WebSocketProvider`**: Implement the `WebSocketContext` and provider to manage the backend connection (`/api/ws`).
- **`AppNavSidebar`**: Implement the sidebar with:
  - Navigation links (Home/Search, Settings).
  - Connection status indicator.
  - Collapse/Expand functionality.
- **`BaseLayout`**: Implement the top-level layout that includes the sidebar and content wrapper.
- **`AppContainer` & `AppHeader`**: Standard wrappers for page content.

### 4. Initial Pages
- **`src/app/page.tsx`**: Default landing page (Home).
- **`src/app/settings/page.tsx`**: A simplified settings page to manage `code_explorer` configurations.

### 5. Integration
- Connect the frontend to the backend API (defaulting to `:8080`).
- Ensure the WebSocket connection correctly reflects the server status.

## Folder Structure to be Created
```
frontend/
├── src/
│   ├── app/
│   │   ├── _components/
│   │   │   ├── app-container.tsx
│   │   │   ├── app-header.tsx
│   │   │   ├── app-nav-sidebar.tsx
│   │   │   ├── base-layout.tsx
│   │   │   └── websocket-provider.tsx
│   │   ├── _jotai/
│   │   │   └── ui-store.ts
│   │   ├── settings/
│   │   │   └── page.tsx
│   │   ├── globals.css
│   │   ├── layout.tsx
│   │   ├── nav-items.ts
│   │   └── page.tsx
│   ├── lib/
│   │   ├── api.ts
│   │   └── utils.ts
│   └── types/
├── next.config.ts
├── package.json
├── postcss.config.mjs
├── tailwind.config.ts
└── tsconfig.json
```
