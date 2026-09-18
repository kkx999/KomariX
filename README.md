<div align="center">

<img src="./docs/komarix.svg" width="120" alt="KomariX Logo">

# KomariX

**轻量、直观、自托管的服务器监控面板**

实时监控服务器运行状态，在一个面板中查看 CPU、内存、磁盘、网络、负载、延迟与在线状态。

[![版本](https://img.shields.io/github/v/release/kkx999/KomariX?label=版本)](https://github.com/kkx999/KomariX/releases)
[![许可证](https://img.shields.io/github/license/kkx999/KomariX?label=许可证)](./LICENSE)

</div>

## 项目介绍

> **项目来源说明：KomariX 基于开源项目 Komari 修改并继续维护。** 在保持核心监控能力和生态兼容性的基础上，KomariX 进行独立品牌、组件、发布与维护。

KomariX 是一个面向个人用户和小型服务器集群的自托管监控面板。

它通过轻量级 Agent 采集服务器运行数据，并在 Web 面板中进行实时展示。你可以用一套面板集中查看多台服务器的在线状态、资源占用、网络流量和历史数据，也可以使用通知、任务、插件和终端等扩展功能。

KomariX 当前基于 **Komari** 进行独立维护。Web、Agent 以及默认 PurCarte 主题所需源码均已归档在本仓库中，日常构建和新节点安装不再依赖 Komari 原仓库。

## 主要功能

- 多服务器集中监控
- CPU、内存、磁盘、负载实时数据
- 网络上传、下载与流量统计
- 在线状态与延迟监控
- 历史监控数据与趋势图表
- 节点分组与排序
- 消息通知
- Web 终端
- 默认内置并启用 PurCarte 磨砂玻璃主题
- 兼容 Komari 主题市场与插件市场
- 使用 KomariX 自有主题/插件市场索引镜像
- 支持主题与插件扩展
- 数据备份与恢复
- 多架构 Linux / Windows 构建
- 自托管部署，监控数据由自己掌控

## 快速安装

推荐使用 Debian / Ubuntu 等支持 systemd 的 Linux 系统，并使用 root 用户执行：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/kkx999/KomariX/main/install-komarix.sh)
```

安装脚本会自动识别系统架构，并提供稳定版与快照版选择。

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

## 版本说明

KomariX 的首个正式版本为 **v1.0.0**。

项目基础：

```text
Komari
↓
KomariX
```

目前保持原版监控逻辑、数据库结构、API、Agent 协议和主要功能行为，优先保证兼容性与稳定性。主题与插件体系继续兼容 Komari 生态；KomariX 同时保存市场索引镜像，默认主题为基于 **PurCarte v1.2.5** 修改的 KomariX 内置版本。PurCarte 原作者署名与 MIT License 保留不变。

## 数据与目录

默认安装目录：

```text
/opt/komarix
```

默认 systemd 服务名：

```text
komarix
```

全新安装统一使用 KomariX 名称。检测到旧版 `/opt/komari`、`komarix.service` 时，安装器会继续按旧路径兼容管理，不强制迁移数据。

## 安全说明

KomariX 包含服务器监控和远程管理能力，请只部署在你拥有或已获得授权管理的服务器上。

建议：

- 为管理后台设置高强度密码
- 优先通过 HTTPS 访问面板
- 不要将管理入口暴露给不可信网络
- 定期备份数据目录
- 及时安装经过验证的正式版本

## 开源说明

KomariX 基于开源项目 Komari 修改并继续维护。

原项目版权声明、MIT License 与 NOTICE 均在仓库中完整保留。KomariX 的后续品牌、维护和版本发布由本仓库独立进行。
