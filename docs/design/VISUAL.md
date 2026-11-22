# GUI visual design notes

Figma: https://www.figma.com/design/lJap2QK7KXc2zzOzgE3ydu/Auto-Deployer  
Page: **AutoDeployer** (do not edit **Page 1**).  
Flows: `docs/design/GUI-FLOWS.md`.

## Metaphor

GitHub-like **job as project**: catalog → job workspace with README home, file tree, code viewer, plus Schedules / Runs / Env tabs.

## Frames

### Global
Jobs catalog · Running · Live log · History and log · Deployments · Doctor · Plan confirmation · Prod confirmation

### Job workspace
Job / Overview · Job / Files · Job / Schedules · Job / Runs · Job / Env

(Older flat `Jobs` / `Env editor` / `Schedules` frames may remain on the canvas as archived references; **job workspace frames are canonical**.)

## Layout — Job / Files (primary visualization)

```text
┌ header: job id · env chip · Plan · Run ──────────────┐
│ tabs: Overview | Files | Schedules | Runs | Env        │
├────────────┬───────────────────────────────────────────┤
│ tree       │  viewer                                   │
│ README.md  │  markdown render  OR  monospace code      │
│ job.yaml   │  Japanese text supported                  │
│ scripts/   │                                           │
│ env/       │                                           │
└────────────┴───────────────────────────────────────────┘
```

Tree ~240px; viewer fills remaining width. Selected file highlighted. README on Overview is full-width (no tree), like a GitHub repo home.

## Tokens

| Token | Value |
|-------|--------|
| Shell / top | `#0F172A` |
| Nav / tree pane | `#1E293B` / `#0F172A` |
| Content bg | `#F1F5F9` |
| Surface | `#FFFFFF` |
| Code pane | `#0F172A` text `#E2E8F0` |
| Dev / UAT / Prod chips | `#0F766E` / `#B45309` / `#B91C1C` |
| Type UI | Inter |
| Type code / log | IBM Plex Mono |
| Screen | 1280×800 |
| Tree width | 240 |

## Accessibility

- Env = **label + color**.
- Prod confirmation stronger red chrome + type gate.
- Tree and tabs keyboard-navigable (Phase 4).
- File paths never shown only as icons.

## Screenshots

`docs/design/screenshots/` — HTML mirror + PNG for PR review without Figma MCP reads.
