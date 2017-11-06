FROM alpine:3.6

ADD build/sanfran-router /
WORKDIR /

CMD [ "./sanfran-router", "-logtostderr" ]

EXPOSE 8080
