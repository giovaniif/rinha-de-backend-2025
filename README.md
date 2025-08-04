# Payment Processor

Sistema de processamento de pagamentos com dois gateways (default e fallback) e load balancer.

## Arquitetura

- 2 instâncias da API Go
- Redis para cache de disponibilidade dos gateways
- Nginx como load balancer
- Health check automático dos payment processors

## Uso

1. Primeiro, suba os payment processors:
```bash
cd temp-rinha/payment-processor
docker-compose up -d
```

2. Depois, suba o sistema principal:
```bash
docker-compose up -d
```

## Endpoints

- `POST /payments` - Processa um pagamento
- `GET /payments-summary?from=<timestamp>&to=<timestamp>` - Resumo dos pagamentos

## Recursos

Total de recursos limitados a:
- CPU: 1.5 unidades
- Memória: 350MB

## Rede

Utiliza a rede `payment-processor` criada pelos payment processors.