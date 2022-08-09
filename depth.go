package huobiapi

import (
	"bytes"
	"net/http"
)

type SpotDepthResponse struct {
	Ch     string `json:"ch"`
	Status string `json:"status"`
	Ts     int64  `json:"ts"`
	Tick   struct {
		Ts      int64       `json:"ts"`
		Version int64       `json:"version"`
		Bids    [][]float64 `json:"bids"`
		Asks    [][]float64 `json:"asks"`
	} `json:"tick"`
}

func (p *Client) SpotDepth(symbol string) (*SpotDepthResponse, error) {
	params := make(map[string]string)
	params["symbol"] = symbol
	params["type"] = "step0"
	res, err := p.sendRequest("spot", http.MethodGet, "/market/depth", nil, &params, false)
	if err != nil {
		return nil, err
	}
	var result SpotDepthResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
