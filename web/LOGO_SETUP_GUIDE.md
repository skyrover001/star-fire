# 自定义Logo配置指南

本指南将帮助您为Star-Fire项目配置自定义的PNG logo。

## 📁 文件放置位置

将您的自定义logo文件放置在以下位置：
```
apps/web-antd/public/logo.png
```

## 🖼️ Logo文件要求

### 推荐规格：
- **格式**: PNG (支持透明背景)
- **尺寸**: 32x32px 到 128x128px (系统会自动缩放到32px)
- **宽高比**: 建议使用正方形 (1:1)
- **背景**: 透明背景，适配明暗主题
- **对比度**: 确保在明暗主题下都有良好的可见性

### 文件命名：
- 主logo: `logo.png`
- 网站图标: `favicon.ico` (可选，16x16px 或 32x32px)

## ⚙️ 配置文件

Logo配置已在以下文件中设置：

### 1. apps/web-antd/src/preferences.ts
```typescript
logo: {
  enable: true,                    // 启用logo显示
  fit: 'contain',                 // 图片适应方式：保持宽高比
  source: '/logo.png',            // logo文件路径
}
```

### 2. apps/web-antd/.env
```properties
VITE_APP_TITLE=Star-Fire          # 应用标题，显示在logo旁边
```

## 🎨 Logo适应方式选项

`fit` 属性可以设置为以下值：

- `'contain'` (推荐): 保持宽高比，完整显示图片
- `'cover'`: 保持宽高比，裁剪以填满容器
- `'fill'`: 拉伸图片以填满容器
- `'none'`: 保持原始尺寸
- `'scale-down'`: 类似contain，但不会放大图片

## 📱 Logo显示位置

您的logo将显示在以下位置：
- 侧边栏顶部 (主要位置)
- 折叠状态下的侧边栏
- 可能的登录页面 (如果有配置)

## 🔧 高级配置

如果需要更多自定义选项，可以在 `preferences.ts` 中添加：

```typescript
logo: {
  enable: true,
  fit: 'contain',
  source: '/logo.png',
  // 可以通过CSS覆盖样式来调整logo大小
}
```

## 🚀 应用更改

1. 将您的PNG logo文件复制到 `apps/web-antd/public/logo.png`
2. (可选) 替换 `apps/web-antd/public/favicon.ico` 为您的网站图标
3. 重启开发服务器：
   ```bash
   pnpm dev
   ```
4. 清除浏览器缓存以确保新logo生效

## 🔍 故障排除

如果logo没有显示：

1. **检查文件路径**: 确保文件位于 `public/logo.png`
2. **检查文件格式**: 确保是有效的PNG文件
3. **清除缓存**: 清除浏览器缓存和应用缓存
4. **检查控制台**: 查看浏览器控制台是否有加载错误
5. **重启服务**: 重启开发服务器

## 💡 提示

- 为了最佳显示效果，建议提供2x分辨率的logo (如64x64px)
- 确保logo在深色和浅色背景下都清晰可见
- 可以通过CSS自定义logo的hover效果和动画
