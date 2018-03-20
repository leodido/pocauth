FROM golang:1.10-alpine as builder

RUN apk update && apk upgrade && apk add --no-cache git pcre-dev gcc musl-dev

ADD . /go/src/github.com/leodido/pocauth/consent
WORKDIR /go/src/github.com/leodido/pocauth/consent

RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure -vendor-only

RUN GOOS=linux go build -ldflags "-extldflags -static" -o /tmp/consent .

# FROM scratch
# COPY --from=builder /tmp/consent /consent
ENTRYPOINT ["/tmp/consent"]
EXPOSE 3000