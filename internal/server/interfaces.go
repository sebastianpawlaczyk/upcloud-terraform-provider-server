package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/request"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/service"
)

func buildNetworkInterfaceRequestFromServerModel(data *serverModel) ([]request.CreateNetworkInterfaceRequest, error) {
	if data == nil {
		return nil, errors.New("serverModel is nil")
	}

	response := make([]request.CreateNetworkInterfaceRequest, 0)
	for i, inter := range data.NetworkInterface {
		response = append(response, request.CreateNetworkInterfaceRequest{
			ServerUUID: data.ID.ValueString(),
			Index:      i + 1,
			IPAddresses: request.CreateNetworkInterfaceIPAddressSlice{
				{
					Family:  inter.IpAddressFamily.ValueString(),
					Address: inter.IpAddress.ValueString(),
				},
			},
			Type:              inter.Type.ValueString(),
			Bootable:          upcloud.FromBool(inter.Bootable.ValueBool()),
			SourceIPFiltering: upcloud.FromBool(inter.SourceIpFiltering.ValueBool()),
		})
	}

	return response, nil
}

func interfacesEquals(a upcloud.ServerInterface, b request.CreateNetworkInterfaceRequest) bool {
	if a.Type != b.Type {
		return false
	}
	if a.Index != b.Index {
		return false
	}
	if len(a.IPAddresses) != 1 || len(b.IPAddresses) != 1 {
		return false
	}
	if a.IPAddresses[0].Family != b.IPAddresses[0].Family {
		return false
	}
	return true
}

func reconfigureServerNetworkInterfaces(ctx context.Context, svc *service.Service, data serverModel, reqs []request.CreateNetworkInterfaceRequest) error {
	// assert server is stopped
	s, err := svc.GetServerDetails(ctx, &request.GetServerDetailsRequest{
		UUID: data.ID.ValueString(),
	})
	if err != nil {
		return err
	}
	if s.State != upcloud.ServerStateStopped {
		return errors.New("server needs to be stopped to alter networks")
	}

	// Try to preserve public (IPv4 or IPv6) and utility network interfaces so that IPs doesn't change
	preserveInterfaces := make(map[int]bool)
	// flush interfaces
	for i, n := range s.Networking.Interfaces {
		if len(reqs) > i && interfacesEquals(n, reqs[i]) {
			preserveInterfaces[n.Index] = true
			continue
		}
		if err := svc.DeleteNetworkInterface(ctx, &request.DeleteNetworkInterfaceRequest{
			ServerUUID: data.ID.ValueString(),
			Index:      n.Index,
		}); err != nil {
			return fmt.Errorf("unable to delete interface #%d; %w", n.Index, err)
		}
	}
	// apply interfaces from state
	for _, r := range reqs {
		if _, ok := preserveInterfaces[r.Index]; ok {
			continue
		}
		if _, err := svc.CreateNetworkInterface(ctx, &r); err != nil {
			return fmt.Errorf("unable to create interface #%d; %w", r.Index, err)
		}
	}

	return nil
}
