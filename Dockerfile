FROM golang:latest AS build-env
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.* /src/
RUN go mod download
COPY *.go .
RUN go build -a -o pandocserver -ldflags="-s -w" -trimpath

FROM pandoc/extra:latest

RUN mkdir -p /app \
    && adduser -D pandocserver \
    && chown -R pandocserver:pandocserver /app

# install additional latex packages
RUN tlmgr install pgf-pie pgfplots

USER pandocserver
WORKDIR /app

COPY --from=build-env /src/pandocserver .

EXPOSE 8080

ENTRYPOINT [ "./pandocserver" ]
