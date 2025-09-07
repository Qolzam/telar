# Contributing to Telar Social

First off, thank you for considering contributing to Telar Social! ❤️ We are building the future of open and intelligent online communities, and we welcome contributions of all kinds. Whether you're reporting a bug, suggesting a feature, improving documentation, or writing code, your help is greatly appreciated.

This document provides a guide for contributing to the project. Please feel free to ask questions by opening an issue or joining our community on Discord.

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
  - [🐛 Reporting Bugs](#-reporting-bugs)
  - [💡 Suggesting Enhancements](#-suggesting-enhancements)
  - [📖 Improving Documentation](#-improving-documentation)
  - [💻 Writing Code](#-writing-code)
- [Development Setup](#development-setup)
- [Our Pull Request Process](#our-pull-request-process)
- [Architectural Philosophy](#architectural-philosophy)
- [Style Guides](#style-guides)

## Code of Conduct

This project and everyone participating in it is governed by the [Telar Code of Conduct](./CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior.

*(**Advisor's Note:** You will need to create a `CODE_OF_CONDUCT.md` file. The [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/) is the industry standard and an excellent choice.)*

## How Can I Contribute?

The best place to start is by joining our community on [**Discord**](https://discord.gg/27Uekrq9gx) and checking out our [**GitHub Issues**](https://github.com/qolzam/telar/issues).

### 🐛 Reporting Bugs

If you find a bug, please ensure the bug has not already been reported by searching on GitHub under [Issues](https://github.com/qolzam/telar/issues). If you're unable to find an open issue addressing the problem, [open a new one](https://github.com/qolzam/telar/issues/new). Be sure to include a **title and clear description**, as much relevant information as possible, and a **code sample or an executable test case** demonstrating the expected behavior that is not occurring.

### 💡 Suggesting Enhancements

We love hearing your ideas for how to make Telar Social better! If you have an idea for an enhancement, please [open a new issue](https://github.com/qolzam/telar/issues/new). Please provide a clear description of the enhancement and the problem it solves.

### 📖 Improving Documentation

Good documentation is key to a great developer experience. If you find something that is confusing, missing, or could be improved, please feel free to open an issue or submit a pull request.

### 💻 Writing Code

If you'd like to contribute code, start by looking through our issues, especially those tagged `good first issue` or `help wanted`. We highly recommend **opening an issue to discuss your proposed changes** before you start working on a pull request. This helps ensure that your work is aligned with the project's goals and prevents wasted effort.

## Development Setup

Getting your local development environment up and running is designed to be as simple as possible.

1.  **Prerequisites:**
    *   Docker & Docker Compose
    *   Go (version 1.19 or higher)
    *   Node.js (version 18 or higher)
    *   `yarn` (our preferred package manager): `yarn install`

2.  **Fork & Clone:**
    *   Fork the repository on GitHub.
    *   Clone your forked repository: `git clone https://github.com/qolzam/telar.git`

3.  **Install & Run:**
    ```bash
    # Navigate into the project directory
    cd telar

    # Install all frontend dependencies using yarn workspaces
    yarn install

    # Start the entire platform (Postgres, Weaviate, Backend, Frontend)
    docker-compose up
    ```

Your local instance of Telar Social should now be running!

## Our Pull Request Process

We follow a standard GitHub flow for pull requests.

1.  **Claim an Issue:** Ensure there is a GitHub issue that describes the work you are doing. If not, create one. Assign the issue to yourself or comment that you are working on it.
2.  **Create a Branch:** Create a new branch from `main` in your forked repository. Please use a descriptive branch name (e.g., `feature/add-poll-feature` or `fix/profile-avatar-bug`).
3.  **Code & Test:**
    *   Make your changes, adhering to the project's architectural philosophy and style guides.
    *   **Add tests!** Your PR will not be accepted without adequate tests for the new code. We aim for high test coverage to maintain quality.
4.  **Commit Your Changes:** We follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification. This helps us maintain a clean and readable git history. Your commit messages should be in the format `feat: A brief summary of the feature` or `fix: A brief summary of the fix`.
5.  **Submit a Pull Request:**
    *   Push your branch to your forked repository.
    *   Open a pull request against the `main` branch of `qolzam/telar`.
    *   In the PR description, **link the issue** that your PR addresses (e.g., `Closes #123`).
    *   Provide a clear summary of the changes you have made.
    *   Ensure all CI checks are passing.

## Architectural Philosophy

To ensure consistency and maintainability, we follow a few key architectural principles. Please keep these in mind when contributing.

*   **Monorepo:** All our code lives in this single repository. This allows for unified tooling, atomic commits across services, and easier dependency management.
*   **Vertical Slices (Backend):** The Go backend is structured by feature, not by layer. All the code for a feature (handlers, services, models) lives in its own directory (e.g., `/apps/api/posts/`). For complex features, we break it down further by "Use Case" (e.g., `/apps/api/auth/login/`).
*   **Spec-First API:** We use OpenAPI to define our API contract *before* writing code. This ensures our API is well-designed and our documentation is always accurate. Please update the relevant `.yaml` file in `/docs/api-reference/` when making API changes.
*   **Unified Frontend:** The Next.js app is a single, unified platform that serves the marketing site, the core application, and the admin dashboard, using domain-based routing.

## Style Guides

*   **Go:** Please follow the standards of [Effective Go](https://go.dev/doc/effective_go) and format your code with `gofmt`.
*   **TypeScript/React:** We use ESLint and Prettier to enforce a consistent style. Please run `yarn lint` and `yarn format` before committing your changes.

Thank you again for your interest in contributing! We look forward to building an amazing platform with you.