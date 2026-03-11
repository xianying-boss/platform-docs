FROM ubuntu:22.04

LABEL maintainer="sandbox-platform"
LABEL description="GUI sandbox runner with Chromium, Playwright, Xvfb, and Office tools"

ENV DEBIAN_FRONTEND=noninteractive
ENV DISPLAY=:99
ENV PYTHONUNBUFFERED=1

# ── System packages ────────────────────────────────────────────────────────────
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Display server
    xvfb \
    x11vnc \
    xdotool \
    # Browser
    chromium-browser \
    chromium-chromedriver \
    # Python runtime
    python3 \
    python3-pip \
    python3-venv \
    # Office tools
    libreoffice \
    libreoffice-calc \
    libreoffice-writer \
    libreoffice-impress \
    # Utilities
    curl \
    wget \
    git \
    jq \
    ca-certificates \
    fonts-liberation \
    fonts-noto-cjk \
    # Clean up
    && rm -rf /var/lib/apt/lists/*

# ── Python dependencies ────────────────────────────────────────────────────────
COPY requirements.txt /tmp/requirements.txt
RUN pip3 install --no-cache-dir -r /tmp/requirements.txt

# ── Tool scripts ───────────────────────────────────────────────────────────────
COPY tools/ /tool/
RUN chmod +x /tool/*.py /tool/*.sh 2>/dev/null || true

# ── Entrypoint ─────────────────────────────────────────────────────────────────
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Create non-root user for running tools.
RUN useradd -m -u 1000 sandbox
USER sandbox

EXPOSE 5900

ENTRYPOINT ["/entrypoint.sh"]
