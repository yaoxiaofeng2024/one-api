"""
旧版 Embeddings 接口
对应路由：POST /v1/engines/:model/embeddings

这是 OpenAI 早期 API 格式，用 /engines/{model}/embeddings 代替 /v1/embeddings?model=xxx。
one-api 保留此路由以兼容旧版客户端。新项目请使用 /v1/embeddings。
"""
import httpx
import json

BASE_URL = "http://localhost:3000"
API_KEY = "sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2"

# 旧版格式：模型名在 URL 路径中，而非请求体
resp = httpx.post(
    f"{BASE_URL}/v1/engines/text-embedding-ada-002/embeddings",
    headers={
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    },
    json={
        "input": "这是旧版 embeddings 接口的示例"
    },
    timeout=30
)

print(f"状态码：{resp.status_code}")
if resp.status_code == 200:
    data = resp.json()
    embedding = data['data'][0]['embedding']
    print(f"向量维度：{len(embedding)}")
    print(f"向量前5个值：{embedding[:5]}")
