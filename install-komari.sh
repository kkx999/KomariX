#!/bin/bash
# Legacy compatibility entrypoint. New installs should use install-komarix.sh.
set -e
exec bash <(curl -fsSL https://raw.githubusercontent.com/kkx999/KomariX/main/install-komarix.sh) "$@"
