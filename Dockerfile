# Stage 1: Build
FROM golang:1.21.4 AS build

WORKDIR /go/src/purity-vision

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o purity-vision .

# Stage 2: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates bash

WORKDIR /root/
COPY --from=build /go/src/purity-vision/purity-vision .

EXPOSE 8080

CMD ["./purity-vision", "-port", "8080"]
