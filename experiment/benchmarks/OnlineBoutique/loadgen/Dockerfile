# Dockerfile - упрощенная версия без лишних файлов
FROM locustio/locust:latest

# Устанавливаем дополнительные инструменты
USER root
RUN apt-get update && apt-get install -y \
    curl \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Создаем директории
RUN mkdir -p /config /home/locust

# Копируем только существующие файлы
COPY locustfile.py /home/locust/locustfile.py

# Устанавливаем права
RUN chown -R locust:locust /home/locust /config

# Создаем entrypoint скрипт
RUN echo '#!/bin/bash\n\
CONFIG_FILE=${CONFIG_FILE:-/config/config.env}\n\
REFRESH_INTERVAL=${REFRESH_INTERVAL:-5}\n\
\n\
echo "=== Locust Load Generator ===\n\
Config file: $CONFIG_FILE\n\
Refresh interval: ${REFRESH_INTERVAL}s\n\
Frontend address: ${FRONTEND_ADDR:-http://localhost:8080}"\n\
\n\
if [ -f "$CONFIG_FILE" ]; then\n\
    source "$CONFIG_FILE"\n\
    echo "Loaded config: USERS=${USERS:-1}, RATE=${RATE:-10}"\n\
fi\n\
\n\
if [ "$HEADLESS" = "true" ]; then\n\
    echo "Starting in headless mode for ${RUN_TIME:-60s}..."\n\
    locust -f /home/locust/locustfile.py \\\n\
           --host=${FRONTEND_ADDR:-http://localhost:8080} \\\n\
           --users=${USERS:-10} \\\n\
           --spawn-rate=${RATE:-10} \\\n\
           --run-time=${RUN_TIME:-60s} \\\n\
           --headless \\\n\
           --html=/tmp/report.html \\\n\
           --loglevel=${LOG_LEVEL:-INFO}\n\
else\n\
    echo "Starting with Web UI on port 8089..."\n\
    locust -f /home/locust/locustfile.py \\\n\
           --host=${FRONTEND_ADDR:-http://localhost:8080} \\\n\
           --web-port=8089 \\\n\
           --web-host=0.0.0.0\n\
fi' > /entrypoint.sh && chmod +x /entrypoint.sh

USER locust

ENTRYPOINT ["/entrypoint.sh"]