# Vision

## The Problem

Existing household task management (AnyList, Todoist, Apple Reminders, Google Tasks) treats tasks as flat lists owned by one person. They don't understand that a household has multiple types of workers with different capabilities:

- **You** — decisions, physical presence, personal tasks
- **Family** — errands, coordination, shared responsibilities
- **Virtual assistants** — phone calls, bookings, calendar management
- **Housekeepers** — cleaning, supplies, maintenance
- **AI agents** — research, drafting, code, monitoring, automation
- **Ad-hoc services** — TaskRabbit, contractors, delivery

No tool today triages tasks to the right handler. You end up being the bottleneck — everything routes through you because the tools assume you're the only worker.

## The Solution

Dochore is a task triage system. Tasks come in, get tagged by handler type, and route to whoever can do them. The person at the center sets the rules and makes decisions. Everything else is delegated.

### What Makes It Different

1. **Handler-aware**: Tasks have a `tag` that maps to a handler type, not just a person. "[AI] research MacBook Air M5 pricing" goes to an agent. "[FAMILY] get Toni to the groomer" goes to a family member. "[VA] call PCC about registration" goes to the VA.

2. **Agent-native**: AI agents are first-class handlers. They can create tasks, claim tasks, update status, and broadcast completions. The MCP server makes dochore accessible from any Claude session.

3. **Inbox triage**: Untagged tasks land in an inbox. Auto-triage rules match patterns to tags. What can't be auto-triaged gets surfaced for manual triage.

4. **Multi-interface**: Admin web UI for the human. MCP server for agents. API for everything else. Future: mobile app, voice input, email-to-task.

5. **Source-agnostic**: Tasks can come from manual entry, agent broadcasts, calendar events, email, voice, screenshots. The source is tracked but the task is the same object regardless.

## Product Trajectory

### Phase 1: Personal Tool
- Admin UI on hyperion
- Go API + MongoDB
- MCP server for agent access
- Manual and agent task creation
- Basic triage rules

### Phase 2: Household Tool
- Family member accounts with their own views
- VA/housekeeper interface (simplified, mobile-friendly)
- Notification system (email/SMS when tasks are assigned)
- Recurring tasks (weekly cleaning checklist, monthly bills)
- Shopping list integration

### Phase 3: Product
- Public signup on dochore.com
- Multi-household support
- Onboarding flow for handler types
- Mobile apps (iOS/Android)
- Integrations (Google Calendar, Apple Reminders import, Alexa/Google Home voice input)
- Marketplace for VA/housekeeper services

## Why Now

- AI agents are real and can do real work — but they need a task interface
- The "loneliness epidemic" means households are doing more alone with fewer support systems
- Gig economy (TaskRabbit, Care.com) created a workforce but no unified interface
- Existing tools optimize for individual productivity, not household orchestration
- AnyList, Todoist, etc. are too limited — they don't understand delegation
