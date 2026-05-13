#!/bin/bash
# entrypoint.sh - Запуск Locust с динамической конфигурацией

# Ждем создания конфиг-файла
CONFIG_FILE=${CONFIG_FILE:-/config/config.env}
REFRESH_INTERVAL=${REFRESH_INTERVAL:-5}

echo "=== Locust Load Generator ==="
echo "Config file: $CONFIG_FILE"
echo "Refresh interval: ${REFRESH_INTERVAL}s"
echo "Frontend address: ${FRONTEND_ADDR:-http://localhost:8080}"

# Запускаем Locust в фоне
if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
    echo "Loaded config: USERS=${USERS:-1}, RATE=${RATE:-10}"
fi

# Запускаем Locust с Web UI (для мониторинга) или в headless режиме
if [ "$HEADLESS" = "true" ]; then
    # Headless режим (без Web UI)
    locust -f /home/locust/locustfile.py \
           --host=${FRONTEND_ADDR:-http://localhost:8080} \
           --users=${USERS:-10} \
           --spawn-rate=${RATE:-10} \
           --run-time=${RUN_TIME:-60s} \
           --headless \
           --html=/tmp/report.html \
           --loglevel=${LOG_LEVEL:-INFO}
else
    # Web UI режим (для динамического управления)
    locust -f /home/locust/locustfile.py \
           --host=${FRONTEND_ADDR:-http://localhost:8080} \
           --web-port=8089 \
           --web-host=0.0.0.0
fi