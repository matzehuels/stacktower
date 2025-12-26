# =============================================================================
# Stage 1: Dependencies (cached unless package.json changes)
# =============================================================================
FROM node:20-alpine AS deps

WORKDIR /app

COPY package*.json ./
RUN npm ci

# =============================================================================
# Stage 2: Build (uses cached deps)
# =============================================================================
FROM deps AS builder

COPY . .
RUN npm run build

# =============================================================================
# Stage 3: Production (minimal nginx image)
# =============================================================================
FROM nginx:alpine-slim

# Copy built files
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]

