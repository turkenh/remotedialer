# build stage
FROM golang:alpine AS build-env
ADD . /src
RUN cd /src && go build -o tunnel-client client/main.go && go build -o tunnel-server server/main.go

# final stage
FROM alpine
WORKDIR /app
COPY --from=build-env /src/tunnel-client /app/
COPY --from=build-env /src/tunnel-server /app/
ENTRYPOINT ./tunnel-client