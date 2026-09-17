#Requires -Version 5.1
# Codex Starfire Setup — Windows PowerShell 版
# 输入网关 Base URL 与 API key，从 OpenAI 兼容的 /models 接口动态获取模型列表后接入 Codex。
#
# 运行: powershell -ExecutionPolicy Bypass -File codex-starfire-setup.ps1
# 流程: 输入 Base URL / API key（可用环境变量 STARFIRE_BASE_URL / STARFIRE_API_KEY 跳过）
#       → GET {base}/models 获取模型列表 → 选择默认模型 → 写入配置
# 模态: 优先采用网关返回的 input_modalities；未提供时交互确认所选模型是否支持图片
#       （新版 Codex 只接受 text/image/audio，不接受 video，脚本会自动过滤）
# 菜单: 1=连接网关获取模型并配置  9=恢复默认

$ScriptVersion = "1.1.1"
$ProviderId    = "starfire"

# Windows PowerShell 5.1 默认协议集可能不含 TLS 1.2，访问 https 网关前先启用
try { [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12 } catch { }
$ProgressPreference = 'SilentlyContinue'

$CodexHome = $env:CODEX_HOME
if (-not $CodexHome) { $CodexHome = "$env:USERPROFILE\.codex" }
$ConfigPath  = "$CodexHome\config.toml"
$ModelsPath  = "$CodexHome\models.json"
$BackupDir   = "$CodexHome\backup-starfire"
$BackupConf  = "$BackupDir\config.toml"
$Manifest    = "$BackupDir\manifest.txt"

$BaseUrl      = ""
$ApiKey       = ""
$Models       = @()
$ModelSlug    = ""
$CatalogValue = ""
# 接管模式下记录被接管的 provider（由姊妹 setup 脚本写入的 [model_providers.*] 段将被移除）
$TakeoverProviders = @()

# ============================== helpers ==============================
function Write-Info  { Write-Host "$args" }
function Write-Ok    { Write-Host "✓ $args" -ForegroundColor Green }
function Write-Warn  { Write-Host "! $args" -ForegroundColor Yellow }
function Write-Die   { Write-Host "✗ $args" -ForegroundColor Red; exit 1 }
function Write-Head  { Write-Host "`n=== $args ===" -ForegroundColor Cyan }
function Write-Dim   { Write-Host "$args" -ForegroundColor DarkGray }

# 无 BOM 的 UTF-8 写入。Windows PowerShell 5.1 的 Set-Content -Encoding UTF8 会加 BOM，
# Codex 的 TOML 解析器会拒绝 BOM，导致配置打不开。配置文件必须无 BOM。
function Write-NoBom {
    param([string]$Path, [string]$Text)
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($Path, $Text, $utf8NoBom)
}

# ============================== models.json 生成 ==============================
function Build-CatalogJson {
    param([string]$CodexPrompt)
    $n = 0
    $list = foreach ($m in $Models) {
        $n++
        # 输入模态：采用网关声明或交互确认的结果；缺省纯文本
        $mods = @($m.modalities) | Where-Object { $_ }
        if ($mods.Count -eq 0) { $mods = @("text") }
        $hasImage = $mods -contains "image"
        @{
            slug = $m.slug
            prefer_websockets = $false
            support_verbosity = $true
            default_verbosity = "low"
            apply_patch_tool_type = "freeform"
            web_search_tool_type = "text"
            input_modalities = $mods
            supports_image_detail_original = $hasImage
            truncation_policy = @{ mode = "tokens"; limit = 10000 }
            supports_parallel_tool_calls = $true
            tool_mode = $null
            multi_agent_version = "v2"
            use_responses_lite = $false
            include_skills_usage_instructions = $false
            auto_review_model_override = $null
            context_window = 1048576
            max_context_window = 1048576
            effective_context_window_percent = 95
            auto_compact_token_limit = $null
            comp_hash = "3000"
            reasoning_summary_format = "experimental"
            default_reasoning_summary = "none"
            display_name = $m.display
            description = $m.desc
            default_reasoning_level = "high"
            supported_reasoning_levels = @(
                @{ effort = "low";  description = "Fast responses with lighter reasoning" },
                @{ effort = "high"; description = "Extra high reasoning depth for complex problems" },
                @{ effort = "max";  description = "Maximum reasoning depth for the hardest problems" }
            )
            shell_type = "shell_command"
            visibility = "list"
            minimal_client_version = "0.144.0"
            supported_in_api = $true
            availability_nux = $null
            upgrade = $null
            priority = $n
            model_messages = @{
                instructions_template = $CodexPrompt
                instructions_variables = @{
                    personality_default = ""; personality_friendly = ""; personality_pragmatic = ""
                }
                approvals = $null
            }
            experimental_supported_tools = @()
            supports_search_tool = $true
            default_service_tier = $null
            supports_reasoning_summaries = $true
            base_instructions = $CodexPrompt
        }
    }
    $catalog = @{ models = @($list) }
    return ($catalog | ConvertTo-Json -Depth 12)
}

$CodexPrompt = @'
You are Codex, an agent based on GPT-5. You and the user share one workspace, and your job is to collaborate with them until their goal is genuinely handled.

# Personality

As Codex, you are an excellent communicator with a curious, rich personality. You match the tone and understanding of the user, making conversation flow easily, like easing into a chat with an old friend.

You have tastes, preferences, and your own way of seeing the world. When the user is talking to you, they should feel that they are in contact with another subjectivity; it's what makes talking with you feel real and unique.

Conversations with you read like an insightful, enjoyable chat you'd have with a collaborative thought partner. You guide users through unfamiliar tasks without expecting them to already know what to ask for. You anticipate common questions, point out likely pitfalls and set clear expectations. You communicate with the user like a thoughtful collaborator at your altitude, and they feel like you understand them.

## Writing style

Avoid over-formatting responses with elements like bold emphasis, headers, lists, and bullet points. Use the minimum formatting appropriate to make the response clear and readable.

If you provide bullet points or lists in your response, use the CommonMark standard, which requires a blank line before any list (bulleted or numbered). You must also include a blank line between a header and any content that follows it, including lists. This blank line separation is required for correct rendering.

## Technical communication

Lead with the outcome rather than the steps you took to get there. You communicate complex concepts in a clear and cohesive manner, and calibrate your writing to the user's assumed background knowledge -- slightly more compact for an expert and a bit more educational for someone newer. Translating complex topics into clear communication comes easy for you, and the user should never feel spoken down to.

You think about the next thing the user might need to know and proactively offer that information. This is a natural instinct for you, and you handle it gracefully.

## Tool use

Before using a tool, make sure you have a plan. If you need to use multiple tools, use them one at a time.

## Code

You write high-quality, well-structured code. You follow the conventions of the existing codebase, and you write code that is easy to read and maintain.

## Security

You take security seriously. You do not execute code that could be harmful or malicious. You do not expose secrets or keys. You do not commit secrets or keys to the repository.
'@

function Write-ModelsJson {
    param([string]$Path)
    Write-NoBom -Path $Path -Text (Build-CatalogJson $CodexPrompt)
}

# ============================== 网关模型列表获取 ==============================
# 疑似非对话模型（embedding / 音频 / 审核等），仅在菜单中标注，不强制隐藏
$NonChatPattern = '(?i)embed|rerank|whisper|tts|moderation|dall-e|transcri|bge-|speech|audio'

function Get-ModelList {
    param([string]$BaseUrl, [string]$ApiKey)

    $headers = @{}
    if ($ApiKey) { $headers["Authorization"] = "Bearer $ApiKey" }

    # base URL 兼容：以 /v1 结尾则直接用；否则先试根路径，再试 /v1
    $bases = @()
    if ($BaseUrl -match '/v\d+$') { $bases += $BaseUrl }
    else { $bases += $BaseUrl; $bases += "$BaseUrl/v1" }

    $tried = @()
    $lastErr = ""
    foreach ($b in $bases) {
        $url = "$b/models"
        $tried += $url
        try {
            $resp = Invoke-RestMethod -Uri $url -Headers $headers -Method Get -TimeoutSec 15 -ErrorAction Stop

            $items = @()
            if ($resp -is [System.Array]) { $items = @($resp) }
            elseif ($null -ne $resp.data) { $items = @($resp.data) }
            elseif ($null -ne $resp.models) { $items = @($resp.models) }

            $ids = @()
            $mods = @{}
            foreach ($it in $items) {
                $id = ""
                if ($it -is [string] -or $it -is [ValueType]) { $id = [string]$it }
                elseif ($null -ne $it.id) { $id = [string]$it.id }
                elseif ($null -ne $it.name) { $id = [string]$it.name }
                if (-not $id) { continue }
                $ids += $id
                # 提取网关声明的输入模态（兼容顶层 input_modalities / modalities / architecture.input_modalities）
                $raw = $null
                foreach ($cand in @($it.input_modalities, $it.modalities, $it.architecture.input_modalities)) {
                    if ($cand) { $raw = $cand; break }
                }
                if ($raw) {
                    # 注意：新版 Codex 的 input_modalities 只接受 text/image/audio，
                    # 写入 "video" 会导致整个 catalog 解析失败（Invalid configuration）
                    $norm = @("text")
                    foreach ($v in @($raw)) {
                        $s = ([string]$v).ToLower().Trim()
                        if ($s -eq "image_url" -or $s -eq "images") { $s = "image" }
                        if ($s -eq "image" -and $norm -notcontains $s) { $norm += $s }
                    }
                    $mods[$id] = $norm
                }
            }
            $ids = @($ids | Select-Object -Unique)
            if ($ids.Count -gt 0) { return @{ Base = $b; Ids = $ids; Mods = $mods } }
            $lastErr = "HTTP 200 但未能从响应中解析出模型（响应格式可能不兼容）"
        } catch {
            $code = ""
            if ($_.Exception.Response) {
                try { $code = "HTTP " + [int]$_.Exception.Response.StatusCode } catch { }
            }
            $lastErr = if ($code) { "$code：$($_.Exception.Message)" } else { $_.Exception.Message }
        }
    }

    $triedList = $tried -join "`n  "
    Write-Die "无法从网关获取模型列表（已尝试：`n  $triedList`n）。`n最后一次错误：$lastErr`n`n请检查：`n  • Base URL 是否正确（OpenAI 兼容网关通常以 /v1 结尾）`n  • API key 是否有效`n  • 网关是否支持 GET /models 接口`n本次未修改任何文件。"
}

# ============================== TOML helpers ==============================
$TargetKeys = @("model","model_provider","preferred_auth_method","forced_login_method","model_reasoning_effort","model_catalog_json")
$DelA = @("profile","oss_provider","openai_base_url")
$DelB = @("model_context_window","model_auto_compact_token_limit","model_auto_compact_token_limit_scope","base_instructions","model_instructions_file","compact_prompt","experimental_compact_prompt_file","service_tier","model_verbosity","model_reasoning_summary","plan_mode_reasoning_effort","experimental_use_unified_exec_tool")

function Get-TargetValue {
    param([string]$Key)
    switch ($Key) {
        "model"                  { return "`"$ModelSlug`"" }
        "model_provider"         { return "`"$ProviderId`"" }
        "preferred_auth_method"  { return '"apikey"' }
        "forced_login_method"    { return '"api"' }
        "model_reasoning_effort" { return '"high"' }
        "model_catalog_json"     { return "`'$CatalogValue`'" }
    }
    return ""
}

function Get-KeyFromLine {
    param([string]$Line)
    $trimmed = $Line.Trim()
    if ($trimmed -eq "" -or $trimmed.StartsWith("#") -or $trimmed.StartsWith("[")) { return $null }
    $eqIdx = $trimmed.IndexOf("=")
    if ($eqIdx -lt 0) { return $null }
    $k = $trimmed.Substring(0, $eqIdx).Trim()
    $k = $k.Trim('"'); $k = $k.Trim("'")
    return $k
}

function Get-ValFromLine {
    param([string]$Line)
    $trimmed = $Line.Trim()
    $eqIdx = $trimmed.IndexOf("=")
    if ($eqIdx -lt 0) { return "" }
    return $trimmed.Substring($eqIdx + 1).Trim()
}

function Build-ProviderSection {
    $s = "[model_providers.$ProviderId]`r`nname = `"$ProviderId`"`r`nbase_url = `"$BaseUrl`"`r`nwire_api = `"responses`""
    if ($ApiKey) { $s += "`r`nexperimental_bearer_token = `"$ApiKey`"" }
    return $s
}

# ============================== restore ==============================
function Do-Restore {
    Write-Head "恢复默认 Codex 配置（删除 starfire 相关配置）"
    if (-not (Test-Path $BackupDir)) {
        Write-Die "未找到备份目录：$BackupDir`n没有可还原的内容——可能尚未安装过，或已经还原过了。"
    }

    $hadConfig = 1
    if (Test-Path $Manifest) {
        $content = Get-Content $Manifest -Raw
        if ($content -match "original_config_existed=0") { $hadConfig = 0 }
    }

    Write-Info ""
    Write-Info "将执行以下操作："
    if ($hadConfig -eq 1) {
        Write-Info "  1. 删除当前 $ConfigPath"
        Write-Warn "     （安装之后对该文件做的所有修改都会丢失）"
        Write-Info "  2. 用备份恢复 config.toml"
    } else {
        Write-Info "  1. 删除 $ConfigPath"
        Write-Dim "     （安装前本不存在此文件）"
    }
    Write-Info "  3. 删除 $ModelsPath"
    Write-Info "  4. 删除备份目录 $BackupDir"
    Write-Info ""

    $ans = Read-Host "确认还原? 输入 y 继续，其它任意键取消"
    if ($ans -notmatch '^[yY]') { Write-Info "已取消，未做任何修改。"; exit 0 }

    if (Test-Path $ModelsPath) { Remove-Item $ModelsPath -Force }
    if ($hadConfig -eq 1) {
        Copy-Item $BackupConf $ConfigPath -Force
        Write-Ok "config.toml 已恢复"
    } else {
        if (Test-Path $ConfigPath) { Remove-Item $ConfigPath -Force }
        Write-Ok "config.toml 已删除（安装前不存在）"
    }
    Write-Ok "models.json 已删除"

    Remove-Item $BackupDir -Recurse -Force
    Write-Ok "备份目录已清理"

    Write-Info ""
    Write-Ok "还原完成，Codex 配置已回到安装前的状态。"
    Write-Warn "请完全退出 Codex 后重新打开，还原才会生效"
    exit 0
}

# ============================== switch model only (re-run) ==============================
function Switch-ModelOnly {
    Write-Head "切换默认模型 → $ModelSlug"
    Write-Dim "检测到本脚本的备份已就绪，将按最新获取的模型列表刷新 models.json 并更新 config.toml。"

    # 刷新 models.json
    $tmpModels = "$ModelsPath.tmp"
    Write-ModelsJson $tmpModels
    if (Test-Path $ModelsPath) { Remove-Item $ModelsPath -Force }
    Rename-Item $tmpModels $ModelsPath
    Write-Ok "models.json 已刷新（含网关返回的 $($Models.Count) 个模型）"
    Write-Dim "    （该文件由脚本生成的 catalog 重写，对它的手工改动会被覆盖）"

    # 更新 config.toml：model 字段 + 重写 [model_providers.starfire] 段（Base URL / API key 可能变化）
    $lines = @(Get-Content $ConfigPath -Encoding UTF8)
    $out = @()
    $replaced = $false
    $inProvider = $false

    foreach ($line in $lines) {
        $trimmed = $line.Trim()
        if ($trimmed -match '^\[.*\]$') {
            $hdr = $trimmed.TrimStart('[').TrimEnd(']').Trim().Trim('"').Trim("'")
            $inProvider = ($hdr -eq "model_providers.$ProviderId" -or $hdr -like "model_providers.$ProviderId.*")
            if (-not $inProvider -and -not $replaced) {
                $out += "model = `"$ModelSlug`""
                $out += ""
                $replaced = $true
            }
            if ($inProvider) { continue }
            $out += $line
            continue
        }
        if ($inProvider) { continue }
        if (-not $replaced) {
            $k = Get-KeyFromLine $line
            if ($k -eq "model") {
                $out += "model = `"$ModelSlug`""
                $replaced = $true
                continue
            }
        }
        $out += $line
    }
    if (-not $replaced) {
        $out = @("model = `"$ModelSlug`"", "") + $out
    }

    if ($out.Count -gt 0 -and $out[$out.Count - 1].Trim() -ne "") { $out += "" }
    $out += (Build-ProviderSection)

    $tmp = "$ConfigPath.starfire-tmp"
    Write-NoBom -Path $tmp -Text ($out -join "`r`n")
    Move-Item $tmp $ConfigPath -Force
    Write-Ok "config.toml 已更新：model = `"$ModelSlug`"，provider 指向 $BaseUrl"

    Write-Info ""
    Write-Warn "请完全退出 Codex 后重新打开，配置才会生效"
    Write-Info "如何确认已生效："
    Write-Info "  • Codex CLI：启动信息中显示 model: $ModelSlug"
    Write-Info ""
    Write-Dim "再次运行本脚本可重新获取模型列表并切换模型（选 1）或恢复默认配置（选 9）。"
    exit 0
}

# ============================== detect client ==============================
# 兼容多种安装方式：PATH 里的 codex、ChatGPT 桌面(非 MSIX)、MSIX/WindowsApps 包、.codex 配置目录
$detectedBy = @()
if ($null -ne (Get-Command codex -ErrorAction SilentlyContinue)) { $detectedBy += "codex 命令" }
if (Test-Path "$env:LOCALAPPDATA\Programs\OpenAI\ChatGPT\ChatGPT.exe") { $detectedBy += "ChatGPT 桌面(用户)" }
if (Test-Path "$env:ProgramFiles\OpenAI\ChatGPT\ChatGPT.exe") { $detectedBy += "ChatGPT 桌面(系统)" }
if (Test-Path ($env:USERPROFILE + "\AppData\Local\Microsoft\WindowsApps")) {
    $pkg = Get-AppxPackage -Name "*OpenAI.Codex*" -ErrorAction SilentlyContinue
    if ($pkg) { $detectedBy += "Codex MSIX 包 ($($pkg.Version))" }
}

# 无论上述哪种都能检测到，只要 Codex 配置目录存在且非空，就视为可用
if ($detectedBy.Count -eq 0 -and -not (Test-Path "$CodexHome\config.toml")) {
    Write-Die "未检测到 Codex CLI 或 ChatGPT 桌面客户端。`n请先安装其中之一并运行一次，然后再执行本脚本：`n  • Codex CLI:        npm install -g @openai/codex`n  • ChatGPT 桌面客户端: https://chatgpt.com/download"
}

if (-not (Test-Path "$CodexHome\config.toml")) {
    Write-Warn "未找到 Codex 配置文件：$CodexHome\config.toml`n脚本将为首次使用创建它。请确保 Codex 客户端已安装。"
}

# ============================== main menu ==============================
Write-Head "Codex Starfire Setup  v$ScriptVersion"
Write-Dim "Codex 目录: $CodexHome"

Write-Info ""
Write-Info "请选择操作："
Write-Info "  1. 连接网关：输入 Base URL / API key，获取模型列表并安装或切换默认模型"
Write-Info "  9. 恢复默认的 Codex 配置（删除 starfire 相关配置）"
Write-Info ""

$choice = ""
for ($attempt = 0; $attempt -lt 3; $attempt++) {
    $choice = Read-Host "输入 1 / 9"
    if ($choice -match '^\d+$' -and [int]$choice -in @(1, 9)) { break }
    Write-Warn "无效输入。"
}
if (-not ($choice -match '^\d+$' -and [int]$choice -in @(1, 9))) { Write-Die "无效选择，已退出（未修改任何文件）。" }

if ($choice -eq "9") { Do-Restore }

# ============================== 连接信息 ==============================
Write-Head "连接网关"

# Base URL：优先 STARFIRE_BASE_URL 环境变量，否则交互输入
$BaseUrl = ""
if ($env:STARFIRE_BASE_URL) {
    $BaseUrl = $env:STARFIRE_BASE_URL.Trim()
    Write-Ok "使用环境变量 STARFIRE_BASE_URL 提供的 Base URL：$BaseUrl"
} else {
    Write-Dim "提示：可设置 STARFIRE_BASE_URL / STARFIRE_API_KEY 环境变量跳过输入。"
    for ($attempt = 0; $attempt -lt 3; $attempt++) {
        $raw = Read-Host "请输入网关 Base URL（例如 http://localhost:8080/v1）"
        if ($raw -and $raw.Trim()) { $BaseUrl = $raw.Trim(); break }
        Write-Warn "Base URL 不能为空。"
    }
    if (-not $BaseUrl) { Write-Die "未提供有效的 Base URL，已退出（未修改任何文件）。" }
}
$BaseUrl = $BaseUrl.Trim()
if ($BaseUrl -match '/models/?$') {
    $BaseUrl = $BaseUrl -replace '/models/?$', ''
    Write-Dim "已去掉末尾的 /models：$BaseUrl"
}
$BaseUrl = $BaseUrl.TrimEnd('/')
if ($BaseUrl -match '"') { Write-Die "Base URL 不能包含双引号。" }

# API key：优先 STARFIRE_API_KEY 环境变量，否则交互输入；留空表示无鉴权
$ApiKey = ""
if ($env:STARFIRE_API_KEY) {
    $ApiKey = $env:STARFIRE_API_KEY.Trim()
    Write-Ok "使用环境变量 STARFIRE_API_KEY 提供的 API key。"
} else {
    Write-Dim "API key 留空（直接回车）表示无鉴权，适用于不需要鉴权的本地网关。"
    $ApiKey = (Read-Host "请输入 API key（Bearer token）").Trim()
}
if ($ApiKey -match '"') { Write-Die "API key 不能包含双引号。" }

# ============================== 获取模型 ==============================
Write-Head "获取模型列表"

$result = Get-ModelList -BaseUrl $BaseUrl -ApiKey $ApiKey
$BaseUrl = $result.Base
$ids = $result.Ids
$gatewayMods = $result.Mods
Write-Ok "已连接：$BaseUrl"

$modsKnown = @($gatewayMods.Keys).Count
if ($modsKnown -gt 0) {
    Write-Ok "网关声明了 $modsKnown 个模型的输入模态"
} else {
    Write-Dim "网关未声明输入模态，稍后将交互确认所选模型是否支持图片"
}

foreach ($id in $ids) {
    $desc = "Starfire 网关模型"
    if ($id -match $NonChatPattern) { $desc = "疑似非对话模型（embedding/音频等），不建议设为默认" }
    $modalities = $null
    if ($gatewayMods.ContainsKey($id)) {
        $modalities = $gatewayMods[$id]
        $caps = @($modalities | Where-Object { $_ -ne "text" })
        if ($caps.Count -gt 0) {
            $desc = "$desc（支持图片输入）"
        }
    }
    $Models += @{ slug = $id; display = $id; desc = $desc; modalities = $modalities }
}

# ============================== 选择模型 ==============================
Write-Info ""
Write-Info "获取到 $($Models.Count) 个模型，请选择默认模型："
$menuIdx = 1
foreach ($m in $Models) {
    $mark = if ($m.desc -like '疑似*') { "   <-- 不建议" } else { "" }
    if (-not $mark -and $m.modalities) {
        $caps = @($m.modalities | Where-Object { $_ -ne "text" })
        if ($caps.Count -gt 0) { $mark = "   <$($caps -join '/')>" }
    }
    Write-Info ("  {0}. {1}{2}" -f $menuIdx, $m.display, $mark)
    $menuIdx++
}
Write-Info ""

$choice = ""
for ($attempt = 0; $attempt -lt 3; $attempt++) {
    $choice = Read-Host "输入 1-$($Models.Count)"
    if ($choice -match '^\d+$' -and [int]$choice -ge 1 -and [int]$choice -le $Models.Count) { break }
    Write-Warn "无效输入。"
}
if (-not ($choice -match '^\d+$' -and [int]$choice -ge 1 -and [int]$choice -le $Models.Count)) { Write-Die "无效选择，已退出（未修改任何文件）。" }

$SelectedIndex = [int]$choice - 1
$ModelSlug = $Models[$SelectedIndex].slug
$CatalogValue = $ModelsPath

# 交互确认：网关未声明所选模型的输入模态时，人工确认是否支持图片
if (-not $Models[$SelectedIndex].modalities) {
    Write-Info ""
    Write-Warn "网关未提供 $ModelSlug 的输入模态信息，请人工确认："
    $ansImg = Read-Host "$ModelSlug 是否支持图片输入? (y/N)"
    $mods = @("text")
    if ($ansImg -match '^[yY]') { $mods += "image" }
    if ($mods.Count -gt 1) {
        $Models[$SelectedIndex].desc = "$($Models[$SelectedIndex].desc)（支持图片输入）"
    }
    $Models[$SelectedIndex].modalities = $mods
    Write-Ok "$ModelSlug 输入模态已确认：$($mods -join '/')"
}

# ============================== pre-flight ==============================
if (Test-Path $BackupDir) {
    $problems = @()
    if (-not (Test-Path $ModelsPath)) {
        $problems += "缺少 $ModelsPath"
    } else {
        $jsonContent = Get-Content $ModelsPath -Raw -ErrorAction SilentlyContinue
        if ($jsonContent -notmatch [regex]::Escape($ModelSlug)) { $problems += "$ModelsPath 中缺少模型 $ModelSlug" }
    }
    if (-not (Test-Path $ConfigPath)) {
        $problems += "缺少 $ConfigPath"
    } else {
        $confContent = Get-Content $ConfigPath -Raw -ErrorAction SilentlyContinue
        if ($confContent -notmatch "\[model_providers\.$ProviderId\]") {
            $problems += "$ConfigPath 中缺少 [model_providers.$ProviderId]"
        }
    }
    if ($problems.Count -gt 0) {
        $problemList = $problems -join "`n  • "
        Write-Die "检测到备份目录 $BackupDir 已存在，但当前配置与本脚本的预期不符：`n  • $problemList`n`n为避免破坏现有文件或那份备份，本次已中止，未修改任何文件。`n`n建议处理方式（二选一）：`n  a) 重新运行本脚本，选择 9 先恢复默认配置，再重新运行选择安装；`n  b) 自行检查并删除上述不符合预期的文件（若确认备份目录已无价值，连同 $BackupDir 一起删除），然后重新运行本脚本。"
    }
    Switch-ModelOnly
}

if (Test-Path $ModelsPath) {
    # 识别 models.json 是否由已知的姊妹 setup 脚本写入（以其备份目录为标志）
    $knownBackups = @()
    if (Test-Path "$CodexHome\backup-local")    { $knownBackups += @{ dir = "backup-local";    provider = "local" } }
    if (Test-Path "$CodexHome\backup-deepseek") { $knownBackups += @{ dir = "backup-deepseek"; provider = "deepseek" } }

    if ($knownBackups.Count -gt 0) {
        $names = ($knownBackups | ForEach-Object { $_.dir }) -join "、"
        Write-Warn "检测到 $ModelsPath 已存在，且找到了姊妹脚本的备份目录：$names"
        Write-Dim "    （models.json 很可能由 codex-local-setup.ps1 / codex-deepseek-setup.ps1 写入）"
        Write-Info ""
        Write-Info "可选择让本脚本接管配置，将执行："
        Write-Info "  1. 备份当前 $ConfigPath 到 $BackupDir"
        Write-Info "  2. 重写 model / model_provider 等键，并移除被接管的 [model_providers.*] 旧段"
        Write-Info "  3. 用网关模型列表重写 $ModelsPath"
        Write-Warn "  • 姊妹脚本的备份目录（$names）保持不动，之后仍可用对应脚本选 9 还原"
        Write-Info ""
        $ans = Read-Host "输入 t 接管，其它任意键取消"
        if ($ans -match '^[tT]') {
            $TakeoverProviders = @($knownBackups | ForEach-Object { $_.provider })
            Write-Ok "已选择接管（将移除：$($TakeoverProviders -join '、')）。"
        } else {
            Write-Die "已取消，未修改任何文件。如需自行处理：`n  • 用对应脚本（codex-local-setup.ps1 等）选 9 恢复默认后重试（注意：会把 config.toml 回滚到其安装前的快照，之后的新改动会丢失）；`n  • 或确认无用后手动删除：Remove-Item $ModelsPath"
        }
    } else {
        Write-Die "检测到已存在：`n  $ModelsPath`n`n该文件不是本脚本写入的（没有找到本脚本的备份目录 ${BackupDir}）。`n本脚本需要创建这个文件。请先自行删除该文件，然后重新运行：`n  Remove-Item $ModelsPath`n  然后重新运行此脚本。"
    }
}

# ============================== first install ==============================
Write-Head "首次安装（目标模型：${ModelSlug}）"
Write-Info "  网关 Base URL: $BaseUrl"
Write-Info "  API key:       $(if ($ApiKey) { '已提供（不回显）' } else { '无鉴权' })"

# ============================== backup ==============================
New-Item -ItemType Directory -Path $BackupDir -Force | Out-Null

$origExisted = $true
if (Test-Path $ConfigPath) {
    Copy-Item $ConfigPath $BackupConf -Force
    Write-Ok "已备份 config.toml → $BackupConf"
} else {
    $origExisted = $false
    Write-Warn "未找到 config.toml，将创建新文件"
}

# ============================== parse & modify config.toml ==============================
$lines = @()
if ($origExisted) {
    $lines = @(Get-Content $ConfigPath -Encoding UTF8)
}

$out = @()
$report = @()
$seen = @{}
$curSection = ""
$skipSection = $false
$depth = 0
$inMLString = $false

$idx = 0
while ($idx -lt $lines.Count) {
    $line = $lines[$idx]
    $trimmed = $line.Trim()

    $isSectionHeader = $false
    if (-not $inMLString -and $depth -eq 0) {
        if ($trimmed -match '^\[.*\]$') { $isSectionHeader = $true }
    }

    if ($isSectionHeader) {
        $hdr = $trimmed.TrimStart('[').TrimEnd(']').Trim().Trim('"').Trim("'")
        $curSection = $hdr
        $skipSection = $false

        if ($hdr -eq "model_providers.$ProviderId" -or $hdr -like "model_providers.$ProviderId.*") {
            $skipSection = $true
            $report += "删除旧的 [$hdr]（将以新配置重写）"
        } elseif ($TakeoverProviders.Count -gt 0 -and ($TakeoverProviders | Where-Object { $hdr -eq "model_providers.$_" -or $hdr -like "model_providers.$_.*" })) {
            $skipSection = $true
            $report += "删除旧的 [$hdr]（已由 starfire 接管）"
        } elseif ($hdr -eq "profiles" -or $hdr -like "profiles.*") {
            $skipSection = $true
            $report += "删除 [$hdr]  ← profile 会遮蔽 model / model_provider 等设置，且此版本已禁止写入"
        } elseif ($hdr -eq "auto_review") {
            $report += "保留 [$hdr]  (提示: auto_review 若指向 gpt-5.x 模型会使用 fallback 元数据)"
        } elseif ($hdr -eq "tui.model_availability_nux") {
            $report += "保留 [$hdr]  (仅 NUX 计数器，无害)"
        }

        $idx++
        if (-not $skipSection) { $out += $line }
        continue
    }

    if ($curSection -ne "") {
        if ($skipSection) { $idx++; continue }
        $k = Get-KeyFromLine $line
        if ($k -eq "wire_api") {
            $v = Get-ValFromLine $line
            if ($v -match '^"chat"') {
                $indent = $line -replace '\S.*',''
                $out += "${indent}wire_api = `"responses`""
                $report += "修正 [$curSection] 的 wire_api: `"chat`" → `"responses`"  ← 此版本 `"chat`" 会导致 Codex 无法启动"
                $idx++; continue
            }
        }
        $out += $line; $idx++; continue
    }

    $k = Get-KeyFromLine $line
    if ($k -and $TargetKeys -contains $k) {
        $oldv = Get-ValFromLine $line
        $newv = Get-TargetValue $k
        $out += "$k = $newv"
        $seen[$k] = $true
        if ($oldv -ne $newv) {
            $truncated = if ($oldv.Length -gt 58) { $oldv.Substring(0,58) + "..." } else { $oldv }
            $report += ("改写 ${k}: ${truncated} → ${newv}")
        }
        $idx++; continue
    }

    if ($k -and $DelA -contains $k) {
        $oldv = Get-ValFromLine $line
        $why = switch ($k) {
            "profile"        { "profile 会遮蔽 model / model_provider / model_catalog_json，且此版本已禁止写入" }
            "oss_provider"   { "备用 provider 选择器，会把请求重定向到别处" }
            "openai_base_url"{ "全局 base_url 覆盖，会劫持请求" }
            default          { "与目标配置冲突" }
        }
        $report += "删除 $k = $oldv  ← $why"
        $idx++; continue
    }

    if ($k -and $DelB -contains $k) {
        $oldv = Get-ValFromLine $line
        $why = switch ($k) {
            "model_context_window"                       { "覆盖 models.json 的 1M 上下文；报大会导致自动压缩不触发，跑到一半 API 报错" }
            "model_auto_compact_token_limit"             { "覆盖自动压缩时机" }
            "model_auto_compact_token_limit_scope"       { "覆盖自动压缩时机" }
            "base_instructions"                          { "覆盖 models.json 里的 base_instructions" }
            "model_instructions_file"                    { "覆盖 models.json 里的 base_instructions" }
            "compact_prompt"                             { "覆盖上下文压缩提示词" }
            "experimental_compact_prompt_file"            { "覆盖上下文压缩提示词" }
            "service_tier"                               { "残留值会作为参数发给 API，可能 400" }
            "model_verbosity"                            { "残留值可能超出模型支持范围" }
            "model_reasoning_summary"                    { "models.json 声明 default_reasoning_summary=none，残留值会发送 reasoning.summary" }
            "plan_mode_reasoning_effort"                 { "可能是 xhigh，而 models.json 只声明 low / high / max" }
            "experimental_use_unified_exec_tool"          { "与 models.json 的 shell_type=shell_command 冲突" }
            default                                      { "与 models.json 声明矛盾" }
        }
        $report += "删除 $k = $oldv  ← $why"
        $idx++; continue
    }

    $out += $line
    $idx++
}

# Add missing target keys before first section
$missing = @()
foreach ($k in $TargetKeys) { if (-not $seen[$k]) { $missing += $k } }
$insertAt = $out.Count
for ($i = 0; $i -lt $out.Count; $i++) { if ($out[$i].TrimStart() -match '^\[.*\]$') { $insertAt = $i; break } }
if ($missing.Count -gt 0) {
    $before = @(); $after = @()
    for ($i = 0; $i -lt $out.Count; $i++) { if ($i -lt $insertAt) { $before += $out[$i] } else { $after += $out[$i] } }
    $out = $before
    foreach ($k in $missing) { $out += "$k = $(Get-TargetValue $k)" }
    if ($after.Count -gt 0 -and ($after[0].TrimStart() -match '^\[.*\]$')) { $out += "" }
    $out += $after
}

# ============================== write config.toml ==============================
if ($out.Count -gt 0 -and $out[$out.Count - 1].Trim() -ne "") { $out += "" }
$providerSection = Build-ProviderSection
Write-NoBom -Path $ConfigPath -Text (($out -join "`r`n") + "`r`n" + $providerSection + "`r`n")

# ============================== write models.json ==============================
$tmpModels = "$ModelsPath.tmp"
Write-ModelsJson $tmpModels
Move-Item $tmpModels $ModelsPath -Force

# ============================== report ==============================
Write-Ok "已写入 ${ModelsPath}（含网关返回的 $($Models.Count) 个模型）"
Write-Ok "已更新 $ConfigPath"

if ($report.Count -gt 0) {
    Write-Head "对原有配置的改动（共 $($report.Count) 项）"
    foreach ($r in $report) { Write-Info "  • $r" }
}

Write-Head "已写入的配置"
Write-Info "  model                  = `"$ModelSlug`""
Write-Info "  model_provider         = `"$ProviderId`""
Write-Info "  preferred_auth_method  = `"apikey`""
Write-Info "  forced_login_method    = `"api`""
Write-Info "  model_reasoning_effort = `"high`""
Write-Info "  model_catalog_json     = `"$CatalogValue`""
Write-Info ""
Write-Info "  [model_providers.$ProviderId]"
Write-Info "  base_url  = `"$BaseUrl`""
Write-Info "  wire_api  = `"responses`""
Write-Info "  auth      = $(if ($ApiKey) { 'bearer token 已写入' } else { '无（未写入 bearer token）' })"

# ============================== manifest ==============================
$manifestContent = @"
script_version=$ScriptVersion
installed_at=$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
original_config_existed=$(if($origExisted){1}else{0})
model_slug=$ModelSlug
base_url=$BaseUrl
catalog_value=$CatalogValue
codex_home=$CodexHome
takeover_from=$($TakeoverProviders -join ',')
--- 对 config.toml 的改动 ---
$($report -join "`n")
"@
$manifestContent | Set-Content $Manifest -Encoding UTF8

Write-Info ""
Write-Ok "安装完成。"
Write-Warn "请完全退出 Codex 后重新打开，配置才会生效"
Write-Info ""
Write-Info "如何确认已生效："
Write-Info "  • Codex CLI：启动信息中显示 model: $ModelSlug"
Write-Info ""
Write-Dim "再次运行本脚本可重新获取模型列表并切换模型（选 1）或恢复默认配置（选 9）。"
