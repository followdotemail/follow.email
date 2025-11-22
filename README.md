<div align="center">

<h1>Follow Email</h1>

Modern email follow-up automation for busy teams, built with a Next.js frontend and Go backend in a single Turborepo workspace.

<p>
  <a href="./Setup.md"><strong>Setup Guide →</strong></a>
  ·
  <a href="apps/hermes/documents/api-documentation.md"><strong>API Docs</strong></a>
  ·
  <a href="apps/hermes/documents/system-design.md"><strong>System Design</strong></a>
</p>

</div>

## ✨ Highlights

- Automated Gmail ingestion and follow-up orchestration backed by Clerk authentication.
- Opinionated monorepo with shared tooling, linting, and build pipelines.
- Production-ready Docker workflow: nginx reverse proxy + backend container, provisioned via Excloud automation.
- Neon-hosted Postgres database and optional integrations with AWS S3, Google Gemini, and Upstash QStash.

## 🧱 Monorepo at a Glance

```
follow.email/
├── apps/frontend/         # Next.js app (App Router, Tailwind, Clerk)
├── apps/hermes/           # Go API (Gin, GORM, Gmail sync)
├── infra/                 # Docker Compose, nginx, deployment assets
├── apps/hermes/deployments/
│   └── provision-dev-instance.py  # Excloud provisioning CLI
├── env.example            # Environment template (Neon, Clerk, Google, etc.)
└── Setup.md               # Detailed setup, provisioning, troubleshooting
```

## 🛠 Tech Stack

| Layer        | Technologies |
| ------------ | ------------ |
| Frontend     | Next.js 15, React 19, TypeScript, Tailwind CSS, Radix UI, Jotai |
| Backend      | Go 1.23, Gin, GORM, Clerk SDK, Gmail API, AWS SDK, Google Gemini |
| Data & Infra | Neon Postgres, Docker Compose, nginx, Turborepo, Excloud provisioning |

Optional integrations: AWS S3 for storage, Upstash QStash for scheduling, Google Gemini for AI-assisted replies.

## 🚀 Getting Started

1. Install prerequisites (Node.js ≥ 18, npm ≥ 9, Go ≥ 1.23, Docker Desktop, Python ≥ 3.9).
2. Copy `env.example` to `.env` and populate with Neon, Clerk, Google, and integration secrets.
3. Follow the environment-specific steps in [`Setup.md`](./Setup.md) for local dev or cloud provisioning.

For day-to-day development you can run the apps directly (`npm run dev`), while `provision-dev-instance.py` automates provisioning and deployment to Excloud.

## 📚 Documentation & Links

- [`Setup.md`](./Setup.md): comprehensive setup, environment configuration, and troubleshooting.
- [`apps/hermes/documents/api-documentation.md`](apps/hermes/documents/api-documentation.md): backend endpoints and models.
- [`apps/hermes/documents/auth-flow.md`](apps/hermes/documents/auth-flow.md): Clerk + Gmail authentication sequence.
- [`apps/hermes/documents/system-design.md`](apps/hermes/documents/system-design.md): architecture overview.

## 🤝 Contributing & Support

Issues and pull requests are welcome. Please open a discussion for major ideas and refer to the setup guide before filing environment-related issues.

---

Built by the Follow Email team • Last updated: November 2025
