# Scalera Traffic Exporter

## Metrics

- scalera_vm_traffic_total
- scalera_vm_traffic_free
- scalera_vm_traffic_used
- scalera_vm_traffic_free_percent
- scalera_vm_traffic_used_percent

### Run

#### Run on your system

```bash
docker run -d -p 9153:9153 -e SCALERA_API_KEY=XXXX -e SCALERA_API_PASSWORD=YYYY \
-e SCALERA_URL="https://URL:PORT" ghcr.io/noorbala7418/scalera-traffic-exporter:latest
```

#### Run on your kubernetes cluster

```bash
kubectl create ns monitoring
kubectl apply -f ./deployment
```

### Tasks

- [ ] Check and list all vms in account.
- [ ] Gather traffic metrics from all vms.
