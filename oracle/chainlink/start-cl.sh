#! /bin/bash
export HOST_IP=(`ipconfig getifaddr en0`) && \
echo HOST_IP=$HOST_IP > '.env' &&\
docker pull smartcontract/chainlink:0.8.18
docker-compose -f 'chainlink/docker-compose.yaml' up -d postgresdb
sleep 5
docker-compose -f 'chainlink/docker-compose.yaml' up -d chainlink-dev
