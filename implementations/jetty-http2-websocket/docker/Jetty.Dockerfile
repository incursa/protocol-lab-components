FROM maven:3.9.11-eclipse-temurin-21-alpine@sha256:922927df2c662cdd47ddb116443d6bec4696cfae3de1a0ddac8fcc7b87ce61ae AS build
WORKDIR /src
RUN apk add --no-cache openssl
COPY pom.xml ./
COPY src ./src
COPY certs ./certs
RUN mvn --batch-mode --no-transfer-progress package \
    && cp target/protocol-lab-jetty-http2-websocket-0.1.0.jar /tmp/server.jar \
    && openssl pkcs12 -export -in certs/leaf.pem -inkey certs/leaf-key.pem -out /tmp/keystore.p12 -name protocol-lab -passout pass:protocol-lab \
    && chmod 0444 /tmp/keystore.p12

FROM eclipse-temurin:21.0.8_9-jre-alpine-3.22@sha256:990397e0495ac088ab6ee3d949a2e97b715a134d8b96c561c5d130b3786a489d
WORKDIR /app
COPY --from=build /tmp/server.jar /app/server.jar
COPY --from=build /tmp/keystore.p12 /app/keystore.p12
ENV PLAB_TARGET_PORT=18452
EXPOSE 18452
USER 10001
ENTRYPOINT ["java", "-jar", "/app/server.jar"]
