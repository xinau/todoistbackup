FROM golang:1.18

WORKDIR /usr/share/repo
COPY . /usr/share/repo

RUN go build -o todoistbackup ./cmd/todoistbackup


FROM debian:latest

LABEL org.opencontainers.image.author="Felix Ehrenpfort <felix@ehrenpfort.de>"

RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=0 /usr/share/repo/todoistbackup /bin/todoistbackup
COPY          LICENSE                       /LICENSE

USER       nobody
ENTRYPOINT [ "/bin/todoistbackup" ]
CMD        [ "--config.file=/etc/todoistbackup/config.json" ]
