
FROM alpine

ADD insantus /insantus
ADD static /static
ADD environments.yml /environments.yml
ADD checks.yml /checks.yml

RUN apk add --update-cache --no-cache ca-certificates \
    && mkdir /data
    
VOLUME /data
EXPOSE 80

ENTRYPOINT ["/insantus", "--db", "/data/insantus.db", "--listen", ":80"]

