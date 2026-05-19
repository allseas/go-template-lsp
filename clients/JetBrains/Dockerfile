FROM golang:tip-trixie

# Install system dependencies
RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y --no-install-recommends \
        openjdk-21-jdk \
        e2fsprogs \
        e2fslibs-dev \
        curl \
        ca-certificates \
        unzip && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Set JAVA_HOME
ENV JAVA_HOME=/usr/lib/jvm/java-21-openjdk-amd64

# Pre-install Gradle 9.0.0
ENV GRADLE_VERSION=9.0.0
ENV GRADLE_HOME=/opt/gradle
ENV GRADLE_USER_HOME=/root/.gradle
RUN curl -fsSL https://services.gradle.org/distributions/gradle-${GRADLE_VERSION}-bin.zip -o /tmp/gradle.zip && \
    unzip -q /tmp/gradle.zip -d /opt && \
    mv /opt/gradle-${GRADLE_VERSION} ${GRADLE_HOME} && \
    rm /tmp/gradle.zip && \
    mkdir -p /tmp/gradle-warmup && cd /tmp/gradle-warmup && \
    touch settings.gradle && \
    ${GRADLE_HOME}/bin/gradle wrapper --gradle-version ${GRADLE_VERSION} && \
    ./gradlew --version && \
    cd / && rm -rf /tmp/gradle-warmup
ENV PATH=${GRADLE_HOME}/bin:$PATH

# Install Node.js (without NVM, directly from NodeSource binary)
ENV NODE_MAJOR=24
RUN curl -fsSL https://nodejs.org/dist/latest-v${NODE_MAJOR}.x/SHASUMS256.txt -o /tmp/SHASUMS256.txt && \
    NODE_FILE=$(grep 'linux-x64.tar.gz' /tmp/SHASUMS256.txt | grep -v 'headers' | awk '{print $2}') && \
    curl -fsSL "https://nodejs.org/dist/latest-v${NODE_MAJOR}.x/${NODE_FILE}" -o /tmp/node.tar.gz && \
    tar -xzf /tmp/node.tar.gz -C /usr/local --strip-components=1 && \
    rm -f /tmp/node.tar.gz /tmp/SHASUMS256.txt

# Ensure /usr/local/bin is in PATH (for node/npm)
ENV PATH=/usr/local/bin:${GRADLE_HOME}/bin:$PATH

# Pre-warm Gradle dependencies (IntelliJ Platform, plugins, Maven artifacts)
# This avoids downloading ~1GB+ of dependencies on every CI run
COPY go-text-template/build.gradle.kts go-text-template/settings.gradle.kts go-text-template/gradle.properties /tmp/gradle-deps/
COPY go-text-template/gradle /tmp/gradle-deps/gradle
COPY go-text-template/gradlew /tmp/gradle-deps/
RUN cd /tmp/gradle-deps && \
    chmod +x gradlew && \
    ./gradlew dependencies --no-configuration-cache || true && \
    rm -rf /tmp/gradle-deps
