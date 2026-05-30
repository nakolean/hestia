# ── Frontend build (Preact + Vite) ──
FROM node:20-alpine AS frontend-build
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# ── Backend build (Go) ──
FROM golang:1.26.3-alpine AS backend-build
ENV GOTOOLCHAIN=auto
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server .

# ── Final image (Alpine) ──
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend-build /app/backend/server ./server
COPY --from=frontend-build /app/frontend/dist ./dist
EXPOSE 8080
CMD ["./server"]
