<h1 align="center">
  <img src="logo.jpg" alt="Eirini">
</h1>

<!-- A spacer -->
<div>&nbsp;</div>

## What is Eirini Controller?

Eirini Controller is a Kubernetes controller that aims to enable Cloud Foundry
to deploy applications as Pods on a Kubernetes cluster. It brings the CF model
to Kubernetes by definig well known Diego abstractions such as Long Running
Processes (LRPs) and Tasks as custom Kubernetes resources.

## Installation

### Prerequisites

- A Kubernetes cluster ([kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) works fine)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [helm](https://helm.sh/docs/intro/install/)
- The eirini-controller system and workloads namespaces need to be created upfront

```
kubectl create ns eirini-controller
kubectl create ns cf-workloads
```

- Secrets containing certificates for the webhooks need to be created. We have
  a script that does that for local dev and testing purposes

```
curl https://raw.githubusercontent.com/cloudfoundry-incubator/eirini-controller/master/deployment/scripts/generate-secrets.sh | bash -s - "*.eirini-controller.svc"
```

### Installing an eirini-controllers release

In ordrer to install eirini-controller to your k8s cluster, run the command below,
replacing `x.y.z` with a [valid release version](https://github.com/cloudfoundry-incubator/eirini-controller/releases)

```bash
VERSION=x.y.z; \
WEBHOOK_CA_BUNDLE="$(kubectl get secret -n eirini-controller eirini-instance-index-env-injector-certs -o jsonpath="{.data['tls\.ca']}")"; \
RESOURCE_VALIDATOR_CA_BUNDLE="$(kubectl get secret -n eirini-controller eirini-resource-validator-certs -o jsonpath="{.data['tls\.ca']}")" \
helm install eirini-controller https://github.com/cloudfoundry-incubator/eirini-controller/releases/download/v$VERSION/eirini-controller-$VERSION.tgz \
  --namespace eirini-controller \
  --set "webhook_ca_bundle=$WEBHOOK_CA_BUNDLE" \
  --set "resource_validator_ca_bundle=$RESOURCE_VALIDATOR_CA_BUNDLE"
```

## Usage

### Running an LRP

```bash
cat <<EOF | kubectl apply -f -
apiVersion: eirini.cloudfoundry.org/v1
kind: LRP
metadata:
  name: mylrp
  namespace: cf-workloads
spec:
  GUID: $(uuidgen)
  diskMB: 256
  image: eirini/dorini
EOF

```

### Running a Task

```bash
cat <<EOF | kubectl apply -f -
apiVersion: eirini.cloudfoundry.org/v1
kind: Task
metadata:
  name: mytask
  namespace: cf-workloads
spec:
  GUID: $(uuidgen)
  image: eirini/busybox
  command: [/bin/sleep 10]
EOF
```
