# etcd monitor

This small tool can be used to monitor the health status pf etcd instance. It can periodically check the the `/health`
endpoint and push results to [AWS CloudWatch](https://aws.amazon.com/cloudwatch/) metrics.

We use this tool to monitor the number of unhealthy etcd instances within etcd cluster.

## Usage

Environment Variables

- `CHECK_INTERVAL` - Time interval of how often to run the check (in seconds). (default: `60`)
- `ETCDMON_CA_FILE` - A PEM eoncoded CA's certificate file.
- `ETCDMON_CERT_FILE` - A PEM eoncoded certificate file.
- `ETCDMON_KEY_FILE` - A PEM encoded private key file.
- `ETCD_ADVERTISE_CLIENT_URLS` - The address of the etcd server. (default: `https://127.0.0.1:2379`)
- `ETCD_NAME` - Name of the etcd cluster. This value will be used as CloudWatch dimension value. (default: `etcd`)
- `METRIC_NAMESPACE` - AWS CloudWatch metric namespace. (default: `etcd`)
- `AWS_REGION` - AWS CloudWatch region. (default: `us-east-1`)

Alternatively CLI flags can be used and will override the value specified in environment variables.

- `-interval=60`
- `-ca-file=/path/to/ca.pem`
- `-cert-file=/path/to/cert.pem`
- `-key-file=/path/to/key.pem`
- `-address=https://127.0.0.1:2379`
- `-name=etcd`
- `-namespace=etcd`
- `-region=us-east-1`

### Docker

This can also be used with docker

```sh
docker run --rm --network=host kasko/etcd-monitor etcd-monitor -address=http://127.0.0.1:2379 -region=eu-west-1
```

## Build

```sh
# Install dependencies
make tools

# Compile binary for linux
make

# Create Docker image
make container

# Simply run without compiling (you can also apply arguments)
go run etcd-monitor.go [-interval=60]
```
## License

For license please check the [LICENSE](LICENSE) file.
