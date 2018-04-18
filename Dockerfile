FROM golang:alpine

WORKDIR /go/src/homelab-plot

COPY . .

RUN apk update
RUN apk add git
RUN go get -d -v ./...
RUN go install -v ./...

CMD ["homelab-plot"]
