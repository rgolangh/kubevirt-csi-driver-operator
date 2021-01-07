FROM registry.svc.ci.openshift.org/openshift/release:golang-1.13 AS builder
WORKDIR /go/src/github.com/openshift/kubevirt-csi-driver-operator
COPY . .
RUN make

FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base
COPY --from=builder /go/src/github.com/openshift/kubevirt-csi-driver-operator/kubevirt-csi-driver-operator /usr/bin/
COPY manifests /manifests

LABEL io.k8s.display-name="OpenShift kubevirt-csi-driver-operator" \
      io.k8s.description="The kubevirt-csi-driver-operator installs and maintains the KubeVirt CSI Driver on a cluster."

# Copy assests into a writable folder. The operator creates more assests during runtime
# Executable expects the assets folder to exist in its working directory
COPY assets /tmp/assets
RUN chmod -R 777 /tmp/assets
WORKDIR /tmp
ENTRYPOINT ["/usr/bin/kubevirt-csi-driver-operator"]

