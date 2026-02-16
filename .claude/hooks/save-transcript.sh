#!/bin/bash
# Installed by cg install — renders Claude Code session transcripts to .transcripts/
set -e

INPUT=$(cat)
TRANSCRIPT_PATH=$(echo "$INPUT" | jq -r '.transcript_path')
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id')

if [ -z "$TRANSCRIPT_PATH" ] || [ "$TRANSCRIPT_PATH" = "null" ]; then
  exit 0
fi

if [ ! -f "$TRANSCRIPT_PATH" ]; then
  exit 0
fi

DEST_DIR="$CLAUDE_PROJECT_DIR/.transcripts"
if [ ! -d "$DEST_DIR" ]; then
  exit 0
fi

cg render --agent claude --file "$TRANSCRIPT_PATH" --format html --out "$DEST_DIR/$SESSION_ID"

cg manifest upsert --agent claude --file "$TRANSCRIPT_PATH" \
  --manifest "$CLAUDE_PROJECT_DIR/.transcripts/manifest.json" \
  --href "$SESSION_ID/index.html"
