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
RUN curl -fsSL https://services.gradle.org/distributions/gradle-${GRADLE_VERSION}-bin.zip -o /tmp/gradle.zip && \
    unzip -q /tmp/gradle.zip -d /opt && \
    mv /opt/gradle-${GRADLE_VERSION} ${GRADLE_HOME} && \
    rm /tmp/gradle.zip
ENV PATH=${GRADLE_HOME}/bin:$PATH

# Install NVM and Node.js
ENV NVM_DIR=/root/.nvm
ENV NODE_VERSION=24

RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.4/install.sh | bash && \
    . "$NVM_DIR/nvm.sh" && \
    nvm install ${NODE_VERSION} && \
    nvm use ${NODE_VERSION} && \
    npm install npm -g

# Make node/npm available in PATH for non-interactive shells
ENV PATH=$NVM_DIR/versions/node/v${NODE_VERSION}/bin:$PATH
