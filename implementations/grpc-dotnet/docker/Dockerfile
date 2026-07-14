FROM mcr.microsoft.com/dotnet/sdk:10.0-noble@sha256:ea8bde36c11b6e7eec2656d0e59101d4462f6bd630730f2c8201ed0572b295d5 AS build
WORKDIR /src
COPY contract/ contract/
COPY source/ source/
RUN dotnet publish source/GrpcDotNet.csproj -c Release -o /out --nologo --self-contained false

FROM mcr.microsoft.com/dotnet/aspnet:10.0-noble@sha256:7644f992230d35cf230017189d4038c0ae0f7388b13f4f7ae1900a155bafb597
WORKDIR /app
COPY --from=build /out/ .
COPY certs/ certs/
EXPOSE 18444
ENTRYPOINT ["dotnet", "GrpcDotNet.dll"]
