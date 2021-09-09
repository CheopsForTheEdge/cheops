Cheops Complete code base \
RUN Broker \
RUN Client \
curl http://IP:8080/deploy \
curl http://IP:8080/get   \
curl http://IP:8080/cross/cluster1,cluster2/nginx     \
curl http://IP:8080/replica/cluster1,cluster2/       <- Deployment value taken from deployment.json (inside client) \
cluster1, cluster2, cluster3 -> added in client   \
docker run -it --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3.8-management &
