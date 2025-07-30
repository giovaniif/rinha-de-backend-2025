# Rinha de Backend 2025 - Arquitetura 1

Este projeto implementa a **Arquitetura 1** para a Rinha de Backend 2025, desenvolvido em Go. O objetivo é criar um intermediador de pagamentos robusto que se conecta aos Payment Processors (default e fallback) com persistência de dados e mecanismos de fail-safe.

## 🏗️ Arquitetura 1 - Diagrama

```
POST /payments → Decide Processor Gateway → Payment Processor Use Case
                       ↓                              ↓
            [Default/Fallback Processor]    → Process Payment
                       ↓                              ↓
                  IS UP? ─────────────────────→ Success/Fails
                                                      ↓
                                           Save Payment Info / Fail Safe
                                                      ↓
                                          TABELA: payments (payment_processor, amount)
```

## 🚀 Componentes da Arquitetura

### 1. **Decide Processor Gateway**
- Verifica se o Default Processor está UP
- Se Default estiver DOWN, verifica o Fallback
- Retorna qual processor usar ou erro se ambos estiverem DOWN

### 2. **Payment Processor Use Case** 
- Orquestra todo o fluxo de processamento de pagamentos
- Implementa a lógica de negócio da Arquitetura 1
- Gerencia Success/Fails paths

### 3. **Process Payment**
- Realiza o processamento do pagamento no processor selecionado
- Implementa timeout e retry logic

### 4. **Save Payment Info & Fail Safe**
- Persiste informações de pagamentos bem-sucedidos
- Implementa mecanismo de fail-safe para erros
- Salva tentativas e estatísticas no banco

### 5. **Tabela Payments**
- `payment_processor`: Qual processor foi usado (default/fallback/none)
- `amount`: Valor do pagamento
- `correlation_id`: ID de correlação único
- Campos adicionais: status, timestamps, etc.

## 📊 Estrutura do Projeto

```
├── cmd/api/                    # Aplicação principal
├── internal/
│   ├── gateway/               # Decide Processor Gateway
│   ├── usecase/               # Payment Processor Use Case  
│   ├── repository/            # Persistência de dados
│   ├── handler/               # Handlers HTTP
│   └── payment/               # Cliente para Payment Processors
├── docker-compose.yml         # Configuração completa
├── nginx.conf                 # Load balancer
└── Dockerfile                 # Imagem Docker da aplicação
```

## 🛠️ Tecnologias Utilizadas

- **Go 1.24.5**: Linguagem de programação
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

### 4. Executar a Arquitetura 1

```bash
# Subir toda a infraestrutura
docker-compose up --build
```

#### Para desenvolvimento local:
```bash
# Apenas a aplicação Go (requer postgres rodando)
go run ./cmd/api
```

## 📡 Endpoints da API

### Endpoint Principal
- `POST /payments` - Processar pagamento com Arquitetura 1

### Endpoints Auxiliares  
- `GET /health` - Health check completo dos componentes
- `GET /payments/history?limit=10` - Histórico de pagamentos
- `GET /payments/stats` - Estatísticas dos processors

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

### Logs da Aplicação
```bash
docker-compose logs -f api01 api02
```

### Acessar Banco de Dados
- **Adminer**: http://localhost:8080
- **Usuário**: rinha_user
- **Senha**: rinha_password
- **Banco**: rinha_payments

## 🎯 Recursos da Arquitetura 1

### ✅ Implementado

- **Decide Processor Gateway** com verificação de status
- **Payment Processor Use Case** com orquestração completa
- **Persistência de dados** com tabela payments
- **Fail Safe** para tentativas com erro
- **Load Balancing** Nginx com round-robin
- **Health Checks** para todos os componentes
- **Estatísticas** de uso dos processors
- **Logs estruturados** para debug
- **Payload correto** da Rinha de Backend 2025
- **Nginx simplificado** com configuração mínima
- **URLs de processors corrigidas** (172.17.0.1 para Docker Linux)

### 🔧 Configurações de Recursos

- **CPU Total**: 1.5 unidades distribuídas
- **Memória Total**: 350MB distribuída
- **PostgreSQL**: 128MB + 0.25 CPU
- **API (2 instâncias)**: 200MB + 0.6 CPU cada
- **Nginx**: 10MB + 0.17 CPU  
- **Adminer**: 32MB + 0.1 CPU

## 🐛 Troubleshooting

### Problemas Comuns

1. **Payment Processors não respondem**
   - Verificar se estão rodando nas portas 8001/8002
   - Checar logs: `docker-compose logs`

2. **Erro de conexão com banco**
   - Aguardar inicialização completa do PostgreSQL
   - Verificar credenciais no config.env

3. **Nginx não consegue acessar APIs**
   - Verificar se as APIs estão healthy
   - Checar network do Docker

4. **Payment Processors não respondem**
   - Configurar manualmente os Payment Processors nas portas 8001/8002
   - Verificar se as portas 8001/8002 estão livres
   - Para Docker Desktop no Windows/Mac, trocar `172.17.0.1` por `host.docker.internal` no docker-compose.yml

### Comandos Úteis

```bash
# Rebuild completo
docker-compose down -v && docker-compose up --build

# Verificar status dos containers
docker-compose ps

# Logs de um serviço específico  
docker-compose logs -f postgres

# Executar comando no banco
docker-compose exec postgres psql -U rinha_user -d rinha_payments

# Ver estrutura da tabela payments
docker-compose exec postgres psql -U rinha_user -d rinha_payments -c "\d payments"
```

## 📈 Próximas Etapas

A **Arquitetura 1** está completa e pronta para as próximas evoluções! 

Este projeto implementa todos os componentes descritos no diagrama da Arquitetura 1, oferecendo uma base sólida para processamento de pagamentos com alta disponibilidade, persistência de dados e mecanismos de recuperação de falhas.

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

**Rinha de Backend 2025** - Arquitetura 1 implementada com ❤️ em Go 