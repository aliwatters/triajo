# Implementation Plan

## MVP — What Gets You Off ALI-TODOS.md

The goal is to replace the flat markdown file with something that understands delegation. MVP is: authenticated admin UI where you can create tasks, triage them to handlers, and agents can interact via MCP.

### MVP Issues (in order)

1. **#1** — Go API with Gin + MongoDB + health endpoint
2. **#21** — Authentication: better-auth with Google OAuth
3. **#2** — Next.js admin app with Tailwind + Shadcn
4. **#3** — Docker Compose for local dev
5. **#5** — Task CRUD API
6. **#6** — Admin UI: task list with filters
7. **#7** — Admin UI: task creation + inbox triage
8. **#8** — triajo-mcp server (agents can create/list/update tasks)
9. **#22** — Invite system + multi-household support

MVP is done when: you can create a task in the UI, tag it [AI], and a Claude agent can pick it up via MCP. And your family/VA can log in and see only their tasks.

### Post-MVP Priority (in order)

10. **#4** — CI pipeline
11. **#9** — Auto-triage rules engine
12. **#10** — agents-mcp integration
13. **#11** — Handler management
14. **#25** — Smart notifications (consolidation, quiet hours)

---

## Phase 1: Foundation (Issues #1–#10, #21–#22)

Get the core loop working: create task → triage → assign → complete. With auth from day one.

### Milestone 1A: Scaffolding + Auth
- [ ] #1 — Initialize Go API with Gin, MongoDB connection, health endpoint
- [ ] #21 — Authentication: better-auth with Google OAuth
- [ ] #2 — Initialize Next.js admin app with Tailwind + Shadcn
- [ ] #3 — Docker Compose for local dev (app, api, nginx)
- [ ] #4 — CI pipeline (lint, test, build)

### Milestone 1B: Core CRUD + Multi-Household
- [ ] #5 — Task CRUD API (create, list, get, update, delete)
- [ ] #22 — Invite system + multi-household support
- [ ] #6 — Admin UI: task list view with filters (status, tag, handler)
- [ ] #7 — Admin UI: task creation form + inbox triage view

### Milestone 1C: Agent Integration
- [ ] #8 — triajo-mcp server (task_create, task_list, task_update, inbox_count)
- [ ] #9 — Auto-triage rules engine (keyword/regex → tag mapping)
- [ ] #10 — agents-mcp integration (listen for broadcasts, create tasks from events)

## Phase 2: Household Features (Issues #11–#16, #23–#26)

Make it useful for the whole household. Multi-step workflows, checklists, notifications.

- [ ] #11 — Handler management (CRUD for family, VA, housekeeper, agents)
- [ ] #12 — Notification system (email/SMS on task assignment)
- [ ] #25 — Smart notifications: consolidation, quiet hours, preferences
- [ ] #13 — Recurring tasks (templates, schedules)
- [ ] #24 — Checklist system: reusable task checklists
- [ ] #23 — Task workflows: multi-step pipelines with state transitions
- [ ] #14 — Shopping list mode (quick-add, check-off, grouped by store)
- [ ] #15 — Housekeeper checklist view (mobile-friendly, simplified)
- [ ] #26 — Photo proof of completion
- [ ] #16 — VA dashboard (assigned tasks, contact info, action buttons)

## Phase 3: Integrations (Issues #17–#20)

Connect to the rest of the ecosystem.

- [ ] #17 — Google Calendar sync (tasks with due dates → calendar events)
- [ ] #18 — Import from Apple Reminders / Todoist / AnyList
- [ ] #19 — Voice input (Whisper transcription → task creation)
- [ ] #20 — Email-to-task (forward email → creates task with metadata)

## Phase 4: Public Product (Future)

- [ ] Public site on triajo.com (waitlist, marketing)
- [ ] Onboarding flow for new households
- [ ] Mobile apps (iOS/Android)
- [ ] Pricing model (freemium: free for 1 household, paid for multi + premium features)
- [ ] Marketplace integrations (TaskRabbit, Care.com for handler discovery)

---

## Architecture Insights from LoanForge A-K Workflow

Jenny's Encompass consulting pipeline (A-K) validated several patterns triajo should adopt:

1. **Multi-step workflows work when each step has a clear handler.** Jenny's steps E-K are repetitive copy-paste across JotForm and Teamwork — the same pattern as household tasks that get delegated to a VA or housekeeper.

2. **State transitions need visibility.** The pipeline view (intake → processing → UAT → deployed → warranty → closed) maps directly to triajo's task lifecycle (inbox → pending → in_progress → completed).

3. **Photo/document proof matters.** Jenny attaches BRDs and Solution Details to each step. Housekeepers need to attach photos. VAs need to attach call notes. Same pattern.

4. **Automation at the boring steps.** Steps E-K are automatable because they're formulaic. Triajo's auto-triage rules and recurring task templates serve the same purpose — automate the parts that don't need a human decision.

## Features Mined from Ideas Repo

The Sunday Reset project (fully specced in `~/git/ideas/sunday-reset/`) contributed:

| Feature | Source | Triajo Application |
|---------|--------|-------------------|
| Operations checklists | sunday-reset/operations | Housekeeper daily/weekly routines |
| Meal train coordination | sunday-reset/support-network | Family meal planning, delegation |
| Crisis support intake | sunday-reset/support-network | Urgent task escalation (24hr response) |
| Volunteer coordination | sunday-reset/operations | Handler assignment and onboarding |
| Circle management | sunday-reset/community-circles | Household groups, handler groups |
| Event reminders | sunday-reset/liturgy | Task due date notifications |
| Smart alert consolidation | wishfire/alert-system | Notification batching, quiet hours |
| Agent marketplace | agent-silk-road | Future: TaskRabbit/Care.com integration |

---

## Development Workflow

Follow travelblog conventions:
- Feature branches off `main`
- PR-based workflow with CI checks
- Conventional commits
- Docker Compose for local dev
- Deploy to hyperion via `docker compose up -d`

## Infrastructure

| Component | Where | Notes |
|-----------|-------|-------|
| MongoDB | hyperion (192.168.4.106:27017) | Existing instance, new `triajo` database |
| Docker | hyperion | Docker Compose, no k8s |
| DNS | /etc/hosts or Tailscale | admin.triajo.test, api.triajo.test |
| Remote access | Tailscale | Access from anywhere on tailnet |
| Public site | Cloudflare Pages | Static landing page, triajo.com |
| Photos/uploads | NAS or Cloudflare R2 | S3-compatible, same as travelblog |
