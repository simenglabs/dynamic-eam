# Design capture — CMMS Distribusi (staging)

Source: `staging.cmms-distribusi.aegis.co.id`, captured 2026-10-19 at 1440x900.
Tokens: [`tokens.css`](tokens.css).

Stack: Next.js + Tailwind CSS v4 + shadcn/ui, icons `lucide-react`, font Inter.
Root font-size **14px** (1rem = 14px), so `text-sm` = 12.25px, `text-xl` = 17.5px.

## Screenshots

| File | Content |
|---|---|
| `00-login.png` | Login card, diagonal-stripe background + teal shapes |
| `01-sidebar-flyout.png` | Icon rail + submenu flyout (Assets) |
| `02-list-table.png` | List page: card, toolbar, data table, badges, pagination |
| `03-form-create.png` | Create form: sections, required/optional labels, inputs, select, lookup, switch, date, right section-nav |
| `04-detail.png` | Detail page |

## Layout

- **Icon rail**: width 105px, bg `--primary-950` `#053947`, logo top, avatar (initials, teal circle) bottom.
  Rail button 73x56, radius 10px, icon `--primary`; hover `primary/20`; active bg `--primary` + white icon.
- **Flyout submenu**: fixed, left 105px, width 243px, bg slate-200 `#e2e8f0`, padding-top 60px.
  Title bold, item h-40px, radius 8px, 12.25px medium, color `#314158`; hover `primary/20`; active `--primary` + white. Overlay dims content.
  Grouped items (e.g. Renkon > PRK) are collapsible.
- **Page header**: breadcrumb (muted, `/` separator) + bold title, white bg, bottom border.
- **Action bar** (form/detail): `← Back` left, `More Actions ▾` (ghost primary) + `Save` (outline, disabled grey) right.
- **Content**: card `rounded-xl border p-6 shadow-sm`, white, radius 12.75px.
- **Section nav** (form/detail): right-side sticky card with 4px teal left border, active item filled teal.

## Components

| Component | Style |
|---|---|
| Button primary | bg `#00a2b9`, white, radius 6.75px, h 36.75px, px 10.5px, 12.25px/500 |
| Button outline | white bg, 1px `#00a2b9` border + text, same size, lucide icon left |
| Button ghost | text `#00a2b9`, no border (More Actions, Back) |
| Input | h 35px, 1px `#e5e5e5`, radius 6.75px, 12.25px; read-only = bottom border only, no box; disabled = grey fill |
| Select | shadcn Select, chevron right; focused border teal |
| Lookup | input + `↗` (maximize-2) icon opens dialog picker |
| Label | 14px bold; required `*` red; `(Optional)` grey normal |
| Divider label | "Asset Details" teal text + teal line |
| Switch | shadcn Switch |
| Collapsible section card | title 17.5px bold, action buttons + chevron right |
| Table | header bg slate-200, bold, sort icon (chevrons-up-down), p 8.75px; rows 38.5px, zebra slate-50, hover `#e5f5f8`; first col = teal underlined link; cell truncate |
| Badge | radius 4.75px, 1px border = text color, 10.5px/500, bg = color at 10–20%. Operating = green-700, Not Ready = slate/zinc |
| Pagination | "Showing x - y of n. Rows per page [10▾]" left; Prev / pages / … / Next right; active page solid teal, others outline teal |
| Search | input with search icon, w ~190px |
| Toast | shadcn toaster region "Notifications (F8)" |

## Menu map (micro frontend zones)

Icon rail order. Each big menu = 1 zone.

| Icon (lucide) | Zone / base path | Submenu |
|---|---|---|
| `house` | `/dashboard` | — |
| `columns-3-cog` | Renkon `/renkon` | PRK (PRK Unit Induk, PRK Unit, Approval PRK), SKAI (SKAI, Approval SKAI), SKKI, KHS (KHS, Approval KHS, Template KHS, Unit KHS), Daftar Material, Daftar Permohonan, Perencanaan, Approval Perencanaan |
| `bolt` | Assets `/assets` | Asset Templates, Asset (T&D), Locations, Asset Approval (Modify), Location Approval (Modify), Run Report Assets |
| `map` | SIG `/sigv2` | Peta Aset CMMS, Aset Belum Bertempat |
| `clipboard-list` | Work Order `/work-order` | Work Order Tracking (T&D), Run Report Work Order |
| `rectangle-ellipsis` | Dynamic `/dynamic` | Component Config, Field Groups |
| `cog` | Master `/master` | Organizations, Map Symbols, Service Address |
| `user-round` | Administration `/administration` | Users Management, Role |

## Issues seen

- `/dashboard` renders full session JSON incl. **access token** on screen. Remove.
- Rail icon buttons have no accessible name (no `aria-label`/tooltip).
- Status badge text clipped in table (last column overflow).
- Typo "Spesifications" on Asset form section title.
