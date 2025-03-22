# NCA（Nano Code Agent）

## 项目简介

NCA是一个轻量级的基于命令行的代码助手工具，基于Go语言开发，旨在帮助开发者提高编码效率。通过连接多种大语言模型API（如Doubao、Qwen、DeepSeek等），NCA能够理解自然语言指令，执行代码分析和生成，以及协助完成各种开发任务。

## 主要功能

- **交互式命令行界面**：提供友好的REPL界面，支持单次查询和持续对话模式
- **多种LLM支持**：支持多种大语言模型API提供商，包括Doubao、Qwen和DeepSeek
- **会话管理**：支持创建检查点并保存对话上下文
- **灵活配置系统**：支持本地和全局配置项
- **管道输入支持**：可以通过管道接收输入内容
- **调试模式**：提供详细的日志记录，便于排查问题

## 安装方法

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/pederhe/nca.git
cd nca

# 构建项目
make build

# 安装到系统路径
make install
```

## 使用方法

### 配置API提供商

```bash
# 设置API提供商
nca config set model doubao-1-5-pro-32k-250115
nca config set api_key your_api_key_here
```

### 基本使用

```bash
# 启动交互式模式
nca

# 一次性查询
nca -p "如何实现一个简单的HTTP服务器？"

# 通过管道传递输入
cat main.go | nca "分析这段代码的性能问题"
```

### 更多命令

```bash
nca help
```
