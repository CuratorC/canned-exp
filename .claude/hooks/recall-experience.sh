#!/bin/bash
# recall-experience.sh — Claude Code UserPromptSubmit hook
# 清洗口语化提问，提取核心语义再检索；未认证时返回认证指南
PROMPT=$(jq -r '.prompt // empty')
[ -z "$PROMPT" ] && exit 0

# 清洗口语化包裹词：去掉常见前缀和语气词
QUERY=$(echo "$PROMPT" | sed \
	-e 's/^你\(还\)\?记得//' \
	-e 's/^你知道//' \
	-e 's/^请问//' \
	-e 's/^帮我//' \
	-e 's/^能不能//' \
	-e 's/^\(我们\|我\)\(之前\)\?//' \
	-e 's/[吗么呢吧啊]+$//' \
	-e 's/[?？!！。]+$//' \
	-e 's/^ *//' \
	-e 's/ *$//')

# 清洗后为空则回退到原始 prompt
QUERY="${QUERY:-$PROMPT}"

# 已认证时检索经验库
RESULT=$(curl -sf --max-time 2 'http://localhost:3000/api/search' \
	-H 'Content-Type: application/json' \
	-d "{\"query\":$(echo "$QUERY" | jq -Rs .),\"top_k\":5}" 2>/dev/null)

if [ $? -eq 0 ] && [ -n "$RESULT" ]; then
	echo "$RESULT"
else
	# 未认证（401），返回认证指南
	curl -sf --max-time 2 'http://localhost:3000/api/auth/guide' 2>/dev/null
fi
