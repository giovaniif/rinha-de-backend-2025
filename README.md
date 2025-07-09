# Rinha de Backend 2025 - Go

Este é um projeto para a Rinha de Backend 2025 desenvolvido em Go. O objetivo é criar um intermediador de pagamentos que se conecta aos Payment Processors (default e fallback).

## Estrutura do Projeto

```
├── cmd/api/           # Aplicação principal
├── internal/
│   ├── handler/       # Handlers HTTP
│   └── payment/       # Cliente para Payment Processors
├── docker-compose.yml # Configuração dos containers
├── nginx.conf         # Configuração do load balancer
└── Dockerfile         # Imagem Docker da aplicação
```

## Funcionalidades

- **Health Check**: Endpoint `/health` que verifica o status dos serviços
- **Processamento de Pagamentos**: Endpoint `/payments` que processa pagamentos
- **Failover**: Tenta primeiro o Payment Processor default, se falhar usa o fallback
- **Load Balancing**: Nginx distribui requisições entre duas instâncias da API

## Executando o Projeto

### 1. Baixar o Payment Processor

```bash
git clone https://github.com/zanfranceschi/rinha-de-backend-2025.git temp-rinha
cd temp-rinha/payment-processor
docker-compose up -d
cd ../..
```

### 2. Configurar as variáveis de ambiente

```bash
cp config.env.example config.env
```

Edite o arquivo `config.env` conforme necessário.

### 3. Executar a aplicação

#### Para desenvolvimento local:
```bash
go run ./cmd/api
```

#### Para produção com Docker:
```bash
docker-compose up --build
```

## Endpoints

- `GET /health` - Health check dos serviços
- `POST /payments` - Processar pagamento

### Exemplo de Payload

```json
{
  "amount": 100.00,
  "currency": "BRL",
  "description": "Pagamento teste"
}
```

## Tecnologias Utilizadas

- **Go**: Linguagem de programação
- **Nginx**: Load balancer
- **Docker**: Containerização
- **Alpine Linux**: Imagem base

## Recursos Utilizados

- **CPU**: 1.5 unidades (distribuído entre os serviços)
- **Memória**: 350MB (distribuído entre os serviços) 