"""
图像生成接口（Images Generations）
对应路由：POST /v1/images/generations
文档：https://platform.openai.com/docs/api-reference/images/create

根据文本描述生成图片（DALL·E 模型），返回图片 URL 或 base64 编码。
"""
from openai import OpenAI

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

# 基本用法：生成一张图片
response = client.images.generate(
    model="dall-e-3",            # 图像生成模型：dall-e-2 或 dall-e-3
    prompt="一只戴着墨镜的猫在海滩上晒太阳，卡通风格",
    n=1,                          # 生成图片数量（dall-e-3 只支持 n=1）
    size="1024x1024",             # 图片尺寸：256x256, 512x512, 1024x1024, 1024x1792, 1792x1024
    quality="standard",           # 图片质量：standard 或 hd（仅 dall-e-3）
    # response_format="url"       # 返回格式：url 或 b64_json
)

print("生成结果：")
for i, image in enumerate(response.data):
    print(f"  图片 {i+1}: {image.url}")
    # 如果使用 b64_json 格式，可以用 image.b64_json 获取 base64 数据
