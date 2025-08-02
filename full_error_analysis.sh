#!/bin/bash

echo "🔬 ANÁLISE COMPLETA DE ERROS PÓS-TESTE K6"
echo "=========================================="
echo "Data/Hora: $(date)"
echo

# Criar diretório para relatórios
mkdir -p reports

# Arquivo do relatório
REPORT_FILE="reports/error_report_$(date +%Y%m%d_%H%M%S).md"

echo "📋 Gerando relatório em: $REPORT_FILE"
echo

# Função para gerar relatório markdown
generate_report() {
    cat > $REPORT_FILE << EOF
# 🔍 Relatório de Análise de Erros - Teste K6

**Data:** $(date)  
**Duração do Teste:** Último período de logs analisado

## 📊 Resumo Executivo

### Principais Métricas de Erro
EOF

    echo "### HTTP Errors (Status >= 400)" >> $REPORT_FILE
    HTTP_ERRORS=$(docker-compose logs api1 api2 | grep "HTTP_ERROR" | wc -l)
    echo "- **Total HTTP Errors:** $HTTP_ERRORS" >> $REPORT_FILE
    
    echo "" >> $REPORT_FILE
    echo "### Errors por Status Code" >> $REPORT_FILE
    echo "- **Status 503 (Service Unavailable):** $(docker-compose logs api1 api2 | grep "status=503" | wc -l)" >> $REPORT_FILE
    echo "- **Status 500 (Internal Server Error):** $(docker-compose logs api1 api2 | grep "status=500" | wc -l)" >> $REPORT_FILE
    echo "- **Status 400 (Bad Request):** $(docker-compose logs api1 api2 | grep "status=400" | wc -l)" >> $REPORT_FILE
    
    echo "" >> $REPORT_FILE
    echo "### Payment Processing Errors" >> $REPORT_FILE
    PAYMENT_FAILED=$(docker-compose logs api1 api2 | grep "PAYMENT_FAILED" | wc -l)
    NO_GATEWAY=$(docker-compose logs api1 api2 | grep "ERROR_NO_GATEWAY\|GATEWAY_UNAVAILABLE" | wc -l)
    QUEUE_FULL=$(docker-compose logs api1 api2 | grep "ERROR_QUEUE_FULL" | wc -l)
    INTERNAL_ERR=$(docker-compose logs api1 api2 | grep "ERROR_INTERNAL" | wc -l)
    
    echo "- **Payment Failed:** $PAYMENT_FAILED" >> $REPORT_FILE
    echo "- **No Gateway Available:** $NO_GATEWAY" >> $REPORT_FILE
    echo "- **Queue Full:** $QUEUE_FULL" >> $REPORT_FILE
    echo "- **Internal Errors:** $INTERNAL_ERR" >> $REPORT_FILE
    
    echo "" >> $REPORT_FILE
    echo "### Health Check Issues" >> $REPORT_FILE
    HEALTH_TIMEOUTS=$(docker-compose logs api1 api2 | grep "Health check failed" | wc -l)
    echo "- **Health Check Timeouts:** $HEALTH_TIMEOUTS" >> $REPORT_FILE
    
    echo "" >> $REPORT_FILE
    echo "## 🔍 Análise Detalhada" >> $REPORT_FILE
    echo "" >> $REPORT_FILE
    
    # Análise do problema principal
    if [ "$NO_GATEWAY" -gt "$PAYMENT_FAILED" ]; then
        echo "### ❌ PROBLEMA PRINCIPAL: Gateway Unavailable" >> $REPORT_FILE
        echo "" >> $REPORT_FILE
        echo "**Diagnóstico:** A maioria dos erros ($NO_GATEWAY de $HTTP_ERRORS) é causada por gateways indisponíveis." >> $REPORT_FILE
        echo "" >> $REPORT_FILE
        echo "**Possíveis Causas:**" >> $REPORT_FILE
        echo "- Payment processors sobrecarregados" >> $REPORT_FILE
        echo "- Rate limiting dos health checks" >> $REPORT_FILE
        echo "- Timeouts de rede" >> $REPORT_FILE
        echo "- Problemas de conectividade" >> $REPORT_FILE
        
    elif [ "$PAYMENT_FAILED" -gt 0 ]; then
        echo "### ❌ PROBLEMA PRINCIPAL: Payment Processing Failures" >> $REPORT_FILE
        echo "" >> $REPORT_FILE
        echo "**Diagnóstico:** $PAYMENT_FAILED falhas no processamento de pagamentos." >> $REPORT_FILE
        echo "" >> $REPORT_FILE
        echo "**Possíveis Causas:**" >> $REPORT_FILE
        echo "- Payment processors retornando status 400/500" >> $REPORT_FILE
        echo "- Problemas de autenticação" >> $REPORT_FILE
        echo "- Payload inválido" >> $REPORT_FILE
        echo "- Rate limiting nos payment processors" >> $REPORT_FILE
    fi
    
    if [ "$QUEUE_FULL" -gt 0 ]; then
        echo "" >> $REPORT_FILE
        echo "### ⚠️ PROBLEMA SECUNDÁRIO: Queue Overflow" >> $REPORT_FILE
        echo "" >> $REPORT_FILE
        echo "**Diagnóstico:** $QUEUE_FULL casos de fila cheia." >> $REPORT_FILE
        echo "" >> $REPORT_FILE
        echo "**Impacto:** Requests rejeitadas com 503 Service Unavailable." >> $REPORT_FILE
        echo "**Solução:** Aumentar capacidade da fila ou workers." >> $REPORT_FILE
    fi
    
    echo "" >> $REPORT_FILE
    echo "## 📈 Recomendações de Melhoria" >> $REPORT_FILE
    echo "" >> $REPORT_FILE
    
    if [ "$HEALTH_TIMEOUTS" -gt 50 ]; then
        echo "### 1. Otimizar Health Checks" >> $REPORT_FILE
        echo "- Aumentar timeout dos health checks" >> $REPORT_FILE
        echo "- Implementar exponential backoff" >> $REPORT_FILE
        echo "- Reduzir frequência quando detectar problemas" >> $REPORT_FILE
        echo "" >> $REPORT_FILE
    fi
    
    if [ "$NO_GATEWAY" -gt 100 ]; then
        echo "### 2. Melhorar Resilência de Gateway" >> $REPORT_FILE
        echo "- Implementar circuit breaker" >> $REPORT_FILE
        echo "- Cache de última disponibilidade conhecida" >> $REPORT_FILE
        echo "- Fallback mais inteligente" >> $REPORT_FILE
        echo "" >> $REPORT_FILE
    fi
    
    if [ "$QUEUE_FULL" -gt 10 ]; then
        echo "### 3. Otimizar Fila de Processamento" >> $REPORT_FILE
        echo "- Aumentar tamanho da fila (atual: 1000)" >> $REPORT_FILE
        echo "- Aumentar número de workers (atual: 4)" >> $REPORT_FILE
        echo "- Implementar backpressure" >> $REPORT_FILE
        echo "" >> $REPORT_FILE
    fi
    
    echo "### 4. Performance Geral" >> $REPORT_FILE
    echo "- Otimizar timeouts HTTP" >> $REPORT_FILE
    echo "- Implementar connection pooling" >> $REPORT_FILE
    echo "- Adicionar métricas de performance" >> $REPORT_FILE
    echo "- Otimizar serialização JSON" >> $REPORT_FILE
    echo "" >> $REPORT_FILE
    
    echo "## 📝 Logs Relevantes" >> $REPORT_FILE
    echo "" >> $REPORT_FILE
    echo "### Últimos 10 Erros HTTP" >> $REPORT_FILE
    echo "\`\`\`" >> $REPORT_FILE
    docker-compose logs api1 api2 | grep "HTTP_ERROR" | tail -10 >> $REPORT_FILE
    echo "\`\`\`" >> $REPORT_FILE
    echo "" >> $REPORT_FILE
    
    echo "### Últimos 10 Payment Failures" >> $REPORT_FILE
    echo "\`\`\`" >> $REPORT_FILE
    docker-compose logs api1 api2 | grep "PAYMENT_FAILED" | tail -10 >> $REPORT_FILE
    echo "\`\`\`" >> $REPORT_FILE
    echo "" >> $REPORT_FILE
    
    echo "---" >> $REPORT_FILE
    echo "*Relatório gerado automaticamente em $(date)*" >> $REPORT_FILE
}

# Gerar relatório
generate_report

echo "✅ Relatório gerado com sucesso!"
echo
echo "📋 RESUMO RÁPIDO:"
echo "════════════════"

# Mostrar resumo no terminal
HTTP_ERRORS=$(docker-compose logs api1 api2 | grep "HTTP_ERROR" | wc -l)
PAYMENT_FAILED=$(docker-compose logs api1 api2 | grep "PAYMENT_FAILED" | wc -l)
NO_GATEWAY=$(docker-compose logs api1 api2 | grep "ERROR_NO_GATEWAY\|GATEWAY_UNAVAILABLE" | wc -l)
QUEUE_FULL=$(docker-compose logs api1 api2 | grep "ERROR_QUEUE_FULL" | wc -l)
HEALTH_TIMEOUTS=$(docker-compose logs api1 api2 | grep "Health check failed" | wc -l)

echo "🔴 HTTP Errors: $HTTP_ERRORS"
echo "💳 Payment Failed: $PAYMENT_FAILED"
echo "🚫 No Gateway: $NO_GATEWAY"
echo "📤 Queue Full: $QUEUE_FULL"
echo "🏥 Health Timeouts: $HEALTH_TIMEOUTS"
echo
echo "📄 Relatório completo: $REPORT_FILE"
echo
echo "🔍 Para ver logs em tempo real durante próximo teste:"
echo "   ./monitor_k6_test.sh"