FROM node:8-alpine

ARG NODE_ENV
ENV NODE_ENV $NODE_ENV

RUN mkdir -p /app
WORKDIR /app

COPY package.json /app
RUN npm install && npm cache clean --force
COPY index.js /app/index.js

CMD [ "node", "./index.js" ]

EXPOSE 81