package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/service"
)

func validateZone(ctx context.Context, service *service.Service, zone string) error {
	zones, err := service.GetZones(ctx)
	if err != nil {
		return err
	}
	availableZones := make([]string, 0)
	for _, z := range zones.Zones {
		if z.ID == zone {
			return nil
		}
		availableZones = append(availableZones, z.ID)
	}
	return fmt.Errorf("expected zone to be one of [%s], got %s", strings.Join(availableZones, ", "), zone)
}
