FROM golang:1.25 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "-s -w" -o /out/sope .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/sope /usr/local/bin/sope
ENTRYPOINT ["/usr/local/bin/sope"]
