import type { Metadata } from "next";
import { Inter, Geist } from "next/font/google";
import "./globals.css";
import { Toaster } from "sonner";
import { BaseLayout } from "./_components/base-layout";
import { WebSocketProvider } from "./_components/websocket-provider";
import { Provider } from "jotai";
import { cn } from "@/lib/utils";

const geist = Geist({subsets:['latin'],variable:'--font-sans'});

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "code_explorer",
  description: "Code exploration and search platform",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={cn("font-sans", geist.variable)}>
      <body className={inter.className}>
        <Provider>
          <WebSocketProvider>
            <BaseLayout>{children}</BaseLayout>
          </WebSocketProvider>
          <Toaster />
        </Provider>
      </body>
    </html>
  );
}
