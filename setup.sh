#!/bin/bash

echo "=== SETUP DO SISTEMA DE PAGAMENTOS ==="
echo

echo "1. Verificando se os payment processors estão rodando..."
if ! docker network ls | grep -q payment-processor; then
    echo "Subindo payment processors..."
    cd temp-rinha/payment-processor
    docker-compose up -d
    cd ../..
    echo "Aguardando 10 segundos para os processors inicializarem..."
    sleep 10
else
    echo "✅ Payment processors já estão rodando"
fi

echo
echo "2. Parando containers antigos (se existirem)..."
docker-compose down

echo
echo "3. Removendo imagens antigas para rebuild..."
docker-compose build --no-cache

echo
echo "4. Subindo os serviços..."
docker-compose up -d

echo
echo "5. Aguardando inicialização completa..."
sleep 15

echo
echo "6. Verificando status..."
docker-compose ps

echo
echo "7. Testando endpoint..."
curl -s "http://localhost:9999/payments-summary?from=2025-08-02T20:37:42.760Z&to=2025-08-02T20:38:52.760Z" | jq . || echo "Erro no teste"

echo
echo "=== SETUP CONCLUÍDO ==="