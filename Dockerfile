FROM golang:alpine AS build-env
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.* /src/
RUN go mod download
COPY . .
RUN go build -a -o app -ldflags="-s -w" -trimpath

FROM pandoc/extra:latest

RUN mkdir -p /app \
    && adduser -D user \
    && chown -R user:user /app

# install additional latex packages
RUN tlmgr update --self && \
    tlmgr install pgf-pie pgfplots

USER user
WORKDIR /app

COPY --from=build-env /src/app .

EXPOSE 8080

ENTRYPOINT [ "./app" ]
