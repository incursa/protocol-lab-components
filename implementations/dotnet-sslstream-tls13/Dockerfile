FROM mcr.microsoft.com/dotnet/sdk:10.0 AS build
WORKDIR /src
COPY src/DotNetSslStreamTls13.csproj ./
RUN dotnet restore DotNetSslStreamTls13.csproj
COPY src/Program.cs ./
RUN dotnet publish DotNetSslStreamTls13.csproj --configuration Release --output /out --no-restore

FROM mcr.microsoft.com/dotnet/runtime:10.0
WORKDIR /app
COPY --from=build /out/ ./
COPY certs/ ./certs/
ENV PLAB_TLS_PORT=8443
EXPOSE 8443/tcp
ENTRYPOINT ["dotnet", "dotnet-sslstream-tls13.dll"]
