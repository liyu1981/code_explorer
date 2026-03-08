import path from "node:path";
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export",
  // Disable image optimization for static exports
  images: {
    unoptimized: true,
  },
  turbopack: {
    root: path.join(__dirname, "."),
  },
  allowedDevOrigins: ["192.168.0.*"],
};

export default nextConfig;
