# begin build container definition
FROM registry.access.redhat.com/ubi8/ubi-minimal as build

# Install clamav
RUN microdnf install -y golang

ENV GOBIN=/bin \
    GOPATH=/go

COPY ./ ./
RUN /usr/bin/go install .


# begin run container definition
FROM registry.access.redhat.com/ubi8/ubi-minimal as run

RUN microdnf install -y git

ADD scripts/ /usr/local/bin/

COPY --from=build /bin/clam-update /usr/local/bin

CMD /usr/local/bin/start.sh
