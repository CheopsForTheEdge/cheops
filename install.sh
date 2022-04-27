apt update -y
apt upgrade -y

apt install -y git

# Golang + Cheops
curl -OL https://golang.org/dl/go1.17.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.17.linux-amd64.tar.gz
echo "export PATH=$PATH:/usr/local/go/bin" >> $HOME/.profile
echo "export GOPATH=$HOME/go" >> $HOME/.profile
source $HOME/.profile
echo "export GOBIN=$GOPATH/bin" >> $HOME/.profile
source $HOME/.profile
go version
go get github.com/gorilla/mux
go get github.com/justinas/alice
go get github.com/arangodb/go-driver
go get github.com/arangodb/go-driver/http
go get github.com/segmentio/ksuid
go get github.com/rabbitmq/amqp091-go
cd $HOME/go/
mkdir src && cd src && git clone https://gitlab.inria.fr/discovery/cheops.git
cd cheops
git checkout mariebind

# ArangoDB
curl -OL https://download.arangodb.com/arangodb38/DEBIAN/Release.key
sudo apt-key add - < Release.key
echo 'deb https://download.arangodb.com/arangodb38/DEBIAN/ /' | sudo tee /etc/apt/sources.list.d/arangodb.list
sudo apt-get install apt-transport-https
sudo apt-get update
sudo DEBIAN_FRONTEND=noninteractive apt-get -y install arangodb3=3.8.0-1