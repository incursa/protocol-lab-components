FROM debian:12.11-slim@sha256:b1a741487078b369e78119849663d7f1a5341ef2768798f7b7406c4240f86aef AS build
RUN apt-get update && apt-get install -y --no-install-recommends build-essential=12.9 cmake=3.25.1-1 protobuf-compiler=3.21.12-3+deb12u1 protobuf-compiler-grpc=1.51.1-3+b1 libprotobuf-dev=3.21.12-3+deb12u1 libgrpc++-dev=1.51.1-3+b1 && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY contract/ contract/
COPY source/ source/
RUN mkdir generated build && protoc -I contract --cpp_out=generated --grpc_out=generated --plugin=protoc-gen-grpc=/usr/bin/grpc_cpp_plugin contract/echo.proto && cmake -S source -B build -DCMAKE_BUILD_TYPE=Release && cmake --build build --parallel

FROM debian:12.11-slim@sha256:b1a741487078b369e78119849663d7f1a5341ef2768798f7b7406c4240f86aef
RUN apt-get update && apt-get install -y --no-install-recommends libprotobuf32=3.21.12-3+deb12u1 libgrpc++1.51=1.51.1-3+b1 ca-certificates=20230311+deb12u1 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /src/build/grpc-cpp-server ./grpc-cpp-server
COPY certs/ certs/
EXPOSE 18444
ENTRYPOINT ["./grpc-cpp-server"]
