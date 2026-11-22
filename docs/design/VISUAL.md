# GUI visual design notes

Figma file: https://www.figma.com/design/lJap2QK7KXc2zzOzgE3ydu/Auto-Deployer  
Page: **AutoDeployer** (do not edit **Page 1**).  
Flows: `docs/design/GUI-FLOWS.md`.

## Frames (match D3 inventory)

| Frame | Notes |
|-------|--------|
| Design notes | Tokens and a11y notes on-canvas |
| Jobs | List + README + empty state |
| Plan confirmation | Masked secrets, Confirm/Cancel |
| Prod confirmation | Red chrome; type `prod` |
| Running | Kill button |
| Live log | Dark pane; Japanese sample `日本語テスト_utf8` |
| History and log | succeeded / skipped / failed + JP log |
| Env editor | Masked + Reveal; exit 4 callout |
| Schedules | Table + reconcile drift |
| Deployments | Promote result; exit 8 error |

## Tokens

| Token | Value |
|-------|--------|
| Shell / top | `#0F172A` |
| Nav | `#1E293B` |
| Content bg | `#F1F5F9` |
| Surface | `#FFFFFF` |
| Text | `#0F172A` / muted `#475569` |
| Dev chip | `#0F766E` |
| UAT chip | `#B45309` |
| Prod chip | `#B91C1C` |
| Type UI | Inter |
| Type log | IBM Plex Mono (fallback Inter) |
| Radius | chip 4 / control 6 / card 8 |
| Nav width | 168 |
| Screen | 1280×800 |

## Accessibility

- Env is **label + color** (never color alone).
- Prod uses stronger border/background on the type gate.
- Body text on white targets ≥ 4.5:1 contrast.
- Kill / Confirm / Reveal are clear text buttons (keyboard focusable in Phase 4).

## Screenshots

Browser captures of the Figma page (when available) live under `docs/design/screenshots/`.  
If captures are missing in a PR, open the Figma page above for visual review.
