# NexusOps Web

Use Node.js 20.19+ or 22.12+ and pnpm. Run `pnpm install` then `pnpm dev`. The development server proxies `/api` to `http://127.0.0.1:8080`; set `VITE_PROXY_TARGET` to change it. `VITE_API_BASE_URL` defaults to `/api/v1`. Copy `.env.example` to `.env.local` for local overrides; Vite variables are public and must not contain secrets.

Run `pnpm typecheck`, `pnpm lint` and `pnpm build` to verify the application. Production hosting must serve the SPA fallback and proxy `/api` to Server over a protected connection. Pages other than the health status are placeholders; ECharts is installed for future real metrics and no sample data is displayed.
