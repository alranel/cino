FROM golang:latest
WORKDIR /cino/cino-server
COPY . ..
RUN go get -v
RUN go build

EXPOSE 8080
ENTRYPOINT [ "cino-server" ]
