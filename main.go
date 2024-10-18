package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/upcloud-terraform-provider-server/upcloud"
)

func main() {
	if err := providerserver.Serve(context.Background(), upcloud.New(), providerserver.ServeOpts{
		Address: "example.com/upcloudltd/upcloud",
	}); err != nil {
		log.Fatal(err)
	}
}
