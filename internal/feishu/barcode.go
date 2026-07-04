package feishu

import (
	"bytes"
	"fmt"
	"image/png"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
)

func GenerateCode128PNG(content string) ([]byte, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("empty barcode content")
	}
	bc, err := code128.Encode(content)
	if err != nil {
		return nil, err
	}
	img, err := barcode.Scale(bc, 360, 80)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
