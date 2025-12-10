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
	SDK  *SDK // SDK instance to expose to plugins (host-side only)
}

// GRPCServer registers the gRPC server for this plugin
func (p *MediaSuitePluginGRPC) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterPluginServiceServer(s, &GRPCServer{Impl: p.Impl, Broker: broker})
	return nil
}

// GRPCClient creates a gRPC client for this plugin
func (p *MediaSuitePluginGRPC) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{
		client: proto.NewPluginServiceClient(c),
		broker: broker,
		sdk:    p.SDK,
	}, nil
}

// GRPCServer is the gRPC server implementation that calls into the plugin
type GRPCServer struct {
	proto.UnimplementedPluginServiceServer
	Impl   MediaSuitePlugin
	Broker *plugin.GRPCBroker
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

// HandleAPI implements the HandleAPI RPC (runs in plugin process)
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

	// Connect to SDK server if host provided one
	if req.SdkServerId != 0 && s.Broker != nil {
		conn, err := s.Broker.Dial(req.SdkServerId)
		if err == nil {
			pluginReq.SDK = &GRPCSDKClient{
				client: proto.NewSDKServiceClient(conn),
			}
		}
		// If SDK connection fails, continue without SDK (plugin can handle missing SDK)
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

	resp := &proto.UIManifestResponse{
		NavItems: navItems,
		Routes:   routes,
	}

	// Convert ConfigSection if present
	if manifest.ConfigSection != nil {
		fields := make([]*proto.ConfigField, len(manifest.ConfigSection.Fields))
		for i, field := range manifest.ConfigSection.Fields {
			protoField := &proto.ConfigField{
				Key:          field.Key,
				Label:        field.Label,
				Description:  field.Description,
				Type:         field.Type,
				Options:      field.Options,
				DefaultValue: field.DefaultValue,
				Required:     field.Required,
				Placeholder:  field.Placeholder,
			}

			if field.Validation != nil {
				validation := &proto.ConfigFieldValidation{
					Min: field.Validation.Min,
					Max: field.Validation.Max,
				}
				if field.Validation.Pattern != "" {
					validation.Pattern = &field.Validation.Pattern
				}
				if field.Validation.ErrorMessage != "" {
					validation.ErrorMessage = &field.Validation.ErrorMessage
				}
				protoField.Validation = validation
			}

			fields[i] = protoField
		}

		resp.ConfigSection = &proto.ConfigSection{
			Title:       manifest.ConfigSection.Title,
			Description: manifest.ConfigSection.Description,
			Fields:      fields,
		}
	}

	return resp, nil
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

// IsIndexer implements the IsIndexer RPC
func (s *GRPCServer) IsIndexer(ctx context.Context, req *proto.IsIndexerRequest) (*proto.IsIndexerResponse, error) {
	isIndexer, err := s.Impl.IsIndexer(ctx)
	if err != nil {
		return &proto.IsIndexerResponse{
			IsIndexer: false,
			Error:     err.Error(),
		}, nil
	}

	return &proto.IsIndexerResponse{IsIndexer: isIndexer}, nil
}

// IsDownloader implements the IsDownloader RPC
func (s *GRPCServer) IsDownloader(ctx context.Context, req *proto.IsDownloaderRequest) (*proto.IsDownloaderResponse, error) {
	isDownloader, err := s.Impl.IsDownloader(ctx)
	if err != nil {
		return &proto.IsDownloaderResponse{
			IsDownloader: false,
			Error:        err.Error(),
		}, nil
	}

	return &proto.IsDownloaderResponse{IsDownloader: isDownloader}, nil
}

// Search implements the Search RPC
func (s *GRPCServer) Search(ctx context.Context, req *proto.IndexerSearchRequest) (*proto.IndexerSearchResponse, error) {
	// Convert proto request to plugin request
	searchReq := &IndexerSearchRequest{
		Query:      req.Query,
		Type:       req.Type,
		Categories: req.Categories,
		TVDBID:     req.Tvdbid,
		TVRageID:   req.Tvrageid,
		Season:     int(req.Season),
		Episode:    int(req.Episode),
		IMDBID:     req.Imdbid,
		TMDBID:     req.Tmdbid,
		Limit:      int(req.Limit),
		Offset:     int(req.Offset),
	}

	// Call plugin implementation
	resp, err := s.Impl.Search(ctx, searchReq)
	if err != nil {
		return &proto.IndexerSearchResponse{
			Error: err.Error(),
		}, nil
	}

	// Convert releases to proto
	protoReleases := make([]*proto.IndexerRelease, len(resp.Releases))
	for i, release := range resp.Releases {
		protoReleases[i] = &proto.IndexerRelease{
			Guid:        release.GUID,
			Title:       release.Title,
			Link:        release.Link,
			Comments:    release.Comments,
			PublishDate: release.PublishDate.Unix(),
			Category:    release.Category,
			Size:        release.Size,
			DownloadUrl: release.DownloadURL,
			Description: release.Description,
			Attributes:  release.Attributes,
			IndexerId:   release.IndexerID,
			IndexerName: release.IndexerName,
		}
	}

	return &proto.IndexerSearchResponse{
		Releases:    protoReleases,
		Total:       int32(resp.Total),
		IndexerId:   resp.IndexerID,
		IndexerName: resp.IndexerName,
	}, nil
}

// GRPCClient is the gRPC client implementation that forwards calls to the plugin
type GRPCClient struct {
	client proto.PluginServiceClient
	broker *plugin.GRPCBroker
	sdk    *SDK // SDK to expose to plugin (host-side only)
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

// HandleAPI calls the plugin's HandleAPI method (runs in host process)
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

	// Start SDK server on host side if SDK is available
	if c.sdk != nil && c.broker != nil {
		sdkServerID := c.broker.NextId()
		// Start SDK server in background - it will accept connections from plugin
		go c.broker.AcceptAndServe(sdkServerID, func(opts []grpc.ServerOption) *grpc.Server {
			server := grpc.NewServer(opts...)
			proto.RegisterSDKServiceServer(server, &GRPCSDKServer{SDK: c.sdk})
			return server
		})
		// Give the server a moment to start accepting
		time.Sleep(50 * time.Millisecond)
		protoReq.SdkServerId = sdkServerID
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

	manifest := &UIManifest{
		NavItems: navItems,
		Routes:   routes,
	}

	// Convert ConfigSection if present
	if resp.ConfigSection != nil {
		fields := make([]ConfigField, len(resp.ConfigSection.Fields))
		for i, protoField := range resp.ConfigSection.Fields {
			field := ConfigField{
				Key:          protoField.Key,
				Label:        protoField.Label,
				Description:  protoField.Description,
				Type:         protoField.Type,
				Options:      protoField.Options,
				DefaultValue: protoField.DefaultValue,
				Required:     protoField.Required,
				Placeholder:  protoField.Placeholder,
			}

			if protoField.Validation != nil {
				validation := &ConfigFieldValidation{
					Min: protoField.Validation.Min,
					Max: protoField.Validation.Max,
				}
				if protoField.Validation.Pattern != nil {
					validation.Pattern = *protoField.Validation.Pattern
				}
				if protoField.Validation.ErrorMessage != nil {
					validation.ErrorMessage = *protoField.Validation.ErrorMessage
				}
				field.Validation = validation
			}

			fields[i] = field
		}

		manifest.ConfigSection = &ConfigSection{
			Title:       resp.ConfigSection.Title,
			Description: resp.ConfigSection.Description,
			Fields:      fields,
		}
	}

	return manifest, nil
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

// IsIndexer calls the plugin's IsIndexer method
func (c *GRPCClient) IsIndexer(ctx context.Context) (bool, error) {
	resp, err := c.client.IsIndexer(ctx, &proto.IsIndexerRequest{})
	if err != nil {
		return false, err
	}

	if resp.Error != "" {
		return false, fmt.Errorf("plugin error: %s", resp.Error)
	}

	return resp.IsIndexer, nil
}

// IsDownloader calls the plugin's IsDownloader method
func (c *GRPCClient) IsDownloader(ctx context.Context) (bool, error) {
	resp, err := c.client.IsDownloader(ctx, &proto.IsDownloaderRequest{})
	if err != nil {
		return false, err
	}

	if resp.Error != "" {
		return false, fmt.Errorf("plugin error: %s", resp.Error)
	}

	return resp.IsDownloader, nil
}

// Search calls the plugin's Search method
func (c *GRPCClient) Search(ctx context.Context, req *IndexerSearchRequest) (*IndexerSearchResponse, error) {
	// Convert to proto request
	protoReq := &proto.IndexerSearchRequest{
		Query:      req.Query,
		Type:       req.Type,
		Categories: req.Categories,
		Tvdbid:     req.TVDBID,
		Tvrageid:   req.TVRageID,
		Season:     int32(req.Season),
		Episode:    int32(req.Episode),
		Imdbid:     req.IMDBID,
		Tmdbid:     req.TMDBID,
		Limit:      int32(req.Limit),
		Offset:     int32(req.Offset),
	}

	// Call plugin
	resp, err := c.client.Search(ctx, protoReq)
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("plugin error: %s", resp.Error)
	}

	// Convert proto releases to plugin releases
	releases := make([]IndexerRelease, len(resp.Releases))
	for i, protoRelease := range resp.Releases {
		releases[i] = IndexerRelease{
			GUID:        protoRelease.Guid,
			Title:       protoRelease.Title,
			Link:        protoRelease.Link,
			Comments:    protoRelease.Comments,
			PublishDate: time.Unix(protoRelease.PublishDate, 0),
			Category:    protoRelease.Category,
			Size:        protoRelease.Size,
			DownloadURL: protoRelease.DownloadUrl,
			Description: protoRelease.Description,
			Attributes:  protoRelease.Attributes,
			IndexerID:   protoRelease.IndexerId,
			IndexerName: protoRelease.IndexerName,
		}
	}

	return &IndexerSearchResponse{
		Releases:    releases,
		Total:       int(resp.Total),
		IndexerID:   resp.IndexerId,
		IndexerName: resp.IndexerName,
	}, nil
}

// ============================================================================
// SDK gRPC Server (host-side)
// ============================================================================

// GRPCSDKServer is the gRPC server that exposes SDK methods to plugins
type GRPCSDKServer struct {
	proto.UnimplementedSDKServiceServer
	SDK *SDK
}

// ConfigGet implements the ConfigGet RPC
func (s *GRPCSDKServer) ConfigGet(ctx context.Context, req *proto.ConfigGetRequest) (*proto.ConfigGetResponse, error) {
	value, err := s.SDK.ConfigGet(ctx, req.Key)
	if err != nil {
		return &proto.ConfigGetResponse{Error: err.Error()}, nil
	}

	jsonValue, err := json.Marshal(value)
	if err != nil {
		return &proto.ConfigGetResponse{Error: err.Error()}, nil
	}

	return &proto.ConfigGetResponse{Value: jsonValue}, nil
}

// ConfigGetString implements the ConfigGetString RPC
func (s *GRPCSDKServer) ConfigGetString(ctx context.Context, req *proto.ConfigGetStringRequest) (*proto.ConfigGetStringResponse, error) {
	value, err := s.SDK.ConfigGetString(ctx, req.Key)
	if err != nil {
		return &proto.ConfigGetStringResponse{Error: err.Error()}, nil
	}

	return &proto.ConfigGetStringResponse{Value: value}, nil
}

// ConfigSet implements the ConfigSet RPC
func (s *GRPCSDKServer) ConfigSet(ctx context.Context, req *proto.ConfigSetRequest) (*proto.ConfigSetResponse, error) {
	var value interface{}
	if err := json.Unmarshal(req.Value, &value); err != nil {
		return &proto.ConfigSetResponse{Error: err.Error()}, nil
	}

	if err := s.SDK.ConfigSet(ctx, req.Key, value); err != nil {
		return &proto.ConfigSetResponse{Error: err.Error()}, nil
	}

	return &proto.ConfigSetResponse{}, nil
}

// ConfigDelete implements the ConfigDelete RPC
func (s *GRPCSDKServer) ConfigDelete(ctx context.Context, req *proto.ConfigDeleteRequest) (*proto.ConfigDeleteResponse, error) {
	if err := s.SDK.ConfigDelete(ctx, req.Key); err != nil {
		return &proto.ConfigDeleteResponse{Error: err.Error()}, nil
	}

	return &proto.ConfigDeleteResponse{}, nil
}

// ============================================================================
// SDK gRPC Client (plugin-side)
// ============================================================================

// GRPCSDKClient is the gRPC client wrapper for SDK calls
type GRPCSDKClient struct {
	client proto.SDKServiceClient
}

// ConfigGet calls the ConfigGet RPC
func (c *GRPCSDKClient) ConfigGet(ctx context.Context, key string) (interface{}, error) {
	resp, err := c.client.ConfigGet(ctx, &proto.ConfigGetRequest{Key: key})
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}

	var value interface{}
	if err := json.Unmarshal(resp.Value, &value); err != nil {
		return nil, err
	}

	return value, nil
}

// ConfigGetString calls the ConfigGetString RPC
func (c *GRPCSDKClient) ConfigGetString(ctx context.Context, key string) (string, error) {
	resp, err := c.client.ConfigGetString(ctx, &proto.ConfigGetStringRequest{Key: key})
	if err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", fmt.Errorf(resp.Error)
	}

	return resp.Value, nil
}

// ConfigSet calls the ConfigSet RPC
func (c *GRPCSDKClient) ConfigSet(ctx context.Context, key string, value interface{}) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	resp, err := c.client.ConfigSet(ctx, &proto.ConfigSetRequest{Key: key, Value: jsonValue})
	if err != nil {
		return err
	}

	if resp.Error != "" {
		return fmt.Errorf(resp.Error)
	}

	return nil
}

// ConfigDelete calls the ConfigDelete RPC
func (c *GRPCSDKClient) ConfigDelete(ctx context.Context, key string) error {
	resp, err := c.client.ConfigDelete(ctx, &proto.ConfigDeleteRequest{Key: key})
	if err != nil {
		return err
	}

	if resp.Error != "" {
		return fmt.Errorf(resp.Error)
	}

	return nil
}
