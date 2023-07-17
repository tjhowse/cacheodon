FROM golang:1.19-alpine AS build

RUN apk update && apk upgrade

RUN apk add --no-cache sqlite sqlite-dev gcc g++ make

WORKDIR /cacheodon

COPY *.go go.sum go.mod /cacheodon/

RUN go build -o /cacheodon/cacheodon

FROM alpine:latest AS runtime

RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /cacheodon

COPY --from=build /cacheodon/cacheodon /cacheodon/cacheodon

CMD ["/cacheodon/cacheodon"]
