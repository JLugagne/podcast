PLAYER_DIR := web/player

.PHONY: site build dev test lint clean

## site: install the player dependencies and build the static site.
site:
	cd $(PLAYER_DIR) && npm ci
	cd $(PLAYER_DIR) && npm run build

## build: alias for site.
build: site

## dev: run the Vite dev server.
dev:
	cd $(PLAYER_DIR) && npm run dev

## test: run the Vitest suite once.
test:
	cd $(PLAYER_DIR) && npm test -- --run

## lint: run Biome.
lint:
	cd $(PLAYER_DIR) && npm run lint

## clean: remove the build output and the generated e2e artifacts.
clean:
	rm -rf $(PLAYER_DIR)/dist
