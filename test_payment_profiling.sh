#!/bin/bash

echo "🔍 TESTANDO PAYMENT PROCESSING PROFILING"
echo "========================================"
echo "Data/Hora: $(date)"
echo

echo "🎯 OBJETIVO: Identificar gargalos no processamento de pagamentos"
echo "   - JSON Serialization time"
echo "   - HTTP Request time"
echo "   - Payment processor latency"
echo "   - Overall payment processing time"
echo

echo "🔧 Preparando ambiente..."
docker-compose down
docker-compose build --no-cache api1 api2
docker-compose up -d
echo "⏱️ Aguardando inicialização..."
sleep 12

echo ""
echo "🧪 TESTE 1: Pagamentos individuais para análise detalhada"
echo "========================================================"

echo "Enviando 5 pagamentos sequenciais..."
for i in {1..5}; do
    echo "Pagamento $i/5..."
    curl -s -X POST -H "Content-Type: application/json" \
         -d "{\"correlationId\": \"prof-test-$i-$(date +%s)\", \"amount\": 1$i.50}" \
         "http://localhost:9999/payments" > /dev/null
    sleep 1
done

echo "⏱️ Aguardando processamento..."
sleep 5

echo ""
echo "📊 ANÁLISE DE PROFILING:"
echo "========================"

echo ""
echo "🐌 PROCESSAMENTOS LENTOS (>500ms):"
SLOW_PAYMENTS=$(docker-compose logs api1 api2 | grep "PAYMENT_PROCESSING_SLOW" | wc -l)
echo "Total de processamentos lentos: $SLOW_PAYMENTS"

if [ $SLOW_PAYMENTS -gt 0 ]; then
    echo ""
    echo "--- Detalhes dos processamentos lentos ---"
    docker-compose logs api1 api2 | grep "PAYMENT_PROCESSING_SLOW" | head -3
fi

echo ""
echo "🌐 REQUISIÇÕES HTTP LENTAS (>200ms):"
SLOW_HTTP=$(docker-compose logs api1 api2 | grep "HTTP_REQUEST_SLOW" | wc -l)
echo "Total de requisições HTTP lentas: $SLOW_HTTP"

if [ $SLOW_HTTP -gt 0 ]; then
    echo ""
    echo "--- Detalhes das requisições HTTP lentas ---"
    docker-compose logs api1 api2 | grep "HTTP_REQUEST_SLOW" | head -5
fi

echo ""
echo "📝 SERIALIZAÇÃO JSON LENTA (>10ms):"
SLOW_JSON=$(docker-compose logs api1 api2 | grep "JSON_SERIALIZATION_SLOW" | wc -l)
echo "Total de serializações JSON lentas: $SLOW_JSON"

if [ $SLOW_JSON -gt 0 ]; then
    echo ""
    echo "--- Detalhes das serializações lentas ---"
    docker-compose logs api1 api2 | grep "JSON_SERIALIZATION_SLOW" | head -3
fi

echo ""
echo "📈 ESTATÍSTICAS GERAIS:"
echo "======================"
TOTAL_SUCCESS=$(docker-compose logs api1 api2 | grep "PAYMENT_SUCCESS" | wc -l)
TOTAL_FAILED=$(docker-compose logs api1 api2 | grep "PAYMENT_FAILED" | wc -l)
echo "Pagamentos bem-sucedidos: $TOTAL_SUCCESS"
echo "Pagamentos falhados: $TOTAL_FAILED"

if [ $TOTAL_SUCCESS -gt 0 ]; then
    SUCCESS_RATE=$((TOTAL_SUCCESS * 100 / (TOTAL_SUCCESS + TOTAL_FAILED)))
    echo "Taxa de sucesso: $SUCCESS_RATE%"
fi

echo ""
echo "🧪 TESTE 2: Carga moderada (20 requests simultâneos)"
echo "===================================================="

echo "Executando carga moderada..."
for i in {1..20}; do
    curl -s -X POST -H "Content-Type: application/json" \
         -d "{\"correlationId\": \"load-test-$i-$(date +%s)\", \"amount\": 5$i.25}" \
         "http://localhost:9999/payments" > /dev/null &
done

wait
sleep 3

echo ""
echo "📊 ANÁLISE APÓS CARGA MODERADA:"
echo "==============================="

NEW_SLOW_PAYMENTS=$(docker-compose logs api1 api2 | grep "PAYMENT_PROCESSING_SLOW" | wc -l)
NEW_SLOW_HTTP=$(docker-compose logs api1 api2 | grep "HTTP_REQUEST_SLOW" | wc -l)
NEW_SLOW_JSON=$(docker-compose logs api1 api2 | grep "JSON_SERIALIZATION_SLOW" | wc -l)

echo "Total processamentos lentos: $NEW_SLOW_PAYMENTS (novos: $((NEW_SLOW_PAYMENTS - SLOW_PAYMENTS)))"
echo "Total HTTP requests lentos: $NEW_SLOW_HTTP (novos: $((NEW_SLOW_HTTP - SLOW_HTTP)))"
echo "Total JSON serializations lentos: $NEW_SLOW_JSON (novos: $((NEW_SLOW_JSON - SLOW_JSON)))"

echo ""
echo "🔍 ANÁLISE DE LATÊNCIA POR COMPONENTE:"
echo "====================================="

echo ""
echo "--- HTTP Request Times (exemplos) ---"
docker-compose logs api1 api2 | grep "HTTP_REQUEST_SLOW" | tail -5

echo ""
echo "--- Gateway Selection Times (contexto) ---"
docker-compose logs api1 api2 | grep "GATEWAY_SELECTED.*time=" | tail -3

echo ""
echo "🎯 DIAGNÓSTICO INICIAL:"
echo "======================="

if [ $NEW_SLOW_HTTP -gt 5 ]; then
    echo "❌ PROBLEMA IDENTIFICADO: HTTP Requests lentos"
    echo "   - Latência alta na comunicação com payment processors"
    echo "   - Possível causa: Network latency ou sobrecarga dos processors"
    echo ""
    echo "🔧 RECOMENDAÇÕES:"
    echo "   1. Implementar connection pooling avançado"
    echo "   2. Ajustar timeouts HTTP"
    echo "   3. Considerar HTTP/2 multiplexing"
    echo "   4. Monitorar latência de rede Docker"
elif [ $NEW_SLOW_JSON -gt 3 ]; then
    echo "❌ PROBLEMA IDENTIFICADO: JSON Serialization lento"
    echo "   - Overhead de serialização maior que esperado"
    echo "   - Possível causa: Struct complexa ou CPU limitado"
    echo ""
    echo "🔧 RECOMENDAÇÕES:"
    echo "   1. Otimizar estruturas de dados"
    echo "   2. Implementar JSON pool/reuse"
    echo "   3. Verificar CPU allocation"
elif [ $NEW_SLOW_PAYMENTS -gt 3 ]; then
    echo "❌ PROBLEMA IDENTIFICADO: Processamento geral lento"
    echo "   - Latência distribuída entre componentes"
    echo "   - Possível causa: Resource contention ou I/O blocking"
    echo ""
    echo "🔧 RECOMENDAÇÕES:"
    echo "   1. Implementar async processing"
    echo "   2. Aumentar workers pool"
    echo "   3. Otimizar lock contention"
else
    echo "✅ PROCESSAMENTO RÁPIDO DETECTADO"
    echo "   - Todos os componentes abaixo dos thresholds"
    echo "   - Investigar outros gargalos se p99 ainda alto"
fi

echo ""
echo "🚀 PRÓXIMO PASSO:"
echo "================"
echo "Execute K6 para carga completa: cd temp-rinha/rinha-test && k6 run rinha.js"
echo "Depois analise: docker-compose logs api1 api2 | grep 'HTTP_REQUEST_SLOW\\|PAYMENT_PROCESSING_SLOW'"