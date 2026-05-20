"""
内容审核接口（Moderations）
对应路由：POST /v1/moderations
文档：https://platform.openai.com/docs/api-reference/moderations

检查文本是否包含违规内容，包括以下类别：
  - hate（仇恨言论）
  - hate/threatening（仇恨+威胁）
  - self-harm（自残）
  - sexual（色情）
  - sexual/minors（涉及未成年人色情）
  - violence（暴力）
  - violence/graphic（血腥暴力）
"""
from openai import OpenAI

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

# 基本用法：审核文本内容
response = client.moderations.create(
    model="text-moderation-latest",  # 审核模型：text-moderation-latest 或 text-moderation-stable
    input="我想了解人工智能的发展历史",
)

result = response.results[0]
print(f"是否被标记：{result.flagged}")

if result.flagged:
    print("\n违规类别：")
    for category, flagged in result.categories:
        if flagged:
            print(f"  ❌ {category}")
    print("\n各类别置信度：")
    for category, score in result.category_scores:
        print(f"  {category}: {score:.4f}")
else:
    print("✅ 内容正常，未检测到违规")

# 审核包含敏感内容的文本
print("\n" + "=" * 50)
response2 = client.moderations.create(
    input="I will hurt you and make you suffer"
)
result2 = response2.results[0]
print(f"是否被标记：{result2.flagged}")
if result2.flagged:
    print("违规类别：")
    for category, flagged in result2.categories:
        if flagged:
            print(f"  ❌ {category}")
