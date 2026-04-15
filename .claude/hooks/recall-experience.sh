#!/bin/bash
# recall-experience.sh — Claude Code UserPromptSubmit hook
# 自动检索经验库，将相关经验注入到对话上下文中
PROMPT=$(jq -r '.prompt // empty')
[ -z "$PROMPT" ] && exit 0

curl -sf --max-time 2 'http://localhost:3100/api/search' \
  -H 'Content-Type: application/json' \
  -d "{\"query\":$(echo "$PROMPT" | jq -Rs .),\"top_k\":3}"
