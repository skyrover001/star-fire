# 配置说明

## 服务器地址配置

### 开发环境配置

在开发环境中，可以通过修改环境变量文件来配置服务器地址：

**文件位置**: `apps/web-antd/.env.development`

```env
# 服务器主机地址（用于客户端连接）
VITE_GLOB_SERVER_HOST=1.94.239.51
```

### 生产环境配置

#### 方式一：编译时配置
在编译前修改环境变量文件：

**文件位置**: `apps/web-antd/.env.production`

```env
# 服务器主机地址（用于客户端连接）
VITE_GLOB_SERVER_HOST=your-production-host.com
```

#### 方式二：编译后动态配置（推荐）
编译完成后，可以直接修改生成的配置文件：

**文件位置**: `dist/_app.config.js`

```javascript
window._VBEN_ADMIN_PRO_APP_CONF_={
  "VITE_GLOB_API_URL": "/api",
  "VITE_GLOB_SERVER_HOST": "your-production-host.com"
};
```

## 配置项说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `VITE_GLOB_API_URL` | API接口地址 | `/api` |
| `VITE_GLOB_SERVER_HOST` | 服务器主机地址（用于客户端连接命令） | `1.94.239.51` |

## 部署建议

1. **Docker 部署**: 可以通过环境变量或者挂载配置文件的方式动态配置
2. **静态部署**: 部署后直接修改 `_app.config.js` 文件
3. **CDN 部署**: 在 CDN 源站修改配置文件

## 使用场景

- **多环境部署**: 同一套代码部署到不同环境，只需修改配置文件
- **动态切换**: 无需重新编译，直接修改配置文件即可切换服务器
- **客户端配置**: 用户可以看到正确的服务器连接命令

## 注意事项

1. 修改配置文件后需要刷新浏览器页面
2. 配置文件为纯 JavaScript，注意语法正确性
3. 建议在修改前备份原配置文件
