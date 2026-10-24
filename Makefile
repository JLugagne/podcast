PLAYER_DIR := web/player
E2E_DIR    := tests/e2e-ui
CONTENT_DIR ?= content

.PHONY: site build dev check test lint e2e-ui clean

## site: install the player dependencies and build the static site.
site:
	cd $(PLAYER_DIR) && npm ci
	cd $(PLAYER_DIR) && CONTENT_DIR=../../$(CONTENT_DIR) npm run build

## build: alias for site.
build: site

## dev: run the Vite dev server (expects `npm run build` data in public/ or a previous build).
dev:
	cd $(PLAYER_DIR) && npm run dev

## check: validate the content tree without building.
check:
	cd $(PLAYER_DIR) && CONTENT_DIR=../../$(CONTENT_DIR) npm run check

## test: run the Vitest suite once.
test:
	cd $(PLAYER_DIR) && npm test -- --run

## lint: run Biome and the e2e TypeScript check.
lint:
	cd $(PLAYER_DIR) && npm run lint
	cd $(E2E_DIR) && npm run typecheck

## e2e-ui: build the site from the e2e content and run the Playwright journey.
e2e-ui:
	cd $(PLAYER_DIR) && npm ci
	node $(E2E_DIR)/fixtures/generate.cjs
	cd $(PLAYER_DIR) && CONTENT_DIR=../../$(E2E_DIR)/content npm run build
	cd $(E2E_DIR) && npm ci && npx playwright install chromium && npm run e2e

## clean: remove the build output and the generated e2e artifacts.
clean:
	rm -rf $(PLAYER_DIR)/dist $(E2E_DIR)/fixtures/generated $(E2E_DIR)/test-results $(E2E_DIR)/screenshots
	find $(E2E_DIR)/content -type d -name media -exec rm -rf {} +
