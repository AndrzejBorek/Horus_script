FROM golang:1.24

ENV USER_API_URL=https://randomuser.me/api/
ENV DEBIAN_FRONTEND=noninteractive
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8


RUN apt-get update && \
    apt-get install -y --no-install-recommends locales ca-certificates && \
    sed -i '/en_US.UTF-8/s/^# //g' /etc/locale.gen && \
    locale-gen en_US.UTF-8 && \
    update-locale LANG=en_US.UTF-8

ADD . /first
WORKDIR /first
RUN go build cmd/first/main.go
