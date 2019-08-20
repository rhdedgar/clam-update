# /usr/local/bin/start.sh will start the service

FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

# Pause indefinitely if asked to do so.
ARG OO_PAUSE_ON_BUILD
RUN test "$OO_PAUSE_ON_BUILD" = "true" && while sleep 10; do true; done || :

# Install clam-update
RUN /usr/bin/yum install -y golang \
                   clamav-update \
                   clamav-unofficial-sigs && \
    /usr/bin/yum clean all

ADD scripts/ /usr/local/bin/

ENV GOBIN=/bin \
    GOPATH=/go

RUN go get github.com/rhdedgar/clam-update && \
    cd /go/src/github.com/rhdedgar/clam-update && \
    go install && \
    cd && \
    rm -rf /go

# Modify permissions needed to run as the clamupdate user
RUN chown -R clamupdate:clamupdate /etc/clamav-unofficial-sigs && \
    chown -R clamupdate:clamupdate /usr/local/sbin /var/log/clamav-unofficial-sigs /var/lib/clamav-unofficial-sigs && \
    chown -R clamupdate:clamupdate /var/lib/clamav/ && \
    chown -R clamupdate:clamupdate /etc/openshift_tools && \
    chown clamupdate:clamupdate /usr/bin/clamav-unofficial-sigs.sh && \
    chown clamupdate:clamupdate /usr/bin/freshclam

# Change shell to the clamupdate user
RUN chsh -s /bin/bash clamupdate

# Edit clamav config file settings
# Add necessary permissions to add arbitrary user
# Make symlinks to /secret custom signature databases and config
RUN sed -i -e 's/reload_dbs="yes"/reload_dbs="no"/' /etc/clamav-unofficial-sigs/clamav-unofficial-sigs.conf && \
    sed -i -e 's/--max-time "$curl_max_time" //' /usr/bin/clamav-unofficial-sigs.sh && \
    sed -i -e 's/--connect-timeout "$curl_connect_timeout"//' /usr/bin/clamav-unofficial-sigs.sh && \
    rm -f /etc/cron.d/clamav-update /etc/cron.d/clamav-unofficial-sigs && \
    chmod -R g+rwX /etc/passwd /etc/group && \
    ln -sf /secrets/openshift_config.cfg /var/lib/clamav/openshift_config.cfg && \
    ln -sf /secrets/openshift_known_vulnerabilities.ldb /var/lib/clamav/openshift_known_vulnerabilities.ldb && \
    ln -sf /secrets/openshift_signatures.db /var/lib/clamav/openshift_signatures.db && \
    ln -sf /secrets/zagg-config-values.yaml /etc/openshift_tools/metric_sender.yaml && \
    ln -sf /secrets/openshift_signatures.hdb /var/lib/clamav/openshift_signatures.hdb && \
    ln -sf /secrets/openshift_signatures.ign2 /var/lib/clamav/openshift_signatures.ign2 && \
    ln -sf /secrets/openshift_signatures.ldb /var/lib/clamav/openshift_signatures.ldb && \
    ln -sf /secrets/openshift_whitelist.sfp /var/lib/clamav/openshift_whitelist.sfp

# run as clamupdate user
USER 999

# Start clam-update processes
ADD clamav-unofficial-sigs.conf /etc/clamav-unofficial-sigs/
CMD /usr/local/bin/start.sh
