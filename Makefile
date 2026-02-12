.PHONY: help build test lint clean migrate docker-dev-up docker-prod-up docker-down docker-logs

# Variables
BINARY_NAME=nexora-core
DOCKER_COMPOSE_DEV=docker-compose.yml
DOCKER_COMPOSE_PROD=docker-compose.prod.yml

help: ## Affiche cette aide
	@echo "Commandes disponibles:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

## --- DEVELOPPEMENT ---

build: ## Compile les binaires en local
	go build -o bin/api ./cmd/api
	@echo "Build terminé dans ./bin"

test: ## Execute tous les tests
	go test -v -race -cover ./...

docker-dev: ## Lance l'environnement de DEV (Standard)
	docker-compose -f $(DOCKER_COMPOSE_DEV) up -d
	@echo "Environnement DEV lancé (Ports: 5432, 6379, 8080)"

## --- PRODUCTION ---

docker-prod: ## Lance l'environnement de PROD (Traefik + SSL)
	docker-compose -f $(DOCKER_COMPOSE_PROD) up -d --build
	@echo "Environnement PROD lancé avec Traefik"

docker-prod-logs: ## Affiche les logs de production en temps réel
	docker-compose -f $(DOCKER_COMPOSE_PROD) logs -f

## --- UTILITAIRES ---

docker-down: ## Arrête tous les conteneurs (indépendamment du fichier)
	docker-compose -f $(DOCKER_COMPOSE_DEV) down
	docker-compose -f $(DOCKER_COMPOSE_PROD) down

clean: ## Nettoie les fichiers générés
	rm -rf bin/
	go clean