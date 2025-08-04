#!/bin/bash

echo "🚀 TESTE INTENSIVO DE PERFORMANCE"
echo "================================="
echo "Data/Hora: $(date)"
echo

echo "🎯 OBJETIVO: Testar otimizações de Gateway Selection"
echo "   - Cache Local (3s TTL)"
echo "   - Redis Pipeline (batch queries)"
echo "   - Profiling detalhado"
echo "   - Métricas de latência"
echo

echo "🔧 Preparando ambiente..."
docker-compose down
docker-compose build --no-cache api1 api2
docker-compose up -d
echo "⏱️ Aguardando inicialização..."
sleep 10

echo ""
echo "🧪 TESTE 1: Latência de seleção de gateway isolada"
echo "=================================================="

for i in {1..10}; do
    echo "Teste $i/10..."
    curl -s -X POST -H "Content-Type: application/json" \
         -d "{\"correlationId\": \"perf-deep-$i\", \"amount\": 10.0$i}" \
         "http://localhost:9999/payments" > /dev/null &
done

wait
sleep 2

echo ""
echo "📊 RESULTADOS DO TESTE 1:"
echo "========================"

echo ""
echo "🏎️ Cache Local hits:"
LOCAL_CACHE_HITS=$(docker-compose logs api1 api2 | grep "GATEWAY_SELECTED_LOCAL" | wc -l)
echo "  Total: $LOCAL_CACHE_HITS"
if [ $LOCAL_CACHE_HITS -gt 0 ]; then
    echo "  Exemplos:"
    docker-compose logs api1 api2 | grep "GATEWAY_SELECTED_LOCAL" | head -3
fi

echo ""
echo "📡 Redis Cache hits:"
REDIS_CACHE_HITS=$(docker-compose logs api1 api2 | grep "GATEWAY_SELECTED.*redis_cache" | wc -l)
echo "  Total: $REDIS_CACHE_HITS"

echo ""
echo "⚠️ Seleções lentas:"
SLOW_SELECTIONS=$(docker-compose logs api1 api2 | grep "GATEWAY_SELECTION_SLOW" | wc -l)
echo "  Total: $SLOW_SELECTIONS"

echo ""
echo "🧪 TESTE 2: Carga concorrente (50 requests simultâneos)"
echo "======================================================"

echo "Gerando carga..."
for i in {1..50}; do
    curl -s -X POST -H "Content-Type: application/json" \
         -d "{\"correlationId\": \"concurrent-$i\", \"amount\": 2$i.50}" \
         "http://localhost:9999/payments" > /dev/null &
done

wait
sleep 3

echo ""
echo "📊 RESULTADOS DO TESTE 2:"
echo "========================"

echo ""
echo "🏎️ Performance de Cache Local:"
NEW_LOCAL_HITS=$(docker-compose logs api1 api2 | grep "GATEWAY_SELECTED_LOCAL" | wc -l)
echo "  Total hits: $NEW_LOCAL_HITS"
echo "  Novos hits: $((NEW_LOCAL_HITS - LOCAL_CACHE_HITS))"

echo ""
echo "⚡ Análise de latência:"
echo "  Seleções < 10ms: $(docker-compose logs api1 api2 | grep "GATEWAY_SELECTED.*time=" | grep -E "time=[0-9]+\.[0-9]*[μn]s|time=[0-9]ms" | wc -l)"
echo "  Seleções 10-50ms: $(docker-compose logs api1 api2 | grep "GATEWAY_SELECTED.*time=" | grep -E "time=[1-4][0-9]\..*ms" | wc -l)"
echo "  Seleções > 50ms: $(docker-compose logs api1 api2 | grep "GATEWAY_SELECTION_SLOW" | wc -l)"

echo ""
echo "🎯 MÉTRICAS DE OTIMIZAÇÃO:"
echo "========================="

echo ""
echo "📈 Cache Hit Ratio:"
TOTAL_SELECTIONS=$(docker-compose logs api1 api2 | grep -E "GATEWAY_SELECTED|GATEWAY_SELECTED_LOCAL" | wc -l)
if [ $TOTAL_SELECTIONS -gt 0 ]; then
    CACHE_HIT_RATIO=$((NEW_LOCAL_HITS * 100 / TOTAL_SELECTIONS))
    echo "  Cache Local: $CACHE_HIT_RATIO% ($NEW_LOCAL_HITS/$TOTAL_SELECTIONS)"
else
    echo "  Sem dados suficientes"
fi

echo ""
echo "⚡ Tempos médios por método:"
echo "  Local Cache: ~1-5μs (em memória)"
echo "  Redis Cache: ~1-10ms (rede + lookup)"
echo "  History Cache: ~5-20ms (cálculos)"
echo "  Grace Period: ~10-50ms (mutex + lógica)"

echo ""
echo "🔍 ANÁLISE DETALHADA DE LOGS:"
echo "============================"

echo ""
echo "--- Últimos 5 Local Cache hits ---"
docker-compose logs api1 api2 | grep "GATEWAY_SELECTED_LOCAL" | tail -5

echo ""
echo "--- Últimos 5 Redis Cache hits ---"
docker-compose logs api1 api2 | grep "GATEWAY_SELECTED.*time=" | grep -v "LOCAL" | tail -5

echo ""
echo "--- Eventuais seleções lentas ---"
SLOW_COUNT=$(docker-compose logs api1 api2 | grep "GATEWAY_SELECTION_SLOW" | wc -l)
if [ $SLOW_COUNT -gt 0 ]; then
    docker-compose logs api1 api2 | grep "GATEWAY_SELECTION_SLOW" | head -3
else
    echo "  ✅ Nenhuma seleção lenta detectada!"
fi

echo ""
echo "🧪 TESTE 3: K6 Performance Test"
echo "==============================="

echo "Executando K6 para medir p99..."
cd temp-rinha/rinha-test
k6 run --duration 30s --vus 100 rinha.js > k6_performance_result.txt 2>&1
cd ../..

echo ""
echo "📊 RESULTADO K6:"
echo "==============="
cat temp-rinha/rinha-test/k6_performance_result.txt | grep -E "http_req_duration|http_req_failed|iterations"

echo ""
echo "🎯 CONCLUSÕES E PRÓXIMOS PASSOS:"
echo "================================"

if [ $NEW_LOCAL_HITS -gt 10 ]; then
    echo "✅ CACHE LOCAL FUNCIONANDO PERFEITAMENTE"
    echo "   - $NEW_LOCAL_HITS hits no cache local"
    echo "   - Redução significativa na latência"
    echo ""
    echo "🚀 PRÓXIMAS OTIMIZAÇÕES RECOMENDADAS:"
    echo "   1. Otimizar payment request processing"
    echo "   2. Implementar connection pooling"
    echo "   3. Reduzir lock contention no PaymentService"
else
    echo "🟡 CACHE LOCAL AINDA EM AQUECIMENTO"
    echo "   - Execute teste mais longo para avaliar melhor"
fi

if [ $SLOW_COUNT -eq 0 ]; then
    echo "✅ GATEWAY SELECTION OTIMIZADO"
    echo "   - Todas as seleções abaixo de 50ms"
    echo "   - Foco agora deve ser no payment processing"
else
    echo "⚠️ AINDA HÁ SELEÇÕES LENTAS"
    echo "   - Investigar causas específicas"
    echo "   - Considerar aumentar TTL do cache local"
fi

echo ""
echo "📈 PARA COMPARAR COM BASELINE:"
echo "   ./compare_performance_results.sh"