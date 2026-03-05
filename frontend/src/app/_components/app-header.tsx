export function AppHeader({ children }: { children: React.ReactNode }) {
  return (
    <header className="h-[60px] border-b px-6 py-3 flex items-center justify-between bg-card">
      {children}
    </header>
  );
}
