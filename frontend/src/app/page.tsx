import { AppContainer } from "./_components/app-container";
import { AppHeader } from "./_components/app-header";

export default function HomePage() {
  return (
    <AppContainer>
      <AppHeader>
        <h1 className="text-xl font-bold text-primary">Home / Search</h1>
      </AppHeader>
      <div className="flex-1 p-6 overflow-auto">
        <div className="max-w-4xl mx-auto space-y-6">
          <div className="p-8 border rounded-lg bg-card text-center space-y-4">
            <h2 className="text-2xl font-semibold">Welcome to code_explorer</h2>
            <p className="text-muted-foreground">
              This is the initial build of the frontend. Search and codebase exploration features will be implemented here.
            </p>
          </div>
        </div>
      </div>
    </AppContainer>
  );
}
