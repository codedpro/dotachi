FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o dotachi-node .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/dotachi-node /usr/local/bin/
EXPOSE 7443
CMD ["dotachi-node"]
