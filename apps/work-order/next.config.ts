import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  basePath: "/work-order",
  transpilePackages: ["@eam/ui"],
};

export default nextConfig;
