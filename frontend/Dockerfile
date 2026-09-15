# syntax=docker/dockerfile:1

FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG NUXT_PUBLIC_API_BASE=http://localhost:18080
ARG NUXT_PUBLIC_DEFAULT_API_KEY=
ENV NUXT_PUBLIC_API_BASE=$NUXT_PUBLIC_API_BASE
ENV NUXT_PUBLIC_DEFAULT_API_KEY=$NUXT_PUBLIC_DEFAULT_API_KEY
RUN npm run build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production
ENV HOST=0.0.0.0
ENV PORT=3000
COPY --from=builder /app/.output ./.output
USER node
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
