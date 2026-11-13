import type { NextConfig } from "next";

// The browser calls /api/... on this app's own origin and Next.js forwards it
// to the Go API. That keeps the session cookie same-site and removes the need
// for CORS. The target is read when `next dev` / `next build` runs.
const apiTarget = process.env.API_PROXY_TARGET ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiTarget}/api/:path*` }];
  },
};

export default nextConfig;
