# Build stage
FROM golang:1.24.5-alpine AS builder

# Instalar dependências necessárias
RUN apk add --no-cache git ca-certificates curl

# Configurar diretório de trabalho
WORKDIR /app

# Copiar arquivos de dependências
COPY go.mod go.sum ./

# Baixar dependências
RUN go mod download

# Copiar código fonte
COPY . .

# Compilar aplicação (sem CGO para evitar problemas de dependências)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# Runtime stage
FROM alpine:latest

# Instalar ca-certificates e curl para health checks
RUN apk --no-cache add ca-certificates curl

# Criar usuário não-root
RUN adduser -D -s /bin/sh appuser

# Configurar diretório de trabalho
WORKDIR /root/

# Copiar binário da aplicação
COPY --from=builder /app/main .

# Copiar arquivo de configuração de exemplo
COPY --from=builder /app/config.env.example ./config.env

# Mudar ownership dos arquivos
RUN chown -R appuser:appuser /root/

# Usar usuário não-root
USER appuser

# Expor porta padrão
EXPOSE 8080

# Labels para metadados
LABEL maintainer="Rinha Backend 2025"
LABEL version="1.0.0"
LABEL description="Arquitetura 1 - Payment Processor Gateway"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Comando para executar a aplicação
CMD ["./main"] 