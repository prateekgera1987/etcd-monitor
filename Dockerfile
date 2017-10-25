FROM scratch
MAINTAINER KASKO Ltd, sysarch@kasko.io

ADD dist/cacert.pem /etc/ssl/ca-bundle.pem
ADD dist/etcd-monitor-linux-amd64 /bin/etcd-monitor

ENV PATH=/bin
ENV TMPDIR=/

CMD ["/bin/etcd-monitor"]
