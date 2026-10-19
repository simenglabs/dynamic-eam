import type { NextConfig } from "next";
import { MENU } from "../../packages/ui/src/menu";

const env = (zone: string, kind: "WEB" | "API") => process.env[`${zone.toUpperCase().replace(/-/g, "_")}_${kind}_URL`];

// Shell = the browser's single entry point. Each top-level menu is rewritten to its zone,
// /api/<zone>/* is rewritten to its Go service (for fetches from client components).
const nextConfig: NextConfig = {
  transpilePackages: ["@eam/ui"],
  async redirects() {
    return [{ source: "/", destination: "/dashboard", permanent: false }];
  },
  async rewrites() {
    return MENU.filter((z) => z.webPort && z.apiPort).flatMap(({ zone, webPort, apiPort }) => {
      const web = env(zone, "WEB") ?? `http://localhost:${webPort}`;
      const api = env(zone, "API") ?? `http://localhost:${apiPort}`;
      return [
        { source: `/${zone}`, destination: `${web}/${zone}` },
        { source: `/${zone}/:path*`, destination: `${web}/${zone}/:path*` },
        { source: `/api/${zone}/:path*`, destination: `${api}/api/${zone}/:path*` },
      ];
    });
  },
};

export default nextConfig;
