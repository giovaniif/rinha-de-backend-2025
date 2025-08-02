#!/bin/bash

echo "🚀 TESTANDO MELHORIAS DE PERFORMANCE"
echo "===================================="
echo "Data/Hora: $(date)"
echo

echo "📋 MELHORIAS IMPLEMENTADAS:"
echo "- Fila aumentada: 1000 → 5000 (5x maior)"
echo "- Workers aumentados: 4 → 8 (2x mais)"
echo "- Reprocessamento quando no gateway available"
echo "- Exponential backoff (2s, 5s, 10s, 15s, 30s)"
echo "- Máximo tentativas: 3 → 5"
echo

echo "🔧 Reconstruindo aplicação..."
docker-compose down
docker-compose build --no-cache api1 api2
docker-compose up -d

echo "⏱️ Aguardando serviços iniciarem..."
sleep 15

echo "🏥 Verificando status dos serviços..."
docker-compose ps

echo "🧪 Fazendo teste básico..."
curl -X POST -H "Content-Type: application/json" \
     -d '{"correlationId": "improvement-test", "amount": 10.00}' \
     "http://localhost:9999/payments"

echo ""
echo "📊 Verificando logs do teste básico..."
docker-compose logs api1 | grep "improvement-test" | tail -5

echo ""
echo "✅ SISTEMA PRONTO PARA TESTE K6!"
echo ""
echo "🎯 PRÓXIMOS PASSOS:"
echo "1. Execute: './monitor_k6_test.sh' em um terminal separado"
echo "2. Execute: 'cd temp-rinha/rinha-test && k6 run rinha.js' em outro terminal"
echo "3. Compare os resultados com o teste anterior"
echo ""
echo "📈 MÉTRICAS PARA COMPARAR:"
echo "- HTTP Errors (anterior: 9739)"
echo "- No Gateway (anterior: 8957)" 
echo "- Queue Full (anterior: 780)"
echo "- Taxa de sucesso (anterior: 59.73%)"
echo "- p99 latency (anterior: 801.63ms)"