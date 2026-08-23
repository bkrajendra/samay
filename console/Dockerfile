# Stage 1: build the React frontend
FROM node:20-alpine AS frontend
WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: build the Go backend, embedding the frontend build output
FROM golang:1.23-alpine AS backend
WORKDIR /backend
COPY backend/go.mod ./
RUN go mod download
COPY backend/ ./
RUN rm -rf web/static
COPY --from=frontend /frontend/dist ./web/static
RUN CGO_ENABLED=0 go build -o /console ./cmd/server

# Stage 3: runtime. Includes the chrony package for the chronyc CLI the
# backend shells out to (see internal/chrony) — it never runs arbitrary
# commands, only fixed chronyc subcommands.
FROM alpine:3.20
RUN apk add --no-cache chrony
COPY --from=backend /console /usr/local/bin/console

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/console"]
