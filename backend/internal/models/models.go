package models

var EstadosValidos = map[string]bool{
	"activo":           true,
	"disponible":       true,
	"reserva":          true, // alias de disponible, se mantiene por compatibilidad
	"en_mantenimiento": true,
	"en_reparacion":    true,
	"averiado":         true,
	"obsoleto":         true,
	"dado_de_baja":     true,
	"extraviado":       true,
}

var TiposComponenteValidos = map[string]bool{
	"Ventilador": true, "SSD": true, "HDD": true, "RAM": true, "Bateria": true,
	"Fuente": true, "Placa": true, "CPU": true, "GPU": true, "Teclado": true,
	"Pantalla": true, "Cargador": true, "Otro": true,
}

var TiposMovimientoValidos = map[string]bool{
	"asignacion":     true,
	"desasignacion":  true,
	"traslado":       true,
	"canibalizacion": true,
	"instalacion":    true,
	"retiro":         true,
}

func EsEstadoValido(estado string) bool {
	return EstadosValidos[estado]
}

func EsTipoComponenteValido(t string) bool {
	if t == "" {
		return true
	}
	return TiposComponenteValidos[t]
}

func EsTipoMovimientoValido(t string) bool {
	return TiposMovimientoValidos[t]
}

type Dispositivo struct {
	ID               string  `json:"id"`
	CodigoActivo     string  `json:"codigo_activo"`
	NumeroSerie      string  `json:"numero_serie"`
	Tipo             string  `json:"tipo"`
	Marca            string  `json:"marca"`
	Modelo           string  `json:"modelo"`
	Especificaciones string  `json:"especificaciones"`
	ValorCompra      float64 `json:"valor_compra"`
	Proveedor        string  `json:"proveedor"`
	Estado           string  `json:"estado"`
	Ubicacion        string  `json:"ubicacion"`
	AsignadoA        *string `json:"asignado_a"`
	FechaCompra      string  `json:"fecha_compra"`
	FechaBaja        *string `json:"fecha_baja"`
	MotivoBaja       *string `json:"motivo_baja"`
	ParentID         *string `json:"parent_id"`
	TipoComponente   *string `json:"tipo_componente"`
	Sede             *string `json:"sede"`
	CreadoEn         string  `json:"creado_en"`
	ActualizadoEn    *string `json:"actualizado_en"`
	Tintas           []NivelTinta  `json:"tintas,omitempty"`
	Componentes      []Dispositivo `json:"componentes,omitempty"`
}

type Movimiento struct {
	ID              string  `json:"id"`
	DispositivoID   string  `json:"dispositivo_id"`
	OrigenParentID  *string `json:"origen_parent_id"`
	DestinoParentID *string `json:"destino_parent_id"`
	TipoMovimiento  string  `json:"tipo_movimiento"`
	Descripcion     *string `json:"descripcion"`
	RealizadoPor    *string `json:"realizado_por"`
	FechaMovimiento string  `json:"fecha_movimiento"`
	CreadoEn        string  `json:"creado_en"`
}

type Mantenimiento struct {
	ID           string  `json:"id"`
	DispositivoID string `json:"dispositivo_id"`
	RealizadoPor string  `json:"realizado_por"`
	Descripcion  string  `json:"descripcion"`
	Costo        float64 `json:"costo"`
	FechaHora    string  `json:"fecha_hora"`
	URLPdf       *string `json:"url_pdf"`
	CreadoEn     string  `json:"creado_en"`
}

type NivelTinta struct {
	ID                 string `json:"id"`
	DispositivoID      string `json:"dispositivo_id"`
	Color              string `json:"color"`
	NivelPorcentaje    int    `json:"nivel_porcentaje"`
	CodigoTinta        *int   `json:"codigo_tinta"`
	UltimaActualizacion string `json:"ultima_actualizacion"`
}

type FotoEvidencia struct {
	ID                  string `json:"id"`
	EntidadTipo         string `json:"entidad_tipo"`
	EntidadID           string `json:"entidad_id"`
	RutaArchivoCifrado  string `json:"ruta_archivo_cifrado"`
	NombreOriginal      string `json:"nombre_original"`
	MimeType            string `json:"mime_type"`
	TamanoBytes         int    `json:"tamano_bytes"`
	CargadoEn           string `json:"cargado_en"`
}

type BajaRequest struct {
	MotivoBaja string `json:"motivo_baja"`
}

type TintasUpdateRequest struct {
	Tintas []struct {
		Color           string `json:"color"`
		NivelPorcentaje int    `json:"nivel_porcentaje"`
		CodigoTinta     *int   `json:"codigo_tinta"`
	} `json:"tintas"`
}
