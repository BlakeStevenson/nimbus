package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blakestevenson/nimbus/internal/plugins/proto"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "NIMBUS_PLUGIN",
	MagicCookieValue: "nimbus-media-suite",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	"media-suite": &MediaSuitePluginGRPC{},
}

// MediaSuitePluginGRPC is the plugin.Plugin implementation for gRPC
type MediaSuitePluginGRPC struct {
	plugin.Plugin
	Impl MediaSuitePlugin
}

// GRPCServer registers the gRPC server for this plugin
func (p *MediaSuitePluginGRPC) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterPluginServiceServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

// GRPCClient creates a gRPC client for this plugin
func (p *MediaSuitePluginGRPC) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewPluginServiceClient(c)}, nil
}

// GRPCServer is the gRPC server implementation that calls into the plugin
type GRPCServer struct {
	proto.UnimplementedPluginServiceServer
	Impl MediaSuitePlugin
}

// Metadata implements the Metadata RPC
func (s *GRPCServer) Metadata(ctx context.Context, req *proto.MetadataRequest) (*proto.MetadataResponse, error) {
	meta, err := s.Impl.Metadata(ctx)
	if err != nil {
		return nil, err
	}

	return &proto.MetadataResponse{
		Id:           meta.ID,
		Name:         meta.Name,
		Version:      meta.Version,
		Description:  meta.Description,
		Capabilities: meta.Capabilities,
	}, nil
}

// APIRoutes implements the APIRoutes RPC
func (s *GRPCServer) APIRoutes(ctx context.Context, req *proto.APIRoutesRequest) (*proto.APIRoutesResponse, error) {
	routes, err := s.Impl.APIRoutes(ctx)
	if err != nil {
		return nil, err
	}

	protoRoutes := make([]*proto.RouteDescriptor, len(routes))
	for i, r := range routes {
		protoRoutes[i] = &proto.RouteDescriptor{
			Method: r.Method,
			Path:   r.Path,
			Auth:   r.Auth,
			Tag:    r.Tag,
		}
	}

	return &proto.APIRoutesResponse{Routes: protoRoutes}, nil
}

// HandleAPI implements the HandleAPI RPC
func (s *GRPCServer) HandleAPI(ctx context.Context, req *proto.HandleAPIRequest) (*proto.HandleAPIResponse, error) {
	// Convert proto request to plugin request
	pluginReq := &PluginHTTPRequest{
		Method:  req.Method,
		Path:    req.Path,
		Query:   make(map[string][]string),
		Headers: make(map[string][]string),
		Body:    req.Body,
		Scopes:  req.Scopes,
	}

	// Convert query parameters
	for k, v := range req.Query {
		pluginReq.Query[k] = v.Values
	}

	// Convert headers
	for k, v := range req.Headers {
		pluginReq.Headers[k] = v.Values
	}

	// Set user ID if present
	if req.UserId != nil {
		uid := *req.UserId
		pluginReq.UserID = &uid
	}

	// Call plugin implementation
	resp, err := s.Impl.HandleAPI(ctx, pluginReq)
	if err != nil {
		return nil, err
	}

	// Convert response headers
	protoHeaders := make(map[string]*proto.StringList)
	for k, v := range resp.Headers {
		protoHeaders[k] = &proto.StringList{Values: v}
	}

	return &proto.HandleAPIResponse{
		StatusCode: int32(resp.StatusCode),
		Headers:    protoHeaders,
		Body:       resp.Body,
	}, nil
}

// UIManifest implements the UIManifest RPC
func (s *GRPCServer) UIManifest(ctx context.Context, req *proto.UIManifestRequest) (*proto.UIManifestResponse, error) {
	manifest, err := s.Impl.UIManifest(ctx)
	if err != nil {
		return nil, err
	}

	navItems := make([]*proto.UINavItem, len(manifest.NavItems))
	for i, item := range manifest.NavItems {
		navItems[i] = &proto.UINavItem{
			Label: item.Label,
			Path:  item.Path,
			Group: item.Group,
			Icon:  item.Icon,
		}
	}

	routes := make([]*proto.UIRoute, len(manifest.Routes))
	for i, route := range manifest.Routes {
		routes[i] = &proto.UIRoute{
			Path:      route.Path,
			BundleUrl: route.BundleURL,
		}
	}

	return &proto.UIManifestResponse{
		NavItems: navItems,
		Routes:   routes,
	}, nil
}

// HandleEvent implements the HandleEvent RPC
func (s *GRPCServer) HandleEvent(ctx context.Context, req *proto.HandleEventRequest) (*proto.HandleEventResponse, error) {
	// Decode JSON data
	var data map[string]interface{}
	if len(req.Data) > 0 {
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return &proto.HandleEventResponse{
				Success: false,
				Error:   err.Error(),
			}, nil
		}
	}

	evt := Event{
		Type:      req.Type,
		Data:      data,
		Timestamp: time.Unix(req.Timestamp, 0),
	}

	err := s.Impl.HandleEvent(ctx, evt)
	if err != nil {
		return &proto.HandleEventResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &proto.HandleEventResponse{Success: true}, nil
}

// GRPCClient is the gRPC client implementation that forwards calls to the plugin
type GRPCClient struct {
	client proto.PluginServiceClient
}

// Metadata calls the plugin's Metadata method
func (c *GRPCClient) Metadata(ctx context.Context) (*PluginMetadata, error) {
	resp, err := c.client.Metadata(ctx, &proto.MetadataRequest{})
	if err != nil {
		return nil, err
	}

	return &PluginMetadata{
		ID:           resp.Id,
		Name:         resp.Name,
		Version:      resp.Version,
		Description:  resp.Description,
		Capabilities: resp.Capabilities,
	}, nil
}

// APIRoutes calls the plugin's APIRoutes method
func (c *GRPCClient) APIRoutes(ctx context.Context) ([]RouteDescriptor, error) {
	resp, err := c.client.APIRoutes(ctx, &proto.APIRoutesRequest{})
	if err != nil {
		return nil, err
	}

	routes := make([]RouteDescriptor, len(resp.Routes))
	for i, r := range resp.Routes {
		routes[i] = RouteDescriptor{
			Method: r.Method,
			Path:   r.Path,
			Auth:   r.Auth,
			Tag:    r.Tag,
		}
	}

	return routes, nil
}

// HandleAPI calls the plugin's HandleAPI method
func (c *GRPCClient) HandleAPI(ctx context.Context, req *PluginHTTPRequest) (*PluginHTTPResponse, error) {
	// Convert to proto request
	protoQuery := make(map[string]*proto.StringList)
	for k, v := range req.Query {
		protoQuery[k] = &proto.StringList{Values: v}
	}

	protoHeaders := make(map[string]*proto.StringList)
	for k, v := range req.Headers {
		protoHeaders[k] = &proto.StringList{Values: v}
	}

	protoReq := &proto.HandleAPIRequest{
		Method:  req.Method,
		Path:    req.Path,
		Query:   protoQuery,
		Headers: protoHeaders,
		Body:    req.Body,
		Scopes:  req.Scopes,
	}

	if req.UserID != nil {
		protoReq.UserId = req.UserID
	}

	// Call plugin
	resp, err := c.client.HandleAPI(ctx, protoReq)
	if err != nil {
		return nil, err
	}

	// Convert response
	headers := make(map[string][]string)
	for k, v := range resp.Headers {
		headers[k] = v.Values
	}

	return &PluginHTTPResponse{
		StatusCode: int(resp.StatusCode),
		Headers:    headers,
		Body:       resp.Body,
	}, nil
}

// UIManifest calls the plugin's UIManifest method
func (c *GRPCClient) UIManifest(ctx context.Context) (*UIManifest, error) {
	resp, err := c.client.UIManifest(ctx, &proto.UIManifestRequest{})
	if err != nil {
		return nil, err
	}

	navItems := make([]UINavItem, len(resp.NavItems))
	for i, item := range resp.NavItems {
		navItems[i] = UINavItem{
			Label: item.Label,
			Path:  item.Path,
			Group: item.Group,
			Icon:  item.Icon,
		}
	}

	routes := make([]UIRoute, len(resp.Routes))
	for i, route := range resp.Routes {
		routes[i] = UIRoute{
			Path:      route.Path,
			BundleURL: route.BundleUrl,
		}
	}

	return &UIManifest{
		NavItems: navItems,
		Routes:   routes,
	}, nil
}

// HandleEvent calls the plugin's HandleEvent method
func (c *GRPCClient) HandleEvent(ctx context.Context, evt Event) error {
	// Encode event data as JSON
	data, err := json.Marshal(evt.Data)
	if err != nil {
		return err
	}

	resp, err := c.client.HandleEvent(ctx, &proto.HandleEventRequest{
		Type:      evt.Type,
		Data:      data,
		Timestamp: evt.Timestamp.Unix(),
	})
	if err != nil {
		return err
	}

	if !resp.Success && resp.Error != "" {
		return fmt.Errorf("plugin error: %s", resp.Error)
	}

	return nil
}
