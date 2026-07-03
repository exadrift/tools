FROM alpine:3 AS certs
ARG NAME=
RUN apk --no-cache add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY ./bin/${NAME} /${NAME}
ENTRYPOINT [ "/${NAME}" ]
