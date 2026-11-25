import type { NextConfig } from "next";

// The browser calls /api/... on this app's own origin and Next.js forwards it
// to the Go API. That keeps the session cookie same-site and removes the need
// for CORS. The target is read when `next dev` / `next build` runs.
const apiTarget = process.env.API_PROXY_TARGET ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // The Docker build (frontend/Dockerfile) sets NEXT_OUTPUT=standalone so
  // `next build` writes a self-contained server to .next/standalone; local
  // `npm run build && npm start` keeps the default output.
  output: process.env.NEXT_OUTPUT === "standalone" ? "standalone" : undefined,
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiTarget}/api/:path*` }];
  },
};

export default nextConfig;
