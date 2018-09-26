FROM golang:1.11

WORKDIR /go/src/butler
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["butler"]
