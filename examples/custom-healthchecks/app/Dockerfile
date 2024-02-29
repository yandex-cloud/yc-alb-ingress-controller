FROM golang:1.19 as builder

COPY ../../../tmp/healthchecktest.go ./healthchecktest.go

RUN GOARCH=amd64 go build  -o app healthchecktest.go

ENTRYPOINT ["./app"]