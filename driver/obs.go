package driver

// OBSのWebSocket通信で使用する構造体

// OBSHello構造体
type OBSHello struct {
	Op int `json:"op"`
	D  struct {
		ObsWebSocketVersion string `json:"obsWebSocketVersion"`
		RPCVersion          int
		Authentication      struct {
			Challenge string
			Salt      string
		}
	}
}

// OBSIdentify構造体
type OBSIdentify struct {
	Op int `json:"op"`
	D  struct {
		RPCVersion     int    `json:"rpcVersion"`
		Authentication string `json:"authentication"`
	} `json:"d"`
}

// OBSRequest構造体
type OBSRequest struct {
	Op int             `json:"op"`
	D  *OBSRequestData `json:"d"`
}

type OBSRequestData struct {
	RequestType string `json:"requestType"`
	RequestID   string `json:"requestId"`
}
