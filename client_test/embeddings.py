import os
import math
from openai import OpenAI

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

MODEL = "text-embedding-v4"

# ==================== 示例1：基本用法 ====================
print("=" * 50)
print("示例1：基本用法 - 单条文本向量化")
print("=" * 50)
resp = client.embeddings.create(
    model=MODEL,
    input="喜欢，以后还来这里买",
    dimensions=256  # 指定向量维度
)
embedding = resp.data[0].embedding
print(f"向量维度: {len(embedding)}")
print(f"向量前5个值: {embedding[:5]}")
print(f"Token 用量: {resp.usage.total_tokens}")

# ==================== 示例2：批量向量化 ====================
print("\n" + "=" * 50)
print("示例2：批量向量化 - 多条文本一次请求")
print("=" * 50)
texts = [
    "机器学习是人工智能的一个分支",
    "深度学习是机器学习的子领域",
    "今天天气真好，适合出去散步",
    "这道菜味道不错，推荐大家尝尝",
    "Python 是一门流行的编程语言",
]
resp2 = client.embeddings.create(
    model=MODEL,
    input=texts,
    dimensions=256
)
print(f"向量化了 {len(resp2.data)} 条文本")
for i, item in enumerate(resp2.data):
    print(f"  [{i}] '{texts[i]}' → {len(item.embedding)} 维, index={item.index}")

# ==================== 示例3：语义相似度计算 ====================
print("\n" + "=" * 50)
print("示例3：语义相似度 - 用余弦相似度衡量文本相关性")
print("=" * 50)

def cosine_similarity(v1, v2):
    """计算两个向量的余弦相似度，范围 [-1, 1]，越接近1越相似"""
    dot = sum(a * b for a, b in zip(v1, v2))
    norm1 = math.sqrt(sum(a * a for a in v1))
    norm2 = math.sqrt(sum(b * b for b in v2))
    return dot / (norm1 * norm2)

vecs = [item.embedding for item in resp2.data]
pairs = [
    (0, 1, "机器学习 vs 深度学习"),
    (0, 2, "机器学习 vs 天气"),
    (2, 3, "天气 vs 菜品评价"),
    (0, 4, "机器学习 vs Python"),
]
for i, j, desc in pairs:
    sim = cosine_similarity(vecs[i], vecs[j])
    print(f"  {desc}: {sim:.4f}")

# ==================== 示例4：语义搜索（最实用场景） ====================
print("\n" + "=" * 50)
print("示例4：语义搜索 - 在文档库中找到与查询最相关的文档")
print("=" * 50)

# 模拟一个小型文档库
documents = [
    "OpenAI 发布了 GPT-4 模型，具备强大的推理能力",
    "Python 3.12 正式发布，性能提升显著",
    "Transformer 架构是现代大语言模型的基础",
    "Docker 容器技术简化了应用部署流程",
    "RAG（检索增强生成）技术结合了搜索和大语言模型",
    "Redis 是一种高性能的键值存储数据库",
    "微服务架构将单体应用拆分为多个独立服务",
    "BERT 模型在自然语言理解任务上取得了突破",
]

# 一次性向量化所有文档 + 查询
query = "大语言模型相关技术"
all_texts = documents + [query]
resp3 = client.embeddings.create(
    model=MODEL,
    input=all_texts,
    dimensions=256
)

# 计算查询与每个文档的相似度
doc_vecs = [item.embedding for item in resp3.data[:-1]]
query_vec = resp3.data[-1].embedding

scores = [(cosine_similarity(query_vec, doc_vec), doc) for doc_vec, doc in zip(doc_vecs, documents)]
scores.sort(reverse=True)

print(f"查询: '{query}'")
print("搜索结果（按相关性排序）:")
for rank, (score, doc) in enumerate(scores, 1):
    bar = "█" * int(score * 50)  # 可视化相似度
    print(f"  {rank}. [{score:.4f}] {bar} {doc}")

# ==================== 示例5：不同维度对比 ====================
print("\n" + "=" * 50)
print("示例5：不同维度的向量对比（维度越低速度越快，但精度略降）")
print("=" * 50)

text = "这是一段用来测试不同向量维度的示例文本"
for dim in [256, 512, 1024, 2048]:
    resp_dim = client.embeddings.create(
        model=MODEL,
        input=text,
        dimensions=dim
    )
    print(f"  dimensions={dim:4d} → 实际维度={len(resp_dim.data[0].embedding)}, tokens={resp_dim.usage.total_tokens}")

# ==================== 示例6：文本分类（基于向量相似度） ====================
print("\n" + "=" * 50)
print("示例6：文本分类 - 用类别描述的向量对文本进行零样本分类")
print("=" * 50)

categories = {
    "科技": "与计算机、人工智能、互联网、软件相关的新闻",
    "体育": "与足球、篮球、奥运会、比赛相关的新闻",
    "财经": "与股票、经济、金融、投资相关的新闻",
    "娱乐": "与电影、音乐、明星、综艺相关的新闻",
}

news_to_classify = [
    "苹果公司发布新款 M4 芯片，性能提升50%",
    "中国队在世界杯预选赛中2:0获胜",
    "沪指大涨3%，突破3500点关口",
    "漫威新电影首周票房突破10亿",
]

# 向量化所有类别描述和待分类文本
all_inputs = list(categories.values()) + news_to_classify
resp4 = client.embeddings.create(
    model=MODEL,
    input=all_inputs,
    dimensions=256
)

cat_vecs = {name: resp4.data[i].embedding for i, name in enumerate(categories)}
news_start = len(categories)

for i, news in enumerate(news_to_classify):
    news_vec = resp4.data[news_start + i].embedding
    cat_scores = {name: cosine_similarity(news_vec, vec) for name, vec in cat_vecs.items()}
    best_cat = max(cat_scores, key=cat_scores.get)
    print(f"  '{news}'")
    print(f"    → 分类: {best_cat} (", end="")
    print(", ".join(f"{k}={v:.3f}" for k, v in sorted(cat_scores.items(), key=lambda x: -x[1])), end="")
    print(")")
