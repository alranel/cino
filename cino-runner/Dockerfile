FROM golang:latest
WORKDIR /cino
COPY . .
RUN cd cino-runner && go get -v
RUN cd cino-runner && go build

ENTRYPOINT [ "cino-runner/cino-runner", "subscribe" ]
