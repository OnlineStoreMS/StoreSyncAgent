package feishu

import (
	"bytes"
	"image/png"
	"testing"
)

func TestGenerateCode128PNG(t *testing.T) {
	pngBytes, err := GenerateCode128PNG("YT1234567890")
	if err != nil {
		t.Fatal(err)
	}
	if len(pngBytes) == 0 {
		t.Fatal("empty png")
	}
	if _, err := png.Decode(bytes.NewReader(pngBytes)); err != nil {
		t.Fatalf("invalid png: %v", err)
	}
}

func TestGenerateCode128PNGEmpty(t *testing.T) {
	if _, err := GenerateCode128PNG("  "); err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestCardBodyElementsFooterBarcode(t *testing.T) {
	elements := cardBodyElements(InteractiveCard{
		Markdown:     "**退货物流：** YT123",
		FooterImgKey: "img_test_key",
		FooterImgAlt: "YT1234567890",
	})
	if len(elements) != 3 {
		t.Fatalf("want 3 elements, got %d", len(elements))
	}
	img, ok := elements[2].(map[string]any)
	if !ok || img["tag"] != "img" || img["img_key"] != "img_test_key" {
		t.Fatalf("unexpected footer img: %#v", elements[2])
	}
}

func TestCardBodyElementsNoFooter(t *testing.T) {
	elements := cardBodyElements(InteractiveCard{Markdown: "hello"})
	if len(elements) != 1 {
		t.Fatalf("want 1 element, got %d", len(elements))
	}
}
