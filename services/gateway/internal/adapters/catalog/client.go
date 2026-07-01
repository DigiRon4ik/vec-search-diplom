package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/marketplace/gateway/internal/domain"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetByIDs(ctx context.Context, ids []int64) (map[int64]domain.Product, error) {
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = fmt.Sprintf("%d", id)
	}

	url := fmt.Sprintf("%s/products?ids=%s", c.baseURL, strings.Join(idStrs, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog service returned %d", resp.StatusCode)
	}

	var result domain.CatalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	m := make(map[int64]domain.Product, len(result.Products))
	for _, p := range result.Products {
		m[p.ID] = p
	}
	return m, nil
}
