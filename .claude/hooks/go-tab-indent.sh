#!/bin/bash
# go-tab-indent.sh — PreToolUse hook
# 对 .go 文件的 Edit 操作，自动将 old_string 和 new_string 中的行首空格缩进转为 tab
# 非 .go 文件直接放行

INPUT=$(cat)

TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# 只拦截 Edit 工具 + .go 文件
if [ "$TOOL_NAME" != "Edit" ] || [[ "$FILE_PATH" != *.go ]]; then
	exit 0
fi

OLD_STRING=$(echo "$INPUT" | jq -r '.tool_input.old_string // empty')
NEW_STRING=$(echo "$INPUT" | jq -r '.tool_input.new_string // empty')

# 检测是否有行首空格缩进（2个及以上连续空格开头的非空行）
NEEDS_FIX=false
if echo "$OLD_STRING" | grep -qP '^  +\S'; then
	NEEDS_FIX=true
fi
if echo "$NEW_STRING" | grep -qP '^  +\S'; then
	NEEDS_FIX=true
fi

if [ "$NEEDS_FIX" = false ]; then
	exit 0
fi

# 行首空格转 tab：反复将行首的连续空格按 4 或 2 为一组替换为 tab
spaces_to_tabs() {
	local line indent rest
	while IFS= read -r line || [ -n "$line" ]; do
		# 提取行首空格
		indent=""
		rest="$line"
		while [ "${rest:0:1}" = " " ]; do
			indent="${indent} "
			rest="${rest:1}"
		done
		# 空行或无缩进行直接输出
		if [ -z "$indent" ]; then
			printf '%s\n' "$line"
			continue
		fi
		# 空格数转为 tab：每 4 空格一个 tab，不足 4 的按 2 空格一个 tab
		local len=${#indent}
		local tabs=""
		while [ "$len" -ge 4 ]; do
			tabs="${tabs}	"
			len=$((len - 4))
		done
		while [ "$len" -ge 2 ]; do
			tabs="${tabs}	"
			len=$((len - 2))
		done
		printf '%s%s\n' "$tabs" "$rest"
	done
}

FIXED_OLD=$(echo "$OLD_STRING" | spaces_to_tabs)
FIXED_NEW=$(echo "$NEW_STRING" | spaces_to_tabs)
REPLACE_ALL=$(echo "$INPUT" | jq -r '.tool_input.replace_all // false')

# 输出修正后的参数
jq -n \
	--arg fp "$FILE_PATH" \
	--arg old "$FIXED_OLD" \
	--arg new "$FIXED_NEW" \
	--argjson ra "$REPLACE_ALL" \
	'{
		hookSpecificOutput: {
			hookEventName: "PreToolUse",
			permissionDecision: "allow",
			permissionDecisionReason: "已自动将空格缩进转换为 tab",
			updatedInput: {
				file_path: $fp,
				old_string: $old,
				new_string: $new,
				replace_all: $ra
			}
		}
	}'
