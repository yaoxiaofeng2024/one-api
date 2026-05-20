文件	接口	说明
1-call_demo.py	POST /v1/chat/completions	对话补全（已有）
2-call_stream.py	POST /v1/chat/completions	流式对话（已有）
3-list_models.py	GET /v1/models	模型列表（已有）
4-completions.py	POST /v1/completions	文本补全（旧版）
5-edits.py	POST /v1/edits	文本编辑（已弃用，用 httpx 直接调用）
6-images_generations.py	POST /v1/images/generations	DALL·E 图像生成
7-embeddings.py	POST /v1/embeddings	文本向量化 + 语义相似度计算
8-audio_transcriptions.py	POST /v1/audio/transcriptions	语音转文字（Whisper）
9-audio_translations.py	POST /v1/audio/translations	语音翻译为英文
10-audio_speech.py	POST /v1/audio/speech	文字转语音（TTS）
11-moderations.py	POST /v1/moderations	内容审核
12-oneapi_proxy.py	ANY /v1/oneapi/proxy/:id/*	自定义代理（one-api 特有，指定渠道直连）
13-legacy_embeddings.py	POST /v1/engines/:model/embeddings	旧版嵌入接口
14-retrieve_model.py	GET /v1/models/:model	查询单个模型详情
