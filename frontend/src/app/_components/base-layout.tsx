"use client";

import React from "react";
import { AppNavSidebar } from "./app-nav-sidebar";

export function BaseLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="h-screen flex bg-background text-foreground">
      <AppNavSidebar />
      <div className="flex-1 flex flex-col relative overflow-hidden">
        <main className="flex-1 flex flex-col overflow-auto">{children}</main>
      </div>
    </div>
  );
}
