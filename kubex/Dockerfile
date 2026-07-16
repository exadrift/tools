FROM alpine:3 AS certs
RUN apk --no-cache add ca-certificates

FROM scratch
ARG NAME=
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY ./bin/${NAME} /${NAME}
ENTRYPOINT [ "/${NAME}" ]
