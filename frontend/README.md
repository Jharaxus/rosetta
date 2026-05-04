# Frontend

React SPA with Vite, TypeScript, React Router, and TanStack Query.

## Stack

- **Framework**: [React 18](https://react.dev) + [TypeScript](https://www.typescriptlang.org)
- **Build**: [Vite 5](https://vitejs.dev) with `@vitejs/plugin-react`
- **Routing**: [React Router v6](https://reactrouter.com)
- **Data fetching**: [TanStack Query v5](https://tanstack.com/query)
- **Dev hot-reload**: Vite dev server with HMR

## Pages

| Path         | Access | Description                          |
|--------------|--------|--------------------------------------|
| `/`          | Public | Welcome page with "Sign in" button   |
| `/dashboard` | Auth   | "Hello World, {display_name}!" page  |

## Authentication Flow

The frontend is entirely **passive** with respect to tokens:

1. "Sign in" is a plain `<a href="/api/auth/login">` — the browser navigates to the backend which starts the OIDC flow
2. After Keycloak redirects back, the backend sets an httpOnly `rosetta_session` cookie
3. `useAuth()` calls `GET /api/auth/me` (with `credentials: 'include'`) to check auth status
4. "Sign out" uses `window.location.href = '/api/auth/logout'` so the browser can follow the 302 → Keycloak end_session redirect chain

**No tokens ever touch JavaScript.** The session cookie is httpOnly and inaccessible from JS.

## Local Development

```bash
cd frontend
npm install
npm run dev     # starts on http://localhost:5173
```

Vite proxies `/api/*` to `http://localhost:8090` (configurable via `VITE_BACKEND_URL`).

## Environment Variables

| Variable          | Description                                   | Default                    |
|-------------------|-----------------------------------------------|----------------------------|
| `VITE_BACKEND_URL`| Backend target for the `/api` proxy           | `http://localhost:8090`    |

## Key Files

```
src/
├── main.tsx                    App entry point
├── router.tsx                  Route definitions
├── api/auth.ts                 getMe() — fetches /api/auth/me
├── hooks/useAuth.ts            useAuth() — wraps getMe() with TanStack Query
├── types/auth.ts               User interface
├── pages/
│   ├── LandingPage.tsx         Welcome + login button
│   └── DashboardPage.tsx       Hello World + logout
└── components/
    ├── ProtectedRoute.tsx      Redirects to / if unauthenticated
    └── LogoutButton.tsx        window.location.href logout
```
