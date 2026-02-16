.PHONY: build install test lint clean examples

BIN := .bin/cg
PKG := ./cmd/cg

build:
	go build -o $(BIN) $(PKG)

install:
	go install $(PKG)

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f $(BIN)

# Regenerate all example outputs from source JSONL files.
examples: build
	# Single session: HTML + terminal
	./$(BIN) render -a claude -f examples/session.jsonl --no-redact --format html -o examples
	./$(BIN) render -a claude -f examples/session.jsonl --no-redact --format terminal -o examples
	mv examples/index.html examples/session.html
	mv examples/index.txt examples/session.txt
	# Subagent demo: build temp dir layout, render, clean up
	rm -rf /tmp/cg-subagent-demo
	mkdir -p /tmp/cg-subagent-demo/sess-main/subagents
	cp reader/claude/testdata/subagent_main.jsonl /tmp/cg-subagent-demo/sess-main.jsonl
	cp reader/claude/testdata/subagent_child.jsonl /tmp/cg-subagent-demo/sess-main/subagents/agent-ae267a1.jsonl
	./$(BIN) render -a claude -f /tmp/cg-subagent-demo/sess-main.jsonl --no-redact --format html -o examples/subagent-demo
	rm -rf /tmp/cg-subagent-demo
