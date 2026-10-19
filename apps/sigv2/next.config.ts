import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  basePath: "/sigv2",
  transpilePackages: ["@eam/ui"],
};

export default nextConfig;
