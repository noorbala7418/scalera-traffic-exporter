package model

type VmList struct {
	Username string `json:"username"`
	VMId     string `json:"vpsid"`
}

type VM struct {
	VpsID         int    `json:"vpsid"`
	VMInformation VMInfo `json:"info"`
}

type VMInfo struct {
	Hostname  string    `json:"hostname"`
	Bandwidth VMTraffic `json:"bandwidth"`
}

type VMTraffic struct {
	TotalTraffic       int     `json:"limit_gb"`
	UsedTraffic        float64 `json:"used_gb"`
	UsedTrafficPercent float64 `json:"percent"`
	FreeTraffic        float64 `json:"free_gb"`
	FreeTrafficPercent float64 `json:"percent_free"`
}
