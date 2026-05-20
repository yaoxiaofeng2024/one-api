"""
文字转语音接口（Audio Speech / TTS）
对应路由：POST /v1/audio/speech
文档：https://platform.openai.com/docs/api-reference/audio/createSpeech

给定文本，生成语音音频。返回的是音频二进制流，需保存为文件。
"""
from openai import OpenAI

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

# 基本用法：将文本转为语音并保存为文件
response = client.audio.speech.create(
    model="tts-1",                  # TTS 模型：tts-1（速度快）或 tts-1-hd（质量高）
    voice="alloy",                  # 语音风格：alloy, echo, fable, onyx, nova, shimmer
    input="你好，欢迎使用文字转语音功能！这是一段测试语音。",
    # 以下为可选参数：
    # response_format="mp3",       # 输出格式：mp3, opus, aac, flac, wav, pcm
    # speed=1.0,                   # 语速：0.25 ~ 4.0，默认 1.0
)

# 将音频流保存到文件
response.stream_to_file("output_speech.mp3")
print("语音已保存到 output_speech.mp3")

# 也可以直接获取二进制数据
response2 = client.audio.speech.create(
    model="tts-1-hd",
    voice="nova",
    input="This is a high quality text-to-speech demo.",
)
with open("output_speech_hd.mp3", "wb") as f:
    f.write(response2.content)
print("HD 语音已保存到 output_speech_hd.mp3")
