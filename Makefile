# @file Makefile
# @brief Makefile pro společnou správu hlavního RIoT stacku a preprocesorů.
#
# @author Dominik Vondruška
# @author Vojtěch Hubáček
#
# @defgroup riot_preprocessors RIoT preprocesory
# @see README.md
#
# @par Autorský podíl
# - Dominik Vondruška: původní provozní koncept preprocesorů napojených na RIoT.
# - Vojtěch Hubáček: úprava pro oddělený repozitář, parametrizaci cesty k RIoT a společné spouštění obou stacků.
COMPOSE ?= docker compose
RIOT_DIR ?= ../RIoT
DOCKER_DIR ?= docker
RIOT_DOCKER_DIR ?= $(RIOT_DIR)/docker
LOG_TAIL ?= 30
ARGS ?= -h

RIOT_COMPOSE = $(COMPOSE) --project-directory "$(RIOT_DIR)" -f "$(RIOT_DIR)/docker-compose.yml"
PREPROCESSORS_COMPOSE = $(COMPOSE) --project-directory "." -f "docker-compose.yml"

ifeq ($(OS),Windows_NT)
	RM_DOCKER_DIR = if exist "$(DOCKER_DIR)" rmdir /S /Q "$(DOCKER_DIR)"
	RM_RIOT_DOCKER_DIR = if exist "$(RIOT_DOCKER_DIR)" rmdir /S /Q "$(RIOT_DOCKER_DIR)"
else
	RM_DOCKER_DIR = rm -rf "$(DOCKER_DIR)"
	RM_RIOT_DOCKER_DIR = rm -rf "$(RIOT_DOCKER_DIR)"
endif

.PHONY: help build run stop clear restart reset prune clear-all reset-all status logs test check-riot-dir

help:
	@echo "Configuration:"
	@echo "  RIOT_DIR=../RIoT        Path to the main RIoT repository"
	@echo ""
	@echo "Stack:"
	@echo "  make build              Start RIoT and preprocessors with image rebuild"
	@echo "  make run                Start RIoT and preprocessors"
	@echo "  make stop               Stop preprocessors and RIoT"
	@echo "  make restart            Stop stack and start it again"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clear              Stop stacks, remove volumes and delete runtime docker dirs"
	@echo "  make reset              Run clear and then build"
	@echo ""
	@echo "Aggressive cleanup:"
	@echo "  make prune              Stop stacks, remove volumes, images and orphan containers"
	@echo "  make clear-all          Run prune and delete runtime docker dirs"
	@echo "  make reset-all          Run clear-all and then build"
	@echo ""
	@echo "Diagnostics:"
	@echo "  make status             Show containers from both stacks"
	@echo "  make logs               Show logs from both stacks"
	@echo "  make logs LOG_TAIL=100  Show last 100 log lines"

check-riot-dir:
	@test -f "$(RIOT_DIR)/docker-compose.yml" || (echo "RIOT_DIR must point to the main RIoT repository with docker-compose.yml. Current value: $(RIOT_DIR)" && exit 1)

build: check-riot-dir
	$(RIOT_COMPOSE) up --build -d
	$(PREPROCESSORS_COMPOSE) up --build -d

run: check-riot-dir
	$(RIOT_COMPOSE) up -d
	$(PREPROCESSORS_COMPOSE) up -d

stop: check-riot-dir
	-$(PREPROCESSORS_COMPOSE) down
	-$(RIOT_COMPOSE) down

clear: check-riot-dir
	-$(PREPROCESSORS_COMPOSE) down -v
	-$(RIOT_COMPOSE) down -v
	@$(RM_DOCKER_DIR)
	@$(RM_RIOT_DOCKER_DIR)

restart: stop run

reset: clear build

prune: check-riot-dir
	-$(PREPROCESSORS_COMPOSE) down -v --rmi local --remove-orphans
	-$(RIOT_COMPOSE) down -v --rmi local --remove-orphans

clear-all: prune
	@$(RM_DOCKER_DIR)
	@$(RM_RIOT_DOCKER_DIR)

reset-all: clear-all build

status: check-riot-dir
	@echo "RIoT:"
	$(RIOT_COMPOSE) ps
	@echo ""
	@echo "Preprocessors:"
	$(PREPROCESSORS_COMPOSE) ps

logs: check-riot-dir
	@echo "RIoT logs:"
	$(RIOT_COMPOSE) logs --tail=$(LOG_TAIL)
	@echo ""
	@echo "Preprocessor logs:"
	$(PREPROCESSORS_COMPOSE) logs --tail=$(LOG_TAIL)
