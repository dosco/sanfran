FROM alpine:3.6

ADD build/sanfran-controller /
WORKDIR /

CMD [ "./sanfran-controller", "-logtostderr" ]

EXPOSE 8080
