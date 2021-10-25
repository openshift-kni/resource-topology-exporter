#!/bin/bash

set -e

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"
RTE_CONTAINER_IMAGE="${RTE_CONTAINER_IMAGE:-quay.io/openshift-kni/resource-topology-exporter:4.9-snapshot}"
RTE_NAMESPACE="${RTE_NAMESPACE:-rte-e2e}"

echo "Deploying using image $RTE_CONTAINER_IMAGE."

echo "Create $RTE_NAMESPACE namespace"
cat << EOF | "$OC_TOOL" apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: "$RTE_NAMESPACE"
EOF

echo "Create RTE config file"
cat << EOF | "$OC_TOOL" apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: rte-config
  namespace: ${RTE_NAMESPACE}
data:
  config.yaml: |
    resources:
      reservedcpus: "0,1"
EOF

RTE_CONTAINER_IMAGE=${RTE_CONTAINER_IMAGE} \
RTE_POLL_INTERVAL=10s \
RTE_NAMESPACE=${RTE_NAMESPACE} \
RTE_CONFIG_FILE=/etc/resource-topology-exporter/config.yaml \
TOPOLOGY_MANAGER_POLICY=single-numa-node \
make gen-manifests | tee rte.yaml

echo "Deploy RTE"
$OC_TOOL adm policy add-scc-to-user privileged system:serviceaccount:"$RTE_NAMESPACE":rte-account
$OC_TOOL create -f rte.yaml

echo "Output cluster info"
$OC_TOOL get nodes
$OC_TOOL get pods -A
$OC_TOOL describe pod -l name=resource-topology || :
$OC_TOOL logs -l name=resource-topology -c resource-topology-exporter-container || :

echo "Check that cluster is ready"
hack/check-ds.sh "$OC_TOOL" "$RTE_NAMESPACE"
$OC_TOOL logs -l name=resource-topology -c resource-topology-exporter-container || :
$OC_TOOL get noderesourcetopologies.topology.node.k8s.io -A -o yaml

echo "Cluster is ready!"
