"""
文本向量化接口（Embeddings）
对应路由：POST /v1/embeddings
文档：https://platform.openai.com/docs/api-reference/embeddings

将文本转换为高维向量，常用于：
  - 语义搜索：将查询和文档都向量化，通过余弦相似度匹配
  - 文本聚类：将相似文本分到同一组
  - 文本分类：用向量作为分类器输入
"""
from openai import OpenAI

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

# 基本用法：将文本转为向量
response = client.embeddings.create(
    model="text-embedding-ada-002",  # 嵌入模型
    input="你好，世界！"               # 要向量化的文本，也可以传列表批量处理
)

# 输出向量信息
embedding = response.data[0].embedding
print(f"模型：{response.model}")
print(f"向量维度：{len(embedding)}")        # ada-002 输出 1536 维
print(f"向量前5个值：{embedding[:5]}")       # 只展示前5个值
print(f"总 token 数：{response.usage.total_tokens}")

# 批量向量化示例
response2 = client.embeddings.create(
    model="text-embedding-ada-002",
    input=[
        "机器学习是人工智能的一个分支",
        "深度学习是机器学习的子领域",
        "今天天气真好"
    ]
)
print(f"\n批量向量化：{len(response2.data)} 条文本，每条 {len(response2.data[0].embedding)} 维")

# 简单语义相似度计算
import math
def cosine_similarity(v1, v2):
    dot = sum(a * b for a, b in zip(v1, v2))
    norm1 = math.sqrt(sum(a * a for a in v1))
    norm2 = math.sqrt(sum(b * b for b in v2))
    return dot / (norm1 * norm2)

vec0 = response2.data[0].embedding  # 机器学习
vec1 = response2.data[1].embedding  # 深度学习
vec2 = response2.data[2].embedding  # 天气

print(f"\n语义相似度：")
print(f"  '机器学习' vs '深度学习': {cosine_similarity(vec0, vec1):.4f}")
print(f"  '机器学习' vs '天气':     {cosine_similarity(vec0, vec2):.4f}")
