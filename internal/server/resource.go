package server

import (
	"context"
	"fmt"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/request"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/service"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/upcloud-terraform-provider-server/internal/utils"
)

var (
	_ resource.Resource                = &serverResource{}
	_ resource.ResourceWithConfigure   = &serverResource{}
	_ resource.ResourceWithImportState = &serverResource{}
)

func NewServerResource() resource.Resource {
	return &serverResource{}
}

type serverResource struct {
	client *service.Service
}

type serverModel struct {
	ID               types.String            `tfsdk:"id"`
	Hostname         types.String            `tfsdk:"hostname"`
	Zone             types.String            `tfsdk:"zone"`
	NetworkInterface []networkInterfaceModel `tfsdk:"network_interface"`
}

type networkInterfaceModel struct {
	IpAddressFamily   types.String `tfsdk:"ip_address_family"`
	IpAddress         types.String `tfsdk:"ip_address"`
	IpAddressFloating types.Bool   `tfsdk:"ip_address_floating"`
	MacAddress        types.String `tfsdk:"mac_address"`
	Type              types.String `tfsdk:"type"`
	Network           types.String `tfsdk:"network"`
	SourceIpFiltering types.Bool   `tfsdk:"source_ip_filtering"`
	Bootable          types.Bool   `tfsdk:"bootable"`
	//additional_ip_address is only for private networks, so I skip it
}

func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *serverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The UpCloud server resource allows the creation, update, and deletion of a cloud server.",

		Attributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The hostname of the UpCloud server.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 128),
				},
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "The zone (region) where the server is deployed.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier (UUID) of the UpCloud server.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"network_interface": schema.ListNestedBlock{
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.IsRequired(),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"ip_address_family": schema.StringAttribute{
							MarkdownDescription: "The type of the primary IP address of this interface (IPv4 or IPv6).",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString(upcloud.IPAddressFamilyIPv4),
							Validators: []validator.String{
								stringvalidator.OneOf(upcloud.IPAddressFamilyIPv4, upcloud.IPAddressFamilyIPv6),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						// for public type network ip_address can not be set,
						"ip_address": schema.StringAttribute{
							MarkdownDescription: "The assigned primary IP address.",
							//Optional:            true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
								resetValueWhenFamilyChanges{},
							},
						},
						"ip_address_floating": schema.BoolAttribute{
							MarkdownDescription: "`true` if the primary IP address is a floating IP address.",
							Computed:            true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"mac_address": schema.StringAttribute{
							MarkdownDescription: "The MAC address assigned to this interface.",
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
								resetValueWhenFamilyChanges{},
							},
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Supported value: `public`.",
							Computed:            true,
							Default:             stringdefault.StaticString("public"),
							Validators: []validator.String{
								stringvalidator.OneOf(
									"public",
								),
							},
						},
						// for public type network can not be set by user
						"network": schema.StringAttribute{
							MarkdownDescription: "The unique ID of a network to attach this network to.",
							//Optional:            true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
								resetValueWhenFamilyChanges{},
							},
						},
						// for public type source_ip_filtering can only be true, so user can not change it
						"source_ip_filtering": schema.BoolAttribute{
							MarkdownDescription: "`true` if source IP filtering is enabled on the interface.",
							Computed:            true,
							//Optional:            true,
							Default: booldefault.StaticBool(true),
						},
						// for public type bootable can only be false, so user can not change it
						"bootable": schema.BoolAttribute{
							MarkdownDescription: "`true` if the interface should be used for network booting.",
							Computed:            true,
							//Optional:            true,
							Default: booldefault.StaticBool(false),
						},
					},
				},
			},
		},
	}
}

func (r *serverResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*service.Service)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *service.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data serverModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateZone(ctx, r.client, data.Zone.ValueString()); err != nil {
		resp.Diagnostics.AddError("Zone Error", fmt.Sprintf("Unable to find provided zone, got error: %s", err))
		return
	}

	networking, diags := buildNetworkInterfaceRequestForServer(data.NetworkInterface)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverReq := &request.CreateServerRequest{
		Hostname:   data.Hostname.ValueString(),
		Zone:       data.Zone.ValueString(),
		Networking: networking,
		StorageDevices: []request.CreateServerStorageDevice{{
			Action:  "clone",
			Size:    20,
			Storage: "01000000-0000-4000-8000-000030240200",
			Tier:    "maxiops",
			Title:   "Ubuntu-24-04-LTS",
		}},
		Title:    fmt.Sprintf("%s %s", data.Hostname.ValueString(), "(terraform resource)"),
		Metadata: upcloud.FromBool(true),
	}

	details, err := r.client.CreateServer(ctx, serverReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create server, got error: %s", err))
		return
	}

	_, err = r.client.WaitForServerState(ctx, &request.WaitForServerStateRequest{
		UUID:         details.UUID,
		DesiredState: upcloud.ServerStateStarted,
	})
	if err != nil {
		resp.Diagnostics.AddError("Server Error", fmt.Sprintf("Unable to start server, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(setServerValues(&data, details)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data serverModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	getRequest := &request.GetServerDetailsRequest{
		UUID: data.ID.ValueString(),
	}

	details, err := r.client.GetServerDetails(ctx, getRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read server details, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(setServerValues(&data, details)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var dataPlan serverModel
	var dataState serverModel

	// Get plan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &dataPlan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get state
	resp.Diagnostics.Append(req.State.Get(ctx, &dataState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Verify if network is updated
	isNetworkReconfigured := false
	if len(dataState.NetworkInterface) != len(dataPlan.NetworkInterface) {
		isNetworkReconfigured = true
	} else {
		for i := range dataState.NetworkInterface {
			if dataState.NetworkInterface[i].IpAddressFamily.ValueString() != dataPlan.NetworkInterface[i].IpAddressFamily.ValueString() {
				isNetworkReconfigured = true
				break
			}
		}
	}

	// Reconfigure network - server needs to be stopped
	if isNetworkReconfigured {
		net, err := buildNetworkInterfaceRequestFromServerModel(&dataPlan)
		if err != nil {
			resp.Diagnostics.AddError("Unable to build network interface", fmt.Sprintf("got error: %s", err))
			return
		}

		if err := utils.VerifyServerStopped(ctx, request.StopServerRequest{UUID: dataPlan.ID.ValueString()}, r.client); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to stop server, got error: %s", err))
			return
		}

		if err := reconfigureServerNetworkInterfaces(ctx, r.client, dataPlan, net); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to refresh interfaces, got error: %s", err))
			return
		}
	}

	_, err := r.client.ModifyServer(ctx, &request.ModifyServerRequest{
		UUID:     dataPlan.ID.ValueString(),
		Hostname: dataPlan.Hostname.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update server, got error: %s", err))
		return
	}

	/// After network reconfiguration - server needs to be started
	if isNetworkReconfigured {
		if err := utils.VerifyServerStarted(ctx, request.StartServerRequest{UUID: dataPlan.ID.ValueString()}, r.client); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to start server, got error: %s", err))
			return
		}
	}

	details, err := r.client.GetServerDetails(ctx, &request.GetServerDetailsRequest{
		UUID: dataPlan.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get server, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(setServerValues(&dataPlan, details)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &dataPlan)...)
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data serverModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := utils.VerifyServerStopped(ctx, request.StopServerRequest{UUID: data.ID.ValueString()}, r.client); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to stop server, got error: %s", err))
		return
	}

	deleteServerRequest := &request.DeleteServerAndStoragesRequest{
		UUID: data.ID.ValueString(),
	}

	if err := r.client.DeleteServerAndStorages(ctx, deleteServerRequest); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete server, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(setServerValues(&data, nil)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildNetworkInterfaceRequestForServer(dataNetworkInterfaces []networkInterfaceModel) (*request.CreateServerNetworking, diag.Diagnostics) {
	if len(dataNetworkInterfaces) == 0 {
		return nil, nil
	}

	interfacesResponse := make(request.CreateServerInterfaceSlice, 0, len(dataNetworkInterfaces))

	for _, inter := range dataNetworkInterfaces {
		interfacesResponse = append(interfacesResponse, request.CreateServerInterface{
			IPAddresses: []request.CreateServerIPAddress{
				{
					Family: inter.IpAddressFamily.ValueString(),
				},
			},
			Network:           inter.Network.ValueString(),
			Type:              inter.Type.ValueString(),
			SourceIPFiltering: upcloud.FromBool(inter.SourceIpFiltering.ValueBool()),
			Bootable:          upcloud.FromBool(inter.Bootable.ValueBool()),
		})
	}

	return &request.CreateServerNetworking{
		Interfaces: interfacesResponse,
	}, nil
}

func setServerValues(data *serverModel, details *upcloud.ServerDetails) diag.Diagnostics {
	var diagsResp diag.Diagnostics

	if data == nil || details == nil {
		return nil
	}

	data.ID = types.StringValue(details.UUID)
	data.Hostname = types.StringValue(details.Hostname)
	data.Zone = types.StringValue(details.Zone)

	data.NetworkInterface = make([]networkInterfaceModel, len(details.Networking.Interfaces))
	for i, iface := range details.Networking.Interfaces {
		networkInterface := networkInterfaceModel{
			MacAddress:        types.StringValue(iface.MAC),
			Type:              types.StringValue(iface.Type),
			SourceIpFiltering: types.BoolValue(iface.SourceIPFiltering.Bool()),
			Bootable:          types.BoolValue(iface.Bootable.Bool()),
			Network:           types.StringValue(iface.Network),
		}

		for _, ip := range iface.IPAddresses {
			networkInterface.IpAddress = types.StringValue(ip.Address)
			networkInterface.IpAddressFamily = types.StringValue(ip.Family)
			networkInterface.IpAddressFloating = types.BoolValue(ip.Floating.Bool())
		}

		data.NetworkInterface[i] = networkInterface
	}

	return diagsResp
}
