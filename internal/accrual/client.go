package accrual

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(base string) *Client {
	return &Client{
		BaseURL: base,
		HTTP:    &http.Client{Timeout: 5 * time.Second},
	}
}

type Response struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`            // REGISTERED|INVALID|PROCESSING|PROCESSED
	Accrual *float64 `json:"accrual,omitempty"` // может отсутствовать
}

// GetInfo ходит в GET /api/orders/{number}
func (c *Client) GetInfo(number string) (resp Response, httpStatus int, retryAfter time.Duration, err error) {
	if c.BaseURL == "" {
		return resp, 0, 0, errors.New("accrual base url empty")
	}
	url := fmt.Sprintf("%s/api/orders/%s", c.BaseURL, number)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	r, err := c.HTTP.Do(req)
	if err != nil {
		return resp, 0, 0, err
	}
	defer r.Body.Close()

	httpStatus = r.StatusCode
	if r.StatusCode == http.StatusTooManyRequests { // 429
		if v := r.Header.Get("Retry-After"); v != "" {
			if sec, _ := strconv.Atoi(v); sec > 0 {
				retryAfter = time.Duration(sec) * time.Second
			}
		}
		return resp, httpStatus, retryAfter, nil
	}
	if r.StatusCode == http.StatusNoContent { // 204 — не зарегистрирован
		return resp, httpStatus, 0, nil
	}
	if r.StatusCode != http.StatusOK { // 500 и прочее
		return resp, httpStatus, 0, nil
	}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&resp); err != nil {
		return resp, httpStatus, 0, err
	}
	return resp, httpStatus, 0, nil
}
