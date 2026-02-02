---
name: ai-cortex
description: Discovers and loads the AI Cortex capability asset library (Skills / Rules / Commands) from GitHub. Use when the user references ai-cortex, wants to use its skills/rules/commands, asks to load AI Cortex, or requests capabilities like README generation, decontextualization, code review, commit messages, or AGENTS.md writing.
---

# AI Cortex Loader

Teaches the agent how to discover, load, and use [AI Cortex](https://github.com/nesnilnehc/ai-cortex): a governable capability asset library (Skills / Rules / Commands) with Spec and tests.

## Quick start

1. **Read the entry** (authoritative source):
   - Fetch and follow: `https://raw.githubusercontent.com/nesnilnehc/ai-cortex/main/AGENTS.md`
2. **Discover indexes**:
   - Load `skills/INDEX.md`, `rules/INDEX.md`, `commands/INDEX.md` from the same Raw base:
   - Base URL: `https://raw.githubusercontent.com/nesnilnehc/ai-cortex/main/`
3. **Match and inject**:
   - Match task intent to `description` / `tags` in each INDEX.
   - Inject **rules first**, then **skills** (full Markdown of each selected RULE/SKILL as context).
   - Use `related_skills` in a SKILL for chained tasks.

## Asset roots

- **Remote**: `https://raw.githubusercontent.com/nesnilnehc/ai-cortex/main/` (replace branch if needed).
- **Local**: If the current repo is a fork or clone of ai-cortex, use repo root; no Raw URL needed.

## Behavior (from AGENTS.md)

- **Rules before skills**: Load relevant rules from `rules/INDEX.md` before any Skill.
- **Self-check**: After producing content from a Skill, run that Skill’s quality/self-check before submitting.
- **Capability questions**: When the user asks “what skills/rules/commands”, read the corresponding INDEX and **list name and purpose**; do not reply with only URLs.
- **Spec as authority**: When creating or editing Skills/Rules/Commands, follow `spec/skill.md`, `spec/rule.md`, `spec/command.md` (under the same base).

## Index URLs (Raw)

| Index | URL |
|-------|-----|
| Entry | `https://raw.githubusercontent.com/nesnilnehc/ai-cortex/main/AGENTS.md` |
| Skills | `https://raw.githubusercontent.com/nesnilnehc/ai-cortex/main/skills/INDEX.md` |
| Rules | `https://raw.githubusercontent.com/nesnilnehc/ai-cortex/main/rules/INDEX.md` |
| Commands | `https://raw.githubusercontent.com/nesnilnehc/ai-cortex/main/commands/INDEX.md` |
| Machine index | `https://raw.githubusercontent.com/nesnilnehc/ai-cortex/main/llms.txt` |

## Common intents → assets

- **AGENTS.md** write/revise → Skill: `write-agents-entry`
- **README** generate → Skill: `generate-standard-readme` or Command: `/readme`
- **Decontextualize / privacy** → Skill: `decontextualize-text`
- **Code review (diff)** → Skill: `review-code` or Command: `/review-code`
- **Codebase review** → Skill: `review-codebase` or Command: `/review-codebase`
- **Commit message** → Skill: `generate-commit-message` or Command: `/generate-commit-message`
- **Skill design/refine** → Skill: `refine-skill-design`
- **Bootstrap remote skills** → Skill: `bootstrap-skills`
- **Chinese technical writing** → Rule: `writing-chinese-technical`

Resolve exact paths from the INDEX tables; append the path from the table to the base URL (e.g. `.../main/skills/generate-standard-readme/SKILL.md`).

## Project without AGENTS.md

If the current project has no AGENTS.md, the agent may create one that references AI Cortex. If AGENTS.md already exists, append a reference to this library per the entry-writing guidance in AI Cortex.
