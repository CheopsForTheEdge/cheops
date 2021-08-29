FROM golang:1.12.0-alpine3.9

RUN mkdir /app
RUN apk update && \
    apk upgrade && \
    apk add git

ADD . /app

RUN go get -d github.com/gorilla/mux
RUN go get -d github.com/justinas/alice
RUN go get -d github.com/segmentio/ksuid

WORKDIR /app/main

RUN go build -o main .

EXPOSE 8080

CMD /app/main/main
