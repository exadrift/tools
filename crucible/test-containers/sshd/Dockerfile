FROM alpine:3

RUN apk update --no-cache && \
  apk add openssh openrc rsync sudo --no-cache
RUN mkdir -p /etc/config/hostkeys

COPY sshd_config /etc/ssh/sshd_config
COPY entrypoint.sh /entrypoint.sh

EXPOSE 22

CMD [ "/entrypoint.sh" ]