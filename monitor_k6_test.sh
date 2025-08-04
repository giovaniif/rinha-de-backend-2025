#!/bin/bash

echo "🔍 MONITOR DE ERROS DURANTE TESTE K6"
echo "======================================"
echo

# Criar diretório para logs se não existir
mkdir -p logs

# Arquivo de monitoramento em tempo real
MONITOR_LOG="logs/k6_realtime_$(date +%Y%m%d_%H%M%S).log"

echo "📝 Logs de monitoramento em: $MONITOR_LOG"
echo

# Função para mostrar estatísticas em tempo real
show_stats() {
    echo "⏱️  $(date '+%H:%M:%S') - ESTATÍSTICAS ATUAIS:"
    echo "═══════════════════════════════════════════"
    
    # Contadores de erro nos últimos 60 segundos
    echo "📊 ERROS HTTP (últimos 60s):"
    docker-compose logs --since=60s api1 api2 2>/dev/null | grep "HTTP_ERROR" | wc -l | xargs printf "   Status >= 400: %s\n"
    
    echo
    echo "💳 ERROS DE PAGAMENTO (últimos 60s):"
    docker-compose logs --since=60s api1 api2 2>/dev/null | grep "PAYMENT_FAILED" | wc -l | xargs printf "   Payment Failed: %s\n"
    docker-compose logs --since=60s api1 api2 2>/dev/null | grep "ERROR_NO_GATEWAY" | wc -l | xargs printf "   No Gateway: %s\n"
    docker-compose logs --since=60s api1 api2 2>/dev/null | grep "ERROR_QUEUE_FULL" | wc -l | xargs printf "   Queue Full: %s\n"
    
    echo
    echo "🏥 STATUS DOS GATEWAYS:"
    # Verificar último status conhecido dos gateways
    REDIS_DEFAULT=$(docker-compose exec -T redis redis-cli get "gateway:default" 2>/dev/null || echo "unavailable")
    REDIS_FALLBACK=$(docker-compose exec -T redis redis-cli get "gateway:fallback" 2>/dev/null || echo "unavailable")
    
    printf "   Default: %s\n" "$REDIS_DEFAULT"
    printf "   Fallback: %s\n" "$REDIS_FALLBACK"
    
    echo
    echo "🔥 ÚLTIMOS ERROS:"
    docker-compose logs --since=30s api1 api2 2>/dev/null | grep -E "(HTTP_ERROR|PAYMENT_FAILED|ERROR_)" | tail -3 | sed 's/^/   /'
    
    echo
    echo "────────────────────────────────────────────"
    echo
}

# Função para salvar snapshot detalhado
save_snapshot() {
    echo "$(date) - SNAPSHOT" >> $MONITOR_LOG
    echo "HTTP Errors: $(docker-compose logs --since=60s api1 api2 2>/dev/null | grep "HTTP_ERROR" | wc -l)" >> $MONITOR_LOG
    echo "Payment Failures: $(docker-compose logs --since=60s api1 api2 2>/dev/null | grep "PAYMENT_FAILED" | wc -l)" >> $MONITOR_LOG
    echo "No Gateway: $(docker-compose logs --since=60s api1 api2 2>/dev/null | grep "ERROR_NO_GATEWAY" | wc -l)" >> $MONITOR_LOG
    echo "Queue Full: $(docker-compose logs --since=60s api1 api2 2>/dev/null | grep "ERROR_QUEUE_FULL" | wc -l)" >> $MONITOR_LOG
    echo "────" >> $MONITOR_LOG
}

echo "🚀 PRONTO PARA MONITORAR!"
echo
echo "📋 INSTRUÇÕES:"
echo "1. Execute este script EM UM TERMINAL SEPARADO"
echo "2. Execute o teste k6 em outro terminal: 'cd temp-rinha/rinha-test && k6 run rinha.js'"
echo "3. Observe as estatísticas aqui em tempo real"
echo "4. Pressione Ctrl+C para parar o monitoramento"
echo
echo "⚡ INICIANDO MONITORAMENTO EM 5 SEGUNDOS..."
sleep 5

# Loop principal de monitoramento
echo "🔄 MONITORAMENTO ATIVO!"
echo

trap 'echo ""; echo "⏹️  Monitoramento parado. Logs salvos em: $MONITOR_LOG"; exit 0' INT

counter=0
while true; do
    clear
    echo "🔍 MONITOR K6 - TEMPO REAL (Ctrl+C para parar)"
    echo "════════════════════════════════════════════"
    echo
    
    show_stats
    save_snapshot
    
    # Mostrar número de ciclos
    counter=$((counter + 1))
    echo "🔄 Ciclo #$counter - Atualizando a cada 10 segundos..."
    
    sleep 10
done