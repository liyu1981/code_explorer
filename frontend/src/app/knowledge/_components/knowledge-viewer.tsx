"use client";

import { useEffect, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import {
  oneDark,
  oneLight,
} from "react-syntax-highlighter/dist/esm/styles/prism";
import { useTheme } from "next-themes";
import {
  Check,
  Copy,
  Maximize2,
  ZoomIn,
  ZoomOut,
  RotateCcw,
  X,
  Move,
} from "lucide-react";
import * as Dialog from "@radix-ui/react-dialog";
import { cn } from "@/lib/utils";
import mermaid from "mermaid";
import { Button } from "@/components/ui/button";

const Mermaid = ({ chart }: { chart: string }) => {
  const ref = useRef<HTMLDivElement>(null);
  const lightboxRef = useRef<HTMLDivElement>(null);
  const { theme } = useTheme();
  const isDark = theme === "dark";
  const [svgContent, setSvgContent] = useState<string>("");
  const [isOpen, setIsOpen] = useState(false);
  const [zoom, setZoom] = useState(1);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  useEffect(() => {
    mermaid.initialize({
      startOnLoad: true,
      theme: isDark ? "dark" : "default",
      securityLevel: "loose",
    });
  }, [isDark]);

  useEffect(() => {
    if (chart) {
      const renderMermaid = async () => {
        try {
          const id = `mermaid-${Math.random().toString(36).substr(2, 9)}`;
          const { svg } = await mermaid.render(id, chart);
          setSvgContent(svg);
          if (ref.current) {
            ref.current.innerHTML = svg;
          }
        } catch (e) {
          console.error("Mermaid render error", e);
        }
      };
      renderMermaid();
    }
  }, [chart]);

  const handleReset = () => {
    setZoom(1);
    setPosition({ x: 0, y: 0 });
  };

  const handleZoomIn = () => setZoom((prev) => Math.min(prev + 0.2, 5));
  const handleZoomOut = () => setZoom((prev) => Math.max(prev - 0.2, 0.5));

  const handleMouseDown = (e: React.MouseEvent) => {
    setIsDragging(true);
    setDragStart({ x: e.clientX - position.x, y: e.clientY - position.y });
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isDragging) {
      setPosition({
        x: e.clientX - dragStart.x,
        y: e.clientY - dragStart.y,
      });
    }
  };

  const handleMouseUp = () => setIsDragging(false);

  return (
    <div className="group relative my-6 bg-muted/30 p-4 rounded-xl border border-border/50 transition-all hover:bg-muted/40">
      <Button
        variant="outline"
        size="icon-xs"
        className="absolute top-3 right-3 opacity-0 group-hover:opacity-100 z-10"
        onClick={() => setIsOpen(true)}
      >
        <Maximize2 className="h-4 w-4" />
      </Button>
      <div
        ref={ref}
        className="mermaid w-full max-w-full overflow-auto flex justify-center"
      />

      <Dialog.Root open={isOpen} onOpenChange={setIsOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-background/80 backdrop-blur-sm z-[100] animate-in fade-in duration-200" />
          <Dialog.Content className="fixed inset-4 md:inset-10 bg-card border border-border shadow-2xl rounded-3xl z-[101] flex flex-col overflow-hidden animate-in zoom-in-95 duration-200">
            <div className="flex items-center justify-between p-4 border-b border-border/50 bg-muted/10">
              <Dialog.Title className="text-sm font-bold uppercase tracking-widest text-muted-foreground flex items-center gap-2">
                <Move className="h-4 w-4" />
                Diagram Viewer
              </Dialog.Title>
              <div className="flex items-center gap-2">
                <div className="flex items-center bg-background/50 border border-border/50 rounded-xl p-1 gap-1">
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    onClick={handleZoomOut}
                    title="Zoom Out"
                  >
                    <ZoomOut className="h-4 w-4" />
                  </Button>
                  <span className="text-[10px] font-mono font-bold w-12 text-center">
                    {Math.round(zoom * 100)}%
                  </span>
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    onClick={handleZoomIn}
                    title="Zoom In"
                  >
                    <ZoomIn className="h-4 w-4" />
                  </Button>
                  <div className="w-px h-4 bg-border/50 mx-1" />
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    onClick={handleReset}
                    title="Reset View"
                  >
                    <RotateCcw className="h-4 w-4" />
                  </Button>
                </div>
                <Dialog.Close asChild>
                  <Button variant="ghost" size="icon-xs">
                    <X className="h-5 w-5" />
                  </Button>
                </Dialog.Close>
              </div>
            </div>

            <div
              className="flex-1 overflow-hidden relative cursor-grab active:cursor-grabbing bg-[#0d1117]/5"
              onMouseDown={handleMouseDown}
              onMouseMove={handleMouseMove}
              onMouseUp={handleMouseUp}
              onMouseLeave={handleMouseUp}
            >
              <div
                style={{
                  transform: `translate(${position.x}px, ${position.y}px) scale(${zoom})`,
                  transformOrigin: "center",
                  transition: isDragging ? "none" : "transform 0.2s ease-out",
                }}
                className="w-full h-full flex items-center justify-center p-20 pointer-events-none"
                dangerouslySetInnerHTML={{ __html: svgContent }}
              />
            </div>

            <div className="p-3 bg-muted/20 border-t border-border/50 flex justify-center">
              <p className="text-[10px] text-muted-foreground font-medium flex items-center gap-2">
                <span className="px-1.5 py-0.5 rounded bg-background border border-border/50">
                  Drag
                </span>{" "}
                to move
                <span className="mx-1">•</span>
                <span className="px-1.5 py-0.5 rounded bg-background border border-border/50">
                  Scroll
                </span>{" "}
                to zoom (coming soon)
              </p>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </div>
  );
};
const CopyButton = ({ text }: { text: string }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      try {
        await navigator.clipboard.writeText(text);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
        return;
      } catch (err) {
        console.error("Clipboard API failed, falling back", err);
      }
    }

    // Fallback for non-secure contexts or missing API
    const textArea = document.createElement("textarea");
    textArea.value = text;
    textArea.style.position = "fixed";
    textArea.style.left = "-9999px";
    textArea.style.top = "0";
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();

    try {
      document.execCommand("copy");
      setCopied(true);
    } catch (err) {
      console.error("Fallback copy failed", err);
    }

    document.body.removeChild(textArea);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <Button
      variant="ghost"
      size="icon-xs"
      onClick={handleCopy}
      title="Copy code"
    >
      {copied ? (
        <Check className="h-3.5 w-3.5 text-green-500" />
      ) : (
        <Copy className="h-3.5 w-3.5" />
      )}
    </Button>
  );
};

export function KnowledgeViewer({ content }: { content: string }) {
  const { theme } = useTheme();
  const isDark = theme === "dark";

  const components: any = {
    code: ({ node, className, children, ...props }: any) => {
      const match = /language-(\w+)/.exec(className || "");
      const isMermaid = match?.[1] === "mermaid";

      if (isMermaid) {
        return <Mermaid chart={String(children).replace(/\n$/, "")} />;
      }

      if (match) {
        const codeString = String(children).replace(/\n$/, "");
        return (
          <div
            className={cn(
              "rounded-xl overflow-hidden border border-border/60 my-6 shadow-sm",
              isDark ? "bg-[#282c34]" : "bg-[#fafafa]",
            )}
          >
            <div className="flex items-center justify-between px-4 py-1.5 border-b border-border/50 bg-muted/30">
              <span className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
                {match[1]}
              </span>
              <CopyButton text={codeString} />
            </div>
            <SyntaxHighlighter
              language={match[1]}
              style={isDark ? oneDark : oneLight}
              showLineNumbers={true}
              lineNumberStyle={{
                minWidth: "3em",
                paddingRight: "1.5em",
                color: isDark ? "#636d83" : "#a0a0a0",
                userSelect: "none",
                textAlign: "right",
                borderRight: `1px solid ${isDark ? "#2c313c" : "#e0e0e0"}`,
                marginRight: "1em",
              }}
              customStyle={{
                margin: 0,
                borderRadius: 0,
                padding: "1.25rem 0",
                fontSize: "13px",
                lineHeight: "1.6",
                backgroundColor: "transparent",
                fontFamily: "var(--font-geist-mono)",
              }}
              codeTagProps={{
                style: {
                  display: "block",
                  padding: "0 1.25rem",
                },
              }}
            >
              {codeString}
            </SyntaxHighlighter>
          </div>
        );
      }

      return (
        <code
          className={cn(
            "bg-muted px-1.5 py-0.5 rounded text-primary font-mono text-[0.9em]",
            className,
          )}
          {...props}
        >
          {children}
        </code>
      );
    },
    table: ({ children }: any) => (
      <div className="overflow-x-auto my-8 border border-border/50 rounded-2xl">
        <table className="w-full border-collapse m-0">{children}</table>
      </div>
    ),
    thead: ({ children }: any) => (
      <thead className="bg-muted/30 text-foreground">{children}</thead>
    ),
    th: ({ children }: any) => (
      <th className="px-6 py-4 text-left text-xs font-bold uppercase tracking-widest border-b border-border/50 text-foreground">
        {children}
      </th>
    ),
    td: ({ children }: any) => (
      <td className="px-6 py-4 text-sm border-b border-border/20 text-foreground/80">
        {children}
      </td>
    ),
    h1: ({ children }: any) => (
      <h1 className="text-3xl font-bold tracking-tight text-foreground mb-8 border-b border-border/20 pb-4">
        {children}
      </h1>
    ),
    h2: ({ children }: any) => (
      <h2 className="text-2xl font-bold tracking-tight text-foreground mt-12 mb-6 border-l-4 border-primary pl-4">
        {children}
      </h2>
    ),
    h3: ({ children }: any) => (
      <h3 className="text-xl font-bold tracking-tight text-foreground mt-8 mb-4">
        {children}
      </h3>
    ),
    p: ({ children }: any) => (
      <p className="leading-relaxed text-foreground/90 mb-6">{children}</p>
    ),
    pre: ({ children }: any) => <pre className="bg-background">{children}</pre>,
    ul: ({ children }: any) => (
      <ul className="space-y-3 mb-8 list-none p-0">{children}</ul>
    ),
    li: ({ children }: any) => (
      <li className="flex gap-3 text-foreground/80">
        <span className="text-primary mt-1.5 flex-shrink-0">•</span>
        <span>{children}</span>
      </li>
    ),
    blockquote: ({ children }: any) => (
      <blockquote className="border-l-4 border-primary/40 bg-primary/5 p-6 rounded-r-2xl italic text-foreground my-8">
        {children}
      </blockquote>
    ),
  };

  return (
    <div className="prose prose-slate dark:prose-invert max-w-none w-full">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {content}
      </ReactMarkdown>
    </div>
  );
}
