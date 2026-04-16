#!/bin/bash
# recall-experience.sh — Claude Code UserPromptSubmit hook
# 未认证时返回认证指南，已认证时检索经验库
PROMPT=$(jq -r '.prompt // empty')
[ -z "$PROMPT" ] && exit 0

# 先尝试已认证的经验检索
RESULT=$(curl -sf --max-time 2 'http://localhost:3100/api/search' \
  -H 'Content-Type: application/json' \
  -d "{\"query\":$(echo "$PROMPT" | jq -Rs .),\"top_k\":3}" 2>/dev/null)

if [ $? -eq 0 ] && [ -n "$RESULT" ]; then
	echo "$RESULT"
else
	# 未认证（401），返回认证指南
	curl -sf --max-time 2 'http://localhost:3100/api/auth/guide' 2>/dev/null
fi
