"""
Prompt Caching 缓存演示

两种主流缓存方式对比：
1. OpenAI — 自动缓存，无需 cache_control 参数，prompt 超 1024 tokens 自动触发
2. Anthropic Claude — 手动标记 cache_control，在 content 块中加 {"type": "ephemeral"}

通过 One-API 中转时，缓存由上游厂商处理，中转站只需正确透传 cache_control 字段。
"""

from openai import OpenAI
import time

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

# ============================================================
# 方式一：OpenAI 风格（自动缓存）
# OpenAI 不需要 cache_control，prompt 前缀超过 1024 tokens 自动缓存
# 通过 usage.prompt_tokens_details.cached_tokens 判断是否命中
# ============================================================

print("=" * 60)
print("OpenAI 风格：自动缓存演示")
print("=" * 60)

# 构造一个长 system prompt（确保超过 1024 tokens 阈值）
long_system_prompt = """你是一个专业的技术顾问。请根据以下规范回答问题：

## 技术规范
1. 所有回答必须基于最新的技术文档
2. 代码示例必须包含完整的错误处理
3. 回答时需考虑性能、安全性和可维护性

## 回答格式要求
- 使用 Markdown 格式
- 代码块需标注语言
- 重要信息使用粗体标记

## 常见问题处理
- 如果问题不明确，请要求用户补充信息
- 如果涉及多个技术栈，请分别说明
- 如果有已知的安全风险，必须明确指出

""" * 10  # 重复使内容超过 1024 tokens

messages = [
    {"role": "system", "content": long_system_prompt},
    {"role": "user", "content": "Python 中如何实现单例模式？"}
]

# 第一次请求：写入缓存
print("\n--- 第一次请求（缓存写入）---")
response1 = client.chat.completions.create(
    model="deepseek-v4-pro",
    messages=messages,
)
print(f"回答: {response1.choices[0].message.content[:100]}...")
print(f"Usage: prompt_tokens={response1.usage.prompt_tokens}, "
      f"completion_tokens={response1.usage.completion_tokens}, "
      f"total_tokens={response1.usage.total_tokens}")
if hasattr(response1.usage, 'prompt_tokens_details') and response1.usage.prompt_tokens_details:
    print(f"  cached_tokens: {response1.usage.prompt_tokens_details.cached_tokens}")
else:
    print(f"  cached_tokens: 不支持/未命中（此模型可能不返回该字段）")

time.sleep(2)

# 第二次请求：追加新消息，前面的 system prompt 应命中缓存
messages.append({"role": "assistant", "content": response1.choices[0].message.content})
messages.append({"role": "user", "content": "那工厂模式呢？"})

print("\n--- 第二次请求（期望缓存命中）---")
response2 = client.chat.completions.create(
    model="deepseek-v4-pro",
    messages=messages,
)
print(f"回答: {response2.choices[0].message.content[:100]}...")
print(f"Usage: prompt_tokens={response2.usage.prompt_tokens}, "
      f"completion_tokens={response2.usage.completion_tokens}, "
      f"total_tokens={response2.usage.total_tokens}")
if hasattr(response2.usage, 'prompt_tokens_details') and response2.usage.prompt_tokens_details:
    print(f"  cached_tokens: {response2.usage.prompt_tokens_details.cached_tokens}")
else:
    print(f"  cached_tokens: 不支持/未命中（此模型可能不返回该字段）")

# ============================================================
# 方式二：Anthropic Claude 风格（手动 cache_control）
# Claude 需要在 content 块中显式标记 cache_control: {"type": "ephemeral"}
# 通过 One-API 转发时，需要在消息的 content 中传入 cache_control
# ============================================================

print("\n" + "=" * 60)
print("Anthropic Claude 风格：手动 cache_control 演示")
print("=" * 60)

# Claude 的 cache_control 是加在 content 块上的
# 通过 OpenAI 兼容接口转发时，需要在 content 中使用结构化格式
claude_messages = [
    {
        "role": "system",
        "content": long_system_prompt
        # 注意：OpenAI 兼容接口中 system 的 content 是字符串，
        # 而 Claude 原生接口中 system 是 content 块数组，可加 cache_control
    },
    {
        "role": "user",
        "content": "解释一下微服务架构的核心组件"
    }
]

# 如果是直接调 Claude 原生 API（非 One-API 转发），格式如下：
# client.messages.create(
#     model="claude-sonnet-4-20250514",
#     system=[
#         {
#             "type": "text",
#             "text": long_system_prompt,
#             "cache_control": {"type": "ephemeral"}  # ← 标记缓存断点
#         }
#     ],
#     messages=[{"role": "user", "content": "..."}]
# )
# 响应中通过 usage.cache_read_input_tokens 和 cache_creation_input_tokens 判断

print("""
Claude 原生 API 的 cache_control 用法（伪代码）：

  请求：
    system=[
        {
            "type": "text",
            "text": "很长的系统提示词...",
            "cache_control": {"type": "ephemeral"}   ← 标记此处为缓存断点
        }
    ],
    messages=[
        {"role": "user", "content": [
            {"type": "text", "text": "大量上下文文档...",
             "cache_control": {"type": "ephemeral"}}, ← 可设多个断点
            {"type": "text", "text": "用户问题"}       ← 断点后的内容不缓存
        ]}
    ]

  响应判断：
    usage.cache_creation_input_tokens = 5000   ← 首次：创建缓存消耗的 tokens
    usage.cache_read_input_tokens = 0          ← 首次：读取缓存为 0

    usage.cache_creation_input_tokens = 0      ← 再次：无新缓存创建
    usage.cache_read_input_tokens = 5000       ← 再次：命中缓存！节省 90% 输入费用

  关键规则：
    1. cache_control 只有一种类型：{"type": "ephemeral"}（临时缓存）
    2. 缓存有效期：5分钟（每次命中自动续期）或 1小时
    3. 最小缓存长度：1024 tokens（Sonnet）/ 2048 tokens（Haiku）
    4. 严格前缀匹配：从第一个 token 开始逐个比对
    5. 稳定内容放前面：system > 工具定义 > 上下文 > 历史对话 > 用户消息
""")

# ============================================================
# 总结对比
# ============================================================

print("=" * 60)
print("OpenAI vs Claude 缓存对比")
print("=" * 60)
print("""
┌──────────────┬──────────────────────┬──────────────────────────┐
│     维度      │       OpenAI         │      Anthropic Claude    │
├──────────────┼──────────────────────┼──────────────────────────┤
│ 激活方式      │ 自动（≥1024 tokens） │ 手动 cache_control 标记  │
│ 标记方式      │ 无需任何参数          │ cache_control: ephemeral │
│ 缓存有效期    │ 不透明，自动管理      │ 5分钟 或 1小时（可选）   │
│ 命中判断      │ cached_tokens 字段   │ cache_read_input_tokens  │
│ 多断点        │ 不支持               │ 支持                     │
│ 最小长度      │ 1024 tokens          │ 1024/2048 tokens         │
│ 读取费用      │ 原价 50%             │ 原价 10%（更划算）       │
└──────────────┴──────────────────────┴──────────────────────────┘
""")
