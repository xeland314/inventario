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

var TiposValidos = map[string]bool{
	// Equipos principales
	"Desktop": true, "Laptop": true, "Servidor": true, "Tablet": true, "Monitor": true,
	"Proyector": true, "Impresora": true, "Scanner": true, "Router": true, "Switch": true,
	"Access Point": true, "Firewall": true, "NAS": true, "UPS": true, "Regulador": true,
	"Telefono IP": true, "Camara IP": true, "Disco Externo": true, "Otro": true,
	// Componentes también válidos como item independiente (cables, discos...)
	"Ventilador": true, "SSD": true, "HDD": true, "NVMe": true, "RAM": true, "Bateria": true,
	"Fuente": true, "Placa": true, "CPU": true, "GPU": true, "Teclado": true,
	"Pantalla": true, "Cargador": true, "Disco": true,
	"Cable Poder": true, "Cable HDMI": true, "Cable VGA": true, "Cable DisplayPort": true,
	"Cable Ethernet": true, "Cable USB": true, "Cable SATA": true, "Patch Cord": true, "Adaptador": true,
	"Bateria UPS": true, "Fusible": true, "Tarjeta Red": true, "Mouse": true, "Mouses": true, "Teclados": true, "Parlantes": true, "Parlante": true, "Microfono": true, "Microfonos": true, "Camara": true, "Camaras": true, "Audifono": true, "Audifonos": true, "Auricular": true, "Auriculares": true, "Disipador": true,
}

var TiposComponenteValidos = map[string]bool{
	// Computo interno
	"Ventilador": true, "SSD": true, "HDD": true, "NVMe": true, "RAM": true, "Bateria": true,
	"Fuente": true, "Placa": true, "CPU": true, "GPU": true, "Teclado": true,
	"Pantalla": true, "Cargador": true,
	// Almacenamiento / discos
	"Disco": true, "Disco Externo": true,
	// Cables
	"Cable Poder": true, "Cable HDMI": true, "Cable VGA": true, "Cable DisplayPort": true,
	"Cable Ethernet": true, "Cable USB": true, "Cable SATA": true, "Patch Cord": true, "Adaptador": true,
	// Energia / regulacion
	"Regulador": true, "UPS": true, "Bateria UPS": true, "Fusible": true,
	// Red / perifericos
	"Tarjeta Red": true, "Mouse": true, "Mouses": true, "Parlante": true, "Parlantes": true, "Microfono": true, "Microfonos": true, "Camara": true, "Camaras": true, "Audifono": true, "Audifonos": true, "Auricular": true, "Auriculares": true, "Disipador": true,
	"Otro": true,
}

var TiposMovimientoValidos = map[string]bool{
	"asignacion":     true,
	"desasignacion":  true,
	"traslado":       true,
	"canibalizacion": true,
	"instalacion":    true,
	"retiro":         true,
}

var RolesValidos = map[string]bool{
	"tecnico": true,
	"lector":  true,
}

func EsRolValido(r string) bool { return RolesValidos[r] }

var EstadosReporteValidos = map[string]bool{
	"abierto":   true,
	"cancelado": true,
	"resuelto":  true,
}

func EsEstadoReporteValido(s string) bool { return EstadosReporteValidos[s] }

func EsEstadoValido(estado string) bool {
	return EstadosValidos[estado]
}

func EsTipoValido(t string) bool {
	if t == "" {
		return true
	}
	return TiposValidos[t]
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

var EstadosEtiquetaValidos = map[string]bool{
	"disponible": true,
	"impreso":    true,
	"asignado":   true,
	"anulado":    true,
}

func EsEstadoEtiquetaValido(s string) bool { return EstadosEtiquetaValidos[s] }

type Etiqueta struct {
	ID            string  `json:"id"`
	Version       int     `json:"version"`
	Estado        string  `json:"estado"`
	DispositivoID *string `json:"dispositivo_id"`
	CodigoBarras  *string `json:"codigo_barras"`
	GeneradoEn    string  `json:"generado_en"`
	ImpresoEn     *string `json:"impreso_en"`
	AsignadoEn    *string `json:"asignado_en"`
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
	Etiqueta         *Etiqueta     `json:"etiqueta,omitempty"`
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

// Auth token-based
type Usuario struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	Rol        string  `json:"rol"`
	CreadoEn   string  `json:"creado_en"`
	Activo     int     `json:"activo"`
}

type Sesion struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	Token    string `json:"token"`
	CreadoEn string `json:"creado_en"`
}

type Reporte struct {
	ID            string  `json:"id"`
	DispositivoID *string `json:"dispositivo_id"`
	EtiquetaID    *string `json:"etiqueta_id"`
	CreadoPor     string  `json:"creado_por"`
	Descripcion   string  `json:"descripcion"`
	Estado        string  `json:"estado"`
	CreadoEn      string  `json:"creado_en"`
	CanceladoEn   *string `json:"cancelado_en"`
	Username      *string `json:"username,omitempty"`
}

type Auditoria struct {
	ID            string  `json:"id"`
	Entidad       string  `json:"entidad"`
	EntidadID     string  `json:"entidad_id"`
	Accion        string  `json:"accion"`
	Campo         *string `json:"campo"`
	ValorAnterior *string `json:"valor_anterior"`
	ValorNuevo    *string `json:"valor_nuevo"`
	UsuarioID     *string `json:"usuario_id"`
	Username      *string `json:"username"`
	Fecha         string  `json:"fecha"`
}

type PlanMantenimiento struct {
	ID            string  `json:"id"`
	DispositivoID *string `json:"dispositivo_id"`
	Tipo          *string `json:"tipo"`
	CadaDias      int     `json:"cada_dias"`
	ProximaFecha  string  `json:"proxima_fecha"`
	Descripcion   string  `json:"descripcion"`
	Activo        int     `json:"activo"`
	CreadoEn      string  `json:"creado_en"`
	ActualizadoEn *string `json:"actualizado_en"`
	CreadoPor     *string `json:"creado_por"`
}

type TomaFisica struct {
	ID          string  `json:"id"`
	Sede        string  `json:"sede"`
	Descripcion *string `json:"descripcion"`
	Estado      string  `json:"estado"`
	CreadoPor   *string `json:"creado_por"`
	CreadoEn    string  `json:"creado_en"`
	CerradoEn   *string `json:"cerrado_en"`
	CerradoPor  *string `json:"cerrado_por"`
}

type TomaItem struct {
	ID            string  `json:"id"`
	TomaID        string  `json:"toma_id"`
	EtiquetaID    *string `json:"etiqueta_id"`
	DispositivoID *string `json:"dispositivo_id"`
	EscaneadoEn   string  `json:"escaneado_en"`
	EscaneadoPor  *string `json:"escaneado_por"`
}
