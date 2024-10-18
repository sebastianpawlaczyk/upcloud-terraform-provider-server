package server

import (
	"testing"

	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/request"
	"github.com/stretchr/testify/assert"
)

func TestInterfacesEquals(t *testing.T) {
	testCases := []struct {
		name   string
		a      upcloud.ServerInterface
		b      request.CreateNetworkInterfaceRequest
		result bool
	}{
		{
			name: "index difference",
			a: upcloud.ServerInterface{
				Index:       1,
				IPAddresses: []upcloud.IPAddress{},
				Type:        "",
			},
			b: request.CreateNetworkInterfaceRequest{
				Index:       0,
				IPAddresses: []request.CreateNetworkInterfaceIPAddress{},
				Type:        "",
			},
			result: false,
		},
		{
			name: "type difference",
			a: upcloud.ServerInterface{
				Index:       0,
				IPAddresses: []upcloud.IPAddress{},
				Type:        upcloud.NetworkTypePublic,
			},
			b: request.CreateNetworkInterfaceRequest{
				Index:       0,
				IPAddresses: []request.CreateNetworkInterfaceIPAddress{},
				Type:        upcloud.NetworkTypePrivate,
			},
			result: false,
		},
		{
			name: "family difference",
			a: upcloud.ServerInterface{
				Index: 0,
				IPAddresses: []upcloud.IPAddress{{
					Family: upcloud.IPAddressFamilyIPv4,
				}},
				Type: upcloud.NetworkTypePublic,
			},
			b: request.CreateNetworkInterfaceRequest{
				Index:       0,
				IPAddresses: []request.CreateNetworkInterfaceIPAddress{},
				Type:        upcloud.NetworkTypePublic,
			},
			result: false,
		},
		{
			name: "perfect match",
			a: upcloud.ServerInterface{
				Index: 0,
				IPAddresses: []upcloud.IPAddress{{
					Family: upcloud.IPAddressFamilyIPv4,
				}},
				Type: upcloud.NetworkTypePublic,
			},
			b: request.CreateNetworkInterfaceRequest{
				Index: 0,
				IPAddresses: []request.CreateNetworkInterfaceIPAddress{{
					Family: upcloud.IPAddressFamilyIPv4,
				}},
				Type: upcloud.NetworkTypePublic,
			},
			result: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.result, interfacesEquals(testCase.a, testCase.b))
		})
	}
}
