package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/marketplace/search/internal/domain"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAll fetches the entire catalog by paginating through the products endpoint.
func (c *Client) GetAll(ctx context.Context) ([]domain.Product, error) {
	const pageSize = 500
	var all []domain.Product
	offset := 0

	for {
		page, total, err := c.fetchPage(ctx, pageSize, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		offset += len(page)

		if len(page) < pageSize || (total > 0 && int64(offset) >= total) {
			break
		}
	}
	return all, nil
}

func (c *Client) fetchPage(ctx context.Context, limit, offset int) ([]domain.Product, int64, error) {
	url := fmt.Sprintf("%s/products?limit=%d&offset=%d", c.baseURL, limit, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("catalog returned %d", resp.StatusCode)
	}

	var result struct {
		Products []domain.Product `json:"products"`
		Total    int64            `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, err
	}
	return result.Products, result.Total, nil
}
