ARG TARGET

FROM alpine:latest AS base
RUN apk --no-cache add gcompat

FROM golang:1.23 AS build
ARG TARGET
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /build/app ./cmd/${TARGET}

FROM base AS RUN
COPY --from=build /build/app /app
CMD ["/app"]
