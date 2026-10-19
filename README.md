# Dynamic EAM

Enterprise Asset Management platform where menus, forms, master data and approval flows are **configured at runtime** (Strapi-style), not coded.

- **Frontend:** Next.js 16 multi-zones (one app per main menu) + shared UI package
- **Backend:** Go (Gin) microservices, DDD + TDD
- **Storage:** MongoDB (data, schemas, audit trail) + Elasticsearch (list & search)

## Quick start

```bash
cp .env.example .env              # set MONGO_URI, ES_URL, JWT_SECRET
make install
ADMIN_PASSWORD=secret123 make dev # http://localhost:3000 — login: admin / secret123
```

Set `JWT_SECRET` in `.env`. Without it, sessions are invalidated on every restart.

| Variable | Purpose |
|---|---|
| `MONGO_URI`, `MONGO_DB` | MongoDB connection |
| `ES_URL`, `ES_REPLICAS` | Elasticsearch (`ES_REPLICAS=0` on a single node) |
| `JWT_SECRET` | Signs login tokens |
| `ADMIN_PASSWORD` | Creates the `admin` superadmin on first run |
| `MEDIA_DIR` | Uploaded files (default `./data/media`) |

## Layout

```
apps/shell            :3000  browser entry: login, auth gate, routes /<zone>/* and /api/<zone>/*
apps/<zone>           :300x  one Next.js app per main menu (basePath /<zone>)
packages/ui                  shared components, design tokens, menu + RBAC keys
services/common              JWT, RBAC, Mongo, error mapping
services/<zone>       :808x  Go API under /api/<zone>
```

| Zone | Web | API | Contents |
|---|---|---|---|
| administration | 3007 | 8087 | users, roles & permissions |
| dynamic | 3005 | 8088 | Content-Type Builder + every dynamic menu |
| renkon / sigv2 / work-order | 3001 / 3003 / 3004 | 8081 / 8083 / 8084 | placeholders |
| assets, master | – | – | menu groups filled by dynamic content types |

Each Go service: `internal/domain` (rules) → `internal/app` (use cases) → `internal/infra` (mongo, elastic, disk) → `internal/http` (handlers, DTOs). `main.go` wires them together.

## Dynamic menus (superadmin)

**Dynamic › Content-Type Builder** creates a menu. It has four tabs:

- **General**: name, which menu it appears in, display label (`{code} - {name}`), restrict records by site.
- **Fields**:
  - Types: text, number, date, select, domain (from Master › Domain), relation, media, component (repeatable), classification attributes.
  - Rules: required, unique, default value.
  - **Option filter (where)** for cascading dropdowns, e.g. location where `site = {site}`.
  - **Auto fill**: from a relation, the user profile, today's date, or the logged-in user. Auto codes use patterns such as `{classification.classificationCode}-{YYYY}-{seq:4}`.
- **Layout**: sections → rows (1–4 columns), drag & drop. Each section is a tab on the entry form.
- **Workflow**:
  - Statuses, with an initial status.
  - Transitions (buttons). Each transition can require its own inputs, e.g. an attachment + attachment type.
  - Per-status views, e.g. an "Approval" menu.

Every record keeps metadata, version, soft delete and a readable **change history**. Asset classification attributes come from **Master › Classification**. Site access is inherited down the site hierarchy (`parent`).

Built-in menus (Master, Assets, User Profile) are seeded from `services/dynamic/internal/seed/*.json`. Seeding only creates what is missing, so existing schemas are never overwritten.

## Access control

- **Administration › Role** sets permissions per menu × action: read, create, update, delete, download.
- Permission keys: `<zone>/<path>`, `ct/<uid>`, `ct/<uid>/v/<view>`, and `ct/<uid>/t/<transition>` (update = may run the transition).
- `superadmin` bypasses all permission checks and is the only role that can build menus.
- Records are scoped to the sites in the user's profile: **Users › Profile & placement**.

## Tests

```bash
cd services && set -a && . ../.env && set +a && go test ./common/... ./administration/... ./dynamic/...   # Mongo tests skip without MONGO_URI
node --test apps/dynamic/app/_lib/*.test.ts packages/ui/src/*.test.ts
(cd apps/dynamic && npx tsc --noEmit)
```

## New main menu

```bash
scripts/new-zone.sh inventory 3008 8089   # Next zone + Go service skeleton
```

Then add the zone to `packages/ui/src/menu.ts`. Routing and the sidebar pick it up automatically.
