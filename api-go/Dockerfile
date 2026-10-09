FROM golang:1.24-alpine AS build

WORKDIR /app

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /matrix-api .

FROM alpine:3.21 AS runtime

WORKDIR /

COPY --from=build /matrix-api /matrix-api

EXPOSE 3000

CMD ["/matrix-api"]
