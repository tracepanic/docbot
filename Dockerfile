FROM golang:1.25-alpine AS build

RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /docbot .

FROM alpine:latest

COPY --from=build /docbot /docbot
COPY migrations /migrations

ENTRYPOINT ["/docbot"]
