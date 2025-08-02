#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: $0 <log_file>"
    exit 1
fi

LOG_FILE=$1

if [ ! -f "$LOG_FILE" ]; then
    echo "Arquivo de log não encontrado: $LOG_FILE"
    exit 1
fi

echo "=== ANÁLISE DETALHADA DE ERROS ==="
echo "Arquivo: $LOG_FILE"
echo

echo "📊 RESUMO DE ERROS:"
echo "HTTP Errors (4xx/5xx): $(grep -c "HTTP_ERROR" $LOG_FILE)"
echo "Payment Failures: $(grep -c "PAYMENT_FAILED" $LOG_FILE)"
echo "No Gateway Available: $(grep -c "ERROR_NO_GATEWAY\|GATEWAY_UNAVAILABLE" $LOG_FILE)"
echo "Queue Full: $(grep -c "ERROR_QUEUE_FULL" $LOG_FILE)"
echo "Internal Errors: $(grep -c "ERROR_INTERNAL" $LOG_FILE)"
echo

echo "🔍 TOP TIPOS DE ERRO:"
echo "Status 503 (Service Unavailable):"
grep "status=503" $LOG_FILE | wc -l
echo "Status 500 (Internal Server Error):"
grep "status=500" $LOG_FILE | wc -l
echo "Status 400 (Bad Request):"
grep "status=400" $LOG_FILE | wc -l
echo

echo "⏱️ PADRÕES TEMPORAIS:"
echo "Últimos erros HTTP:"
grep "HTTP_ERROR" $LOG_FILE | tail -5
echo

echo "🎯 PRINCIPAIS PROBLEMAS IDENTIFICADOS:"
if grep -q "ERROR_NO_GATEWAY" $LOG_FILE; then
    echo "❌ GATEWAY UNAVAILABLE - Payment processors indisponíveis"
fi
if grep -q "ERROR_QUEUE_FULL" $LOG_FILE; then
    echo "❌ QUEUE OVERFLOW - Fila de pagamentos cheia"
fi
if grep -q "Health check failed" $LOG_FILE; then
    echo "❌ HEALTH CHECK TIMEOUT - Payment processors não respondem"
fi
if grep -q "status=500" $LOG_FILE; then
    echo "❌ INTERNAL SERVER ERROR - Erros internos da aplicação"
fi

echo
echo "📝 Para investigação detalhada, veja o arquivo completo: $LOG_FILE"
