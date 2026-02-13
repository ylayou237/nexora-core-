# --- Etape 1: Builder ---
FROM golang:1.24-alpine AS builder

# Git est souvent nécessaire pour récupérer des dépendances Go
RUN apk add --no-cache git

WORKDIR /app

# Optimisation du cache Docker : on ne télécharge les deps que si go.mod change
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compilation statique optimisée
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /nexora-app ./cmd/api/main.go

# --- Etape 2: Runtime (Production) ---
FROM alpine:3.19

# Sécurité : Création d'un utilisateur non-privilégié
RUN addgroup -S nexgroup && adduser -S nexuser -G nexgroup

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /home/nexuser/

# Copie sélective depuis le builder
COPY --from=builder --chown=nexuser:nexgroup /nexora-app .
COPY --from=builder --chown=nexuser:nexgroup /app/migrations ./migrations
COPY --from=builder --chown=nexuser:nexgroup /app/configs ./configs

# Utilisation de l'utilisateur non-root
USER nexuser

# Ports standards + RADIUS
EXPOSE 8080 1812/udp 1813/udp

ENTRYPOINT ["./nexora-app"]