import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

interface BadgeProps {
  children: ReactNode;
  variant?: "default" | "success" | "destructive" | "primary" | "outline";
  className?: string;
  icon?: ReactNode;
}

export function Badge({
  children,
  variant = "default",
  className,
  icon,
}: BadgeProps) {
  const variants = {
    default: "bg-muted text-muted-foreground border-border",
    success: "bg-green-500/10 text-green-500 border-green-500/20",
    destructive: "bg-destructive/10 text-destructive border-destructive/20",
    primary: "bg-primary/10 text-primary border-primary/20",
    outline: "bg-transparent border-border",
  };

  return (
    <div
      className={cn(
        "inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider border",
        variants[variant],
        className,
      )}
    >
      {icon}
      {children}
    </div>
  );
}
