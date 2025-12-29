<p align="right">
   <strong>中文</strong> | <a href="./README.en.md">English</a>
</p>
<div>

![start-fire](/unit/img/logo.png)

# Star Fire

MaaS层个人算力服务平台

下载release的版本。

### server端（linux系统）

1. 安装nginx，将starfire.conf放到nginx conf.d目录下，测试并生效配置
2. 运行starfire


### user端

1. 使用邮箱注册并登录

#### 分享模型
1. PC客户端模式（目前支持mac和windows，图形客户端，见release）
![img.png](unit/img/pc.png)
2. 命令行模式

   (1) 下载客户端，或本地编译 make client （build/client目录下）

   (2) 在模型广场页面点击注册到Star Fire 获取注册token

   (3) 注册客户端： 

       （windows）starfire.exe -host (host) -token {register token} -ippm {input prices per million tokens, default 4.0} -oppm {output prices per million tokens, default 8.0}
       
       （macos）： starfire -host {host} -token {register token} -ippm {input prices per million tokens, default 4.0} -oppm {output prices per million tokens, default 8.0} 

   (4) 本地使用ollama 运行模型，客户端会自动将模型信息推送到server端，准备提供服务 

   (5) 可以在我的收益页面查看自己所有提供模型的收益情况

#### 使用模型
1. 在模型广场页面选择模型
2. 在API密钥页面创建获取API密钥 
3. 使用 /v1/models 获取所有模型列表
4. 使用 /v1/chat/completions 对话
5. 可以在我的使用页面查看自己使用模型的情况

### 体验地址
http://115.190.26.60/
