# AGENTS.md

This repository is governed by `PURIA.md`.
Agents MUST read `PURIA.md` before any action.
`PURIA.md` is the single source of truth for agent behavior, engineering style, workflow, git rules, commits, testing, releases, and project-specific doctrine.
Before doing anything, agents MUST read `PURIA.md`.
If `PURIA.md` is missing, unreadable, or unclear:

→ STOP  
→ DO NOT modify files  
→ report the problem

There is no fallback behavior.
`PURIA.md` is the only source of truth.

---

## Execution

- Do NOT infer conventions
- Do NOT adopt undocumented patterns
- Use ONLY rules defined in `PURIA.md`

If something looks like a convention but is not defined:

→ append it to `HITL.md`  
→ do NOT use it

---

## OpenAPI Spec Integrity

The OpenAPI spec at `internal/api/openapi.json` MUST remain valid JSON at all times.

After ANY edit to the file:

→ run `python3 -c "import json; json.load(open('internal/api/openapi.json'))"`  
→ if validation fails, the edit MUST be corrected before the task is complete

Never append or prepend to the file using shell tools (`cat >>`, `sed`, etc.). Always use the Write tool for the full file, or the Edit tool with exact string matching. The file structure is:

```json
{
  "openapi": "3.0.3",
  "info": { ... },
  "tags": [ ... ],
  "paths": {
    "/endpoint": { ... }
  },
  "components": {
    "schemas": { ... }
  }
}
```

A common failure mode: path entries leaking outside the `"paths"` object. When adding a new endpoint, ensure it goes inside `"paths": { ... }` before the closing `},` that precedes `"components"`.
