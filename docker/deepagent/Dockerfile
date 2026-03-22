FROM golang:1.25-bookworm

RUN apt-get update && \
    apt-get install -y --no-install-recommends bash git ripgrep curl ca-certificates && \
    rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
    apt-get update && \
    apt-get install -y --no-install-recommends nodejs && \
    corepack enable && \
    corepack prepare pnpm@latest --activate && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

CMD ["sleep", "infinity"]
