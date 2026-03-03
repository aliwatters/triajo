# Dochore

> Do your chores. Route tasks to humans, family, VAs, and AI agents.

**Domain:** dochore.com (registered)

## What It Is

Dochore is a task management system built for a household that operates with multiple types of workers: the person themselves, family members, virtual assistants, housekeepers, AI agents, and ad-hoc services like TaskRabbit.

Tasks come in from everywhere — voice, screenshots, agent broadcasts, calendar events, manual entry. Dochore triages them to the right handler based on tags and rules.

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

- **Public site** (dochore.com): Landing page, nothing functional yet
- **Admin UI** (admin.dochore.test): The real interface, runs on hyperion
- **API** (api.dochore.test): Go API with Gin, serves admin UI and MCP/agent integrations
- **Database**: MongoDB 8.0 on hyperion (existing cluster)

## Key Integrations

- **MCP Server**: Agents can create, query, update, and triage tasks
- **Agent broadcast bus**: Listens to `agents-mcp` for task-relevant events
- **Calendar**: Tasks with deadlines sync to Google Calendar via gsuite-mcp
- **Notifications**: Email/SMS alerts for time-sensitive tasks

## Getting Started

```bash
# Clone
git clone git@github.com:aliwatters/dochore.git
cd dochore

# Dev environment
docker compose up

# Access
# Admin UI: http://admin.dochore.test:3000
# API: http://api.dochore.test:8080
```

## Status

**Phase:** Initialization — repo created, architecture defined, issues filed.
