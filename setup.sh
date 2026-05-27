#!/bin/bash
set -e

# ==============================================================================
# КОНФИГУРАЦИЯ
# ==============================================================================
APP_NAME="honeypot"
INSTALL_DIR="/opt/$APP_NAME"
SERVICE_FILE="/etc/systemd/system/honeypot.service"
GO_VERSION="1.25.4"

# Внимание: Укажите здесь свой репозиторий в формате "user/repo",
# откуда будут качаться релизы (например "vlessenc/go_videostream")
GITHUB_REPO="username/repository"
# ==============================================================================

# Проверка root-прав
if [ "$EUID" -ne 0 ]; then
  echo "[-] Пожалуйста, запустите скрипт с правами root (sudo ./setup.sh)"
  exit 1
fi

install_go() {
    echo "[*] Проверка установки Golang..."
    if ! command -v go &> /dev/null; then
        echo "[*] Go не найден. Установка Go $GO_VERSION..."
        wget -q --show-progress "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
        rm -rf /usr/local/go
        tar -C /usr/local -xzf /tmp/go.tar.gz
        ln -sf /usr/local/go/bin/go /usr/bin/go
        rm /tmp/go.tar.gz
        echo "[+] Go успешно установлен."
    else
        echo "[+] Go уже установлен: $(go version)"
    fi
}

build_app() {
    echo "[*] Сборка приложения из исходников..."
    go mod tidy
    go build -o honeypot cmd/honeypot/main.go
    echo "[+] Сборка завершена."
}

setup_systemd() {
    echo "[*] Настройка systemd сервиса..."
    mkdir -p "$INSTALL_DIR"
    cp honeypot "$INSTALL_DIR/honeypot"
    
    # Копируем конфиг только если его нет, чтобы не затереть пользовательские настройки при обновлении
    if [ ! -f "$INSTALL_DIR/config.yaml" ]; then
        if [ -f "config.yaml" ]; then
            cp config.yaml "$INSTALL_DIR/config.yaml"
        else
            echo "[!] Файл config.yaml не найден в текущей директории! Поместите его вручную в $INSTALL_DIR/config.yaml"
        fi
    fi

    cat <<EOF > "$SERVICE_FILE"
[Unit]
Description=Go-Honeypot VideoStream Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/honeypot -config $INSTALL_DIR/config.yaml
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable honeypot
    systemctl restart honeypot
    echo "[+] Сервис honeypot успешно установлен и запущен!"
    echo "    Проверка логов: journalctl -u honeypot -f"
}

update_github() {
    echo "[*] Обновление до последней версии из GitHub Releases..."
    
    if [ "$GITHUB_REPO" == "username/repository" ]; then
        echo "[-] Ошибка: Вы не указали GITHUB_REPO внутри файла setup.sh"
        exit 1
    fi
    
    # Получаем URL последнего релиза
    LATEST_URL=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep "browser_download_url" | grep "linux-amd64" | cut -d '"' -f 4 | head -n 1)
    
    if [ -z "$LATEST_URL" ]; then
        echo "[-] Ошибка: Не удалось найти архив (linux-amd64) для релиза в $GITHUB_REPO"
        exit 1
    fi

    echo "[*] Скачивание $LATEST_URL ..."
    wget -q --show-progress "$LATEST_URL" -O /tmp/honeypot_latest
    chmod +x /tmp/honeypot_latest

    echo "[*] Остановка старого сервиса..."
    systemctl stop honeypot || true

    mkdir -p "$INSTALL_DIR"
    mv /tmp/honeypot_latest "$INSTALL_DIR/honeypot"

    echo "[*] Запуск сервиса..."
    systemctl start honeypot
    echo "[+] Успешно обновлено из GitHub (Systemd сервис запущен)!"
}

echo "=========================================================="
echo "          Установка и Управление Go-Honeypot              "
echo "=========================================================="
echo "1) Установить с нуля (Установит Go + Сборка + Systemd)"
echo "2) Пересобрать локально (Соберет код и перезапустит сервис)"
echo "3) Обновить через GitHub (Скачает последний собранный релиз)"
echo "4) Полностью удалить сервис"
echo "0) Выход"
echo "=========================================================="
read -p "Выберите действие [0-4]: " ACTION

case $ACTION in
    1)
        install_go
        build_app
        setup_systemd
        ;;
    2)
        install_go
        build_app
        systemctl stop honeypot || true
        cp honeypot "$INSTALL_DIR/honeypot"
        systemctl start honeypot
        echo "[+] Локальное обновление завершено!"
        ;;
    3)
        update_github
        ;;
    4)
        systemctl stop honeypot || true
        systemctl disable honeypot || true
        rm -f "$SERVICE_FILE"
        systemctl daemon-reload
        echo "[+] Сервис honeypot успешно удален из системы."
        ;;
    0)
        exit 0
        ;;
    *)
        echo "[-] Неверный выбор."
        ;;
esac
