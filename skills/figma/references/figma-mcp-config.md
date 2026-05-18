# Figma MCP config reference

This reference is for the current official Figma remote MCP flow with Codex.

## Recommended Codex setup

Figma's current official docs recommend the remote MCP server for Codex:

```toml
[mcp_servers.figma]
url = "https://mcp.figma.com/mcp"
```

- For Codex, authenticate through the built-in OAuth flow after adding the server.
- The current official Codex setup does not require a manually managed `FIGMA_OAUTH_TOKEN`, region headers, or RMCP feature flags.
- Verify the connection with `codex mcp list`. A healthy setup should show the Figma server as `enabled` with `Auth = OAuth`.

## Official Codex install paths

- Codex app (preferred):
  1. Open `Settings -> MCP servers`
  2. Install and authenticate the Figma server
  3. Install Figma's companion skills from the Skills UI

- Codex CLI (manual):

```sh
codex mcp add figma --url https://mcp.figma.com/mcp
```

Then complete the OAuth prompt.

## Write-to-canvas prerequisites

For `use_figma` workflows, the official docs currently require:

- Remote Figma MCP server
- A `Full` Figma seat
- Edit permission to the target file
- The file URL or selection URL in the prompt/context
- Figma's write-focused skills for Codex, especially `Figma Use`

If reads work (`whoami`, `get_design_context`, `get_screenshot`) but writes do not visibly land in Figma, check whether the write skill set is installed in addition to the base server connection.

## Verification checklist

- `codex mcp list` shows `figma` as `enabled` and `OAuth`
- `whoami` returns the expected Figma account
- `whoami` shows a `Full` seat for the plan that owns the file
- The target file is editable by that account
- The Codex environment has the write-focused Figma skills installed

## Usage reminders

- The remote server is link-based for file context. Give Codex the file URL or node URL when you want it to target a specific Figma file or selection.
- For read workflows, follow the normal design-context flow: `get_design_context -> get_metadata` if needed -> `get_screenshot`.
- For write workflows, prefer smaller incremental edits and validate the resulting file after each `use_figma` operation.
