#!/bin/bash
# recall-experience.sh — Claude Code UserPromptSubmit hook (global)
# 清洗口语化提问，提取核心语义再检索；未认证时引导用户配置 token

CANNED_EXP_URL="${CANNED_EXP_URL:-http://localhost:3000}"
TOKEN_FILE="$HOME/.claude/canned-exp-token"
NO_AUTH_FLAG="$HOME/.claude/canned-exp-no-auth"

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

# 构建 curl 参数：有 token 则带上 Authorization header
CURL_ARGS=(-s --max-time 2 "${CANNED_EXP_URL}/api/search"
	-H 'Content-Type: application/json'
	-d "{\"query\":$(echo "$QUERY" | jq -Rs .),\"top_k\":5}")

if [ -f "$TOKEN_FILE" ]; then
	TOKEN=$(tr -d '[:space:]' < "$TOKEN_FILE" 2>/dev/null)
	if [ -n "$TOKEN" ]; then
		CURL_ARGS+=(-H "Authorization: Bearer ${TOKEN}")
		rm -f "$NO_AUTH_FLAG"
	fi
fi

# 发起请求，捕获 HTTP 状态码和响应体
RESP=$(curl -w '\n%{http_code}' "${CURL_ARGS[@]}" 2>/dev/null) || exit 0
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
	rm -f "$NO_AUTH_FLAG"
	echo "$BODY"
	exit 0
fi

if [ "$HTTP_CODE" = "401" ] && [ ! -f "$NO_AUTH_FLAG" ]; then
	touch "$NO_AUTH_FLAG"
	cat << GUIDE
{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"[canned-exp] 未认证，经验检索已跳过。请运行以下命令配置 token（替换为你的 6 位 TOTP 验证码）：\ncurl -sf -X POST ${CANNED_EXP_URL}/api/auth/login -H 'Content-Type: application/json' -d '{\"totp_code\":\"000000\",\"service\":\"hook\"}' | jq -r '.data.token' > ~/.claude/canned-exp-token\ntoken 有效期内无需重复配置。"}}
GUIDE
fi
