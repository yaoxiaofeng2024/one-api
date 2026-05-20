import os
from openai import OpenAI

client = OpenAI(
    # 各地域的API Key不同。获取API Key：https://help.aliyun.com/zh/model-studio/get-api-key
    api_key="sk-a73ee6f79ce54b7ca96f9b3e947f924e",
    # 以下是北京地域base-url，如果使用新加坡地域的模型，需要将base_url替换为：https://dashscope-intl.aliyuncs.com/compatible-mode/v1
    base_url="https://dashscope.aliyuncs.com/compatible-mode/v1",
)

resp = client.embeddings.create(
    model="text-embedding-v4",
    input=["喜欢，以后还来这里买"],
    # 将向量维度设置为 256
    dimensions=256
)
print(f"向量维度: {len(resp.data[0].embedding)}")
print(resp)

