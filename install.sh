#!/usr/bin/env sh
#
# Script to install Better DNS as a service on Linux systems.
#
# Primary goal is to support Raspberry Pi devices on Raspbian, but others may be supported in the future.
#

# Defaults and static variables
ARCH="unknown"
BINARY="better-dns-unknown"
BINARY_DST="/usr/sbin/better-dns"
CONFIG="/etc/better-dns.yaml"
OS="linux"
SOURCE="https://github.com/lietu/better-dns/releases/latest/download/"

# Composite variables
CONFIG_SOURCE="$SOURCE/better-dns-server.yaml"


# ----- TEMPLATES ----- #

function make_systemd_config {
  cat <<-EOF
[Unit]
Description=Better DNS
After=network.target

[Service]
ExecStart=$BINARY_DST -config $CONFIG
ExecStop=kill -2 \$MAINPID
Restart=always
User=root
Type=idle

[Install]
WantedBy=multi-user.target
EOF
}

# ----- FUNCTIONS ----- #

function detect_arch() {
  ARCH=$(uname -m)

  # TODO: Probably need a lot more special cases here
  if [[ "$ARCH" == "armv7l" ]]; then
    ARCH="arm"
  fi

  if [[ "$ARCH" == "x86_64" ]]; then
    ARCH="amd64"
  fi

  echo "Detected architecture as $ARCH"
}

function pick_binary() {
  BINARY="better-dns-${OS}-${ARCH}"
  echo "Will use $BINARY"
}

function download() {
  src="$1"
  dst="$2"

  echo "Downloading $src to $dst"

  if (which curl >/dev/null); then
    curl -L -o "$dst" "$src"
  elif (which wget >/dev/null); then
    wget "$src" -O "$dst"
  else
    echo "No curl or wget found, dunno how to download the binary."
    exit 1
  fi
}

function install_binary() {
  download "$SOURCE/$BINARY" "$BINARY_DST"
  chmod 0700 "$BINARY_DST"
}

function install_config() {
  download "$CONFIG_SOURCE" "$CONFIG"
  chmod 0600 "$CONFIG"
}

function install_service() {
  if (which systemctl >/dev/null); then
    SERVICE="better-dns.service"
    SERVICE_CONF="/etc/systemd/system/$SERVICE"

    echo "Installing systemd service $SERVICE to $SERVICE_CONF"

    make_systemd_config > "$SERVICE_CONF"
    chmod 0600 "$SERVICE_CONF"

    echo "Enabling and starting service via systemd"
    systemctl enable "$SERVICE"
    systemctl start "$SERVICE"

    echo "To update configuration after changes run:"
    echo "  systemctl restart $SERVICE"
  else
    echo "Don't know how to install service."
    exit 1
  fi
}

# ----- LOGIC ----- #

# Try to self-elevate privileges if needed
if [ "$UID" -ne 0 ]; then
  # Try to self-elevate
  echo "Root privileges required for installation, trying to self-elevate via sudo."
  exec sudo bash "$0" "$@"
fi

# Detect where we're running on
detect_arch
pick_binary

# Install
install_binary
install_config
install_service

echo
echo "Binary is installed at $BINARY_DST"
echo "Configuration is at $CONFIG"
echo
echo "All done! Enjoy Better DNS."
