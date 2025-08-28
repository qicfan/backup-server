# 运行阶段
FROM alpine:latest
# 设置时区
ENV TZ=Asia/Shanghai
ENV PATH=/app:$PATH
# 安装依赖
RUN apk add --update libheif
RUN apk add --update ffmpeg
RUN apk add --update imagemagick
ENV GIN_MODE=release
# 设置工作目录
WORKDIR /app
# 创建必要的目录
COPY BACKUP-SERVER .
VOLUME ["/app/config", "/upload"]
# 暴露端口
EXPOSE 12334
# 启动命令
CMD ["./BACKUP-SERVER"]