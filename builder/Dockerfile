FROM alpine:3.6

RUN rm -rf /var/cache/apk/* && \
    rm -rf /tmp/*

RUN apk update \
    && apk add --no-cache

RUN apk add --update nodejs-npm \
    && rm -rf /var/cache/apk/*

ADD build/sanfran-builder /
WORKDIR /

ENV npm_config_cache=/data

VOLUME [ "/data" ]
CMD [ "./sanfran-builder", "-logtostderr" ]

EXPOSE 8080
