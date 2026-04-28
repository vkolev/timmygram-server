package qr

import (
	"bytes"
	"encoding/json"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

type QRPayload struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
}

func GenerateQRCode(payload QRPayload) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	qrCode, err := qr.Encode(string(payloadBytes), qr.M, qr.Auto)
	if err != nil {
		return nil, err
	}

	qrCode, err = barcode.Scale(qrCode, 300, 300)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, qrCode); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
