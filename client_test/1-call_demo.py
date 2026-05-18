from openai import OpenAI

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

response = client.chat.completions.create(
    # model="deepseek-v4-pro",
    model="glm-4-7-251222",
    messages=[
        {"role": "user", "content": "你好，请介绍一下你自己"}
    ]
)

print(response.choices[0].message.content)




