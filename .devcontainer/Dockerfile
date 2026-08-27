FROM golang:1.26.6

RUN apt-get update && apt-get install -y \
    make \
    curl \
    wget \
    vim \
    jq \
    unzip \
    bash-completion \
    && rm -rf /var/lib/apt/lists/*

RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/

RUN curl -Lo ./kind "https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64" && \
    chmod +x ./kind && \
    mv ./kind /usr/local/bin/

RUN curl -L https://istio.io/downloadIstio | sh - && \
    cd istio-* && \
    mv bin/istioctl /usr/local/bin/ && \
    cd .. && \
    rm -rf istio-*

RUN curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 && \
    chmod 700 get_helm.sh && \
    ./get_helm.sh && \
    rm get_helm.sh

RUN useradd -m -u 1000 vscode && \
    chown -R vscode:vscode /go

RUN echo "PS1='\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ '" >> /home/vscode/.bashrc && \
    echo "alias ll='ls -alF'" >> /home/vscode/.bashrc && \
    echo "alias la='ls -A'" >> /home/vscode/.bashrc && \
    echo "alias l='ls -CF'" >> /home/vscode/.bashrc && \
    echo "alias k='kubectl'" >> /home/vscode/.bashrc && \
    echo "alias kg='kubectl get'" >> /home/vscode/.bashrc && \
    echo "alias kd='kubectl describe'" >> /home/vscode/.bashrc && \
    echo "source /usr/share/bash-completion/bash_completion" >> /home/vscode/.bashrc && \
    echo "source <(kubectl completion bash)" >> /home/vscode/.bashrc && \
    echo "source <(istioctl completion bash)" >> /home/vscode/.bashrc && \
    echo "source <(helm completion bash)" >> /home/vscode/.bashrc && \
    chown vscode:vscode /home/vscode/.bashrc

USER vscode
WORKDIR /workspaces/CoolPA

RUN istioctl version --remote=false && kubectl version --client && kind version
FROM golang:1.26.6

RUN apt-get update && apt-get install -y \
    make \
    curl \
    wget \
    vim \
    jq \
    unzip \
    bash-completion \
    && rm -rf /var/lib/apt/lists/*

RUN curl -LO "https://dl.k8s.io/release/v1.35.3/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/

RUN curl -Lo ./kind "https://kind.sigs.k8s.io/dl/v0.23.0/kind-linux-amd64" && \
    chmod +x ./kind && \
    mv ./kind /usr/local/bin/

RUN curl -L https://github.com/istio/istio/releases/download/1.28.0/istio-1.28.0-linux-amd64.tar.gz | tar xz && \
    mv istio-1.28.0/bin/istioctl /usr/local/bin/ && \
    rm -rf istio-1.28.0

RUN curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 && \
    chmod 700 get_helm.sh && \
    ./get_helm.sh && \
    rm get_helm.sh

RUN useradd -m -u 1000 vscode && \
    chown -R vscode:vscode /go

RUN echo "PS1='\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ '" >> /home/vscode/.bashrc && \
    echo "alias ll='ls -alF'" >> /home/vscode/.bashrc && \
    echo "alias la='ls -A'" >> /home/vscode/.bashrc && \
    echo "alias l='ls -CF'" >> /home/vscode/.bashrc && \
    echo "alias k='kubectl'" >> /home/vscode/.bashrc && \
    echo "alias kg='kubectl get'" >> /home/vscode/.bashrc && \
    echo "alias kd='kubectl describe'" >> /home/vscode/.bashrc && \
    echo "source /usr/share/bash-completion/bash_completion" >> /home/vscode/.bashrc && \
    echo "source <(kubectl completion bash)" >> /home/vscode/.bashrc && \
    echo "source <(istioctl completion bash)" >> /home/vscode/.bashrc && \
    echo "source <(helm completion bash)" >> /home/vscode/.bashrc && \
    chown vscode:vscode /home/vscode/.bashrc

USER vscode
WORKDIR /workspaces/CoolPA
