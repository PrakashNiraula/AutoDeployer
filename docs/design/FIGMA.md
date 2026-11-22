# Figma

- File: https://www.figma.com/design/lJap2QK7KXc2zzOzgE3ydu/Auto-Deployer
- File key: `lJap2QK7KXc2zzOzgE3ydu`
- Owner: Prakash Niraula (plan "21st tech", admin)
- **AutoDeployer page:** all product UI frames (D4). Do **not** edit or delete `Page 1` (unrelated podcast app).

## Pages

| Page | Purpose |
|------|---------|
| Page 1 | Unrelated podcast app — leave alone |
| AutoDeployer | Design notes + GUI screens for D4 |

## Frames on AutoDeployer

Design notes · Jobs · Plan confirmation · Prod confirmation · Running · Live log · History and log · Env editor · Schedules · Deployments

See `docs/design/VISUAL.md` for tokens and `docs/design/GUI-FLOWS.md` for behavior.

## Connecting Figma in Cursor

1. In Cursor open Settings → Tools and MCP, add:
   ```json
   {
     "mcpServers": {
       "figma": { "url": "https://mcp.figma.com/mcp" }
     }
   }
   ```
2. Connect / authenticate with the Figma account that can edit this file.
3. Check access with `whoami`. Prefer **write** tools (`use_figma`) on Starter; avoid unnecessary MCP reads (20/month limit on Starter).
4. Rate limits: https://developers.figma.com/docs/figma-mcp-server/rate-limits-access/

## Naming

Frames match `docs/design/GUI-FLOWS.md`.
