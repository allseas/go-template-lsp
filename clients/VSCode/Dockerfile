FROM golang:1.26.2-trixie

# Install system dependencies
RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y --no-install-recommends \
        xvfb \
        xauth \
        chromium \
        libnss3 \
        libatk1.0-0 \
        libatk-bridge2.0-0 \
        libcups2 \
        libdrm2 \
        libxkbcommon0 \
        libxcomposite1 \
        libxdamage1 \
        libxfixes3 \
        libxrandr2 \
        libgbm1 \
        libasound2 \
        curl \
        ca-certificates \
        unzip && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Install Node.js 24 directly from binary distribution
ENV NODE_MAJOR=24
RUN curl -fsSL https://nodejs.org/dist/latest-v${NODE_MAJOR}.x/SHASUMS256.txt -o /tmp/SHASUMS256.txt && \
    NODE_FILE=$(grep 'linux-x64.tar.gz' /tmp/SHASUMS256.txt | grep -v 'headers' | awk '{print $2}') && \
    curl -fsSL "https://nodejs.org/dist/latest-v${NODE_MAJOR}.x/${NODE_FILE}" -o /tmp/node.tar.gz && \
    tar -xzf /tmp/node.tar.gz -C /usr/local --strip-components=1 && \
    rm -f /tmp/node.tar.gz /tmp/SHASUMS256.txt

ENV PATH=/usr/local/bin:$PATH

# Install ts-node and typescript globally for build scripts
RUN npm install -g ts-node typescript
