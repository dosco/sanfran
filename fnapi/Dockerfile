FROM alpine:3.6

ADD build/sanfran-fnapi /
WORKDIR /

CMD [ "./sanfran-fnapi", "-logtostderr" ]

EXPOSE 8080
