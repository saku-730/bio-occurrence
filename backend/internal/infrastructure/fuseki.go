package infrastructure

import (
	"io"
	"net/http"
	"strings" // ★追加: strings.NewReaderを使うために必要
	"time"
)

// FusekiClient: Fusekiとの通信を担当する構造体
type FusekiClient struct {
	BaseURL   string
	UpdateURL string
	QueryURL  string
	Username  string
	Password  string
	Client    *http.Client
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

// Send: 共通のクエリ送信メソッド
// method: "GET" or "POST"
// url: 送信先URL
// contentType: ヘッダー指定 (空文字なら設定しない)
// body: 送信するデータ (SPARQLクエリなど)
func (fc *FusekiClient) Send(method, url, contentType, body string) (*http.Response, error) {
	var bodyReader io.Reader
	
	// bodyがある場合はReaderに変換する
	if body != "" {
		bodyReader = strings.NewReader(body) // ★ここを修正しました
	}

	// リクエストの作成
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// ヘッダー設定
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Basic認証の設定
	req.SetBasicAuth(fc.Username, fc.Password)

	// 送信実行
	return fc.Client.Do(req)
}
