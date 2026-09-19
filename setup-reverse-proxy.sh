#!/usr/bin/env bash
set -euo pipefail

DEFAULT_PORT="25774"
NGINX_CONF="/etc/nginx/conf.d/komarix.conf"

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

install_nginx() {
    if command -v nginx >/dev/null 2>&1; then
        return
    fi

    info "未检测到 Nginx，正在自动安装..."
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update
        DEBIAN_FRONTEND=noninteractive apt-get install -y nginx curl
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y nginx curl
    elif command -v yum >/dev/null 2>&1; then
        yum install -y nginx curl
    else
        die "未找到支持的包管理器，请先手动安装 Nginx 和 curl。"
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

valid_port() {
    [[ "$1" =~ ^[0-9]+$ ]] && (( $1 >= 1 && $1 <= 65535 ))
}

install_nginx

read -rp "请输入 KomariX 域名（例如 monitor.example.com）: " DOMAIN
read -rp "请输入 KomariX 面板端口 [默认 ${DEFAULT_PORT}]: " PORT
PORT="${PORT:-${DEFAULT_PORT}}"
DOMAIN="$(normalize_domain "${DOMAIN}")"

valid_domain "${DOMAIN}" || die "域名格式不正确：${DOMAIN:-<空>}"
valid_port "${PORT}" || die "端口必须是 1-65535 之间的数字。"

if ! curl -fsS --max-time 5 "http://127.0.0.1:${PORT}/" >/dev/null 2>&1; then
    echo "警告：当前无法通过 http://127.0.0.1:${PORT}/ 访问 KomariX。"
    echo "如果 KomariX 尚未启动或使用了其他端口，请先确认后再继续。"
    read -rp "仍然继续配置反向代理？[y/N]: " CONTINUE
    [[ "${CONTINUE}" =~ ^[Yy]$ ]] || exit 1
fi

mkdir -p "$(dirname "${NGINX_CONF}")"
BACKUP_FILE=""

if [[ -f "${NGINX_CONF}" ]]; then
    BACKUP_FILE="${NGINX_CONF}.bak.$(date +%Y%m%d%H%M%S)"
    cp -a "${NGINX_CONF}" "${BACKUP_FILE}"
    echo "已备份原配置：${BACKUP_FILE}"
fi

info "写入 KomariX Nginx 反向代理配置..."
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

if ! nginx -t; then
    if [[ -n "${BACKUP_FILE}" && -f "${BACKUP_FILE}" ]]; then
        cp -a "${BACKUP_FILE}" "${NGINX_CONF}"
    else
        rm -f "${NGINX_CONF}"
    fi
    die "Nginx 配置测试失败，已回滚本次修改。"
fi

systemctl enable nginx >/dev/null 2>&1 || true
systemctl restart nginx

echo
echo "=============================================="
echo "KomariX Nginx 反向代理配置完成"
echo "域名：http://${DOMAIN}"
echo "后端：http://127.0.0.1:${PORT}"
echo "配置：${NGINX_CONF}"
echo
echo "下一步可执行 SSL 一键脚本开启 HTTPS："
echo "bash <(curl -fsSL https://raw.githubusercontent.com/kkx999/KomariX/main/setup-https.sh)"
echo "=============================================="
