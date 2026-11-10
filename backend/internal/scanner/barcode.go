package scanner

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/ean"
	"github.com/boombuler/barcode/qr"
)

// SupportedBarcodeFormats lista los formatos admitidos.
var SupportedBarcodeFormats = map[string]string{
	"ean13":   "EAN-13 (Retail / Supermercado)",
	"code128": "Code 128 (Alfanumérico / Logística)",
	"code39":  "Code 39 (Alfanumérico Industrial)",
	"upca":    "UPC-A (Retail USA / 12 dígitos)",
	"ean8":    "EAN-8 (Empaques compactos)",
	"qrcode":  "Código QR (2D / Matricial)",
}

// NormalizeFormat normaliza el nombre del formato.
func NormalizeFormat(formatName string) string {
	fmtLower := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(formatName, "-", ""), "_", ""))
	switch fmtLower {
	case "qr", "qrcode":
		return "qrcode"
	case "ean13", "ean":
		return "ean13"
	case "code128", "c128":
		return "code128"
	case "code39", "c39":
		return "code39"
	case "upca", "upc":
		return "upca"
	case "ean8":
		return "ean8"
	default:
		return "code128"
	}
}

// EncodeBarcode crea la instancia de código de barras según el formato.
func EncodeBarcode(formatName, code string) (barcode.Barcode, error) {
	fmtNorm := NormalizeFormat(formatName)
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return nil, fmt.Errorf("el código no puede estar vacío")
	}

	switch fmtNorm {
	case "qrcode":
		return qr.Encode(cleanCode, qr.M, qr.Auto)
	case "ean13", "upca", "ean8":
		// Si es upca de 12 dígitos o ean13
		bc, err := ean.Encode(cleanCode)
		if err != nil {
			// Si falla por dígito verificador y tiene 13 caracteres, reintentar con 12
			if len(cleanCode) == 13 {
				if bc2, err2 := ean.Encode(cleanCode[:12]); err2 == nil {
					return bc2, nil
				}
			}
			return nil, err
		}
		return bc, nil
	case "code39":
		return code39.Encode(strings.ToUpper(cleanCode), false, true)
	case "code128":
		return code128.Encode(cleanCode)
	default:
		return code128.Encode(cleanCode)
	}
}

// GeneratePNG genera los bytes de la imagen PNG escalada.
func GeneratePNG(formatName, code string, width, height int) ([]byte, error) {
	bc, err := EncodeBarcode(formatName, code)
	if err != nil {
		return nil, err
	}

	if width <= 0 {
		width = 360
	}
	if height <= 0 {
		height = 120
	}
	if NormalizeFormat(formatName) == "qrcode" {
		if height < width {
			height = width
		} else {
			width = height
		}
	}

	scaled, err := barcode.Scale(bc, width, height)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, scaled); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GenerateSVG genera un SVG vectorial para nitidez perfecta en navegadores e impresiones.
func GenerateSVG(formatName, code string) (string, error) {
	bc, err := EncodeBarcode(formatName, code)
	if err != nil {
		return "", err
	}

	bounds := bc.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return "", fmt.Errorf("dimensiones inválidas del código")
	}

	fmtNorm := NormalizeFormat(formatName)
	is2D := fmtNorm == "qrcode"

	// Dimensiones visuales SVG recomendadas
	svgWidth := 340
	svgHeight := 100
	if is2D {
		svgWidth = 240
		svgHeight = 240
	}

	var rects strings.Builder
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := bc.At(x, y)
			r, g, b, _ := c.RGBA()
			// Si es barra negra (oscura)
			if r < 32768 && g < 32768 && b < 32768 {
				rects.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="1" height="1" fill="#000000"/>`, x-bounds.Min.X, y-bounds.Min.Y))
			}
		}
	}

	textElem := ""
	if !is2D {
		textElem = fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-family="monospace, monospace" font-size="12" fill="#0f172a">%s</text>`, w/2, h+14, code)
	}

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" style="background:#ffffff; max-width:100%%; height:auto;">
  <rect width="100%%" height="100%%" fill="#ffffff"/>
  <g transform="translate(0, 0)">
    %s
  </g>
  %s
</svg>`, w, h+(func() int {
		if is2D {
			return 0
		}
		return 18
	}()), svgWidth, svgHeight, rects.String(), textElem)

	return svg, nil
}

// CheckColorIsDark verifica si un pixel es oscuro.
func checkColorIsDark(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r < 32768 && g < 32768 && b < 32768
}

var _ = image.Black
