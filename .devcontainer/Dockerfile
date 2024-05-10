# You can use any debian based image you want
FROM mcr.microsoft.com/vscode/devcontainers/go:1.20-bullseye

RUN VERSION=$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt) \
    && curl -Lo kubectl "https://storage.googleapis.com/kubernetes-release/release/$VERSION/bin/linux/amd64/kubectl" \
    && install kubectl /usr/local/bin/kubectl
