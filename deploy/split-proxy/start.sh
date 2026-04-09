#!/usr/bin/env bash
set -euo pipefail

: "${UPSTREAM_PROXY_HOST:?UPSTREAM_PROXY_HOST is required}"
: "${UPSTREAM_PROXY_PORT:?UPSTREAM_PROXY_PORT is required}"

direct_domains="${DIRECT_DOMAINS:-localhost host.docker.internal kirors-kiro}"
direct_cidrs="${DIRECT_CIDRS:-127.0.0.0/8 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16 169.254.0.0/16 100.64.0.0/10 ::1/128 fc00::/7 fe80::/10}"
squid_http_port="${SQUID_HTTP_PORT:-3128}"
log_dir="/var/log/squid"
spool_dir="/var/spool/squid"
access_log_path="${log_dir}/access.log"
cache_log_path="${log_dir}/cache.log"

peer_login=""
if [[ -n "${UPSTREAM_PROXY_LOGIN:-}" ]]; then
  peer_login=" login=${UPSTREAM_PROXY_LOGIN}"
fi

mkdir -p "${log_dir}" "${spool_dir}"
touch "${access_log_path}" "${cache_log_path}"

if id proxy >/dev/null 2>&1; then
  chown -R proxy:proxy "${log_dir}" "${spool_dir}"
fi

cat >/etc/squid/squid.conf <<EOF
http_port ${squid_http_port}
visible_hostname split-proxy

acl allowed_clients src 10.0.0.0/8
acl allowed_clients src 172.16.0.0/12
acl allowed_clients src 192.168.0.0/16
acl allowed_clients src 100.64.0.0/10
acl allowed_clients src fc00::/7

acl SSL_ports port 443
acl SSL_ports port 8443
acl Safe_ports port 80
acl Safe_ports port 21
acl Safe_ports port 443
acl Safe_ports port 70
acl Safe_ports port 210
acl Safe_ports port 280
acl Safe_ports port 488
acl Safe_ports port 591
acl Safe_ports port 777
acl Safe_ports port 1025-65535

acl direct_hosts dstdomain ${direct_domains}
acl direct_dst dst ${direct_cidrs}

http_access deny !Safe_ports
http_access deny CONNECT !SSL_ports
http_access allow allowed_clients
http_access allow localhost
http_access deny all

cache_peer ${UPSTREAM_PROXY_HOST} parent ${UPSTREAM_PROXY_PORT} 0 no-query default name=upstream_parent${peer_login}
cache_peer_access upstream_parent deny direct_hosts
cache_peer_access upstream_parent deny direct_dst
cache_peer_access upstream_parent allow all

always_direct allow direct_hosts
always_direct allow direct_dst
never_direct allow all

cache deny all
access_log stdio:${access_log_path}
cache_log stdio:${cache_log_path}
coredump_dir ${spool_dir}
pid_filename none
EOF

exec squid -N -f /etc/squid/squid.conf
