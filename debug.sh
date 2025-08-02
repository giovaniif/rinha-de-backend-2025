#!/bin/bash

echo "=== DIAGNÓSTICO DO SISTEMA DE PAGAMENTOS ==="
echo

echo "1. Verificando se a rede payment-processor existe..."
if docker network ls | grep -q payment-processor; then
    echo "✅ Rede payment-processor encontrada"
else
    echo "❌ Rede payment-processor NÃO encontrada"
    echo "   Você precisa subir os payment processors primeiro!"
    echo "   cd temp-rinha/payment-processor && docker-compose up -d"
    exit 1
fi

echo
echo "2. Verificando status dos containers..."
docker-compose ps

echo
echo "3. Verificando logs dos serviços..."
echo "--- Logs do nginx ---"
docker-compose logs nginx | tail -10

echo
echo "--- Logs do api1 ---"
docker-compose logs api1 | tail -10

echo
echo "--- Logs do api2 ---"
docker-compose logs api2 | tail -10

echo
echo "--- Logs do redis ---"
docker-compose logs redis | tail -5

echo
echo "4. Testando conectividade interna..."
echo "Testando se api1 responde na porta 8080..."
docker-compose exec nginx wget -q --spider http://api1:8080/payments-summary?from=2025-08-02T20:37:42.760Z\&to=2025-08-02T20:38:52.760Z && echo "✅ api1 OK" || echo "❌ api1 falhou"

echo "Testando se api2 responde na porta 8080..."
docker-compose exec nginx wget -q --spider http://api2:8080/payments-summary?from=2025-08-02T20:37:42.760Z\&to=2025-08-02T20:38:52.760Z && echo "✅ api2 OK" || echo "❌ api2 falhou"

echo
echo "5. Verificando se Redis está acessível..."
docker-compose exec redis redis-cli ping && echo "✅ Redis OK" || echo "❌ Redis falhou"

echo
echo "=== FIM DO DIAGNÓSTICO ==="