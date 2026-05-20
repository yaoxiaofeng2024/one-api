"""
语音转文字接口（Audio Transcriptions）
对应路由：POST /v1/audio/transcriptions
文档：https://platform.openai.com/docs/api-reference/audio/createTranscription

上传音频文件，模型（Whisper）将其转录为文本。
支持 mp3、mp4、mpeg、mpga、m4a、wav、webm 等格式。
"""
from openai import OpenAI

client = OpenAI(
    api_key="sk-CKHWDcpDc17l0kqF53C91f1fE907493aA13a01Ee55A6F1C2",
    base_url="http://localhost:3000/v1"
)

# 基本用法：上传音频文件进行转录
# 请确保本地有对应的音频文件
audio_file = open("test_audio.mp3", "rb")  # 替换为实际的音频文件路径

response = client.audio.transcriptions.create(
    model="whisper-1",           # 语音识别模型
    file=audio_file,             # 音频文件对象
    # 以下为可选参数：
    # language="zh",             # 指定音频语言（ISO 639-1 格式），可提高准确率
    # response_format="json",    # 输出格式：json, text, srt, verbose_json, vtt
    # temperature=0.0,           # 采样温度
)

print("转录结果：")
print(response.text)

# verbose_json 格式可返回更多详情
audio_file2 = open("test_audio.mp3", "rb")
response2 = client.audio.transcriptions.create(
    model="whisper-1",
    file=audio_file2,
    response_format="verbose_json",
    timestamp_granularities=["word", "segment"]  # 返回词级和段落级时间戳
)
print(f"\n详细结果：语言={response2.language}, 时长={response2.duration}s")
