# ServeEz Design System

## Overview & Creative North Star

ServeEz is an AI-native cluster orchestrator. The product is a serious operations
dashboard for engineers who run infrastructure. The design must feel **engineered,
calm, and precise** — like Grafana, Datadog, or a well-made developer console.
It must never feel like a flashy marketing page or a generic AI-generated mockup.

**The North Star: "Calm Instrumentation."** High information density without chaos.
A restrained neutral palette, quiet grid lines, and one intentional accent used
sparingly. The interface should feel like the inside of a well-built instrument,
not a party.

## Color & Tonal Architecture

### Palette
We use a **neutral charcoal base** with **cool undertones** and a single restrained
**accent** (a desaturated indigo-blue). No neon gradients, no purple glass, no
violet-on-black. Color is reserved for status and meaning, not decoration.

| Token | Hex | Usage |
|-------|-----|-------|
| `surface` | `#0e1116` | App background (canvas) |
| `surface_container` | `#151a21` | Sidebar, panels |
| `surface_container_high` | `#1c232c` | Cards, tables, inputs |
| `surface_container_highest` | `#242c36` | Hover states, popovers |
| `on_surface` | `#e6e9ee` | Primary text |
| `on_surface_variant` | `#9aa4b2` | Secondary text, labels |
| `on_surface_muted` | `#6b7683` | Tertiary text, captions |
| `outline` | `#2b333d` | Hairline borders (1px, 12-18% opacity) |
| `primary` | `#4c8dff` | Accent — interactive, selected, links |
| `primary_container` | `#1b2a44` | Selected row / active pill background |
| `success` | `#3fb950` | Healthy nodes, up states |
| `warning` | `#d29922` | Degraded states |
| `error` | `#f85149` | Down states, destructive |
| `info` | `#58a6ff` | AI suggestions, neutral system info |

### Rules
- **No gradients.** Flat surfaces only. No glassmorphism, no backdrop-blur chrome.
- **Borders are hairline and quiet.** Use `outline` at low opacity; never thick or loud.
- **Status colors are semantic only.** Never use green/red/amber for decoration.
- **Dark mode only.** ServeEz is a monitoring tool; assume operators live in dark.

## Typography

- **UI:** `Inter` (system fallback `-apple-system, "Segoe UI", sans-serif`).
- **Numeric/data:** `"SF Mono", "JetBrains Mono", ui-monospace, monospace` — tabular
  figures for metrics, versions, hashes, and IDs.
- **Scale** (restrained, no giant display type):
  - `13px` — table cells, descriptions
  - `14px` — body, list items
  - `16px` — section titles
  - `20px` — page headers
  - `24px` — key metrics on cards
- Weights: 400 body, 500 emphasis, 600 headers. All-caps only for tiny 11px
  uppercase labels with letter-spacing `0.06em`.

## Layout & Density

- **Fixed left sidebar** (220-240px) with nav + a compact system-status footer.
- **Header bar** (48-52px) with page title, environment tag, and global search.
- **Content grid** with 12-16px gaps. Dense but breathable: `padding: 12-16px`
  inside cards.
- Cards are `surface_container_high` with a 6px radius and a quiet 1px hairline.
- No decorative shadows. Elevation = background tone shift only.

## Components

### Navigation
- Active item: `primary_container` background + `primary` 3px left indicator.
- Inactive: `on_surface_variant`, hover → `on_surface`.
- Nav item height 36px, 12px gap between groups. Group labels are the tiny
  all-caps muted style.

### Metric cards
- Label (tiny all-caps muted) on top, large value below (24px, mono), a small
  delta/sub-line under the value. Sparkline only if it adds signal.

### Tables
- Header row: 11px all-caps muted label. Rows: 13px, `padding: 8px 12px`, hairline
  dividers, hover = `surface_container`. No zebra stripes.
- Status rendered as `<dot> + label` (e.g., `● Healthy`), dot 8px.

### Buttons
- **Primary:** `primary` background, `on_surface`-white text, 6px radius, 32px height.
- **Secondary:** transparent, 1px `outline` hairline, `on_surface` text.
- **Ghost/Icon:** text-only or icon, no border.
- Hover = `+6%` lightness; active = `-6%`. No glow, no gradient.

### Inputs & selects
- `surface_container_highest` fill, 6px radius, 1px hairline, 32px height.
- Focus ring: 2px `primary` at 40% opacity. Placeholder = `on_surface_muted`.

### Status / badges
- Chip: 11px label, 4px radius, tinted container (`success`@12% bg, `success` text).
- Dot before the text for statuses.

### Terminal / log blocks
- `#0a0c10` fill, `ui-monospace` 12px, left gutter with line numbers `on_surface_muted`.
- Use only for real output (commands, YAML, logs) — not decorative boxes.

## Content rules (anti "AI-look")

- Use **real, specific product data**: node names like `node-a1`, IPs `10.0.0.12`,
  images `ghcr.io/acme/api:1.4.2`, real-looking metrics (`cpu 34.2%`, `mem 1.2GB`).
- No lorem ipsum, no emoji, no generic placeholder art, no robots/space/cloud
  illustrations, no "Hello, world" headers.
- Headers are declarative and dry: **"Cluster Overview"**, **"Services"**,
  **"Cost Comparison"**, **"Alerts"**, **"AI Control"**.
- Copy is technical and terse. No exclamation points, no marketing adjectives.

## Screen inventory (ServeEz GUI)

1. **Cluster Overview** — stat cards (nodes, workloads, uptime, cost $/hr) + node
   health grid (dot grid grouped by node) + recent activity feed + mini alerts.
2. **Services** — table of workloads: name, image, replicas, node, CPU, memory,
   state (running/healthy/degraded). Filters + search.
3. **Cost Comparison** — form (vCPU, memory GB, runtime hrs) + result table across
   AWS/Azure/GCP: provider, instance, region, on-demand vs spot, monthly cost,
   savings %; best recommendation highlighted.
4. **Alerts** — list of alerts: severity chip, node, message, timestamp, status;
   empty state with a calm "No active alerts" message.
5. **AI Control** — chat panel: message thread, intent chips (scale/migrate/restart),
   confidence meter, "preview action" cards with confirm buttons.

Keep the layout language identical across all five screens: same sidebar, same
header, same card grammar, same type ramp. Consistency is the brand.