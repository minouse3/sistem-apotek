# Agent Instructions

Follow these rules on every task in this repository.

## Mandatory project specifications

Read and follow every normative Markdown specification in the root directory except `README.MD`:

- [PRD.MD](PRD.MD): Product requirements, RBAC matrix, service-layer conditional rules, tech stack, and project structure.
- [DATABASE.MD](DATABASE.MD): Authoritative 19-table PostgreSQL 17 schema, `ON DELETE RESTRICT` foreign keys, base-unit storage rules, `NUMERIC(14,4)` costs, indexes, and ERD.
- [DESIGN.MD](DESIGN.MD): Stillwell visual direction, Bootstrap 5.3 foundation, desktop (`216px` sticky white sidebar) and narrow (`<992px` bar and offcanvas) shells, colors, Roboto (`400..700`) typography scale, Phosphor Bold (`15px` to `17px`) icons, and component finish rules.
- [WEBSITE-GUIDELINES.md](WEBSITE-GUIDELINES.md): Interface copy rules, inline field error placement, demo-data labeling, working-controls-only UI, and code/comment hygiene.

For Go coding style, `http.ServeMux` route registration, and `html/template` composition patterns, refer to `references/go-class/` as a structural reference while following the rules in `PRD.MD` and `DESIGN.MD`.

## Strict restrictions

1. **Never access `.env`.** Do not read, open, search, print, edit, or delete `.env` unless the user explicitly tells you to in the current request. Use `.env.example` when checking expected environment variable names.
2. **Never make unrequested changes.** Stay strictly within the user's request. If a request is unclear, lacks detail, conflicts with `PRD.MD`, `DESIGN.MD`, `WEBSITE-GUIDELINES.md`, or `DATABASE.MD`, or risks harming or breaking the project, stop, propose a solution first, and wait for the user's agreement before making changes.
3. **Propagate configuration and structural changes.** Whenever the user requests changes to critical configurations or project structures (such as database schema, directory layout, environment keys, or Docker config), update every file referencing the old setup and update the affected `.MD` specification files (`PRD.MD`, `DATABASE.MD`, `DESIGN.MD`, `WEBSITE-GUIDELINES.md`) as needed.
4. **Never invent or bypass business constraints.** Preserve all RBAC checks, financial field omissions for `kasir` and `staf`, 30-day FEFO expiry blocking, blind opname hiding of `stok_sistem`, and `ON DELETE RESTRICT` schema integrity.

## Response style

Keep responses straightforward, clear, and directly aligned with the user's instructions. Do not add unnecessary commentary, extra explanations, or unsolicited recaps of product rules or website specifications; focus only on what the user asked for.

## Mandatory skill usage

Always load and apply **Antislop** skills along with any domain-specific skills in `.agents/skills/` that match the task before making edits:

- **Every task (mandatory):** Read and apply the relevant Antislop skills (`antislop` plus the domain-specific `antislop-*` skill below).
- **UI, layout, CSS, HTML, or frontend preview changes:** Load `antislop`, `antislop-ui`, `antislop-human`, and `antislop-layoutmobile` (plus `redesign-existing-projects` or `minimalist-ui` when relevant), and follow `DESIGN.MD` for visual direction.
- **Documentation, copywriting, or interface text:** Load `antislop-copywriting` and `humanizer`, and apply `WEBSITE-GUIDELINES.md`.
- **Code edits, refactoring, or comment cleanup:** Load `antislop-code`, `code-simplification`, and `source-driven-development` when verifying Go, PostgreSQL, or Bootstrap APIs.
- **Debugging, build errors, or test work:** Load `debugging-and-error-recovery` and `test-driven-development`.
