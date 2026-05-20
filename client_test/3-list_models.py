from openai import OpenAI

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

# 查询可用模型列表
models = client.models.list()

print(models)

print("可用模型列表：")
print("-" * 50)
for model in models.data:
    print(f"  {model.id}")
print(f"\n共 {len(models.data)} 个模型")
