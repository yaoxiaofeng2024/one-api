"""
语音翻译接口（Audio Translations）
对应路由：POST /v1/audio/translations
文档：https://platform.openai.com/docs/api-reference/audio/createTranslation

上传任意语言的音频文件，模型将其翻译为英文文本并输出。
与 transcriptions 的区别：transcriptions 是"原语言转录"，translations 是"翻译为英文"。
"""
from openai import OpenAI

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

# 基本用法：上传音频文件，翻译为英文
audio_file = open("test_audio.mp3", "rb")  # 替换为实际的音频文件路径

response = client.audio.translations.create(
    model="whisper-1",           # 语音翻译模型
    file=audio_file,             # 音频文件（任意语言）
    # 以下为可选参数：
    # response_format="json",    # 输出格式：json, text, srt, verbose_json, vtt
    # temperature=0.0,           # 采样温度
)

print("翻译结果（英文）：")
print(response.text)

# 注意：translations 不支持指定 language 参数，
# 模型会自动检测源语言并翻译为英文
