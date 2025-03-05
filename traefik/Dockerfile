FROM golang:1.21-alpine
WORKDIR /app
COPY main.go .
RUN go mod init config-server && \
    go mod tidy && \
    go build -o configserver
EXPOSE 9000
CMD ["./configserver"]