"""
查询单个模型详情
对应路由：GET /v1/models/:model
文档：https://platform.openai.com/docs/api-reference/models/retrieve

根据模型ID查询该模型的详细信息，包括创建时间、归属组织等。
"""
from openai import OpenAI

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

# 查询单个模型详情
model = client.models.retrieve("gpt-3.5-turbo")

print(f"模型ID：{model.id}")
print(f"创建时间：{model.created}")
print(f"归属：{model.owned_by}")
print(f"完整信息：{model}")
