FROM golang:1.19 as builder

COPY ./simple_svc.go ./simple_svc.go

RUN GOARCH=amd64 go build  -o app simple_svc.go

ENTRYPOINT ["./app"]