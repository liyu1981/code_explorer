export function AppContainer({ children }: { children: React.ReactNode }) {
  return <div className="flex-1 flex flex-col h-full overflow-hidden">{children}</div>;
}
