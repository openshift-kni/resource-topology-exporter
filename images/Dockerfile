FROM  quay.io/centos/centos:centos7.9.2009
ADD _out/resource-topology-exporter /bin/resource-topology-exporter
RUN mkdir /etc/resource-topology-exporter/ && \
    touch /etc/resource-topology-exporter/config.yaml
ENTRYPOINT ["/bin/resource-topology-exporter"]
