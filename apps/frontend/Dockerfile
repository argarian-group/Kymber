FROM node:24-alpine AS build

WORKDIR /workspace

RUN npm install --global pnpm@11.5.1
COPY apps/frontend/package.json apps/frontend/pnpm-lock.yaml ./apps/frontend/
WORKDIR /workspace/apps/frontend
RUN pnpm install --frozen-lockfile

WORKDIR /workspace
COPY apps/frontend/ ./apps/frontend/
COPY assets/ ./assets/
WORKDIR /workspace/apps/frontend
RUN pnpm build

FROM nginx:alpine

COPY --from=build /workspace/apps/frontend/dist/ /usr/share/nginx/html/
COPY apps/frontend/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
