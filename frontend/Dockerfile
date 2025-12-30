# Stacktower Frontend - Development with hot reload
FROM node:20-alpine

WORKDIR /app

# Install dependencies (cached layer)
COPY package*.json ./
RUN npm ci

# Source is mounted as volume for hot-reload
EXPOSE 3000

CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
