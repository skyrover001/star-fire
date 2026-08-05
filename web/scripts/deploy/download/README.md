# 客户端安装包下载目录

将客户端安装包放入此目录，构建 Docker 镜像时会自动复制到 nginx 的 `/var/www/starfire/download/`，供模型广场的 `/download/` 路径下载。

## 需要的文件

| 文件 | 对应下载链接 | 说明 |
|------|-------------|------|
| `starfire.rar` | `/download/windows/starfire.rar` | Windows 客户端压缩包 |
| `starfire.zip` | `/download/macos/starfire.zip` | macOS 客户端压缩包 |
| `starfire.tar.gz` | `/download/linux/starfire.tar.gz` | Linux 客户端压缩包 |

## 使用方式

1. 将 `starfire.rar`、`starfire.zip`、`starfire.tar.gz` 放入本目录。
2. 重新构建前端镜像：

   ```bash
   docker compose build frontend
   docker compose up -d frontend
   ```

3. 验证下载链接可访问：

   ```
   http://<host>:8080/download/windows/starfire.rar
   http://<host>:8080/download/macos/starfire.zip
   http://<host>:8080/download/linux/starfire.tar.gz
   ```

> 注意：如果文件不存在，nginx 会返回 404，前端会提示下载失败（而不是之前的 HTML 解析报错）。
