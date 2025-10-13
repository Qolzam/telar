# Telar SDK

TypeScript SDK for the Telar Social Platform API.

## Overview

This SDK provides a clean, type-safe interface for communicating with the Telar backend services. It handles HTTP requests, authentication, error handling, and data serialization.

## Architecture

- **Auth API**: Calls Next.js BFF routes (`/api/auth/*`) for security-sensitive operations
- **Future APIs**: Will call Go backend directly for data operations (posts, profiles, etc.)

## Usage

```typescript
import { createTelarSDK } from '@telar/sdk';

const sdk = createTelarSDK();

// Login
await sdk.auth.login({ username: 'user@example.com', password: 'password' });

// Get session
const session = await sdk.auth.getSession();
```

## Development

```bash
# Build
pnpm build

# Watch mode
pnpm dev
```

