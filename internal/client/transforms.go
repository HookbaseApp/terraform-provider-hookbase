package client

import (
	"context"
	"fmt"
)

type Transform struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"organizationId"`
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	Description    *string `json:"description"`
	Code           string  `json:"code"`
	// API returns camelCase on GET, snake_case on POST — handle both
	TransformType      string `json:"transformType"`
	TransformTypeSnake string `json:"transform_type"`
	InputFormat        string `json:"inputFormat"`
	InputFormatSnake   string `json:"input_format"`
	OutputFormat       string `json:"outputFormat"`
	OutputFormatSnake  string `json:"output_format"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

// GetTransformType returns the transform type from whichever field was populated.
func (t *Transform) GetTransformType() string {
	if t.TransformType != "" {
		return t.TransformType
	}
	return t.TransformTypeSnake
}

// GetInputFormat returns the input format from whichever field was populated.
func (t *Transform) GetInputFormat() string {
	if t.InputFormat != "" {
		return t.InputFormat
	}
	return t.InputFormatSnake
}

// GetOutputFormat returns the output format from whichever field was populated.
func (t *Transform) GetOutputFormat() string {
	if t.OutputFormat != "" {
		return t.OutputFormat
	}
	return t.OutputFormatSnake
}

type CreateTransformRequest struct {
	Name          string  `json:"name"`
	Slug          string  `json:"slug,omitempty"`
	Description   *string `json:"description,omitempty"`
	Code          string  `json:"code"`
	TransformType *string `json:"transformType,omitempty"`
	InputFormat   *string `json:"inputFormat,omitempty"`
	OutputFormat  *string `json:"outputFormat,omitempty"`
}

type UpdateTransformRequest struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	Code         *string `json:"code,omitempty"`
	InputFormat  *string `json:"inputFormat,omitempty"`
	OutputFormat *string `json:"outputFormat,omitempty"`
}

type transformResponse struct {
	Transform Transform `json:"transform"`
}

func (c *Client) CreateTransform(ctx context.Context, req CreateTransformRequest) (*Transform, error) {
	var resp transformResponse
	if err := c.Post(ctx, "/transforms", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Transform, nil
}

func (c *Client) GetTransform(ctx context.Context, id string) (*Transform, error) {
	var resp transformResponse
	if err := c.Get(ctx, fmt.Sprintf("/transforms/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp.Transform, nil
}

func (c *Client) UpdateTransform(ctx context.Context, id string, req UpdateTransformRequest) (*Transform, error) {
	var resp transformResponse
	if err := c.Patch(ctx, fmt.Sprintf("/transforms/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Transform, nil
}

func (c *Client) DeleteTransform(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/transforms/%s", id))
}
