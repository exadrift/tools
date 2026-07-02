FROM alpine:3

RUN apk update --no-cache && \
  apk add openssh-client --no-cache

COPY entrypoint.sh /entrypoint.sh

CMD [ "/entrypoint.sh" ]