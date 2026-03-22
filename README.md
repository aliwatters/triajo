# Ginla

> Get it done. Route tasks to humans, family, VAs, and AI agents.

**Domains:** ginla.com (primary), dochore.com (alternative branding)

## What It Is

Ginla is a task management system built for a household that operates with multiple types of workers: the person themselves, family members, virtual assistants, housekeepers, AI agents, and ad-hoc services like TaskRabbit.

Tasks come in from everywhere — voice, screenshots, agent broadcasts, calendar events, manual entry. Ginla triages them to the right handler based on tags and rules.

## Tags

| Tag | Who | Example |
|-----|-----|---------|
| `[ME]` | Only I can do this | Physical presence, decisions, personal |
| `[AI]` | Claude/Gemini/agents | Code work, research, drafting, monitoring |
| `[VA]` | Virtual assistant | Phone calls, bookings, calendar management |
| `[FAMILY]` | Family member (named) | Dog grooming, errands, coordination |
| `[HOUSEKEEPER]` | Housekeeper | Cleaning, supplies, household maintenance |
| `[DELEGATE]` | Anyone capable | Needs assignment |

## Architecture

Next.js + Go API + MongoDB. See [ARCHITECTURE.md](ARCHITECTURE.md).

- **Public site** (ginla.com): Landing page, nothing functional yet
- **Admin UI** (admin.ginla.test): The real interface, runs on hyperion
- **API** (api.ginla.test): Go API with Gin, serves admin UI and MCP/agent integrations
- **Database**: MongoDB 8.0 on hyperion (existing cluster)

## Key Integrations

- **MCP Server**: Agents can create, query, update, and triage tasks
- **Agent broadcast bus**: Listens to `agents-mcp` for task-relevant events
- **Calendar**: Tasks with deadlines sync to Google Calendar via gsuite-mcp
- **Notifications**: Email/SMS alerts for time-sensitive tasks

## Getting Started

```bash
# Clone
git clone git@github.com:aliwatters/ginla.git
cd ginla

# Dev environment
docker compose up

# Access
# Admin UI: http://admin.ginla.test:3000
# API: http://api.ginla.test:8080
```

## Status

**Phase:** Initialization — repo created, architecture defined, issues filed.
