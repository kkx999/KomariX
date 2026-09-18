FROM alpine:3.21

WORKDIR /app

# Docker buildx 会在构建时自动填充这些变量
ARG TARGETOS
ARG TARGETARCH

COPY --chmod=755 komarix-agent-${TARGETOS}-${TARGETARCH} /app/komarix-agent

RUN touch /.komarix-agent-container

ENTRYPOINT ["/app/komarix-agent"]
# 运行时请指定参数
# Please specify parameters at runtime.
# eg: docker run komarix-agent -e example.com -t token
CMD ["--help"]
