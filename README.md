# upcloud-terraform-provider-server

## Steps to Run the Provider Locally

```shell
go mod tidy        # Clean up the Go module dependencies
go build -o upcloud-terraform-provider-server_v0.1.0  # Build the provider
```

```shell
mkdir -p ~/.terraform.d/plugins/local/upcloud/0.1.0/darwin_amd64/  # For macOS
mkdir -p ~/.terraform.d/plugins/local/upcloud/0.1.0/linux_amd64/   # For Linux
```

```shell
cp upcloud-terraform-provider-server_v0.1.0 ~/.terraform.d/plugins/local/upcloud/0.1.0/darwin_amd64/  # On macOS
```

Then use files from `./examples` folder.
