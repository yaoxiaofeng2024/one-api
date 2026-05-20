"""
文本编辑接口（Edits）— 已弃用
对应路由：POST /v1/edits
文档：https://platform.openai.com/docs/api-reference/edits

该接口可给定一段文本和指令，让模型修改文本。
OpenAI 已将其标记为弃用，建议用 chat/completions 替代。
one-api 仍保留了此路由的转发支持。
"""
from openai import OpenAI

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

# 注意：OpenAI SDK 1.0+ 已移除 edits 接口，需用 httpx 直接调用
import httpx
import json

resp = httpx.post(
    "http://localhost:3000/v1/edits",
    headers={
        "Authorization": "Bearer sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
        "Content-Type": "application/json"
    },
    json={
        "model": "text-davinci-edit-001",
        "input": "今天天气很好，我想出去走一走。",
        "instruction": "将这段话改成英文"
    },
    timeout=30
)

print(f"状态码：{resp.status_code}")
print(f"响应：{json.dumps(resp.json(), indent=2, ensure_ascii=False)}")
