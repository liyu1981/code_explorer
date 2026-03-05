import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { Toaster } from "sonner";
import { BaseLayout } from "./_components/base-layout";
import { WebSocketProvider } from "./_components/websocket-provider";

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
    <html lang="en">
      <body className={inter.className}>
        <WebSocketProvider>
          <BaseLayout>{children}</BaseLayout>
        </WebSocketProvider>
        <Toaster />
      </body>
    </html>
  );
}
