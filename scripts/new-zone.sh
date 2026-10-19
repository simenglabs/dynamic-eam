#!/usr/bin/env bash
# Usage: scripts/new-zone.sh <zone> <web-port> <api-port>
#        SERVICE_ONLY=1 scripts/new-zone.sh <zone> <web-port> <api-port>   # Go service only
# Creates a Next.js zone (apps/<zone>, basePath /<zone>) + Go service (services/<zone>).
# Then add the zone entry in packages/ui/src/menu.ts with the same ports.
set -euo pipefail
cd "$(dirname "$0")/.."
Z=$1 WEB=$2 API=$3
if [ "${SERVICE_ONLY:-}" != 1 ]; then
[ -e "apps/$Z" ] && { echo "apps/$Z already exists"; exit 1; }

mkdir -p "apps/$Z/app/[[...slug]]"
cat > "apps/$Z/package.json" <<EOF
{
  "name": "@eam/$Z",
  "private": true,
  "scripts": {
    "dev": "next dev -p $WEB",
    "build": "next build",
    "start": "next start -p $WEB"
  },
  "dependencies": {
    "@eam/ui": "workspace:*",
    "lucide-react": "^1.59.0",
    "next": "16.4.2",
    "react": "^19.3.1",
    "react-dom": "^19.3.1"
  },
  "devDependencies": {
    "@tailwindcss/postcss": "^4.3.3",
    "@types/node": "^24",
    "@types/react": "^19",
    "@types/react-dom": "^19",
    "tailwindcss": "^4.3.3",
    "typescript": "^5"
  }
}
EOF
cp apps/shell/tsconfig.json apps/shell/postcss.config.mjs apps/shell/app/globals.css "apps/$Z/" 2>/dev/null || true
mv "apps/$Z/globals.css" "apps/$Z/app/globals.css"
cp packages/ui/src/icon.svg "apps/$Z/app/icon.svg" # favicon
cat > "apps/$Z/next.config.ts" <<EOF
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  basePath: "/$Z",
  transpilePackages: ["@eam/ui"],
};

export default nextConfig;
EOF
cat > "apps/$Z/app/layout.tsx" <<EOF
import "./globals.css";
import { AppLayout } from "@eam/ui";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return <AppLayout zone="$Z">{children}</AppLayout>;
}
EOF
cat > "apps/$Z/app/[[...slug]]/page.tsx" <<EOF
import { ZonePage } from "@eam/ui";

export default function Page(props: { params: Promise<{ slug?: string[] }> }) {
  return <ZonePage zone="$Z" params={props.params} />;
}
EOF

fi # SERVICE_ONLY

mkdir -p "services/$Z/internal/http"
cat > "services/$Z/main.go" <<EOF
// Service $Z. main = composition root.
package main

import (
	"net/http"

	"eam/services/common"
	zonehttp "eam/services/$Z/internal/http"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	zonehttp.Register(r.Group("/api/$Z"))
	r.Run(":" + common.Env("PORT", "$API"))
}
EOF
cat > "services/$Z/internal/http/handler.go" <<EOF
// Package http: HTTP adapter for service $Z.
// DDD layout when adding features: internal/domain/<aggregate> (entity + repository port, tests first),
// internal/app (use cases), internal/infra (mongo, etc.), then handlers here with
// az.Require("$Z/<submenu>", common.Read) — key = path in packages/ui/src/menu.ts. Full example: services/administration.
package http

import (
	"net/http"

	"eam/services/common"
	"github.com/gin-gonic/gin"
)

func Register(rg *gin.RouterGroup) {
	a := rg.Group("", common.Auth())
	a.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"zone": "$Z"}) })
}
EOF
cat > "services/$Z/internal/http/handler_test.go" <<EOF
package http

import (
	"net/http/httptest"
	"testing"
	"time"

	"eam/services/common"
	"github.com/gin-gonic/gin"
)

func TestPingRequiresLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r.Group("/api/$Z"))
	tok, _ := common.Sign("u1", "u1", "U1", time.Hour)
	for _, c := range []struct {
		auth string
		want int
	}{{"", 401}, {"Bearer " + tok, 200}} {
		req := httptest.NewRequest("GET", "/api/$Z/ping", nil)
		if c.auth != "" {
			req.Header.Set("Authorization", c.auth)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != c.want {
			t.Errorf("auth=%v: %d, want %d", c.auth != "", w.Code, c.want)
		}
	}
}
EOF
(cd "services/$Z" && { [ -f go.mod ] || go mod init "eam/services/$Z" >/dev/null 2>&1; } \
  && go mod edit -require=eam/services/common@v0.0.0 -replace=eam/services/common=../common && go get github.com/gin-gonic/gin@v1.12.0 >/dev/null 2>&1 && go mod tidy >/dev/null 2>&1)
(cd services && { [ -f go.work ] || go work init; } && go work use "./$Z")
echo "zone $Z created (web :$WEB, api :$API). Add an entry in packages/ui/src/menu.ts."
