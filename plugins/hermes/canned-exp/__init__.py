"""canned-exp experience recall plugin for Hermes/OpenClaw.

Injects relevant experiences from the canned-exp knowledge base
into every user message via the pre_llm_call hook.
"""

import httpx
import os
import re

_API_URL = os.getenv("CANNED_EXP_URL", "http://localhost:3000/api/search")
_API_KEY = os.getenv("CANNED_EXP_API_KEY", "")
_TIMEOUT = 3
_TOP_K = 5


def _clean_query(text: str) -> str:
    """Strip colloquial wrappers to extract semantic core."""
    text = re.sub(r"^(你还?记得|你知道|请问|帮我|能不能)", "", text)
    text = re.sub(r"^(我们|我)(之前)?", "", text)
    text = re.sub(r"[吗么呢吧啊?？!！。]+$", "", text)
    return text.strip() or text


def recall_experience(user_message: str, **kwargs) -> dict | None:
    """pre_llm_call hook — search canned-exp and inject results."""
    try:
        headers = {"Content-Type": "application/json"}
        if _API_KEY:
            headers["Authorization"] = f"Bearer {_API_KEY}"

        query = _clean_query(user_message)
        resp = httpx.post(
            _API_URL,
            json={"query": query, "top_k": _TOP_K},
            headers=headers,
            timeout=_TIMEOUT,
        )
        resp.raise_for_status()
        data = resp.json()

        hook = data.get("hookSpecificOutput", {})
        context = hook.get("additionalContext")
        if not context:
            return None

        return {"context": context}
    except Exception:
        return None


def register(ctx):
    ctx.register_hook("pre_llm_call", recall_experience)
