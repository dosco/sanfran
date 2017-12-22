FROM alpine:3.6

ADD build/sanfran-fnapi /
WORKDIR /

VOLUME [ "/data" ]
CMD [ "./sanfran-fnapi", "-logtostderr" ]

EXPOSE 8080
