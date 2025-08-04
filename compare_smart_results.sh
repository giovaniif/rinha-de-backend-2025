#!/bin/bash

echo "🧠 COMPARAÇÃO: SMART GATEWAY vs BÁSICO"
echo "======================================"
echo "Data/Hora: $(date)"
echo

echo "📊 RESULTADOS ANTERIORES (Sistema Básico):"
echo "══════════════════════════════════════════"
echo "🔴 HTTP Errors: 9739 → 1 (99% redução)"
echo "💳 Payment Failed: 211 → 182"
echo "🚫 No Gateway: 8957 → 27682 (ainda alto!)"
echo "📤 Queue Full: 780 → 0 (100% redução)"
echo "✅ Taxa Sucesso: ~59.73% → ~99.99%"
echo "⏱️ p99 Latency: 801.63ms → 499.36ms"
echo

echo "📈 RESULTADOS ATUAIS (Smart Gateway System):"
echo "═══════════════════════════════════════════"

# Coletar métricas atuais
HTTP_ERRORS=$(docker-compose logs api1 api2 | grep "HTTP_ERROR" | wc -l)
NO_GATEWAY=$(docker-compose logs api1 api2 | grep "ERROR_NO_GATEWAY\|GATEWAY_UNAVAILABLE" | wc -l)
SMART_HEALTH_CHECKS=$(docker-compose logs api1 api2 | grep "SMART_HEALTH_CHECK" | wc -l)
GATEWAY_RECOVERIES=$(docker-compose logs api1 api2 | grep "GATEWAY_RECOVERED" | wc -l)
HISTORY_USAGE=$(docker-compose logs api1 api2 | grep "GATEWAY_SELECTED_FROM_HISTORY" | wc -l)
GRACE_PERIOD_USAGE=$(docker-compose logs api1 api2 | grep "GATEWAY_SELECTED_GRACE_PERIOD" | wc -l)
STATE_CHANGES=$(docker-compose logs api1 api2 | grep "GATEWAY_STATE_CHANGE" | wc -l)
ADAPTIVE_TIMEOUTS=$(docker-compose logs api1 api2 | grep "HEALTH_CHECK_INTERVAL_ADJUSTED" | wc -l)
PAYMENT_SUCCESS=$(docker-compose logs api1 api2 | grep "PAYMENT_SUCCESS" | wc -l)

echo "🔴 HTTP Errors: $HTTP_ERRORS"
echo "🚫 No Gateway: $NO_GATEWAY"
echo "✅ Payment Success: $PAYMENT_SUCCESS"
echo

echo "🧠 SMART FEATURES EM AÇÃO:"
echo "═════════════════════════"
echo "🔍 Smart Health Checks Executados: $SMART_HEALTH_CHECKS"
echo "🔄 Gateway Recoveries Detectados: $GATEWAY_RECOVERIES"
echo "💾 Uso do Cache Histórico: $HISTORY_USAGE"
echo "⏳ Uso do Grace Period: $GRACE_PERIOD_USAGE"
echo "🎛️ Mudanças de Estado: $STATE_CHANGES"
echo "⚡ Ajustes de Timeout: $ADAPTIVE_TIMEOUTS"

echo ""
echo "🎯 ANÁLISE DE EFETIVIDADE:"
echo "═════════════════════════"

if [ $HISTORY_USAGE -gt 0 ]; then
    echo "✅ Cache Histórico funcionando - $HISTORY_USAGE usos"
else
    echo "🟡 Cache Histórico ainda não foi necessário"
fi

if [ $GRACE_PERIOD_USAGE -gt 0 ]; then
    echo "✅ Grace Period funcionando - $GRACE_PERIOD_USAGE usos"
else
    echo "🟡 Grace Period ainda não foi necessário"
fi

if [ $GATEWAY_RECOVERIES -gt 0 ]; then
    echo "✅ Sistema detectou $GATEWAY_RECOVERIES recuperações de gateway"
else
    echo "🟡 Nenhuma recuperação de gateway detectada ainda"
fi

if [ $ADAPTIVE_TIMEOUTS -gt 0 ]; then
    echo "✅ Timeouts adaptativos funcionando - $ADAPTIVE_TIMEOUTS ajustes"
else
    echo "🟡 Timeouts adaptativos ainda não ajustaram"
fi

echo ""
echo "🔍 LOGS RELEVANTES DOS SMART FEATURES:"
echo "════════════════════════════════════"

echo ""
echo "--- Smart Health Checks (últimos 5) ---"
docker-compose logs api1 api2 | grep "SMART_HEALTH_CHECK" | tail -5

echo ""
echo "--- Gateway State Changes ---"
docker-compose logs api1 api2 | grep "GATEWAY_STATE_CHANGE" | tail -3

echo ""
echo "--- Cache Histórico em Uso ---"
docker-compose logs api1 api2 | grep "GATEWAY_SELECTED_FROM_HISTORY" | tail -3

echo ""
echo "--- Grace Period em Uso ---"
docker-compose logs api1 api2 | grep "GATEWAY_SELECTED_GRACE_PERIOD" | tail -3

echo ""
echo "🎯 PRÓXIMOS PASSOS PARA OTIMIZAÇÃO:"
echo "═══════════════════════════════════"

if [ $NO_GATEWAY -gt 1000 ]; then
    echo "📊 RECOMENDAÇÃO: Mesmo com Smart Gateway, ainda há muitos 'No Gateway'"
    echo "   - Investigar se payment processors estão realmente sobrecarregados"
    echo "   - Considerar implementar Circuit Breaker Pattern"
    echo "   - Avaliar se precisamos de mais instâncias dos payment processors"
fi

if [ $HISTORY_USAGE -eq 0 ] && [ $GRACE_PERIOD_USAGE -eq 0 ]; then
    echo "📊 RECOMENDAÇÃO: Smart features ainda não foram testadas intensivamente"
    echo "   - Execute teste K6 mais longo para estressar o sistema"
    echo "   - Simule falhas nos payment processors para testar recuperação"
fi

echo ""
echo "🚀 PARA TESTE INTENSIVO:"
echo "   cd temp-rinha/rinha-test && k6 run rinha.js"
echo ""
echo "📊 PARA RELATÓRIO COMPLETO:"
echo "   ./full_error_analysis.sh"