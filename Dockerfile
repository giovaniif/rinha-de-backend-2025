FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copiar arquivos de dependências
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fonte
COPY . .

# Compilar o aplicativo
RUN go build -o main ./cmd/api

# Imagem final
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copiar o binário
COPY --from=builder /app/main .

# Expor a porta
EXPOSE 8080

# Comando para executar
CMD ["./main"] 