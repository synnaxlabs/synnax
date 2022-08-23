VERSION 0.6

build:
    FROM node:18.7.0-alpine3.16
    WORKDIR /pluto
    FROM node:18.7.0-alpine3.16
    COPY package.json ./
    COPY yarn.lock ./
    RUN yarn install --frozen-lockfile
    COPY . .
    RUN yarn build
    SAVE ARTIFACT .