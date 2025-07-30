# Rinha de Backend 2025 - Arquitetura 2

Este projeto implementa a **Arquitetura 2** para a Rinha de Backend 2025, desenvolvido em Go. A evolução da Arquitetura 1 agora inclui **Redis Cache** e **Gateway Instance** rodando em paralelo com health checks automáticos.

## 🏗️ Arquitetura 2 - Diagrama

```
POST /payments → Redis Cache ← Gateway Instance (5s health checks)
                      ↓              ↓
            Decide Processor Gateway → Payment Processor Use Case
                      ↓                      ↓
            [Default/Fallback Processor] → Process Payment
                      ↓                      ↓
                 Cache Update         Success/Fails
                                           ↓
                                Saves Payment Info / Fail Safe
                                           ↓
                             TABELA: payments (payment_processor, amount)
```

## 🚀 Componentes da Arquitetura 2

### 1. **Redis Cache** 🆕
- Armazena o último gateway disponível 
- TTL de 30 segundos para performance
- Cache invalidation automático quando gateway fica down
- Chaves: `rinha:available_gateway`, `rinha:default_status`, `rinha:fallback_status`

### 2. **Gateway Instance** 🆕
- **Roda em paralelo** à aplicação principal
- **Health checks automáticos** a cada 5 segundos
- Atualiza o cache Redis automaticamente
- Graceful shutdown integrado

### 3. **Decide Processor Gateway** (Evoluído)
- **Cache-first approach**: Consulta Redis antes de verificar diretamente
- Fallback para verificação direta se cache não estiver disponível
- Logs detalhados com emojis para debugging

### 4. **Payment Processor Use Case** (Mantido)
- Mesma lógica de negócio da Arquitetura 1
- Compatível com o novo sistema de cache

### 5. **Process Payment + Persistence** (Mantido)
- Salva informações de pagamentos bem-sucedidos
- Fail Safe para tentativas com erro
- Tabela payments com todos os campos necessários

## 📊 Estrutura do Projeto

```
├── cmd/api/                    # Aplicação principal (Arquitetura 2)
├── internal/
│   ├── cache/                 # 🆕 Redis Cache management
│   ├── gateway/               # Gateway + Gateway Instance  
│   ├── usecase/               # Payment Processor Use Case  
│   ├── repository/            # Persistência de dados
│   ├── handler/               # Handlers HTTP
│   └── payment/               # Cliente para Payment Processors
├── docker-compose.yml         # Inclui Redis
├── nginx.conf                 # Load balancer otimizado
└── Dockerfile                 # Imagem Docker da aplicação
```

## 🛠️ Tecnologias Utilizadas

- **Go 1.24.5**: Linguagem de programação
- **Redis 7**: Cache para gateway decisions 🆕
- **PostgreSQL**: Banco de dados para persistência
- **Nginx**: Load balancer com algoritmo round-robin
- **Docker & Docker Compose**: Containerização
- **Alpine Linux**: Imagem base otimizada

## 🚀 Executando o Projeto

### 1. Pré-requisitos

- Docker e Docker Compose
- Payment Processors rodando nas portas 8001 (default) e 8002 (fallback)

### 2. Configurar Payment Processors (se necessário)

Configure manualmente os Payment Processors:
```bash
# Default Processor na porta 8001
git clone https://github.com/zanfranceschi/rinha-de-backend-2025.git processor-default
cd processor-default/payment-processor
PORT=8001 docker-compose up -d
cd ../..

# Fallback Processor na porta 8002  
git clone https://github.com/zanfranceschi/rinha-de-backend-2025.git processor-fallback
cd processor-fallback/payment-processor
PORT=8002 docker-compose up -d
cd ../..
```

### 3. Configurar variáveis de ambiente

```bash
cp config.env.example config.env
# Edite config.env conforme necessário
```

### 4. Executar a Arquitetura 2

```bash
# Subir toda a infraestrutura (inclui Redis)
docker-compose up --build
```

#### Para desenvolvimento local:
```bash
# Apenas a aplicação Go (requer postgres e redis rodando)
go run ./cmd/api
```

## 📡 Endpoints da API

### Endpoint Principal
- `POST /payments` - Processar pagamento com Arquitetura 2

### Endpoints Auxiliares  
- `GET /health` - Health check completo dos componentes
- `GET /payments/history?limit=10` - Histórico de pagamentos
- `GET /payments/stats` - Estatísticas dos processors
- `GET /payments-summary?from=YYYY-MM-DDTHH:mm:ss.sssZ&to=YYYY-MM-DDTHH:mm:ss.sssZ` - Resumo de pagamentos por período

### Exemplo de Payload (Rinha de Backend 2025)

```json
{
  "correlationId": "4a7901b8-7d26-4d9d-aa19-4dc1c7cf60b3",
  "amount": 19.90
}
```

### Exemplo de Resposta (Sucesso)

```json
{
  "success": true,
  "payment": {
    "id": "pay_123456",
    "correlationId": "4a7901b8-7d26-4d9d-aa19-4dc1c7cf60b3",
    "status": "completed",
    "amount": 19.90,
    "fee": 1.50,
    "processedAt": "2025-01-XX..."
  },
  "processor_used": "default",
  "processing_time": "245ms",
  "saved_to_db": true
}
```

## 🔍 Monitoramento e Debug

### Health Check Completo
```bash
curl http://localhost:9999/health
```

### Processar Pagamento
```bash
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{
    "correlationId": "4a7901b8-7d26-4d9d-aa19-4dc1c7cf60b3",
    "amount": 19.90
  }'
```

### Verificar Histórico  
```bash
curl http://localhost:9999/payments/history?limit=5
```

### Estatísticas dos Processors
```bash
curl http://localhost:9999/payments/stats
```

### Resumo de Pagamentos por Período
```bash
curl "http://localhost:9999/payments-summary?from=2025-01-01T00:00:00.000Z&to=2025-12-31T23:59:59.999Z"
```

### Monitorar Redis Cache 🆕
```bash
# Conectar no Redis
docker exec -it rinha-redis redis-cli

# Verificar chaves do cache
KEYS rinha:*

# Ver gateway disponível atual
GET rinha:available_gateway

# Ver status dos processors
GET rinha:default_status
GET rinha:fallback_status
```

### Logs da Aplicação
```bash
docker-compose logs -f api01 api02
```

### Acessar Banco de Dados
- **Adminer**: http://localhost:8080
- **Usuário**: rinha_user
- **Senha**: rinha_password
- **Banco**: rinha_payments

## 🎯 Recursos da Arquitetura 2

### ✅ Implementado (Novos da Arquitetura 2)

- **Redis Cache** com TTL e invalidação automática
- **Gateway Instance** rodando em paralelo
- **Health checks automáticos** a cada 5 segundos
- **Cache-first approach** para decisões de gateway
- **Graceful shutdown** integrado
- **Logs aprimorados** com emojis para debugging

### ✅ Mantido da Arquitetura 1

- **Payment Processor Use Case** com orquestração completa
- **Persistência de dados** com tabela payments
- **Fail Safe** para tentativas com erro
- **Load Balancing** Nginx com round-robin
- **Health Checks** para todos os componentes
- **Estatísticas** de uso dos processors
- **Payload correto** da Rinha de Backend 2025

### 🔧 Configurações de Recursos

- **CPU Total**: 1.5 unidades distribuídas
- **Memória Total**: 350MB distribuída
- **PostgreSQL**: 128MB + 0.25 CPU
- **Redis**: 64MB + 0.1 CPU 🆕
- **API (2 instâncias)**: 256MB + 0.65 CPU cada
- **Nginx**: 32MB + 0.1 CPU  
- **Adminer**: 32MB + 0.05 CPU

## 🌟 Diferenças da Arquitetura 1 para Arquitetura 2

| Aspecto | Arquitetura 1 | Arquitetura 2 |
|---------|---------------|---------------|
| **Gateway Decision** | Verificação direta a cada request | Cache Redis first, verificação sob demanda |
| **Health Checks** | Por request, síncronos | Background automático a cada 5s |
| **Performance** | ~50ms por decisão | ~5ms (cache hit) |
| **Resiliência** | Fail fast | Cache + fallback para verificação direta |
| **Observabilidade** | Logs básicos | Logs detalhados + cache monitoring |
| **Memória** | ~350MB | ~414MB (+Redis) |
| **Complexidade** | Simples | Moderada |

## 🐛 Troubleshooting

### Problemas Comuns

1. **Payment Processors não respondem**
   - Configurar manualmente os Payment Processors nas portas 8001/8002
   - Verificar se as portas 8001/8002 estão livres
   - Para Docker Desktop no Windows/Mac, trocar `172.17.0.1` por `host.docker.internal` no docker-compose.yml

2. **Redis connection failed**
   - Verificar se o container Redis está rodando
   - Checar logs: `docker-compose logs redis`
   - Verificar conectividade: `docker exec rinha-redis redis-cli ping`

3. **Gateway Instance não está funcionando**
   - Verificar logs: `docker-compose logs api01 api02`
   - Confirmar se Redis está acessível
   - Validar variável `REDIS_URL`

4. **Cache não está sendo atualizado**
   - Verificar se Gateway Instance está rodando
   - Monitorar chaves Redis: `docker exec rinha-redis redis-cli KEYS rinha:*`
   - Conferir TTL: `docker exec rinha-redis redis-cli TTL rinha:available_gateway`

### Comandos Úteis

```bash
# Rebuild completo
docker-compose down -v && docker-compose up --build

# Verificar status dos containers
docker-compose ps

# Logs de um serviço específico  
docker-compose logs -f redis

# Executar comando no banco
docker-compose exec postgres psql -U rinha_user -d rinha_payments

# Ver estrutura da tabela payments
docker-compose exec postgres psql -U rinha_user -d rinha_payments -c "\d payments"

# Monitorar Redis em tempo real
docker exec rinha-redis redis-cli MONITOR

# Limpar cache Redis
docker exec rinha-redis redis-cli FLUSHALL
```

## 📈 Próximas Etapas

A **Arquitetura 2** está completa e pronta para evolução adicional! 

Este projeto implementa uma solução robusta com cache inteligente, health checks automáticos e alta performance para processamento de pagamentos.

### Campos da Tabela Payments

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `id` | SERIAL | ID único do registro |
| `payment_id` | VARCHAR(255) | ID do pagamento retornado pelo processor |
| `correlation_id` | VARCHAR(255) | ID de correlação da requisição |
| `payment_processor` | VARCHAR(50) | Processor usado (default/fallback/none) |
| `amount` | DECIMAL(10,2) | Valor do pagamento |
| `status` | VARCHAR(20) | Status do pagamento |
| `fee` | DECIMAL(10,2) | Taxa cobrada |
| `error_message` | TEXT | Mensagem de erro (se houver) |
| `processed_at` | TIMESTAMP | Timestamp do processamento |
| `created_at` | TIMESTAMP | Timestamp de criação |

---

**Rinha de Backend 2025** - Arquitetura 2 implementada com ❤️ em Go + Redis 