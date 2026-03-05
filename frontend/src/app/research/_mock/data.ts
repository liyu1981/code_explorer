import { Source } from "../_components/source-card";

export const MOCK_REPORT_1 = `I have analyzed the codebase regarding **initial setup**. The project is a sophisticated TypeScript-based React application utilizing Next.js 16.

### Core Component Pattern
The application uses a modular pattern for its UI components. For instance, the WebSocket management is handled via a custom React hook pattern to ensure state synchronization across the app:

\`\`\`typescript
import { useState, useEffect, useCallback } from 'react';

interface WebSocketHook {
  isConnected: boolean;
  lastMessage: any;
  send: (msg: string) => void;
}

export function useSocket(url: string): WebSocketHook {
  const [socket, setSocket] = useState<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    const ws = new WebSocket(url);
    ws.onopen = () => setIsConnected(true);
    ws.onclose = () => setIsConnected(false);
    setSocket(ws);
    return () => ws.close();
  }, [url]);

  const send = useCallback((msg: string) => {
    socket?.send(msg);
  }, [socket]);

  return { isConnected, lastMessage: null, send };
}
\`\`\`

### State Management
State is managed using **Jotai**, which provides atomic state updates. This is particularly useful for the research session history, where each turn is appended to a global state atom.`;

export const MOCK_SOURCES_1: Source[] = [
  {
    id: "1",
    path: "src/hooks/useSocket.ts",
    snippet:
      "export function useSocket(url: string): WebSocketHook {\n  const [socket, setSocket] = useState<WebSocket | null>(null);",
  },
  {
    id: "2",
    path: "src/store/research.ts",
    snippet: "export const researchSessionsAtom = atom<ResearchSession[]>([]);",
  },
];

export const MOCK_REPORT_2 = `Deepening the analysis, I have examined the Markdown rendering implementation.

### Technical Implementation
The system uses \`react-markdown\` combined with \`rehype-highlight\` for syntax highlighting. The implementation details can be found in the core components:

\`\`\`typescript
import ReactMarkdown from "react-markdown";
import rehypeHighlight from "rehype-highlight";

interface MarkdownProps {
  content: string;
  className?: string;
}

export const MarkdownRenderer: React.FC<MarkdownProps> = ({ content, className }) => {
  return (
    <div className={className}>
      <ReactMarkdown 
        rehypePlugins={[rehypeHighlight]}
        components={{
          code({ node, inline, className, children, ...props }) {
            return !inline ? (
              <pre className="rounded-lg bg-gray-100 p-4">
                <code className={className} {...props}>
                  {children}
                </code>
              </pre>
            ) : (
              <code className="bg-gray-200 px-1 rounded" {...props}>
                {children}
              </code>
            );
          }
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
};
\`\`\`

### Styling Strategy
The project adopts **Tailwind CSS 4** for styling, leveraging the new \`@theme\` configuration for a more robust design system.`;

export const MOCK_SOURCES_2: Source[] = [
  {
    id: "3",
    path: "src/components/Markdown.tsx",
    snippet:
      "export const MarkdownRenderer: React.FC<MarkdownProps> = ({ content, className }) => {",
  },
  {
    id: "4",
    path: "tailwind.config.ts",
    snippet: "@theme {\n  --color-primary: oklch(0.205 0 0);\n}",
  },
];
