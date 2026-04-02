FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o dotachi-cp .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/dotachi-cp /usr/local/bin/
EXPOSE 8080
CMD ["dotachi-cp"]
