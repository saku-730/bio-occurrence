package infrastructure

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// FusekiClient: Fusekiとの通信を担当する構造体
type FusekiClient struct {
	BaseURL  string
	UpdateURL string
	QueryURL  string
	Username string
	Password string
	Client   *http.Client
}

func NewFusekiClient(baseURL, user, pass string) *FusekiClient {
	return &FusekiClient{
		BaseURL:   baseURL,
		UpdateURL: baseURL + "/update",
		QueryURL:  baseURL + "/query",
		Username:  user,
		Password:  pass,
		Client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// SendQuery: 共通のクエリ送信メソッド
func (fc *FusekiClient) Send(method, url, contentType, body string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader =  // ... (strings.NewReaderなどをここに書くか、呼び出し元でやるか)
		// 簡易化のため、ここでは strings.NewReader(body) をラップする形にするなら import "strings" 必要
	}
    // ... 実装の詳細は元の sendToFuseki / queryFuseki を汎用化してここに置く
    return nil, nil // (長くなるので省略、概念だけ)
}
