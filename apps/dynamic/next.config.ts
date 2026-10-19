import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  basePath: "/dynamic",
  transpilePackages: ["@eam/ui"],
};

export default nextConfig;
