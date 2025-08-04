#!/bin/bash

echo "🚀 ANÁLISE DE PERFORMANCE - GATEWAY SELECTION"
echo "=============================================="
echo "Data/Hora: $(date)"
echo

echo "🔍 COLETANDO DADOS DE PROFILING..."
echo "=================================="

# Limpar logs antigos para análise limpa
docker-compose restart api1 api2
sleep 5

echo "🧪 Executando teste de carga focado em latência..."
curl -X POST -H "Content-Type: application/json" \
     -d '{"correlationId": "perf-test-1", "amount": 10.00}' \
     "http://localhost:9999/payments" &

curl -X POST -H "Content-Type: application/json" \
     -d '{"correlationId": "perf-test-2", "amount": 15.50}' \
     "http://localhost:9999/payments" &

curl -X POST -H "Content-Type: application/json" \
     -d '{"correlationId": "perf-test-3", "amount": 25.75}' \
     "http://localhost:9999/payments" &

wait

echo "⏱️ Aguardando processamento..."
sleep 3

echo ""
echo "📊 ANÁLISE DE GATEWAY SELECTION TIMING:"
echo "======================================="

echo ""
echo "🔍 SELEÇÕES LENTAS (>50ms):"
SLOW_SELECTIONS=$(docker-compose logs api1 api2 | grep "GATEWAY_SELECTION_SLOW" | wc -l)
echo "Total de seleções lentas: $SLOW_SELECTIONS"

if [ $SLOW_SELECTIONS -gt 0 ]; then
    echo ""
    echo "--- Detalhes das seleções lentas ---"
    docker-compose logs api1 api2 | grep "GATEWAY_SELECTION_SLOW" | head -5
fi

echo ""
echo "⚡ TEMPOS DE SELEÇÃO POR MÉTODO:"
echo "==============================="

echo ""
echo "Redis Cache (método mais rápido):"
REDIS_SELECTIONS=$(docker-compose logs api1 api2 | grep "GATEWAY_SELECTED.*redis_cache" | wc -l)
echo "  Quantidade: $REDIS_SELECTIONS"
if [ $REDIS_SELECTIONS -gt 0 ]; then
    echo "  Exemplo:"
    docker-compose logs api1 api2 | grep "GATEWAY_SELECTED.*time=" | head -1
fi

echo ""
echo "Historical Cache:"
HISTORY_SELECTIONS=$(docker-compose logs api1 api2 | grep "GATEWAY_SELECTED_FROM_HISTORY" | wc -l)
echo "  Quantidade: $HISTORY_SELECTIONS"

echo ""
echo "Grace Period:"
GRACE_SELECTIONS=$(docker-compose logs api1 api2 | grep "GATEWAY_SELECTED_GRACE_PERIOD" | wc -l)
echo "  Quantidade: $GRACE_SELECTIONS"

echo ""
echo "🕐 ANÁLISE DE COMPONENTES DE LATÊNCIA:"
echo "======================================"

echo ""
echo "Redis Lookup Time:"
docker-compose logs api1 api2 | grep "GATEWAY_SELECTION_SLOW" | \
    grep -o "redis=[0-9.]*[a-z]*" | head -5

echo ""
echo "History Lookup Time:"
docker-compose logs api1 api2 | grep "GATEWAY_SELECTION_SLOW" | \
    grep -o "history=[0-9.]*[a-z]*" | head -5

echo ""
echo "Grace Period Time:"
docker-compose logs api1 api2 | grep "GATEWAY_SELECTION_SLOW" | \
    grep -o "grace=[0-9.]*[a-z]*" | head -5

echo ""
echo "🎯 DIAGNÓSTICO E RECOMENDAÇÕES:"
echo "==============================="

if [ $SLOW_SELECTIONS -gt 0 ]; then
    echo "❌ PROBLEMA IDENTIFICADO: Seleções de gateway lentas detectadas"
    echo ""
    echo "📋 POSSÍVEIS CAUSAS:"
    echo "1. Redis com alta latência"
    echo "2. Lock contention no RWMutex"
    echo "3. Cálculos históricos custosos"
    echo "4. Múltiplas consultas Redis sequenciais"
    echo ""
    echo "🔧 PRÓXIMAS OTIMIZAÇÕES RECOMENDADAS:"
    echo "1. Implementar cache local em memória"
    echo "2. Usar pipeline Redis para múltiplas consultas"
    echo "3. Reduzir granularidade dos locks"
    echo "4. Cache de resultado da última seleção"
else
    echo "✅ GATEWAY SELECTION PERFORMÁTICO"
    echo "   Todas as seleções estão abaixo de 50ms"
    echo ""
    echo "🔍 PROCURANDO OUTROS GARGALOS..."
    echo ""
    echo "📊 PRÓXIMA ANÁLISE: Payment Request Timing"
fi

echo ""
echo "📊 ESTATÍSTICAS GERAIS:"
echo "======================"
echo "Total de logs de gateway selection: $(docker-compose logs api1 api2 | grep "GATEWAY_SELECTED" | wc -l)"
echo "Total de requests processados: $(docker-compose logs api1 api2 | grep "PAYMENT_SUCCESS\|PAYMENT_FAILED" | wc -l)"

echo ""
echo "🚀 PARA TESTE COMPLETO DE PERFORMANCE:"
echo "======================================"
echo "1. Execute: './test_performance_deep.sh' (teste intensivo)"
echo "2. Execute: 'cd temp-rinha/rinha-test && k6 run rinha.js' (carga completa)"
echo "3. Compare com: './compare_performance_results.sh'"