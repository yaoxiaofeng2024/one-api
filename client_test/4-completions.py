"""
文本补全接口（Completions）
对应路由：POST /v1/completions
文档：https://platform.openai.com/docs/api-reference/completions

这是 OpenAI 最早的文本生成接口，给定一段 prompt，模型会续写文本。
现已逐步被 chat/completions 替代，但仍可用于纯文本续写场景。
"""
from openai import OpenAI

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

# 基本用法：给定 prompt，模型续写
response = client.completions.create(
    model="gpt-3.5-turbo-instruct",  # 注意：completions 接口用 instruct 模型
    prompt="从前有座山，山里有座庙，",
    max_tokens=100,
    temperature=0.7
)

print("续写结果：")
print(response.choices[0].text)
print(f"\n用量：输入 {response.usage.prompt_tokens} tokens, 输出 {response.usage.completion_tokens} tokens")
