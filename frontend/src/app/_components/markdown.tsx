"use client";

import ReactMarkdown from "react-markdown";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import { oneLight } from "react-syntax-highlighter/dist/esm/styles/prism";
import remarkGfm from "remark-gfm";
import { cn } from "@/lib/utils";

interface MarkdownProps {
  content: string;
  className?: string;
}

export function Markdown({ content, className }: MarkdownProps) {
  return (
    <div
      className={cn(
        "prose prose-lg dark:prose-invert max-w-none prose-headings:font-bold prose-p:leading-relaxed prose-pre:p-0 prose-pre:bg-transparent",
        className,
      )}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          pre: ({ children }) => (
            <div className="not-prose rounded-xl overflow-hidden border border-border/60 bg-[#fafafa] my-6 shadow-sm">
              {children}
            </div>
          ),
          code: ({ inline, className, children, ...props }: any) => {
            const match = /language-(\w+)/.exec(className || "");
            const language = match ? match[1] : "text";
            const codeString = String(children).replace(/\n$/, "");

            if (!inline && match) {
              return (
                <SyntaxHighlighter
                  language={language}
                  style={oneLight}
                  showLineNumbers={true}
                  lineNumberStyle={{
                    minWidth: "3em",
                    paddingRight: "1.5em",
                    color: "#a0a0a0",
                    userSelect: "none",
                    textAlign: "right",
                    borderRight: "1px solid #e0e0e0",
                    marginRight: "1em",
                  }}
                  customStyle={{
                    margin: 0,
                    padding: "1.25rem 0",
                    fontSize: "13px",
                    lineHeight: "1.6",
                    background: "#fafafa",
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
                  {codeString}
                </SyntaxHighlighter>
              );
            }

            return (
              <code
                className={cn(
                  "bg-muted/80 px-1.5 py-0.5 rounded text-xs font-mono font-medium",
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
