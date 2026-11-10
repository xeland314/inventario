package scanner

// SampleProduct representa un producto del catálogo de demostración para pruebas con el escáner.
type SampleProduct struct {
	Code        string  `json:"code"`
	Format      string  `json:"format"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Description string  `json:"description"`
	BadgeColor  string  `json:"badge_color"`
}

// SampleProducts catálogo de pruebas integrado directamente desde escaner_codigo_barras.
var SampleProducts = []SampleProduct{
	{
		Code:        "7501031311309",
		Format:      "ean13",
		Name:        "Refresco Coca-Cola 600ml",
		Category:    "Bebidas",
		Price:       18.50,
		Stock:       45,
		Description: "Bebida carbonatada clásica en botella PET.",
		BadgeColor:  "emerald",
	},
	{
		Code:        "7501000111203",
		Format:      "ean13",
		Name:        "Pan Blanco de Caja Bimbo Grande",
		Category:    "Panadería",
		Price:       42.00,
		Stock:       28,
		Description: "Pan de molde enriquecido con vitaminas y minerales.",
		BadgeColor:  "emerald",
	},
	{
		Code:        "7501008045612",
		Format:      "ean13",
		Name:        "Leche Entera Alpura 1 Litro",
		Category:    "Lácteos",
		Price:       26.50,
		Stock:       60,
		Description: "Leche 100% pura de vaca pasteurizada.",
		BadgeColor:  "emerald",
	},
	{
		Code:        "9780132350884",
		Format:      "ean13",
		Name:        "Libro: Clean Code (Robert C. Martin)",
		Category:    "Libros",
		Price:       550.00,
		Stock:       12,
		Description: "Manual de desarrollo ágil de software y buenas prácticas.",
		BadgeColor:  "emerald",
	},
	{
		Code:        "012345678905",
		Format:      "upca",
		Name:        "Cereal Cheerios Avena 500g",
		Category:    "Despensa",
		Price:       65.00,
		Stock:       30,
		Description: "Cereal de avena integral en caja familiar.",
		BadgeColor:  "amber",
	},
	{
		Code:        "049000000443",
		Format:      "upca",
		Name:        "Bebida Sprite Lima-Limón 355ml",
		Category:    "Bebidas",
		Price:       16.00,
		Stock:       52,
		Description: "Refresco sabor lima limón en lata de aluminio.",
		BadgeColor:  "amber",
	},
	{
		Code:        "SKU-LAPTOP-M3",
		Format:      "code128",
		Name:        "Laptop ThinkPad Ultra 16 Pulgadas",
		Category:    "Electrónica",
		Price:       18499.00,
		Stock:       8,
		Description: "Portátil empresarial con procesador de alto rendimiento y 32GB RAM.",
		BadgeColor:  "indigo",
	},
	{
		Code:        "SN-SYM-05E0-1200",
		Format:      "code128",
		Name:        "Escáner Láser Symbol Technologies USB",
		Category:    "Hardware POS",
		Price:       1250.00,
		Stock:       15,
		Description: "Lector óptico de código de barras 1D de alta velocidad.",
		BadgeColor:  "indigo",
	},
	{
		Code:        "PKG-FEDEX-98214",
		Format:      "code128",
		Name:        "Paquete Despacho Prioritario FedEx",
		Category:    "Logística",
		Price:       240.00,
		Stock:       1,
		Description: "Guía de envío rastreable internacional.",
		BadgeColor:  "indigo",
	},
	{
		Code:        "MOTOR-MOD44",
		Format:      "code39",
		Name:        "Motor NEMA 23 Paso a Paso",
		Category:    "Componentes",
		Price:       820.00,
		Stock:       22,
		Description: "Motor de alta precisión para máquinas CNC e impresoras 3D.",
		BadgeColor:  "cyan",
	},
	{
		Code:        "LOTE-OCT-2026",
		Format:      "code39",
		Name:        "Materia Prima Polímero Grado A",
		Category:    "Insumos",
		Price:       3400.00,
		Stock:       50,
		Description: "Tambor de resina plástica certificada de alta resistencia.",
		BadgeColor:  "cyan",
	},
	{
		Code:        "96385074",
		Format:      "ean8",
		Name:        "Chicles Trident Menta 18s",
		Category:    "Golosinas",
		Price:       14.00,
		Stock:       95,
		Description: "Goma de mascar sin azúcar sabor menta fresca.",
		BadgeColor:  "rose",
	},
	{
		Code:        "12345670",
		Format:      "ean8",
		Name:        "Barra Chocolate Amargo 30g",
		Category:    "Confitería",
		Price:       22.00,
		Stock:       70,
		Description: "Chocolate 70% cacao puro de origen sostenible.",
		BadgeColor:  "rose",
	},
	{
		Code:        "https://github.com",
		Format:      "qrcode",
		Name:        "Enlace GitHub Repositorio",
		Category:    "Digital 2D",
		Price:       0.00,
		Stock:       1,
		Description: "Código QR estándar matricial con URL web.",
		BadgeColor:  "violet",
	},
	{
		Code:        "PROD:SYM-SCAN;LOTE:2026;PRECIO:1250",
		Format:      "qrcode",
		Name:        "Ficha Técnica Estructurada Symbol",
		Category:    "Digital 2D",
		Price:       1250.00,
		Stock:       1,
		Description: "Matriz QR con datos estructurados para lectura industrial.",
		BadgeColor:  "violet",
	},
}

// FindSampleProductByCode busca en el catálogo demo si el código coincide.
func FindSampleProductByCode(code string) *SampleProduct {
	for _, p := range SampleProducts {
		if p.Code == code {
			cp := p
			return &cp
		}
	}
	return nil
}
