<div align="center">

<img src="./docs/komarix.svg" width="120" alt="KomariX Logo">

# KomariX

**轻量、直观、自托管的服务器监控面板**

实时监控服务器运行状态，在一个面板中查看 CPU、内存、磁盘、网络、负载、延迟与在线状态。

[![版本](https://img.shields.io/github/v/release/kkx999/KomariX?label=版本)](https://github.com/kkx999/KomariX/releases)
[![许可证](https://img.shields.io/github/license/kkx999/KomariX?label=许可证)](./LICENSE)

[**资源投稿**](https://komarix.666101.xyz/) · 为 KomariX 提交第三方主题或插件，审核通过后发布至官方市场。

</div>

## 项目介绍

> **项目来源说明：KomariX 基于开源项目 Komari 修改并继续维护。** 在保持核心监控能力和生态兼容性的基础上，KomariX 进行独立品牌、组件、发布与维护。

KomariX 是一个面向个人用户和小型服务器集群的自托管监控面板。

它通过轻量级 Agent 采集服务器运行数据，并在 Web 面板中进行实时展示。你可以用一套面板集中查看多台服务器的在线状态、资源占用、网络流量和历史数据，也可以使用通知、主题和插件等扩展功能。

Web、Agent、默认 PurCarte 主题以及主题/插件市场均由 KomariX 独立维护。KomariX 保持对旧 Komari 数据、主题与插件格式的兼容，并提供从 Komari 迁移到 KomariX 的持续适配。为降低远程控制攻击面，KomariX 已移除 Web 终端、任意远程命令执行和远程文件管理能力，专注于服务器监控。

## 主要功能

- 多服务器集中监控
- CPU、内存、磁盘、负载实时数据
- 网络上传、下载与流量统计
- 在线状态与延迟监控
- 历史监控数据与趋势图表
- 节点分组与排序
- 消息通知
- 默认内置并启用 PurCarte 磨砂玻璃主题
- 兼容第三方旧主题与插件格式；旧主题支持 `komari-theme.json`，并兼容 ZIP 根目录或唯一一级外层目录打包
- 使用 KomariX 自有主题/插件市场索引镜像
- 支持第三方主题与插件扩展、投稿与审核
- 支持从 Komari 迁移数据并持续处理兼容项
- 数据备份与恢复（恢复前验证、失败自动回滚）
- 管理后台登录防爆破与可配置审计日志保留/清理
- 移除 Web 终端、任意远程命令执行与远程文件管理，降低远程控制攻击面
- Panel / Agent 正式版下载校验 SHA256，升级失败保留或恢复旧版本
- 多架构 Linux / Windows 构建
- 自托管部署，监控数据由自己掌控

## 快速安装

推荐使用 Debian / Ubuntu 等支持 systemd 的 Linux 系统，并使用 root 用户执行：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/kkx999/KomariX/main/install-komarix.sh)
```

安装脚本会自动识别系统架构，并提供稳定版与快照版选择。正式版本下载会校验 `SHA256SUMS`；升级会先下载并校验新二进制，若新版本启动失败则自动恢复升级前版本。

默认面板端口：

```text
25774
```

安装完成后，在浏览器中访问：

```text
http://服务器IP:25774
```

首次打开后按照页面提示完成初始化即可。

## 支持架构

正式版本目前提供以下构建：

| 系统 | 架构 |
| --- | --- |
| Linux | amd64 |
| Linux | arm64 |
| Linux | 386 |
| Linux | riscv64 |
| Linux | loong64 |
| Windows | amd64 |
| Windows | arm64 |
| Windows | 386 |

最新正式版可在 [Releases](https://github.com/kkx999/KomariX/releases) 下载。

## 文档与资源

- [版本更新记录](./CHANGELOG.md)
- [正式版本下载](https://github.com/kkx999/KomariX/releases)
- [主题与插件资源投稿](https://komarix.666101.xyz/)

## 从 Komari 迁移

KomariX 保持对旧 Komari 数据和 Agent 监控协议的兼容。通过完整数据迁移升级到 KomariX 后，原有 Komari Agent 可以继续连接并上报监控数据，不要求立即重装。

为了后续持续使用 KomariX 的更新与新功能，建议逐步将旧 Komari Agent 替换为 KomariX Agent。迁移单个节点时，先在 KomariX 后台打开**原有节点**并复制该节点当前生成的一键安装命令，然后先清理旧 Komari Agent，再安装 KomariX Agent。

### 卸载旧 Komari Agent

> 仅用于从原 Komari 迁移到 KomariX 的被监控机。

```bash
systemctl stop komari-agent 2>/dev/null || true; systemctl disable komari-agent 2>/dev/null || true; rm -f /etc/systemd/system/komari-agent.service; systemctl daemon-reload; systemctl reset-failed komari-agent 2>/dev/null || true; rm -f /opt/komari/agent; echo "旧 Komari Agent 已清理完成"
```

随后执行该节点在 KomariX 后台生成的一键安装命令，即可安装并启动 `komarix-agent`。

> **不要删除 KomariX 后台中的原有节点再重新创建。** 继续使用迁移后的原节点，可以保留节点身份以及已有历史监控数据。上面的命令只删除旧 Komari Agent 服务与 `/opt/komari/agent`，不会删除整个 `/opt/komari` 目录。

## 卸载 KomariX Agent

> 仅用于已经安装 KomariX Agent 的被监控机。

默认 Linux / systemd 安装可使用：

```bash
systemctl stop komarix-agent 2>/dev/null || true; systemctl disable komarix-agent 2>/dev/null || true; rm -f /etc/systemd/system/komarix-agent.service; systemctl daemon-reload; systemctl reset-failed komarix-agent 2>/dev/null || true; rm -rf /opt/komarix-agent; echo "KomariX Agent 已卸载"
```

该命令只卸载被监控机上的 KomariX Agent，不会删除 KomariX 面板中的节点或历史监控数据。

## 数据与目录

默认安装目录：

```text
/opt/komarix
```

默认 systemd 服务名：

```text
komarix
```

全新安装统一使用 KomariX 名称、目录和 systemd 服务标识。

## 安全说明

KomariX 定位为服务器监控面板，不提供 Web 终端、任意远程命令执行或远程文件管理能力，以减少不必要的远程控制攻击面。

建议：

- 为管理后台设置高强度密码，并建议启用 2FA
- 优先通过 HTTPS 访问面板
- 不要将管理入口暴露给不可信网络
- 定期备份数据目录，并保留至少一份独立于当前服务器的备份
- 根据使用规模设置审计日志保留天数与最大条数
- 及时安装经过 SHA256 校验的正式版本

## 开源说明

原项目版权声明、MIT License 与 NOTICE 均在仓库中完整保留。KomariX 的后续品牌、维护和版本发布由本仓库独立进行。
