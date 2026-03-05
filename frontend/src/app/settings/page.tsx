"use client";

import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";

export default function SettingsPage() {
  return (
    <AppContainer>
      <AppHeader>
        <h1 className="text-xl font-bold text-primary">System Settings</h1>
      </AppHeader>
      <div className="flex-1 p-6 overflow-auto">
        <div className="max-w-4xl mx-auto space-y-8">
          <section className="space-y-4">
            <h2 className="text-lg font-semibold border-b pb-2">
              Configuration
            </h2>
            <div className="p-4 border rounded bg-muted/20">
              <p className="text-sm text-muted-foreground italic">
                Settings and system information will be displayed here.
              </p>
            </div>
          </section>
        </div>
      </div>
    </AppContainer>
  );
}
