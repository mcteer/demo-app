FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /server /server

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
