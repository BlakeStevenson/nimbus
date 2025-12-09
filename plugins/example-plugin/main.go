package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/blakestevenson/nimbus/internal/plugins"
	"github.com/hashicorp/go-plugin"
)

// ExamplePlugin implements the MediaSuitePlugin interface
type ExamplePlugin struct{}

// Metadata returns plugin metadata
func (p *ExamplePlugin) Metadata(ctx context.Context) (*plugins.PluginMetadata, error) {
	return &plugins.PluginMetadata{
		ID:           "example-plugin",
		Name:         "Example Plugin",
		Version:      "0.1.0",
		Description:  "A simple example plugin demonstrating the Nimbus plugin system",
		Capabilities: []string{"api", "ui"},
	}, nil
}

// APIRoutes returns the HTTP routes this plugin provides
func (p *ExamplePlugin) APIRoutes(ctx context.Context) ([]plugins.RouteDescriptor, error) {
	return []plugins.RouteDescriptor{
		{
			Method: "GET",
			Path:   "/api/plugins/example/hello",
			Auth:   "none", // Public endpoint
			Tag:    "",
		},
		{
			Method: "GET",
			Path:   "/api/plugins/example/status",
			Auth:   "session", // Requires authentication
			Tag:    "",
		},
	}, nil
}

// HandleAPI handles HTTP requests for this plugin's routes
func (p *ExamplePlugin) HandleAPI(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	switch req.Path {
	case "/api/plugins/example/hello":
		return p.handleHello(ctx, req)
	case "/api/plugins/example/status":
		return p.handleStatus(ctx, req)
	default:
		return &plugins.PluginHTTPResponse{
			StatusCode: http.StatusNotFound,
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
			Body:       []byte(`{"error":"Not found"}`),
		}, nil
	}
}

// handleHello returns a simple greeting
func (p *ExamplePlugin) handleHello(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	response := map[string]interface{}{
		"message": "Hello from the Example Plugin!",
		"version": "0.1.0",
		"plugin":  "example-plugin",
	}

	body, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return &plugins.PluginHTTPResponse{
		StatusCode: http.StatusOK,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: body,
	}, nil
}

// handleStatus returns plugin status (requires authentication)
func (p *ExamplePlugin) handleStatus(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	response := map[string]interface{}{
		"status":  "running",
		"plugin":  "example-plugin",
		"version": "0.1.0",
	}

	// If user is authenticated, include user info
	if req.UserID != nil {
		response["user_id"] = *req.UserID
		response["authenticated"] = true
	} else {
		response["authenticated"] = false
	}

	body, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return &plugins.PluginHTTPResponse{
		StatusCode: http.StatusOK,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: body,
	}, nil
}

// UIManifest returns the UI configuration for this plugin
func (p *ExamplePlugin) UIManifest(ctx context.Context) (*plugins.UIManifest, error) {
	return &plugins.UIManifest{
		NavItems: []plugins.UINavItem{
			{
				Label: "Example Plugin",
				Path:  "/plugins/example-plugin",
				Icon:  "puzzle",
			},
		},
		Routes: []plugins.UIRoute{
			{
				Path:      "/plugins/example-plugin",
				BundleURL: "/src/plugins-example-plugin.tsx",
			},
		},
	}, nil
}

// HandleEvent handles system events (not implemented in this example)
func (p *ExamplePlugin) HandleEvent(ctx context.Context, evt plugins.Event) error {
	// This example plugin doesn't handle events
	return nil
}

func main() {
	// Create plugin instance
	examplePlugin := &ExamplePlugin{}

	// Serve the plugin using go-plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.Handshake,
		Plugins: map[string]plugin.Plugin{
			"media-suite": &plugins.MediaSuitePluginGRPC{
				Impl: examplePlugin,
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
