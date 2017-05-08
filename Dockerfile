FROM alpine
MAINTAINER Kelsey Hightower <kelsey.hightower@gmail.com>
ADD gopath/bin/secrets-controller /secrets-controller
ENTRYPOINT ["/secrets-controller"]
