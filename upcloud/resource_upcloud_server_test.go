package upcloud

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Only boilerplate for Acceptance tests

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"upcloud": providerserver.NewProtocol6WithError(New()()),
}

func testAccPreCheck(_ *testing.T) {}

func TestAccServerResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccUpcloudServerResourceConfig("zone-01"),
				Check:  resource.ComposeAggregateTestCheckFunc(),
			},
			// ImportState testing
			{
				ResourceName:      "upcloud_server.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccUpcloudServerResourceConfig("zone-01"),
				Check:  resource.ComposeAggregateTestCheckFunc(),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccUpcloudServerResourceConfig(configurableAttribute string) string {
	return fmt.Sprintf(`
resource "upcloud_server" "test" {
  zone = %[1]q
}
`, configurableAttribute)
}
