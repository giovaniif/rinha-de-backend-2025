#!/bin/bash

echo "📊 COMPARAÇÃO DE RESULTADOS - ANTES vs DEPOIS"
echo "============================================="
echo "Data/Hora: $(date)"
echo

# Métricas do teste anterior
echo "📋 RESULTADOS ANTERIORES (baseline):"
echo "════════════════════════════════════"
echo "🔴 HTTP Errors: 9739"
echo "💳 Payment Failed: 211"
echo "🚫 No Gateway: 8957 (92% dos problemas!)"
echo "📤 Queue Full: 780"
echo "🏥 Health Timeouts: 144"
echo "📈 Taxa Sucesso: ~59.73%"
echo "⏱️ p99 Latency: 801.63ms"
echo "💰 Inconsistências: 25879"
echo

echo "📊 RESULTADOS ATUAIS:"
echo "═══════════════════"

# Coletar métricas atuais
HTTP_ERRORS=$(docker-compose logs api1 api2 | grep "HTTP_ERROR" | wc -l)
PAYMENT_FAILED=$(docker-compose logs api1 api2 | grep "PAYMENT_FAILED" | wc -l)
NO_GATEWAY=$(docker-compose logs api1 api2 | grep "ERROR_NO_GATEWAY\|GATEWAY_UNAVAILABLE" | wc -l)
GATEWAY_RETRY=$(docker-compose logs api1 api2 | grep "GATEWAY_RETRY" | wc -l)
QUEUE_FULL=$(docker-compose logs api1 api2 | grep "ERROR_QUEUE_FULL" | wc -l)
HEALTH_TIMEOUTS=$(docker-compose logs api1 api2 | grep "Health check failed" | wc -l)
PAYMENT_SUCCESS=$(docker-compose logs api1 api2 | grep "PAYMENT_SUCCESS" | wc -l)

echo "🔴 HTTP Errors: $HTTP_ERRORS"
echo "💳 Payment Failed: $PAYMENT_FAILED"  
echo "🚫 No Gateway: $NO_GATEWAY"
echo "🔄 Gateway Retries: $GATEWAY_RETRY (NEW!)"
echo "📤 Queue Full: $QUEUE_FULL"
echo "🏥 Health Timeouts: $HEALTH_TIMEOUTS"
echo "✅ Payment Success: $PAYMENT_SUCCESS"

echo ""
echo "📈 CÁLCULO DE MELHORIAS:"
echo "═══════════════════════"

# Calcular reduções percentuais
if [ $HTTP_ERRORS -lt 9739 ]; then
    REDUCTION=$(((9739 - HTTP_ERRORS) * 100 / 9739))
    echo "🎯 HTTP Errors reduzidos em: ${REDUCTION}%"
else
    INCREASE=$(((HTTP_ERRORS - 9739) * 100 / 9739))
    echo "⚠️ HTTP Errors aumentaram em: ${INCREASE}%"
fi

if [ $NO_GATEWAY -lt 8957 ]; then
    REDUCTION=$(((8957 - NO_GATEWAY) * 100 / 8957))
    echo "🎯 No Gateway reduzido em: ${REDUCTION}%"
else
    INCREASE=$(((NO_GATEWAY - 8957) * 100 / 8957))
    echo "⚠️ No Gateway aumentou em: ${INCREASE}%"
fi

if [ $QUEUE_FULL -lt 780 ]; then
    REDUCTION=$(((780 - QUEUE_FULL) * 100 / 780))
    echo "🎯 Queue Full reduzido em: ${REDUCTION}%"
else
    INCREASE=$(((QUEUE_FULL - 780) * 100 / 780))
    echo "⚠️ Queue Full aumentou em: ${INCREASE}%"
fi

echo ""
echo "🔍 ANÁLISE QUALITATIVA:"
echo "══════════════════════"

if [ $GATEWAY_RETRY -gt 0 ]; then
    echo "✅ Sistema agora reprocessa quando gateway indisponível ($GATEWAY_RETRY retries)"
fi

if [ $QUEUE_FULL -lt 100 ]; then
    echo "✅ Problema de fila cheia praticamente resolvido"
elif [ $QUEUE_FULL -lt 780 ]; then
    echo "🟡 Problema de fila cheia melhorou mas ainda existe"
else
    echo "❌ Problema de fila cheia persiste"
fi

if [ $PAYMENT_SUCCESS -gt 1000 ]; then
    echo "✅ Alto número de pagamentos processados com sucesso"
elif [ $PAYMENT_SUCCESS -gt 100 ]; then
    echo "🟡 Número moderado de pagamentos bem-sucedidos"
else
    echo "❌ Poucos pagamentos processados com sucesso"
fi

echo ""
echo "🎯 RECOMENDAÇÕES:"
echo "════════════════"

if [ $NO_GATEWAY -gt 1000 ]; then
    echo "- Investigar por que gateways ficam indisponíveis"
    echo "- Considerar aumentar timeout dos health checks"
    echo "- Implementar circuit breaker mais inteligente"
fi

if [ $QUEUE_FULL -gt 100 ]; then
    echo "- Considerar aumentar ainda mais a fila (atual: 5000)"
    echo "- Adicionar mais workers (atual: 8)"
fi

if [ $HEALTH_TIMEOUTS -gt 50 ]; then
    echo "- Otimizar configurações de rede com payment processors"
    echo "- Implementar health check mais resiliente"
fi

echo ""
echo "📝 Para análise detalhada, execute:"
echo "   ./full_error_analysis.sh"