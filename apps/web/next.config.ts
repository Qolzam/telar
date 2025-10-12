import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  // Explicitly set the monorepo root to avoid lockfile detection issues
  outputFileTracingRoot: path.resolve(__dirname, "../.."),
};

export default nextConfig;
