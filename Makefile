.PHONY: all build install doctor dist clean skills-embed

BINARY_NAME=pi-gomemory
BIN_DIR=bin
DIST_DIR=dist
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "v1.0.0")

all: build

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BIN_DIR)/$(BINARY_NAME)-setup .

install: build
	@mkdir -p $(HOME)/.local/bin
	cp $(BIN_DIR)/$(BINARY_NAME)-setup $(HOME)/.local/bin/.pi-gomemory-setup.tmp
	mv -f $(HOME)/.local/bin/.pi-gomemory-setup.tmp $(HOME)/.local/bin/pi-gomemory-setup
	./$(BIN_DIR)/$(BINARY_NAME)-setup install

doctor: build
	./$(BIN_DIR)/$(BINARY_NAME)-setup doctor

dist:
	@bash scripts/build-release.sh $(VERSION)

# HCD-080: copia en skills/ las SKILL.md que instaló `mem` (mismo orden que
# memSkillSources), saltando las que resuelven dentro de ~/.pi-gomemory.
MEM_SKILL_SOURCES = $(HOME)/.claude/skills $(HOME)/.config/opencode/skills $(HOME)/.codex/skills $(HOME)/.gemini/config/skills
skills-embed:
	@for d in skills/*/; do \
		n=$$(basename $$d); \
		for s in $(MEM_SKILL_SOURCES); do \
			f=$$s/$$n/SKILL.md; [ -f $$f ] || continue; \
			case $$(realpath $$f) in $(HOME)/.pi-gomemory/*) continue;; esac; \
			if ! cmp -s $$f $$d/SKILL.md; then cp $$f $$d/SKILL.md; echo "✓ $$n ← $$s"; fi; \
			break; \
		done; \
	done

clean:
	rm -rf $(BIN_DIR)/$(BINARY_NAME)-setup $(DIST_DIR)
