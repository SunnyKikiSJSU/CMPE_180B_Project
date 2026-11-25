# Next.js web tier. Multi-stage: install, build, then a small runtime image
# with only the standalone server (no node_modules, no source).

# Next.js 16 needs Node.js 20.9 or newer.
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

FROM node:22-alpine AS build
WORKDIR /app
# Where Next.js forwards /api/*. It is baked into the build (rewrites are
# resolved by `next build`), so Compose passes it as a build argument:
# http://api:8080 is the API container on the Compose network.
ARG API_PROXY_TARGET=http://api:8080
ENV API_PROXY_TARGET=${API_PROXY_TARGET} \
    NEXT_OUTPUT=standalone \
    NEXT_TELEMETRY_DISABLED=1
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:22-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    # Listen on all IPv4 interfaces so Compose and the health check can reach it.
    HOSTNAME=0.0.0.0
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
USER node
EXPOSE 3000
HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=5 \
  CMD wget -qO- http://127.0.0.1:3000/ >/dev/null || exit 1
CMD ["node", "server.js"]
