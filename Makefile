SHELL := $(shell if [ -x /bin/zsh ]; then echo /bin/zsh; elif command -v bash >/dev/null 2>&1; then command -v bash; elif command -v sh >/dev/null 2>&1; then command -v sh; elif command -v pwsh >/dev/null 2>&1; then command -v pwsh; else echo /bin/sh; fi)

ROOT_DIR := $(CURDIR)
SERVER_DIR := $(ROOT_DIR)/server
CLIENT_DIR := $(ROOT_DIR)/client
RUN_DIR := $(ROOT_DIR)/.run
SERVER_PID := $(RUN_DIR)/server.pid
CLIENT_PID := $(RUN_DIR)/client.pid
SERVER_LOG := $(RUN_DIR)/server.log
CLIENT_LOG := $(RUN_DIR)/client.log
GO_CACHE := /tmp/kazhuta-go-cache

.PHONY: help install start start-server start-client stop stop-server stop-client restart test test-server test-client build build-client status clean-run

help:
	@echo "Kazhuta commands:"
	@echo "  make install       Install frontend dependencies"
	@echo "  make start         Start backend and frontend in background"
	@echo "  make stop          Stop backend and frontend"
	@echo "  make restart       Stop and start both servers"
	@echo "  make status        Show running server status"
	@echo "  make test          Run backend and frontend tests"
	@echo "  make build         Build the React frontend"
	@echo "  make clean-run     Remove local pid/log files"

install:
	cd $(CLIENT_DIR) && npm install

start: start-server start-client
	@echo "Backend:  http://127.0.0.1:8080"
	@echo "Frontend: http://127.0.0.1:5173"

start-server:
	@mkdir -p $(RUN_DIR)
	@if [ -f $(SERVER_PID) ] && kill -0 "$$(cat $(SERVER_PID))" 2>/dev/null; then \
		echo "Backend already running with pid $$(cat $(SERVER_PID))"; \
	else \
		cd $(SERVER_DIR) && GOCACHE=$(GO_CACHE) nohup go run ./cmd/server > $(SERVER_LOG) 2>&1 & echo $$! > $(SERVER_PID); \
		echo "Started backend with pid $$(cat $(SERVER_PID))"; \
	fi

start-client:
	@mkdir -p $(RUN_DIR)
	@if [ -f $(CLIENT_PID) ] && kill -0 "$$(cat $(CLIENT_PID))" 2>/dev/null; then \
		echo "Frontend already running with pid $$(cat $(CLIENT_PID))"; \
	else \
		cd $(CLIENT_DIR) && nohup npm run dev -- --host 127.0.0.1 > $(CLIENT_LOG) 2>&1 & echo $$! > $(CLIENT_PID); \
		echo "Started frontend with pid $$(cat $(CLIENT_PID))"; \
	fi

stop: stop-client stop-server

stop-server:
	@if [ -f $(SERVER_PID) ] && kill -0 "$$(cat $(SERVER_PID))" 2>/dev/null; then \
		kill "$$(cat $(SERVER_PID))"; \
		echo "Stopped backend"; \
	else \
		echo "Backend is not running"; \
	fi
	@rm -f $(SERVER_PID)

stop-client:
	@if [ -f $(CLIENT_PID) ] && kill -0 "$$(cat $(CLIENT_PID))" 2>/dev/null; then \
		kill "$$(cat $(CLIENT_PID))"; \
		echo "Stopped frontend"; \
	else \
		echo "Frontend is not running"; \
	fi
	@rm -f $(CLIENT_PID)

restart: stop start

status:
	@if [ -f $(SERVER_PID) ] && kill -0 "$$(cat $(SERVER_PID))" 2>/dev/null; then \
		echo "Backend running with pid $$(cat $(SERVER_PID))"; \
	else \
		echo "Backend stopped"; \
	fi
	@if [ -f $(CLIENT_PID) ] && kill -0 "$$(cat $(CLIENT_PID))" 2>/dev/null; then \
		echo "Frontend running with pid $$(cat $(CLIENT_PID))"; \
	else \
		echo "Frontend stopped"; \
	fi

test: test-server test-client

test-server:
	cd $(SERVER_DIR) && GOCACHE=$(GO_CACHE) go test ./...

test-client:
	cd $(CLIENT_DIR) && npm test

build: build-client

build-client:
	cd $(CLIENT_DIR) && npm run build

clean-run:
	rm -rf $(RUN_DIR)
