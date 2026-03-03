# Implementation Plan

## Phase 1: Foundation (Issues #1–#7)

Get the core loop working: create task → triage → assign → complete.

### Milestone 1A: Scaffolding
- [ ] #1 — Initialize Go API with Gin, MongoDB connection, health endpoint
- [ ] #2 — Initialize Next.js admin app with Tailwind + Shadcn
- [ ] #3 — Docker Compose for local dev (app, api, nginx)
- [ ] #4 — CI pipeline (lint, test, build)

### Milestone 1B: Core CRUD
- [ ] #5 — Task CRUD API (create, list, get, update, delete)
- [ ] #6 — Admin UI: task list view with filters (status, tag, handler)
- [ ] #7 — Admin UI: task creation form + inbox triage view

### Milestone 1C: Agent Integration
- [ ] #8 — triajo-mcp server (task_create, task_list, task_update, inbox_count)
- [ ] #9 — Auto-triage rules engine (keyword/regex → tag mapping)
- [ ] #10 — agents-mcp integration (listen for broadcasts, create tasks from events)

## Phase 2: Household Features (Issues #11–#16)

Make it useful for the whole household.

- [ ] #11 — Handler management (CRUD for family, VA, housekeeper, agents)
- [ ] #12 — Notification system (email/SMS on task assignment)
- [ ] #13 — Recurring tasks (templates, schedules)
- [ ] #14 — Shopping list mode (quick-add, check-off, grouped by store)
- [ ] #15 — Housekeeper checklist view (mobile-friendly, simplified)
- [ ] #16 — VA dashboard (assigned tasks, contact info, action buttons)

## Phase 3: Integrations (Issues #17–#20)

Connect to the rest of the ecosystem.

- [ ] #17 — Google Calendar sync (tasks with due dates → calendar events)
- [ ] #18 — Import from Apple Reminders / Todoist / AnyList
- [ ] #19 — Voice input (Whisper transcription → task creation)
- [ ] #20 — Email-to-task (forward email → creates task with metadata)

## Phase 4: Public Product (Future)

- [ ] Public site on triajo.com (waitlist, marketing)
- [ ] Multi-household / multi-tenant support
- [ ] Mobile apps
- [ ] Onboarding flow
- [ ] Pricing model

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
| Public site | Cloudflare Pages or Vercel | Static, no server needed |
