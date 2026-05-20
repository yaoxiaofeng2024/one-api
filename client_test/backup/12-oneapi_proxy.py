"""
自定义代理接口（OneAPI Proxy）
对应路由：ANY /v1/oneapi/proxy/:channelid/*target

这是 one-api 特有的接口，不属于 OpenAI 标准 API。
用于直接指定渠道ID和目标路径，跳过自动渠道选择，将请求原样转发到指定渠道。

适用场景：
  - 调试特定渠道的连通性
  - 访问上游服务商的非标准接口（如 OpenAI 的 /v1/organizations/... 等管理接口）
  - 绕过 one-api 的渠道分发逻辑，直接控制请求路由

URL 格式：/v1/oneapi/proxy/{渠道ID}/{上游目标路径}
  例如：/v1/oneapi/proxy/1/chat/completions
  表示：使用渠道ID=1，转发到该渠道的 /chat/completions
"""
import httpx
import json

BASE_URL = "http://localhost:3000"
API_KEY = "sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2"

# 示例1：通过指定渠道ID发送聊天请求
# 假设渠道ID=1，目标是 /chat/completions
resp = httpx.post(
    f"{BASE_URL}/v1/oneapi/proxy/1/chat/completions",
    headers={
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    },
    json={
        "model": "gpt-3.5-turbo",
        "messages": [
            {"role": "user", "content": "你好"}
        ]
    },
    timeout=30
)
print(f"示例1 - 指定渠道聊天：状态码={resp.status_code}")
if resp.status_code == 200:
    data = resp.json()
    print(f"  回复：{data['choices'][0]['message']['content']}")

# 示例2：通过指定渠道发送 embeddings 请求
# 假设渠道ID=1，目标是 /embeddings
resp2 = httpx.post(
    f"{BASE_URL}/v1/oneapi/proxy/1/embeddings",
    headers={
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    },
    json={
        "model": "text-embedding-ada-002",
        "input": "Hello world"
    },
    timeout=30
)
print(f"\n示例2 - 指定渠道嵌入：状态码={resp2.status_code}")
if resp2.status_code == 200:
    data = resp2.json()
    print(f"  向量维度：{len(data['data'][0]['embedding'])}")

# 示例3：访问上游的非标准接口
# 例如某些服务商提供的特有接口（如查询模型详情等）
# 假设渠道ID=1，目标是 /models
resp3 = httpx.get(
    f"{BASE_URL}/v1/oneapi/proxy/1/models",
    headers={
        "Authorization": f"Bearer {API_KEY}",
    },
    timeout=30
)
print(f"\n示例3 - 指定渠道查询模型：状态码={resp3.status_code}")
