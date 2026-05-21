import anthropic

client = anthropic.Anthropic()

message = client.messages.create(
    #model="deepseek-v4-pro",
    model="deepseek-v4-flash",
    max_tokens=1000,
    system="You are a helpful assistant.",
    messages=[
        {
            "role": "user",
            "content": [
                {
                    "type": "text",
                    "text": "Hi, how are you?"
                }
            ]
        }
    ]
)
print(message.content)

"""
linux：
export ANTHROPIC_BASE_URL=https://api.deepseek.com/anthropic
export ANTHROPIC_API_KEY="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2"

win：
$env:ANTHROPIC_BASE_URL = "https://api.deepseek.com/anthropic"
$env:ANTHROPIC_API_KEY = "sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2"
"""
