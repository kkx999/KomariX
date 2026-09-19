# KomariX 安装与部署教程

> [!WARNING]
> **KomariX 当前处于测试阶段，仅建议用于测试、体验和开发验证，不建议用于长期生产环境。**
> 重要数据请定期备份，并至少保留一份服务器之外的副本。

本教程适用于 KomariX Panel 的常规 Linux 部署，推荐使用 Debian / Ubuntu 等支持 systemd 的系统，并使用 `root` 用户执行。

---

## 一、准备工作

建议准备：

- 一台 Linux VPS
- root 权限
- 一个已经解析到 VPS 公网 IP 的域名
- 公网 TCP 80 / 443 端口
- 如使用 Cloudflare DNS 验证，可不开放 80 端口用于证书验证，但 443 仍需要开放用于正常访问

KomariX 默认端口：

```text
25774
```

默认安装目录：

```text
/opt/komarix
```

默认 systemd 服务：

```text
komarix
```

---

## 二、一键安装 KomariX

在全新机器上执行：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/kkx999/KomariX/main/install-komarix.sh)
```

安装脚本会自动识别系统架构，并提供稳定版与快照版选择。

正式版下载会校验 Release 中的 `SHA256SUMS`，升级时会先下载并验证新二进制，启动失败会尝试恢复升级前版本。

推荐选择：

```text
稳定版
```

监听端口没有特殊需求时保持默认：

```text
25774
```

安装完成后先访问：

```text
http://服务器IP:25774
```

按照页面安装向导创建管理员账号并完成初始化。

---

## 三、Nginx 反向代理一键配置

KomariX 提供独立的一键反向代理脚本。

执行：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/kkx999/KomariX/main/setup-reverse-proxy.sh)
```

脚本会要求输入：

```text
KomariX 域名
KomariX 面板端口
```

如果安装 KomariX 时没有修改端口，直接使用默认值：

```text
25774
```

脚本会自动：

1. 检查并安装 Nginx
2. 创建 KomariX 独立反向代理配置
3. 配置真实 IP 转发头
4. 配置 WebSocket 反向代理
5. 关闭代理缓冲
6. 检查 Nginx 配置语法
7. 配置失败时自动恢复旧配置
8. 启动并启用 Nginx

Nginx 配置文件：

```text
/etc/nginx/conf.d/komarix.conf
```

配置完成后可先访问：

```text
http://你的KomariX域名
```

---

## 四、SSL 证书与 HTTPS 一键配置

先确认域名已经正确解析，然后执行：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/kkx999/KomariX/main/setup-https.sh)
```

脚本会要求输入：

```text
KomariX 域名
用于申请 SSL 证书的邮箱
KomariX 面板端口
```

如果已经运行过反向代理脚本，HTTPS 脚本会尝试自动读取当前 Nginx 配置中的 KomariX 后端端口。

然后选择证书验证方式：

```text
1. HTTP 验证
2. Cloudflare DNS 验证
```

### 方式一：HTTP 验证

适合公网 80 端口可以正常访问的服务器。

要求：

```text
域名已经解析到当前 VPS
TCP 80 可以从公网访问
TCP 443 可以从公网访问
```

选择：

```text
1. HTTP 验证
```

脚本使用 acme.sh + Let's Encrypt 自动申请和安装证书。

### 方式二：Cloudflare DNS 验证

如果域名 DNS 托管在 Cloudflare，可以选择：

```text
2. Cloudflare DNS 验证
```

DNS 验证申请证书时不要求公网开放 80 端口。

Cloudflare 凭据支持：

```text
1. API Token（推荐）
2. Global API Key
```

推荐使用 API Token，并尽量只给最小权限：

```text
Zone → DNS → Edit
Zone → Zone → Read
```

Zone Resources 建议限制为：

```text
Include → Specific zone → 你的域名
```

输入 Token / Global API Key 时，敏感内容不会回显在终端。

> Cloudflare 凭据会由 acme.sh 保存，用于后续自动续期证书。请保护好 `/root/.acme.sh/`。

### HTTPS 脚本会自动完成

1. 检查或安装 Nginx、curl 和证书续期组件
2. 配置 KomariX HTTP 反向代理
3. 安装或检查 acme.sh
4. 使用 Let's Encrypt 申请证书
5. 安装证书
6. 开启 HTTPS
7. 配置 HTTP 自动跳转 HTTPS
8. 配置 WebSocket 反向代理
9. 检查 Nginx 配置
10. 重载 Nginx
11. 配置 acme.sh 自动续期

SSL 证书目录：

```text
/etc/nginx/ssl/komarix/你的域名/
```

完成后访问：

```text
https://你的KomariX域名
```

---

## 五、常用服务命令

查看 KomariX 状态：

```bash
systemctl status komarix
```

重启：

```bash
systemctl restart komarix
```

停止：

```bash
systemctl stop komarix
```

启动：

```bash
systemctl start komarix
```

查看实时日志：

```bash
journalctl -u komarix -f
```

检查 Nginx 配置：

```bash
nginx -t
```

重载 Nginx：

```bash
systemctl reload nginx
```

---

## 六、数据备份

KomariX 默认运行目录为：

```text
/opt/komarix
```

主要持久化数据位于：

```text
/opt/komarix/data
```

其中默认包含主数据库 `komarix.db`、监控数据 `metrics.db` 以及其他运行数据。

如果需要重装 VPS，建议**备份整个 data 目录**，不要只备份单个数据库。

执行：

```bash
systemctl stop komarix

tar -C /opt/komarix -czf /root/komarix-data-backup.tar.gz data

systemctl start komarix
```

备份文件：

```text
/root/komarix-data-backup.tar.gz
```

请在重装系统前把该文件下载到自己的电脑或其他服务器保存。

---

## 七、数据恢复

先在新系统中正常安装一次 KomariX，然后把备份文件上传到：

```text
/root/komarix-data-backup.tar.gz
```

执行：

```bash
systemctl stop komarix

mv /opt/komarix/data /opt/komarix/data.before-restore.$(date +%Y%m%d%H%M%S)

tar -C /opt/komarix -xzf /root/komarix-data-backup.tar.gz

systemctl start komarix
```

然后检查：

```bash
systemctl status komarix
```

确认面板和数据均正常后，再自行删除旧的 `data.before-restore.*` 目录。

> 如果你把监控数据库切换到了外部 MySQL / PostgreSQL，仅备份本机 `/opt/komarix/data` 不包含外部数据库中的监控数据，请同时对外部数据库单独备份。

---

## 八、升级 KomariX

重新执行官方安装管理脚本：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/kkx999/KomariX/main/install-komarix.sh)
```

然后选择升级相关选项即可。

正式版本升级会校验 SHA256；新版本无法正常启动时，脚本会尽量恢复升级前的二进制文件。

---

## 九、卸载 KomariX

如需彻底卸载，请先确认已经完成数据备份。

停止并删除服务：

```bash
systemctl stop komarix 2>/dev/null || true
systemctl disable komarix 2>/dev/null || true
rm -f /etc/systemd/system/komarix.service
systemctl daemon-reload
systemctl reset-failed komarix 2>/dev/null || true
```

删除 KomariX：

```bash
rm -rf /opt/komarix
```

如果同时需要删除本教程创建的 Nginx 反代和证书：

```bash
rm -f /etc/nginx/conf.d/komarix.conf
rm -rf /etc/nginx/ssl/komarix
nginx -t && systemctl reload nginx
```

如果不再使用 acme.sh，可自行卸载；如果服务器上还有其他站点使用 acme.sh，**不要删除** `/root/.acme.sh/`。

---

## 十、端口与防火墙

直接使用 IP + KomariX 端口访问时，需要开放安装时设置的 Panel 端口，例如：

```text
25774/tcp
```

使用 Nginx + HTTPS 后，对公网通常只需要：

```text
80/tcp
443/tcp
```

建议让 KomariX Panel 端口仅供本机 Nginx 访问，并根据实际防火墙环境限制公网直接访问。

---

## 十一、安全建议

- 管理员密码使用高强度随机密码
- 建议开启 2FA
- 优先使用 HTTPS
- 定期备份 `/opt/komarix/data`
- Cloudflare API Token 使用最小权限
- 不要把备份文件、证书私钥或 Cloudflare 凭据公开上传
- 正式环境优先使用稳定版，而不是 Snapshot
- KomariX 当前仍处于测试阶段，重要环境请保留独立监控与数据备份方案
