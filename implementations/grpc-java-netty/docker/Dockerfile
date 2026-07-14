FROM maven:3.9.11-eclipse-temurin-21@sha256:6fdc855a6ed81d288ca7ca37ac6ff5e9308b612485c0801d70b25a858c83d237 AS build
WORKDIR /src
COPY source/pom.xml ./pom.xml
COPY contract/ src/main/proto/
COPY source/src/ src/
RUN mvn -B -DskipTests package

FROM eclipse-temurin:21.0.8_9-jre@sha256:66bb900643426ad01996d25bada7d56751913f9cec3b827fcb715d2ec9a0fbfc
WORKDIR /app
COPY --from=build /src/target/grpc-java-netty-target-0.1.0.jar app.jar
COPY certs/ certs/
EXPOSE 18444
ENTRYPOINT ["java","-jar","app.jar"]
