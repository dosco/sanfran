FROM node:9-alpine

ENV TINI_VERSION v0.16.1-sf
ADD https://github.com/dosco/tini/releases/download/${TINI_VERSION}/tini-static /tini
RUN chmod +x /tini
ENTRYPOINT ["/tini", "-r", "--"]

ARG NODE_ENV
ENV NODE_ENV $NODE_ENV

RUN mkdir -p /app
WORKDIR /app

COPY package.json /app
RUN npm install && npm cache clean --force
COPY server.js /app/

VOLUME [ "/shared" ]

CMD ["/usr/local/bin/node", "server.js"]

EXPOSE 8081