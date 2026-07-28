# build
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/game .

# run
FROM alpine:3.24.1
WORKDIR /app
COPY --from=build /out/game .
EXPOSE 8080
ENTRYPOINT ["/app/game"]