# Softaculous Traffic Exporter

## Metrics

- softaculous_vm_traffic_total
- softaculous_vm_traffic_free
- softaculous_vm_traffic_used
- softaculous_vm_traffic_free_percent
- softaculous_vm_traffic_used_percent

### Run

#### Run on your system

```bash
docker run -d -p 9153:9153 -e SOFTACULOUS_API_KEY=XXXX -e SOFTACULOUS_API_PASSWORD=YYYY \
-e SOFTACULOUS_URL="https://URL:PORT" -e SOFTACULOUS_SCRAPE_SCHEDULE=10 -e SOFTACULOUS_IGNORE_SSL=true ghcr.io/noorbala7418/softaculous-traffic-exporter:latest
```

#### Run on your kubernetes cluster

```bash
kubectl create ns monitoring
kubectl apply -f ./deployment
```

### Tasks

- [ ] Check and list all vms in account.
- [ ] Gather traffic metrics from all vms.
