#!/bin/bash

echo "🚀 TESTANDO OTIMIZAÇÕES HTTP CLIENT + CIRCUIT BREAKER"
echo "===================================================="
echo "Data/Hora: $(date)"
echo

echo "🎯 OTIMIZAÇÕES IMPLEMENTADAS:"
echo "  ✅ HTTP Client timeout: 8s → 15s"
echo "  ✅ Connection pooling: MaxIdleConns=100, IdleTimeout=90s"
echo "  ✅ Circuit breaker: 5 failures → OPEN por 30s"
echo "  ✅ Keep-alive connections: Enabled"
echo "  ✅ Compression disabled para menor latência"
echo

echo "🔧 1. PURGE DOS PAYMENT PROCESSORS"
echo "=================================="
echo "Limpando dados anteriores..."
docker-compose exec api1 wget --post-data="" --header="X-Rinha-Token: 123" -q -O - "http://payment-processor-default:8080/admin/purge-payments" > /dev/null
docker-compose exec api1 wget --post-data="" --header="X-Rinha-Token: 123" -q -O - "http://payment-processor-fallback:8080/admin/purge-payments" > /dev/null
echo "✅ Purge concluído"

echo ""
echo "🔄 2. REBUILD E RESTART"
echo "======================="
echo "Aplicando otimizações..."
docker-compose down -q
docker-compose build --no-cache api1 api2 -q
docker-compose up -d
echo "⏱️ Aguardando inicialização..."
sleep 15

echo ""
echo "🧪 3. TESTE BÁSICO - CIRCUIT BREAKER"
echo "===================================="

echo "Enviando 10 requests para verificar circuit breaker..."
for i in {1..10}; do
    echo -n "Request $i/10: "
    response=$(curl -s -w "%{http_code}" -X POST -H "Content-Type: application/json" \
                    -d "{\"correlationId\": \"cb-test-$i-$(date +%s)\", \"amount\": 1$i.99}" \
                    "http://localhost:9999/payments")
    echo "$response"
    sleep 0.5
done

echo ""
echo "📊 4. ANÁLISE INICIAL DOS LOGS"
echo "=============================="

echo "🔄 Circuit Breaker Events:"
CB_CREATED=$(docker-compose logs api1 api2 | grep "CIRCUIT_BREAKER_CREATED" | wc -l)
CB_OPENED=$(docker-compose logs api1 api2 | grep "CIRCUIT_BREAKER_OPENED" | wc -l)
CB_BLOCKED=$(docker-compose logs api1 api2 | grep "CIRCUIT_BREAKER_BLOCKED" | wc -l)
echo "  - Circuit breakers criados: $CB_CREATED"
echo "  - Circuit breakers abertos: $CB_OPENED"
echo "  - Requests bloqueados: $CB_BLOCKED"

echo ""
echo "⚡ HTTP Client Performance:"
HTTP_SLOW=$(docker-compose logs api1 api2 | grep "HTTP_REQUEST_SLOW" | wc -l)
PAYMENT_SLOW=$(docker-compose logs api1 api2 | grep "PAYMENT_PROCESSING_SLOW" | wc -l)
echo "  - HTTP requests lentos (>200ms): $HTTP_SLOW"
echo "  - Payment processing lento (>500ms): $PAYMENT_SLOW"

echo ""
echo "🏃 5. TESTE DE CARGA MODERADA"
echo "============================="

echo "Executando 50 requests simultâneos..."
start_time=$(date +%s)

for i in {1..50}; do
    curl -s -X POST -H "Content-Type: application/json" \
         -d "{\"correlationId\": \"load-test-$i-$(date +%s)\", \"amount\": $(($i + 10)).50}" \
         "http://localhost:9999/payments" > /dev/null &
done

wait
end_time=$(date +%s)
duration=$((end_time - start_time))

echo "✅ 50 requests processados em ${duration}s"

echo ""
echo "📈 6. ANÁLISE APÓS CARGA"
echo "========================"

sleep 3

NEW_HTTP_SLOW=$(docker-compose logs api1 api2 | grep "HTTP_REQUEST_SLOW" | wc -l)
NEW_PAYMENT_SLOW=$(docker-compose logs api1 api2 | grep "PAYMENT_PROCESSING_SLOW" | wc -l)
NEW_CB_OPENED=$(docker-compose logs api1 api2 | grep "CIRCUIT_BREAKER_OPENED" | wc -l)
NEW_CB_BLOCKED=$(docker-compose logs api1 api2 | grep "CIRCUIT_BREAKER_BLOCKED" | wc -l)

TOTAL_SUCCESS=$(docker-compose logs api1 api2 | grep "PAYMENT_SUCCESS" | wc -l)
TOTAL_FAILED=$(docker-compose logs api1 api2 | grep "PAYMENT_FAILED" | wc -l)

echo "📊 ESTATÍSTICAS FINAIS:"
echo "  - HTTP requests lentos: $NEW_HTTP_SLOW (delta: $((NEW_HTTP_SLOW - HTTP_SLOW)))"
echo "  - Payment processing lento: $NEW_PAYMENT_SLOW (delta: $((NEW_PAYMENT_SLOW - PAYMENT_SLOW)))"
echo "  - Circuit breakers abertos: $NEW_CB_OPENED"
echo "  - Requests bloqueados por CB: $NEW_CB_BLOCKED"
echo "  - Pagamentos bem-sucedidos: $TOTAL_SUCCESS"
echo "  - Pagamentos falhados: $TOTAL_FAILED"

if [ $TOTAL_SUCCESS -gt 0 ] && [ $TOTAL_FAILED -gt 0 ]; then
    SUCCESS_RATE=$((TOTAL_SUCCESS * 100 / (TOTAL_SUCCESS + TOTAL_FAILED)))
    echo "  - Taxa de sucesso: $SUCCESS_RATE%"
fi

echo ""
echo "🔍 7. EXEMPLOS DE LOGS RELEVANTES"
echo "================================="

if [ $NEW_CB_OPENED -gt 0 ]; then
    echo ""
    echo "🔴 Circuit Breaker Opened (exemplos):"
    docker-compose logs api1 api2 | grep "CIRCUIT_BREAKER_OPENED" | head -3
fi

if [ $NEW_CB_BLOCKED -gt 0 ]; then
    echo ""
    echo "🚫 Requests Bloqueados por Circuit Breaker:"
    docker-compose logs api1 api2 | grep "CIRCUIT_BREAKER_BLOCKED" | head -3
fi

if [ $NEW_HTTP_SLOW -gt 0 ]; then
    echo ""
    echo "🐌 HTTP Requests ainda lentos (>200ms):"
    docker-compose logs api1 api2 | grep "HTTP_REQUEST_SLOW" | tail -3
fi

echo ""
echo "⚡ HTTP Client Optimization Logs:"
docker-compose logs api1 api2 | grep "HTTP_CLIENT_OPTIMIZED"

echo ""
echo "🎯 8. DIAGNÓSTICO"
echo "================="

if [ $NEW_HTTP_SLOW -lt 10 ] && [ $SUCCESS_RATE -gt 85 ]; then
    echo "✅ OTIMIZAÇÕES FUNCIONANDO!"
    echo "   - HTTP requests rápidos: $(($NEW_HTTP_SLOW)) < 10 ✅"
    echo "   - Taxa de sucesso alta: $SUCCESS_RATE% > 85% ✅"
    echo "   - Circuit breaker protegendo sistema ✅"
    echo ""
    echo "🚀 PRÓXIMO PASSO: Execute K6 para teste completo"
    echo "   cd temp-rinha/rinha-test && k6 run rinha.js"
elif [ $NEW_CB_OPENED -gt 2 ]; then
    echo "⚠️ CIRCUIT BREAKERS MUITO ATIVOS"
    echo "   - Muitos circuit breakers abertos: $NEW_CB_OPENED"
    echo "   - Payment processors sobrecarregados"
    echo "   - Considere ajustar thresholds ou scaling"
elif [ $NEW_HTTP_SLOW -gt 20 ]; then
    echo "⚠️ AINDA TEMOS HTTP TIMEOUTS"
    echo "   - HTTP requests lentos: $NEW_HTTP_SLOW > 20"
    echo "   - Timeout de 15s pode não ser suficiente"
    echo "   - Payment processors podem estar limitados"
else
    echo "🟡 RESULTADOS MISTOS"
    echo "   - HTTP slow: $NEW_HTTP_SLOW"
    echo "   - Success rate: $SUCCESS_RATE%"
    echo "   - Circuit breakers: $NEW_CB_OPENED abertos"
    echo "   - Necessário K6 para análise completa"
fi

echo ""
echo "📋 COMPARAÇÃO ESPERADA COM K6:"
echo "  - p99 anterior: ~1000ms"
echo "  - p99 target: ~300ms (-70%)"
echo "  - Error rate anterior: 1.8%"
echo "  - Error rate target: <0.5%"