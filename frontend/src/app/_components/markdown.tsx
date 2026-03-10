"use client";

import ReactMarkdown from "react-markdown";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import {
  oneLight,
  oneDark,
} from "react-syntax-highlighter/dist/esm/styles/prism";
import remarkGfm from "remark-gfm";
import { useTheme } from "next-themes";
import { cn } from "@/lib/utils";

interface MarkdownProps {
  content: string;
  className?: string;
}

export function Markdown({ content, className }: MarkdownProps) {
  const { theme } = useTheme();
  const isDark = theme === "dark";

  return (
    <div
      className={cn(
        "prose prose-slate dark:prose-invert max-w-none prose-headings:text-foreground prose-p:text-foreground/90 prose-strong:text-foreground prose-code:text-foreground prose-headings:font-bold prose-p:leading-relaxed prose-pre:p-0 prose-pre:bg-transparent",
        className,
      )}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          pre: ({ children }) => (
            <div className="not-prose rounded-xl overflow-hidden border border-border/60 bg-muted/30 my-6 shadow-sm">
              {children}
            </div>
          ),
          code: ({ node, inline, className, children, ...props }: any) => {
            const match = /language-(\w+)/.exec(className || "");
            const language = match ? match[1] : "text";

            // Extract text content safely from children
            const codeString = Array.isArray(children)
              ? children.join("")
              : typeof children === "string"
                ? children
                : String(children || "");

            if (!inline && match) {
              return (
                <SyntaxHighlighter
                  language={language}
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
                    padding: "1.25rem 0",
                    fontSize: "13px",
                    lineHeight: "1.6",
                    background: isDark ? "#282c34" : "#fafafa",
                    fontFamily: "var(--font-geist-mono)",
                  }}
                  codeTagProps={{
                    style: {
                      display: "block",
                      padding: "0 1.25rem",
                    },
                  }}
                  {...props}
                >
                  {codeString.replace(/\n$/, "")}
                </SyntaxHighlighter>
              );
            }

            return (
              <code
                className={cn(
                  "bg-muted/80 px-1.5 py-0.5 rounded text-xs font-mono font-medium text-foreground",
                  className,
                )}
                {...props}
              >
                {children}
              </code>
            );
          },
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
