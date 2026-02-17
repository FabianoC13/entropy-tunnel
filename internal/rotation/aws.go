package rotation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// AWSController rotates endpoints via AWS Lambda@Edge.
type AWSController struct {
	NoOpController
	region    string
	accessKey string
	secretKey string
	client    *http.Client
}

// NewAWSController creates an AWS Lambda@Edge rotation backend.
func NewAWSController(region, accessKey, secretKey string, logger *zap.Logger) *AWSController {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AWSController{
		NoOpController: NoOpController{
			endpoints: make([]*Endpoint, 0),
			logger:    logger,
			stopCh:    make(chan struct{}),
		},
		region:    region,
		accessKey: accessKey,
		secretKey: secretKey,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Rotate deploys a new AWS Lambda function and creates a function URL.
func (a *AWSController) Rotate(ctx context.Context) (*Endpoint, error) {
	a.mu.Lock()
	a.counter++
	functionName := fmt.Sprintf("entropy-lambda-%d-%d", time.Now().Unix(), a.counter)
	a.mu.Unlock()

	a.logger.Info("deploying new aws lambda",
		zap.String("name", functionName),
		zap.String("region", a.region),
	)

	// 1. Create Lambda function
	if err := a.createFunction(ctx, functionName); err != nil {
		return nil, fmt.Errorf("create lambda %s: %w", functionName, err)
	}

	// 2. Create function URL for direct invocation
	funcURL, err := a.createFunctionURL(ctx, functionName)
	if err != nil {
		// Cleanup on failure
		_ = a.deleteFunction(ctx, functionName)
		return nil, fmt.Errorf("create function URL %s: %w", functionName, err)
	}

	ep := &Endpoint{
		ID:        functionName,
		Address:   funcURL,
		Region:    a.region,
		Provider:  "aws",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Metadata: map[string]string{
			"function_name": functionName,
			"type":          "lambda",
		},
	}

	a.mu.Lock()
	a.endpoints = append(a.endpoints, ep)
	a.mu.Unlock()

	a.logger.Info("aws lambda deployed",
		zap.String("id", ep.ID),
		zap.String("url", funcURL),
	)

	return ep, nil
}

// Retire tears down an AWS Lambda endpoint.
func (a *AWSController) Retire(ctx context.Context, ep *Endpoint) error {
	if ep.Provider != "aws" {
		return a.NoOpController.Retire(ctx, ep)
	}

	a.logger.Info("retiring aws lambda", zap.String("name", ep.ID))

	if err := a.deleteFunction(ctx, ep.ID); err != nil {
		a.logger.Warn("failed to delete lambda", zap.Error(err))
	}

	a.mu.Lock()
	for i, e := range a.endpoints {
		if e.ID == ep.ID {
			a.endpoints = append(a.endpoints[:i], a.endpoints[i+1:]...)
			break
		}
	}
	a.mu.Unlock()

	return nil
}

func (a *AWSController) createFunction(ctx context.Context, name string) error {
	apiURL := fmt.Sprintf("https://lambda.%s.amazonaws.com/2015-03-31/functions", a.region)

	payload, _ := json.Marshal(map[string]any{
		"FunctionName": name,
		"Runtime":      "nodejs20.x",
		"Handler":      "index.handler",
		"Role":         "arn:aws:iam::role/entropy-lambda-role",
		"Code": map[string]any{
			"ZipFile": lambdaProxyCode(),
		},
		"Timeout":    30,
		"MemorySize": 128,
		"Tags": map[string]string{
			"project":  "entropy-tunnel",
			"rotation": "auto",
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	a.signRequest(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("lambda create error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (a *AWSController) createFunctionURL(ctx context.Context, name string) (string, error) {
	apiURL := fmt.Sprintf(
		"https://lambda.%s.amazonaws.com/2021-10-31/functions/%s/url",
		a.region, name,
	)

	payload, _ := json.Marshal(map[string]any{
		"AuthType": "NONE",
		"InvokeMode": "RESPONSE_STREAM",
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	a.signRequest(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		FunctionURL string `json:"FunctionUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.FunctionURL, nil
}

func (a *AWSController) deleteFunction(ctx context.Context, name string) error {
	apiURL := fmt.Sprintf(
		"https://lambda.%s.amazonaws.com/2015-03-31/functions/%s",
		a.region, name,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, apiURL, nil)
	if err != nil {
		return err
	}
	a.signRequest(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// signRequest adds AWS Signature V4 headers.
// Simplified implementation for MVP; production should use aws-sdk-go-v2.
func (a *AWSController) signRequest(req *http.Request) {
	req.Header.Set("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))
	// In production, this would use proper SigV4:
	//   signer := v4.NewSigner()
	//   signer.SignHTTP(ctx, credentials, req, payloadHash, "lambda", region, time.Now())
	// For now, we set the access key placeholder.
	req.Header.Set("Authorization",
		fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s/%s/lambda/aws4_request",
			a.accessKey,
			time.Now().UTC().Format("20060102"),
			a.region,
		),
	)
}

// lambdaProxyCode returns base64-encoded Lambda proxy function.
func lambdaProxyCode() string {
	// Lambda handler that proxies WebSocket connections to the tunnel server.
	// In production, this would be a proper ZIP file.
	return "UEsDBBQAAAAIAA==" // placeholder zip
}
