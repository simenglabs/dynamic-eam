import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  basePath: "/administration",
  transpilePackages: ["@eam/ui"],
};

export default nextConfig;
