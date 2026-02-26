package cloudflare

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/InTheForests/wgcf/config"
	"github.com/InTheForests/wgcf/openapi"
	"github.com/InTheForests/wgcf/util"
	"github.com/InTheForests/wgcf/wireguard"
	"github.com/cockroachdb/errors"
)

const (
	ApiUrl     = "https://api.cloudflareclient.com"
	ApiVersion = "v0a1922"
)

var (
	DefaultHeaders = map[string]string{
		"User-Agent":        "okhttp/3.12.1",
		"CF-Client-Version": "a-6.3-1922",
	}
	DefaultTransport = &http.Transport{
		// Match app's TLS config or API will reject us with code 403 error 1020
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			MaxVersion: tls.VersionTLS12},
		ForceAttemptHTTP2: false,
		// From http.DefaultTransport
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
)

func NewApiClient(authToken *string, transport *http.Transport) *openapi.APIClient {
	if transport == nil {
		transport = DefaultTransport
	}
	httpClient := http.Client{Transport: transport}
	apiClient := openapi.NewAPIClient(&openapi.Configuration{
		DefaultHeader: DefaultHeaders,
		UserAgent:     DefaultHeaders["User-Agent"],
		Debug:         false,
		Servers: []openapi.ServerConfiguration{
			{URL: ApiUrl},
		},
		HTTPClient: &httpClient,
	})
	if authToken != nil {
		apiClient.GetConfig().DefaultHeader["Authorization"] = "Bearer " + *authToken
	}
	return apiClient
}

func Register(apiClient *openapi.APIClient, publicKey *wireguard.Key, deviceModel string) (*openapi.Register200Response, error) {
	timestamp := util.GetTimestamp()
	result, _, err := apiClient.DefaultAPI.
		Register(nil, ApiVersion).
		RegisterRequest(openapi.RegisterRequest{
			FcmToken:  "", // not empty on actual client
			InstallId: "", // not empty on actual client
			Key:       publicKey.String(),
			Locale:    "en_US",
			Model:     deviceModel,
			Tos:       timestamp,
			Type:      "Android",
		}).Execute()
	return result, errors.WithStack(err)
}

type SourceDevice openapi.GetSourceDevice200Response

func GetSourceDevice(apiClient *openapi.APIClient, ctx *config.Context) (*SourceDevice, error) {
	apiClient.GetConfig().DefaultHeader["Authorization"] = "Bearer " + ctx.AccessToken
	result, _, err := apiClient.DefaultAPI.
		GetSourceDevice(nil, ApiVersion, ctx.DeviceId).
		Execute()
	return (*SourceDevice)(result), errors.WithStack(err)
}

type Account openapi.Account

func GetAccount(apiClient *openapi.APIClient, ctx *config.Context) (*Account, error) {
	apiClient.GetConfig().DefaultHeader["Authorization"] = "Bearer " + ctx.AccessToken
	result, _, err := apiClient.DefaultAPI.
		GetAccount(nil, ctx.DeviceId, ApiVersion).
		Execute()
	castResult := (*Account)(result)
	return castResult, errors.WithStack(err)
}

func UpdateLicenseKey(apiClient *openapi.APIClient, ctx *config.Context) (*openapi.UpdateAccount200Response, error) {
	apiClient.GetConfig().DefaultHeader["Authorization"] = "Bearer " + ctx.AccessToken
	result, _, err := apiClient.DefaultAPI.
		UpdateAccount(nil, ctx.DeviceId, ApiVersion).
		UpdateAccountRequest(openapi.UpdateAccountRequest{License: ctx.LicenseKey}).
		Execute()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return result, nil
}

type BoundDevice openapi.BoundDevice

func GetBoundDevices(apiClient *openapi.APIClient, ctx *config.Context) ([]BoundDevice, error) {
	apiClient.GetConfig().DefaultHeader["Authorization"] = "Bearer " + ctx.AccessToken
	result, _, err := apiClient.DefaultAPI.
		GetBoundDevices(nil, ctx.DeviceId, ApiVersion).
		Execute()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var castResult []BoundDevice
	for _, device := range result {
		castResult = append(castResult, BoundDevice(device))
	}
	return castResult, nil
}

func GetSourceBoundDevice(apiClient *openapi.APIClient, ctx *config.Context) (*BoundDevice, error) {
	result, err := GetBoundDevices(apiClient, ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return FindDevice(result, ctx.DeviceId)
}

func UpdateSourceBoundDeviceName(apiClient *openapi.APIClient, ctx *config.Context, targetDeviceId string, newName string) (*BoundDevice, error) {
	return updateSourceBoundDevice(apiClient, ctx, targetDeviceId, openapi.UpdateBoundDeviceRequest{
		Name: &newName,
	})
}

func UpdateSourceBoundDeviceActive(apiClient *openapi.APIClient, ctx *config.Context, targetDeviceId string, active bool) (*BoundDevice, error) {
	return updateSourceBoundDevice(apiClient, ctx, targetDeviceId, openapi.UpdateBoundDeviceRequest{
		Active: &active,
	})
}

func updateSourceBoundDevice(apiClient *openapi.APIClient, ctx *config.Context, targetDeviceId string, data openapi.UpdateBoundDeviceRequest) (*BoundDevice, error) {
	apiClient.GetConfig().DefaultHeader["Authorization"] = "Bearer " + ctx.AccessToken
	result, _, err := apiClient.DefaultAPI.
		UpdateBoundDevice(nil, ctx.DeviceId, ApiVersion, targetDeviceId).
		UpdateBoundDeviceRequest(data).
		Execute()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var castResult []BoundDevice
	for _, device := range result {
		castResult = append(castResult, BoundDevice(device))
	}
	return FindDevice(castResult, ctx.DeviceId)
}

func DeleteBoundDevice(apiClient *openapi.APIClient, ctx *config.Context, targetDeviceId string) error {
	apiClient.GetConfig().DefaultHeader["Authorization"] = "Bearer " + ctx.AccessToken
	if _, err := apiClient.DefaultAPI.
		DeleteBoundDevice(nil, ctx.DeviceId, ApiVersion, targetDeviceId).
		Execute(); err != nil {
		return err
	}
	return nil
}
