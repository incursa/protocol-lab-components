FROM node:24.4.1-alpine@sha256:820e86612c21d0636580206d802a726f2595366e1b867e564cbc652024151e8a
WORKDIR /app
COPY source/package.json source/package-lock.json ./
RUN npm ci --omit=dev && npm cache clean --force
COPY contract/ contract/
COPY source/server.js ./server.js
COPY certs/ certs/
EXPOSE 18444
ENTRYPOINT ["node", "server.js"]
