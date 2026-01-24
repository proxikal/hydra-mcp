# Line Limit Update: 200 → 250

**Date:** 2026-01-24
**Reason:** More realistic limit while maintaining discipline

## Updated Policy

- **Standard limit:** 250 lines per file
- **Rare exception:** Up to 300 lines for highly complex single-concern modules
  - Requires explicit justification
  - Examples: Core state machines, complex protocol handlers
- **Absolute max:** 300 lines (no exceptions beyond this)
- **Generated code:** Exempt from limits

## Previous Policy
- Hard limit: 200 lines
- No exceptions

## Files Updated

### Core Documentation
- `PRD.md` - Updated rule and implementation notes
- `README.md` - Updated key standards
- `CONTRIBUTING.md` - Updated file size limit section (2 locations)

### Hydra Architect Skill (Claude)
- `~/.claude/skills/hydra-architect/skill.md`
  - Updated goal, core mandates, all gates (0, 1, 5, 6, 7, 8)
  - Updated quick reference
- `~/.claude/skills/hydra-architect/patterns/splitting.md`
  - Updated thresholds, examples, decision tree
- `~/.claude/skills/hydra-architect/patterns/modularity.md`
  - Updated file limits

### Hydra Architect Skill (Gemini)
- `.gemini/skills/hydra-architect/skill.md`
- `.gemini/skills/hydra-architect/patterns/modularity.md`

## Current Status

After update, violations against 250-line limit:
- `internal/supervisor/process.go`: 352 lines (+102) - Core supervisor logic
- `internal/cli/cli_test.go`: 988 lines (+738) - CLI test suite

Both exceed the 300-line absolute max and require splitting.

Files within 250-300 range (rare exceptions): **0**

## Next Steps

Optional cleanup (technical debt, not blocking):
1. Split `supervisor/process.go` by lifecycle stage (start/stop/restart)
2. Split `cli/cli_test.go` into separate test files by command
