FROM maven:3.9.11-eclipse-temurin-21-alpine@sha256:922927df2c662cdd47ddb116443d6bec4696cfae3de1a0ddac8fcc7b87ce61ae AS build
WORKDIR /src
COPY pom.xml ./
COPY src ./src
RUN mvn --batch-mode --no-transfer-progress package \
    && cp target/protocol-lab-jetty-websocket-0.1.0.jar /tmp/server.jar

FROM eclipse-temurin:21.0.8_9-jre-alpine-3.22@sha256:990397e0495ac088ab6ee3d949a2e97b715a134d8b96c561c5d130b3786a489d
WORKDIR /app
COPY --from=build /tmp/server.jar /app/server.jar
ENV PLAB_TARGET_PORT=18081
EXPOSE 18081
USER 10001
ENTRYPOINT ["java", "-jar", "/app/server.jar"]
