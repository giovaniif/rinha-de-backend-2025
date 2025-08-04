#!/bin/bash

echo "🧠 TESTANDO SMART GATEWAY SYSTEM"
echo "================================="
echo "Data/Hora: $(date)"
echo

echo "📋 FUNCIONALIDADES IMPLEMENTADAS:"
echo "🔄 Health Check Inteligente:"
echo "   - Timeout adaptativo baseado em histórico"
echo "   - Estados: healthy → degraded → unavailable"
echo "   - Grace period antes de marcar como indisponível"
echo "   - Intervalos adaptativos baseados no estado"
echo "   - Rate limiting detection"
echo ""
echo "💾 Cache de Última Disponibilidade:"
echo "   - Histórico dos últimos 5 minutos"
echo "   - Fallback para gateway mais recentemente healthy"
echo "   - Estados degraded ainda utilizáveis"
echo "   - TTL inteligente baseado na confiabilidade"
echo ""
echo "⚡ Adaptive Timeouts:"
echo "   - Baseado no response time médio dos últimos 10 requests"
echo "   - Timeout = 3x média + 2s de margem"
echo "   - Limites: 3s mín, 10s máx"
echo

echo "🔧 Reconstruindo aplicação com Smart Gateway..."
docker-compose down
docker-compose build --no-cache api1 api2
docker-compose up -d

echo "⏱️ Aguardando inicialização..."
sleep 15

echo "🏥 Verificando status inicial dos gateways..."
docker-compose exec redis redis-cli keys "gateway:*"

echo ""
echo "🧪 Fazendo testes com diferentes cenários..."

echo ""
echo "TEST 1: Pagamento normal (deve funcionar)"
curl -X POST -H "Content-Type: application/json" \
     -d '{"correlationId": "smart-test-1", "amount": 15.50}' \
     "http://localhost:9999/payments"

echo ""
echo "TEST 2: Pagamento após falha (deve usar fallback ou historical)"
curl -X POST -H "Content-Type: application/json" \
     -d '{"correlationId": "smart-test-2", "amount": 25.75}' \
     "http://localhost:9999/payments"

echo ""
echo "TEST 3: Verificando summary"
curl -s "http://localhost:9999/payments-summary?from=2025-08-01T00:00:00Z&to=2025-08-03T00:00:00Z" | jq .

echo ""
echo "📊 LOGS DOS SMART HEALTH CHECKS (últimos 20):"
docker-compose logs api1 api2 | grep -E "(SMART_HEALTH_CHECK|GATEWAY_)" | tail -20

echo ""
echo "✅ SMART GATEWAY SYSTEM ATIVO!"
echo ""
echo "🎯 PARA TESTE COMPLETO K6:"
echo "1. Execute: './monitor_k6_test.sh' (monitoramento)"
echo "2. Execute: 'cd temp-rinha/rinha-test && k6 run rinha.js' (teste)"
echo "3. Execute: './compare_smart_results.sh' (análise)"
echo ""
echo "🔍 Observe os novos logs:"
echo "   - SMART_HEALTH_CHECK: verificações inteligentes"
echo "   - GATEWAY_STATE_CHANGE: mudanças de estado"
echo "   - GATEWAY_SELECTED_FROM_HISTORY: uso do cache histórico"
echo "   - GATEWAY_SELECTED_GRACE_PERIOD: uso de gateway degraded"