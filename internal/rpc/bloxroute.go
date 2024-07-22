package rpc

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/iqbalbaharum/go-solana-mev-bot/internal/config"
)

type BloxRouteResponse struct {
	Signature string `json:"signature"`
}

func SubmitBloxRouteTransaction(transaction *solana.Transaction, useStakedRPCs bool) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	msg, err := transaction.MarshalBinary()

	if err != nil {
		return "", err
	}

	requestBody := map[string]interface{}{
		"transaction": map[string]string{
			"content": base64.StdEncoding.EncodeToString(msg),
		},
		"skipPreFlight":          true,
		"frontRunningProtection": false,
		"useStakedRPCs":          useStakedRPCs,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", config.BloxRouteUrl, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Authorization", config.BloxRouteToken)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var reader io.ReadCloser

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
	case "deflate":
		reader, err = zlib.NewReader(resp.Body)
	default:
		reader = resp.Body
	}

	if err != nil {
		return "", err
	}
	defer reader.Close()

	var responseBody strings.Builder
	if _, err := io.Copy(&responseBody, reader); err != nil {
		return "", err
	}

	var response BloxRouteResponse
	if err := json.Unmarshal([]byte(responseBody.String()), &response); err != nil {
		return "", err
	}

	if response.Signature == "" {
		return "", errors.New("no signature returned from BloxRoute")
	}

	return response.Signature, nil
}
