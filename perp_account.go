package huobiapi

import (
	"bytes"
	"net/http"
	"strings"
)

type PerpCrossAccountInfoResponse struct {
	Status string `json:"status"`
	Data   []struct {
		MarginMode        string               `json:"margin_mode"`
		MarginAccount     string               `json:"margin_account"`
		MarginAsset       string               `json:"margin_asset"`
		MarginBalance     float64              `json:"margin_balance"`
		MarginStatic      float64              `json:"margin_static"`
		MarginPosition    float64              `json:"margin_position"`
		MarginFrozen      float64              `json:"margin_frozen"`
		ProfitReal        float64              `json:"profit_real"`
		ProfitUnreal      float64              `json:"profit_unreal"`
		WithdrawAvailable float64              `json:"withdraw_available"`
		RiskRate          float64              `json:"risk_rate"`
		ContractDetail    []ContractDetailData `json:"contract_detail"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

type ContractDetailData struct {
	Symbol           string  `json:"symbol"`
	ContractCode     string  `json:"contract_code"`
	MarginPosition   float64 `json:"margin_position"`
	MarginFrozen     float64 `json:"margin_frozen"`
	MarginAvailable  float64 `json:"margin_available"`
	ProfitUnreal     float64 `json:"profit_unreal"`
	LiquidationPrice float64 `json:"liquidation_price"`
	LeverRate        int     `json:"lever_rate"`
	AdjustFactor     float64 `json:"adjust_factor"`
}

func (p *Client) PerpAccountInfo() (*PerpCrossAccountInfoResponse, error) {
	res, err := p.sendRequest("swap", http.MethodPost, "/linear-swap-api/v1/swap_cross_account_info", nil, nil, true)
	if err != nil {
		return nil, err
	}
	var result PerpCrossAccountInfoResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type PerpCrossAccountPositionResponse struct {
	Status string `json:"status"`
	Data   struct {
		Positions         []PositionData       `json:"positions"`
		MarginMode        string               `json:"margin_mode"`
		MarginAccount     string               `json:"margin_account"`
		MarginAsset       string               `json:"margin_asset"`
		MarginBalance     float64              `json:"margin_balance"`
		MarginStatic      float64              `json:"margin_static"`
		MarginPosition    float64              `json:"margin_position"`
		MarginFrozen      float64              `json:"margin_frozen"`
		ProfitReal        float64              `json:"profit_real"`
		ProfitUnreal      float64              `json:"profit_unreal"`
		WithdrawAvailable float64              `json:"withdraw_available"`
		RiskRate          float64              `json:"risk_rate"`
		ContractDetail    []ContractDetailData `json:"contract_detail"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

func (p *Client) PerpAccountPositionInfo() (*PerpCrossAccountPositionResponse, error) {
	params := make(map[string]string)
	params["margin_account"] = "USDT"
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("swap", http.MethodPost, "/linear-swap-api/v1/swap_cross_account_position_info", body, &params, true)
	if err != nil {
		return nil, err
	}
	var result PerpCrossAccountPositionResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type PerpPositionResponse struct {
	Status string         `json:"status"`
	Data   []PositionData `json:"data"`
	Ts     int64          `json:"ts"`
}

func (p *Client) PerpPositionInfo() (*PerpPositionResponse, error) {
	res, err := p.sendRequest("swap", http.MethodPost, "/linear-swap-api/v1/swap_cross_position_info", nil, nil, true)
	if err != nil {
		return nil, err
	}
	var result PerpPositionResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type PositionData struct {
	Symbol         string  `json:"symbol"`
	ContractCode   string  `json:"contract_code"`
	Volume         float64 `json:"volume"`
	Available      float64 `json:"available"`
	Frozen         float64 `json:"frozen"`
	CostOpen       float64 `json:"cost_open"`
	CostHold       float64 `json:"cost_hold"`
	ProfitUnreal   float64 `json:"profit_unreal"`
	ProfitRate     float64 `json:"profit_rate"`
	LeverRate      int     `json:"lever_rate"`
	PositionMargin float64 `json:"position_margin"`
	Direction      string  `json:"direction"`
	Profit         float64 `json:"profit"`
	LastPrice      float64 `json:"last_price"`
	MarginAsset    string  `json:"margin_asset"`
	MarginMode     string  `json:"margin_mode"`
	MarginAccount  string  `json:"margin_account"`
}

type PerpIsoAccountPositionResponse struct {
	Status string `json:"status"`
	Data   []struct {
		Symbol            string         `json:"symbol"`
		ContractCode      string         `json:"contract_code"`
		MarginBalance     float64        `json:"margin_balance"`
		MarginPosition    float64        `json:"margin_position"`
		MarginFrozen      float64        `json:"margin_frozen"`
		MarginAvailable   float64        `json:"margin_available"`
		ProfitReal        float64        `json:"profit_real"`
		ProfitUnreal      float64        `json:"profit_unreal"`
		RiskRate          float64        `json:"risk_rate"`
		WithdrawAvailable float64        `json:"withdraw_available"`
		LiquidationPrice  float64        `json:"liquidation_price"`
		LeverRate         int            `json:"lever_rate"`
		AdjustFactor      float64        `json:"adjust_factor"`
		MarginStatic      float64        `json:"margin_static"`
		Positions         []PositionData `json:"positions"`
		MarginAsset       string         `json:"margin_asset"`
		MarginMode        string         `json:"margin_mode"`
		MarginAccount     string         `json:"margin_account"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

func (p *Client) PerpIsoAccountPositionInfo(symbol string) (*PerpIsoAccountPositionResponse, error) {
	params := make(map[string]string)
	params["contract_code"] = strings.ToUpper(symbol)
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	res, err := p.sendRequest("swap", http.MethodPost, "/linear-swap-api/v1/swap_account_position_info", body, nil, true)
	if err != nil {
		return nil, err
	}
	var result PerpIsoAccountPositionResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type PerpIsoAccountResponse struct {
	Status string                `json:"status"`
	Data   []*PerpIsoAccountData `json:"data"`
	Ts     int64                 `json:"ts"`
}

type PerpIsoAccountData struct {
	Symbol            string  `json:"symbol"`
	MarginBalance     float64 `json:"margin_balance"`
	MarginPosition    float64 `json:"margin_position"`
	MarginFrozen      float64 `json:"margin_frozen"`
	MarginAvailable   float64 `json:"margin_available"`
	ProfitReal        float64 `json:"profit_real"`
	ProfitUnreal      float64 `json:"profit_unreal"`
	RiskRate          float64 `json:"risk_rate"`
	WithdrawAvailable float64 `json:"withdraw_available"`
	LiquidationPrice  float64 `json:"liquidation_price"`
	LeverRate         int     `json:"lever_rate"`
	AdjustFactor      float64 `json:"adjust_factor"`
	MarginStatic      float64 `json:"margin_static"`
	ContractCode      string  `json:"contract_code"`
	MarginAsset       string  `json:"margin_asset"`
	MarginMode        string  `json:"margin_mode"`
	MarginAccount     string  `json:"margin_account"`
}

func (p *Client) PerpIsoAccountInfo() (*PerpIsoAccountResponse, error) {
	res, err := p.sendRequest("swap", http.MethodPost, "/linear-swap-api/v1/swap_account_info", nil, nil, true)
	if err != nil {
		return nil, err
	}
	var result PerpIsoAccountResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Client) PerpIsoPositionInfo() (*PerpPositionResponse, error) {
	res, err := p.sendRequest("swap", http.MethodPost, "/linear-swap-api/v1/swap_position_info", nil, nil, true)
	if err != nil {
		return nil, err
	}
	var result PerpPositionResponse
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
