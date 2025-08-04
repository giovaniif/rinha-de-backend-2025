#!/bin/bash

echo "=== COLETANDO LOGS DE ERRO DURANTE TESTE K6 ==="
echo "Data/Hora: $(date)"
echo

# Criar diretório para logs se não existir
mkdir -p logs

# Arquivo principal de log
LOG_FILE="logs/error_analysis_$(date +%Y%m%d_%H%M%S).log"

echo "Logs sendo salvos em: $LOG_FILE"
echo

# Função para coletar logs em tempo real
collect_logs() {
    echo "=== ANÁLISE DE ERROS - $(date) ===" >> $LOG_FILE
    echo >> $LOG_FILE
    
    echo "--- ERRORS HTTP (Status >= 400) ---" >> $LOG_FILE
    docker-compose logs api1 api2 | grep "HTTP_ERROR" | tail -50 >> $LOG_FILE
    echo >> $LOG_FILE
    
    echo "--- PAYMENT FAILURES ---" >> $LOG_FILE
    docker-compose logs api1 api2 | grep "PAYMENT_FAILED\|ERROR_NO_GATEWAY\|ERROR_QUEUE_FULL" | tail -50 >> $LOG_FILE
    echo >> $LOG_FILE
    
    echo "--- GATEWAY UNAVAILABLE ---" >> $LOG_FILE
    docker-compose logs api1 api2 | grep "GATEWAY_UNAVAILABLE\|Health check failed" | tail -20 >> $LOG_FILE
    echo >> $LOG_FILE
    
    echo "--- INTERNAL ERRORS ---" >> $LOG_FILE
    docker-compose logs api1 api2 | grep "ERROR_INTERNAL\|Failed to" | tail -20 >> $LOG_FILE
    echo >> $LOG_FILE
    
    echo "--- CONTADORES DE ERRO ---" >> $LOG_FILE
    echo "HTTP Errors:" >> $LOG_FILE
    docker-compose logs api1 api2 | grep "HTTP_ERROR" | wc -l >> $LOG_FILE
    echo "Payment Failures:" >> $LOG_FILE
    docker-compose logs api1 api2 | grep "PAYMENT_FAILED" | wc -l >> $LOG_FILE
    echo "No Gateway Available:" >> $LOG_FILE
    docker-compose logs api1 api2 | grep "ERROR_NO_GATEWAY\|GATEWAY_UNAVAILABLE" | wc -l >> $LOG_FILE
    echo "Queue Full Errors:" >> $LOG_FILE
    docker-compose logs api1 api2 | grep "ERROR_QUEUE_FULL" | wc -l >> $LOG_FILE
    echo >> $LOG_FILE
    
    echo "================================" >> $LOG_FILE
    echo >> $LOG_FILE
}

# Coletar logs iniciais
collect_logs

echo "Logs iniciais coletados!"
echo "Execute 'docker-compose logs -f api1 api2 | grep \"ERROR\\|HTTP_ERROR\\|PAYMENT_FAILED\"' para acompanhar em tempo real"
echo
echo "Para análise completa após o teste, execute:"
echo "  ./analyze_errors.sh $LOG_FILE"
echo

# Criar script de análise
cat > analyze_errors.sh << 'EOF'
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
EOF

chmod +x analyze_errors.sh

echo "Scripts criados:"
echo "  - collect_error_logs.sh (atual)"
echo "  - analyze_errors.sh (para análise posterior)"
echo "  - Logs em: logs/"