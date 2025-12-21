<br/>
<p align="center">
  <a href="https://github.com/qolzam/telar">
    <img src="https://raw.githubusercontent.com/red-gold/red-gold-web/master/website/static/img/logos/telar-social-logo/profile.png" alt="Telar Platform Logo" width="200">
  </a>

  <h3 align="center">The Telar Platform</h3>

  <p align="center">
    The open-source platform that enables developers to launch AI-powered, production-ready social features in minutes, not months.
    <br />
    <br />
    <a href="https://github.com/qolzam/telar/issues">Report Bug</a>
    ·
    <a href="https://github.com/qolzam/telar/issues">Request Feature</a>
  </p>
</p>

<p align="center">
    <a href="https://github.com/qolzam/telar/actions/workflows/ci.yml"><img src="https://github.com/qolzam/telar/actions/workflows/ci.yml/badge.svg" alt="CI Status"></a>
    <a href="https://github.com/qolzam/telar/blob/main/LICENSE"><img src="https://img.shields.io/github/license/qolzam/telar" alt="License"></a>
    <a href="https://discord.gg/27Uekrq9gx"><img src="https://img.shields.io/discord/1401496628933955664?logo=discord&label=community" alt="Discord"></a>
    <a href="https://github.com/qolzam/telar/stargazers"><img src="https://img.shields.io/github/stars/qolzam/telar?style=social" alt="GitHub stars"></a>
</p>

---

## 🚀 What is Telar?

Building community features from scratch is incredibly complex. Telar is a complete, **production-ready platform** that provides all the core functionality you need, allowing you to bypass months of development and focus on what makes your community unique.

It's a single, unified monorepo built on a modern, high-performance stack, designed for developers who value speed, flexibility, and the trust of open-source.

| Feature                      | The Telar Platform                                                         | The Old Way                                                                 |
| ---------------------------- | -------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| **Unified Platform**         | ✅ A complete, integrated solution in one repository.                        | ❌ Gluing together dozens of libraries for auth, profiles, posts, etc.      |
| **AI-Powered**               | ✅ Core features for moderation, engagement, and search built-in.            | ❌ Writing complex AI integrations from scratch.                            |
| **High-Performance Backend** | ✅ Modern, scalable Go backend using a professional "Vertical Slice" architecture. | ❌ Older, monolithic frameworks (Rails, PHP) or slow serverless functions.  |
| **Open Source (MIT)**        | ✅ 100% transparent, flexible, and free. No vendor lock-in.                 | ❌ Opaque, restrictive, and expensive closed-source SaaS platforms.         |
| **Flexible Architecture**    | ✅ Deploy as a single "modular monolith" or as true microservices on Kubernetes. | ❌ Locked into a single deployment model that can't scale with you. |

<br/>

## 🤖 The AI-Powered Advantage

Telar is designed from the ground up to leverage AI, solving the most painful problems of community management.

*   **💡 Community Ignition Toolkit:** Solves the "empty room" problem with AI-powered conversation starters and automated weekly summaries.

*   **🛡️ AI Co-Moderator:** Solves the "toxicity & trolls" problem with a 24/7 AI assistant that uses sentiment and intent analysis to proactively flag harmful content.

*   **⚡ Content Supercharger:** Solves the "buried knowledge" problem with a RAG-based AI search that understands natural language, allowing users to ask questions and get synthesized answers with sources.

Underpinning these features is a **standalone, provider-agnostic AI engine**, designed to be flexible, cost-effective, and powerful. It can run on anything from local, open-source models to high-performance inference APIs.

<br/>

## 🏁 Getting Started in 5 Minutes

Our platform is being architected to run entirely on Docker for a seamless developer experience. The full `docker-compose` setup is a primary goal of the current refactor.


**Prerequisites:** Docker & Docker Compose

```bash
# NOTE: The full docker-compose setup is in progress. This is the target command.

# 1. Clone the repository
git clone https://github.com/qolzam/telar.git

# 2. Navigate into the directory
cd telar

# 3. Start the entire platform
docker-compose up
```

Your Telar instance is now running!
*   **API:** `http://localhost:9099`
*   **Web App:** `http://localhost:3000`

For more detailed setup and configuration, please see our [**Full Documentation**](./docs/README.md).

<br/>

## 🛠️ Tech Stack

Telar is built on a modern, robust, and scalable set of technologies chosen for performance and developer experience.

*   **Backend:** Go (Golang)
*   **Frontend:** Next.js (React)
*   **Database:** PostgreSQL (with JSONB)
*   **Vector Database:** Weaviate
*   **Deployment:** Docker, Kubernetes
*   **AI Engine:** A provider-agnostic service supporting:
    *   **Ollama:** For free, private, self-hosted open-source models.
    *   **Groq:** For high-performance, low-latency cloud inference.
    *   **OpenAI:** For compatibility with the standard commercial API.

<br/>

## 🚧 Project Status & Roadmap

Telar is under active development. We are building in public and are currently focused on completing the core platform migration and shipping the v1 AI Engine.

### Status Legend
*   `📋 Planned`: Not started
*   `🏗️ In Progress`: Foundational work/refactoring
*   `🚀 In Development`: Actively building new features
*   `✅ Stable`: Ready for production use

### Current Status

| Component                 | Directory           | Status                                        |
| ------------------------- | ------------------- | --------------------------------------------- |
| **Headless API**          | `/apps/api`         | 🏗️ In Progress                                |
| **Unified Web Client**    | `/apps/web`         |  🏗️ In Progress                             |
| **Standalone AI Engine**  | `/apps/ai-engine`   |  🚀 In Development (See PR [#1](https://github.com/Qolzam/telar/pull/1))    |
| **TypeScript SDK**        | `/packages/sdk`     | 📋 Planned                                    |

### The Roadmap

*   **Q3 2025: The Foundation**
    *   [x] Complete repository analysis & consolidation plan.
    *   [ ] Complete migration of all backend services.
    *   [ ] Build the core functionality of the Unified Next.js frontend.

*   **Q4 2025: Public Launch**
    *   [ ] Ship v1.0 of the **Standalone AI Engine**, featuring the provider-agnostic RAG pipeline.
    *   [ ] Release the first version of the TypeScript SDK.
    *   [ ] Launch our managed hosting solution.

*   **2026: Scale & Grow**
    *   [ ] Introduce advanced AI features (AI Co-Moderator).
    *   [ ] Launch the Marketplace for themes and plugins.

<br/>

## 🤝 Contributing

We are building Telar in the open, and we welcome contributions of all kinds!

1.  **Join our Community:** The best place to start is our [**Discord Server**](https://discord.gg/27Uekrq9gx).
2.  **Find an Issue:** Check out our [**Open Issues**](https://github.com/qolzam/telar/issues), especially those marked `good first issue`.
3.  **Read our Guidelines:** Please see our [**Contributing Guide**](./CONTRIBUTING.md).

<br/>

## 📜 License

Telar is open-source software licensed under the [MIT License](./LICENSE).
