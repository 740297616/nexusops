FROM node:24.11.1-bookworm-slim AS dependencies
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@11.25.0 --activate
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

FROM dependencies AS development
COPY . .
EXPOSE 80
CMD ["pnpm", "dev", "--host", "0.0.0.0", "--port", "80", "--strictPort"]

FROM dependencies AS build
COPY . .
RUN pnpm build

FROM nginx:1.27.5-alpine AS production
COPY nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
