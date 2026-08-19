"""Localization helpers for the StarFire desktop client."""

import locale
import platform


DEFAULT_LANGUAGE = "zh_CN"
SUPPORTED_LANGUAGES = ("zh_CN", "en_US")

LANGUAGE_NAMES = {
    "zh_CN": "简体中文",
    "en_US": "English",
}

TRANSLATIONS = {
    "zh_CN": {
        "app.title": "StarFire MaaS 算力分享APP",
        "common.error": "错误",
        "common.info": "提示",
        "common.success": "成功",
        "common.warning": "警告",
        "language.label": "语言:",
    },
    "en_US": {
        "app.title": "StarFire MaaS Compute Sharing",
        "common.error": "Error",
        "common.info": "Notice",
        "common.success": "Success",
        "common.warning": "Warning",
        "language.label": "Language:",
    },
}

ENGLISH_SOURCE_TRANSLATIONS = {
    "StarFire MaaS 算力分享APP": "StarFire MaaS Compute Sharing",
    "算力分享应用": "Compute Sharing",
    "正在启动...": "Starting...",
    "正在初始化...": "Initializing...",
    "正在加载组件...": "Loading components...",
    "准备就绪...": "Ready...",
    "🔌 模型接入方式": "🔌 Model Provider",
    "Ollama (本地)": "Ollama (Local)",
    "llama.cpp (开发中)": "llama.cpp (Coming Soon)",
    "代理模式": "Proxy",
    "并发请求数:": "Parallel requests:",
    "(空值=自动，推荐4或1)": "(blank=auto; 4 or 1 recommended)",
    "💡 每个模型同时处理的最大并行请求数。默认根据可用内存自动选择4或1": "💡 Maximum concurrent requests per model. The default is 4 or 1 based on available memory.",
    "正在检查模型服务状态...": "Checking model service...",
    "📦 已安装的模型": "📦 Available Models",
    "分类": "Category",
    "模型名称": "Model",
    "大小": "Size",
    "修改时间": "Modified",
    "状态:": "Status:",
    " ● 运行中 ": " ● Running ",
    " ○ 未运行 ": " ○ Stopped ",
    "🔄 刷新": "🔄 Refresh",
    "▶️ 运行": "▶️ Run",
    "⏹️ 停止": "⏹️ Stop",
    "🔔 测试通知": "🔔 Test Notification",
    "📋 运行日志": "📋 Runtime Log",
    "🌟 Starfire 算力注册": "🌟 StarFire Registration",
    "⚙️ 配置参数": "⚙️ Configuration",
    "服务器地址:": "Server URL:",
    "用户名:": "Username:",
    "密码:": "Password:",
    "登录状态:": "Login status:",
    " ○ 未登录 ": " ○ Signed out ",
    "总收益:": "Total earnings:",
    "最新收益:": "Latest earnings:",
    "⚙ 模型配置": "⚙ Model Configuration",
    "选择注册模型并配置价格": "Select models and configure pricing",
    "🔐 登录": "🔐 Sign In",
    "💾 保存配置": "💾 Save",
    "💰 刷新收益": "💰 Refresh Earnings",
    "🎮 算力控制": "🎮 Compute Control",
    " ● 未运行 ": " ● Stopped ",
    " ○ TCP未启动 ": " ○ TCP stopped ",
    " ● TCP运行中 ": " ● TCP running ",
    "💡 TCP服务器地址: 127.0.0.1:19527 (自动启动)": "💡 TCP server: 127.0.0.1:19527 (starts automatically)",
    "▶️ 启动算力注册": "▶️ Start Registration",
    "⏹️ 停止算力注册": "⏹️ Stop Registration",
    "📊 Starfire 日志": "📊 StarFire Log",
    "💡 提示: 需要 starfire.exe 与本程序在同一目录": "💡 starfire.exe must be in the same folder as this application",
    "显示窗口": "Show Window",
    "退出程序": "Quit",
    "StarFire MaaS 算力分享": "StarFire MaaS",
    "语言:": "Language:",
    "✓ 代理模式 - 请配置 Base URL 和 API Key": "✓ Proxy mode - configure Base URL and API Key",
    "🔐 安全验证": "🔐 Security Check",
    "换一题": "New Question",
    " ● 已登录 ": " ● Signed in ",
    "✓ 登录": "✓ Sign In",
    "✗ 取消": "✗ Cancel",
    "🔄 刷新模型列表": "🔄 Refresh Models",
    "💾 保存所有价格": "💾 Save All Prices",
    "🔄 刷新模型": "🔄 Refresh Models",
    "💾 保存模型配置": "💾 Save Configuration",
    "注册选中": "Register Selected",
    "取消注册选中": "Unregister Selected",
    "全部注册": "Register All",
    "全部取消": "Clear All",
    "保存": "Save",
    "🔍 搜索:": "🔍 Search:",
    "仅显示已注册": "Registered only",
    "引擎": "Engine",
    "输入价格(¥/M)": "Input Price (¥/M)",
    "输出价格(¥/M)": "Output Price (¥/M)",
    "缓存输入价格(¥/M)": "Cached Input (¥/M)",
    "✗ 未检测到 Ollama": "✗ Ollama not detected",
    "名称:": "Name:",
    "启用": "Enabled",
    "新增/更新": "Add / Update",
    "删除": "Delete",
    "启用选中": "Enable Selected",
    "停用选中": "Disable Selected",
    "名称": "Name",
    "状态": "Status",
}


def normalize_language(language):
    """Map a locale identifier to one of the supported languages."""
    if not language:
        return None

    normalized = str(language).replace("-", "_").lower()
    if normalized.startswith("zh"):
        return "zh_CN"
    if normalized.startswith("en"):
        return "en_US"
    return None


def detect_system_language():
    """Detect the system UI language without changing the process locale."""
    candidates = []

    if platform.system() == "Windows":
        try:
            import ctypes

            buffer = ctypes.create_unicode_buffer(85)
            if ctypes.windll.kernel32.GetUserDefaultLocaleName(buffer, len(buffer)):
                candidates.append(buffer.value)
        except (AttributeError, OSError):
            pass

    try:
        candidates.append(locale.getlocale()[0])
    except (ValueError, TypeError):
        pass

    for candidate in candidates:
        language = normalize_language(candidate)
        if language:
            return language

    return DEFAULT_LANGUAGE


class I18n:
    """Translate stable keys with Chinese fallback for incomplete locales."""

    def __init__(self, language=None):
        self.language = normalize_language(language) or detect_system_language()

    def set_language(self, language):
        normalized = normalize_language(language)
        if normalized is None:
            raise ValueError(f"Unsupported language: {language}")
        self.language = normalized

    def translate(self, key, **kwargs):
        text = TRANSLATIONS.get(self.language, {}).get(key)
        if text is None:
            text = TRANSLATIONS[DEFAULT_LANGUAGE].get(key, key)
        return text.format(**kwargs) if kwargs else text

    def translate_source(self, source_text):
        if self.language == "en_US":
            return ENGLISH_SOURCE_TRANSLATIONS.get(source_text, source_text)
        return source_text

    def select(self, chinese, english, **kwargs):
        template = english if self.language == "en_US" else chinese
        return template.format(**kwargs) if kwargs else template

    __call__ = translate


def refresh_widget_texts(widget, translator):
    """Refresh text-bearing Tk widgets in place while preserving their state."""
    try:
        current_text = widget.cget("text")
        source_text = getattr(widget, "_i18n_source_text", None)
        rendered_text = getattr(widget, "_i18n_rendered_text", None)
        if source_text is None or (rendered_text is not None and current_text != rendered_text):
            source_text = current_text
            widget._i18n_source_text = source_text
        translated_text = translator.translate_source(source_text)
        widget.configure(text=translated_text)
        widget._i18n_rendered_text = translated_text
    except Exception:
        pass

    try:
        children = widget.winfo_children()
    except Exception:
        children = ()

    for child in children:
        refresh_widget_texts(child, translator)