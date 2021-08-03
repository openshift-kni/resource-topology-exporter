FROM  quay.io/centos/centos:centos7.9.2009
ADD _out/resource-topology-exporter /bin/resource-topology-exporter
RUN mkdir /etc/resource-topology-exporter-config/
# initialize with empty content
RUN echo "{}" > /etc/resource-topology-exporter-config/resources.json
ENTRYPOINT ["/bin/resource-topology-exporter"]
