FROM alpine:3.20

RUN apk add --no-cache chrony \
    && mkdir -p /var/lib/chrony /var/log/chrony

EXPOSE 123/udp

ENTRYPOINT ["chronyd", "-d", "-f", "/etc/chrony/chrony.conf"]
