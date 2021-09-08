RUN Broker
RUN Client 
curl http://172.16.192.21:8080/deploy \
curl http://172.16.192.21:8080/get   \
docker run -it --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3.8-management &
