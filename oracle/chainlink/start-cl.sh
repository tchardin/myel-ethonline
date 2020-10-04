#! /bin/bash
export HOST_IP=(`ipconfig getifaddr en0`) && \
echo HOST_IP=$HOST_IP > '.env' &&\
docker-compose -f 'chainlink/docker-compose.yaml' up
