# Figma

- File: https://www.figma.com/design/lJap2QK7KXc2zzOzgE3ydu/Auto-Deployer
- File key: `lJap2QK7KXc2zzOzgE3ydu`
- Owner: Prakash Niraula (plan "21st tech", admin)

## State at project start

The file currently holds **three podcast-app mobile frames** on `Page 1` ("01 Press play", "02 Go deeper", "03 Keep exploring", app name "sonora"). They are unrelated to AutoDeployer and are not a design to follow.

Rule: do not edit or delete `Page 1`. Create a new page named `AutoDeployer` and put all work there, unless the owner says otherwise on issue #4.

## Connecting Figma in Cursor

1. In Cursor open Settings, then Tools and MCP (named "MCP" in some versions), and add a new MCP server:
   ```json
   {
     "mcpServers": {
       "figma": { "url": "https://mcp.figma.com/mcp" }
     }
   }
   ```
2. Click Connect or Authenticate and finish the Figma sign-in in the browser. Use the same Figma account that can edit the file above.
3. Check access: ask Cursor to call `whoami`, then read the file with the file key above. If it fails, open the Figma file and confirm that account has edit access.
4. Rate limits: Figma limits MCP tool calls by plan and seat. This file is on a Starter plan, which may allow very few calls. Check https://developers.figma.com/docs/figma-mcp-server/rate-limits-access/ before starting. If the limit blocks the work, stop and tell the owner; do not switch tools silently.

## Naming

Frames are named after the screens in `docs/design/GUI-FLOWS.md` (issue #3), for example `Jobs`, `Plan confirmation`, `Prod confirmation`, `Running`, `History and log`, `Env editor`, `Schedules`, `Deployments`.
