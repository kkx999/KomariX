#!/usr/bin/env bash
set -euo pipefail

DEFAULT_PORT="25774"
NGINX_CONF="/etc/nginx/conf.d/komarix.conf"
ACME_SH="/root/.acme.sh/acme.sh"

die() {
    echo "错误：$*" >&2
    exit 1
}

info() {
    echo
    echo "==> $*"
}

if [[ "${EUID}" -ne 0 ]]; then
    die "请使用 root 用户运行此脚本。"
fi

install_dependencies() {
    local need_install=0
    command -v nginx >/dev/null 2>&1 || need_install=1
    command -v curl >/dev/null 2>&1 || need_install=1

    if [[ "${need_install}" -eq 0 ]]; then
        return
    fi

    info "正在安装 Nginx、curl 和证书续期所需组件..."
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update
        DEBIAN_FRONTEND=noninteractive apt-get install -y nginx curl cron ca-certificates
        systemctl enable --now cron >/dev/null 2>&1 || true
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y nginx curl cronie ca-certificates
        systemctl enable --now crond >/dev/null 2>&1 || true
    elif command -v yum >/dev/null 2>&1; then
        yum install -y nginx curl cronie ca-certificates
        systemctl enable --now crond >/dev/null 2>&1 || true
    else
        die "未找到支持的包管理器，请先手动安装 Nginx、curl 和 cron。"
    fi
}

normalize_domain() {
    local value="$1"
    value="${value#http://}"
    value="${value#https://}"
    value="${value%%/*}"
    value="${value%.}"
    printf '%s' "${value}"
}

valid_domain() {
    [[ "$1" =~ ^([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)+[A-Za-z]{2,63}$ ]]
}

valid_email() {
    [[ "$1" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]]
}

valid_port() {
    [[ "$1" =~ ^[0-9]+$ ]] && (( $1 >= 1 && $1 <= 65535 ))
}

detect_panel_port() {
    local detected=""
    if [[ -f "${NGINX_CONF}" ]]; then
        detected="$(sed -nE 's/^[[:space:]]*proxy_pass[[:space:]]+http:\/\/127\.0\.0\.1:([0-9]+);.*/\1/p' "${NGINX_CONF}" | head -n1)"
    fi
    printf '%s' "${detected:-${DEFAULT_PORT}}"
}

write_http_proxy() {
    cat > "${NGINX_CONF}" <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};

    client_max_body_size 64m;

    location / {
        proxy_pass http://127.0.0.1:${PORT};
        proxy_http_version 1.1;

        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
        proxy_buffering off;
    }
}
EOF
}

write_https_proxy() {
    cat > "${NGINX_CONF}" <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};

    return 301 https://\$host\$request_uri;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name ${DOMAIN};

    ssl_certificate ${SSL_DIR}/fullchain.cer;
    ssl_certificate_key ${SSL_DIR}/private.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    client_max_body_size 64m;

    location / {
        proxy_pass http://127.0.0.1:${PORT};
        proxy_http_version 1.1;

        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;

        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
        proxy_buffering off;
    }
}
EOF
}

install_dependencies

read -rp "请输入 KomariX 域名（例如 monitor.example.com）: " DOMAIN
read -rp "请输入用于申请 SSL 证书的邮箱: " EMAIL

DOMAIN="$(normalize_domain "${DOMAIN}")"
valid_domain "${DOMAIN}" || die "域名格式不正确：${DOMAIN:-<空>}"
valid_email "${EMAIL}" || die "邮箱格式不正确：${EMAIL}"

DETECTED_PORT="$(detect_panel_port)"
read -rp "请输入 KomariX 面板端口 [默认/检测到 ${DETECTED_PORT}]: " PORT
PORT="${PORT:-${DETECTED_PORT}}"
valid_port "${PORT}" || die "端口必须是 1-65535 之间的数字。"

SSL_DIR="/etc/nginx/ssl/komarix/${DOMAIN}"
mkdir -p "$(dirname "${NGINX_CONF}")" "${SSL_DIR}"

BACKUP_FILE=""
if [[ -f "${NGINX_CONF}" ]]; then
    BACKUP_FILE="${NGINX_CONF}.bak.$(date +%Y%m%d%H%M%S)"
    cp -a "${NGINX_CONF}" "${BACKUP_FILE}"
    echo "已备份原 Nginx 配置：${BACKUP_FILE}"
fi

echo
echo "请选择 SSL 证书验证方式："
echo "1. HTTP 验证（要求域名已解析到本机，公网 80 端口可访问）"
echo "2. Cloudflare DNS 验证（不要求开放 80 端口）"
read -rp "请选择 [1-2]: " VERIFY_METHOD

case "${VERIFY_METHOD}" in
    1) VERIFY_NAME="HTTP" ;;
    2) VERIFY_NAME="Cloudflare DNS" ;;
    *) die "请选择 1 或 2。" ;;
esac

info "准备 Nginx HTTP 反向代理..."
write_http_proxy
if ! nginx -t; then
    if [[ -n "${BACKUP_FILE}" && -f "${BACKUP_FILE}" ]]; then
        cp -a "${BACKUP_FILE}" "${NGINX_CONF}"
    else
        rm -f "${NGINX_CONF}"
    fi
    die "Nginx 配置测试失败，已回滚。"
fi
systemctl enable nginx >/dev/null 2>&1 || true
systemctl restart nginx

info "安装或检查 acme.sh..."
if [[ ! -x "${ACME_SH}" ]]; then
    curl -fsSL https://get.acme.sh | sh -s email="${EMAIL}"
fi
[[ -x "${ACME_SH}" ]] || die "acme.sh 安装失败。"
"${ACME_SH}" --set-default-ca --server letsencrypt

info "使用 ${VERIFY_NAME} 验证申请 Let's Encrypt 证书..."
set +e
if [[ "${VERIFY_METHOD}" == "1" ]]; then
    echo "请确认 ${DOMAIN} 已解析到当前 VPS，并且公网 TCP 80 可以访问。"
    "${ACME_SH}" --issue --nginx -d "${DOMAIN}"
    ACME_RC=$?
else
    echo
    echo "Cloudflare 凭据方式："
    echo "1. API Token（推荐）"
    echo "2. Global API Key"
    read -rp "请选择 [1-2]: " CF_METHOD

    case "${CF_METHOD}" in
        1)
            read -rsp "请输入 Cloudflare API Token: " CF_TOKEN_INPUT
            echo
            [[ -n "${CF_TOKEN_INPUT}" ]] || die "Cloudflare API Token 不能为空。"
            export CF_Token="${CF_TOKEN_INPUT}"
            unset CF_Account_ID CF_Zone_ID CF_Key CF_Email 2>/dev/null || true
            ;;
        2)
            read -rp "请输入 Cloudflare 登录邮箱: " CF_EMAIL_INPUT
            read -rsp "请输入 Cloudflare Global API Key: " CF_KEY_INPUT
            echo
            valid_email "${CF_EMAIL_INPUT}" || die "Cloudflare 登录邮箱格式不正确。"
            [[ -n "${CF_KEY_INPUT}" ]] || die "Cloudflare Global API Key 不能为空。"
            export CF_Email="${CF_EMAIL_INPUT}"
            export CF_Key="${CF_KEY_INPUT}"
            unset CF_Token CF_Account_ID CF_Zone_ID 2>/dev/null || true
            ;;
        *)
            die "请选择 1 或 2。"
            ;;
    esac

    "${ACME_SH}" --issue --dns dns_cf -d "${DOMAIN}"
    ACME_RC=$?
fi
set -e

if [[ "${ACME_RC}" -ne 0 && "${ACME_RC}" -ne 2 ]]; then
    echo "SSL 证书申请失败。"
    if [[ "${VERIFY_METHOD}" == "1" ]]; then
        echo "请检查域名解析、公网 80 端口以及防火墙。"
    else
        echo "请检查 Cloudflare Token/API Key 以及 Zone 权限。"
    fi
    exit "${ACME_RC}"
fi

info "安装证书..."
"${ACME_SH}" --install-cert -d "${DOMAIN}" \
    --key-file "${SSL_DIR}/private.key" \
    --fullchain-file "${SSL_DIR}/fullchain.cer" \
    --reloadcmd "systemctl reload nginx"

chmod 600 "${SSL_DIR}/private.key"

info "启用 HTTPS 和 HTTP 自动跳转..."
write_https_proxy

if ! nginx -t; then
    if [[ -n "${BACKUP_FILE}" && -f "${BACKUP_FILE}" ]]; then
        cp -a "${BACKUP_FILE}" "${NGINX_CONF}"
        nginx -t && systemctl reload nginx || true
    fi
    die "HTTPS Nginx 配置测试失败，已尝试恢复旧配置。"
fi

systemctl reload nginx

echo
echo "=============================================="
echo "KomariX HTTPS 配置完成"
echo "验证方式：${VERIFY_NAME}"
echo "访问地址：https://${DOMAIN}"
echo "后端地址：http://127.0.0.1:${PORT}"
echo "Nginx 配置：${NGINX_CONF}"
echo "证书目录：${SSL_DIR}"
echo "acme.sh 会自动续期证书，并在续期后重载 Nginx。"
echo "=============================================="
