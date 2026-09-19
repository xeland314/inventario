package db

import (
	"database/sql"
	"log"
)

const Schema = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS dispositivos (
    id TEXT PRIMARY KEY,
    codigo_activo TEXT UNIQUE NOT NULL,
    numero_serie TEXT NOT NULL,
    tipo TEXT NOT NULL,
    marca TEXT NOT NULL,
    modelo TEXT NOT NULL,
    especificaciones TEXT,
    valor_compra REAL NOT NULL DEFAULT 0.0,
    proveedor TEXT NOT NULL,
    estado TEXT NOT NULL DEFAULT 'activo' CHECK (estado IN ('activo','disponible','reserva','en_mantenimiento','en_reparacion','averiado','obsoleto','dado_de_baja','extraviado')),
    ubicacion TEXT NOT NULL,
    asignado_a TEXT,
    fecha_compra TEXT NOT NULL,
    fecha_baja TEXT,
    motivo_baja TEXT,
    parent_id TEXT REFERENCES dispositivos(id) ON DELETE SET NULL,
    tipo_componente TEXT,
    sede TEXT,
    creado_en TEXT NOT NULL DEFAULT (datetime('now')),
    actualizado_en TEXT
);

CREATE TABLE IF NOT EXISTS mantenimientos (
    id TEXT PRIMARY KEY,
    dispositivo_id TEXT NOT NULL,
    realizado_por TEXT NOT NULL,
    descripcion TEXT NOT NULL,
    costo REAL DEFAULT 0.0,
    fecha_hora TEXT NOT NULL,
    url_pdf TEXT,
    creado_en TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (dispositivo_id) REFERENCES dispositivos(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS niveles_tinta (
    id TEXT PRIMARY KEY,
    dispositivo_id TEXT NOT NULL,
    color TEXT NOT NULL,
    nivel_porcentaje INTEGER NOT NULL CHECK (nivel_porcentaje BETWEEN 0 AND 100),
    codigo_tinta INTEGER,
    ultima_actualizacion TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (dispositivo_id) REFERENCES dispositivos(id) ON DELETE CASCADE,
    UNIQUE(dispositivo_id, color)
);

CREATE TABLE IF NOT EXISTS fotos_evidencia (
    id TEXT PRIMARY KEY,
    entidad_tipo TEXT NOT NULL,
    entidad_id TEXT NOT NULL,
    ruta_archivo_cifrado TEXT NOT NULL,
    nombre_original TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    tamano_bytes INTEGER NOT NULL,
    cargado_en TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS movimientos (
    id TEXT PRIMARY KEY,
    dispositivo_id TEXT NOT NULL REFERENCES dispositivos(id) ON DELETE CASCADE,
    origen_parent_id TEXT REFERENCES dispositivos(id) ON DELETE SET NULL,
    destino_parent_id TEXT REFERENCES dispositivos(id) ON DELETE SET NULL,
    tipo_movimiento TEXT NOT NULL CHECK (tipo_movimiento IN ('asignacion','desasignacion','traslado','canibalizacion','instalacion','retiro')),
    descripcion TEXT,
    realizado_por TEXT,
    fecha_movimiento TEXT NOT NULL DEFAULT (datetime('now')),
    creado_en TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS etiquetas (
    id TEXT PRIMARY KEY,
    version INTEGER NOT NULL CHECK (version IN (4,7)),
    estado TEXT NOT NULL CHECK (estado IN ('disponible','impreso','asignado','anulado')) DEFAULT 'disponible',
    dispositivo_id TEXT UNIQUE REFERENCES dispositivos(id) ON DELETE SET NULL,
    codigo_barras TEXT UNIQUE,
    generado_en TEXT NOT NULL DEFAULT (datetime('now')),
    impreso_en TEXT,
    asignado_en TEXT
);

CREATE INDEX IF NOT EXISTS idx_fotos_entidad ON fotos_evidencia(entidad_tipo, entidad_id);
CREATE INDEX IF NOT EXISTS idx_mantenimientos_dispositivo ON mantenimientos(dispositivo_id);
CREATE INDEX IF NOT EXISTS idx_dispositivos_estado ON dispositivos(estado);
CREATE INDEX IF NOT EXISTS idx_dispositivos_ubicacion ON dispositivos(ubicacion);
CREATE INDEX IF NOT EXISTS idx_movimientos_dispositivo ON movimientos(dispositivo_id);
CREATE INDEX IF NOT EXISTS idx_etiquetas_estado ON etiquetas(estado);
CREATE INDEX IF NOT EXISTS idx_etiquetas_dispositivo ON etiquetas(dispositivo_id);
`

func InitDB(database *sql.DB) error {
	// Si la tabla dispositivos ya existe (BD antigua), no intentar recrearla con Schema completo
	// que incluye parent_id etc y CHECK nuevo; primero migrar.
	var tableExists int
	database.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='dispositivos'`).Scan(&tableExists)
	if tableExists == 0 {
		if _, err := database.Exec(Schema); err != nil {
			return err
		}
	} else {
		// BD existente: asegurar que las tablas nuevas existan (movimientos, etiquetas etc) sin re-crear dispositivos
		database.Exec(`
		CREATE TABLE IF NOT EXISTS movimientos (
			id TEXT PRIMARY KEY,
			dispositivo_id TEXT NOT NULL REFERENCES dispositivos(id) ON DELETE CASCADE,
			origen_parent_id TEXT REFERENCES dispositivos(id) ON DELETE SET NULL,
			destino_parent_id TEXT REFERENCES dispositivos(id) ON DELETE SET NULL,
			tipo_movimiento TEXT NOT NULL CHECK (tipo_movimiento IN ('asignacion','desasignacion','traslado','canibalizacion','instalacion','retiro')),
			descripcion TEXT,
			realizado_por TEXT,
			fecha_movimiento TEXT NOT NULL DEFAULT (datetime('now')),
			creado_en TEXT NOT NULL DEFAULT (datetime('now'))
		);
		`)
		database.Exec(`
		CREATE TABLE IF NOT EXISTS etiquetas (
			id TEXT PRIMARY KEY,
			version INTEGER NOT NULL CHECK (version IN (4,7)),
			estado TEXT NOT NULL CHECK (estado IN ('disponible','impreso','asignado','anulado')) DEFAULT 'disponible',
			dispositivo_id TEXT UNIQUE REFERENCES dispositivos(id) ON DELETE SET NULL,
			codigo_barras TEXT UNIQUE,
			generado_en TEXT NOT NULL DEFAULT (datetime('now')),
			impreso_en TEXT,
			asignado_en TEXT
		);
		`)
		// También asegurar tablas clásicas si faltan
		database.Exec(`CREATE TABLE IF NOT EXISTS fotos_evidencia (id TEXT PRIMARY KEY, entidad_tipo TEXT NOT NULL, entidad_id TEXT NOT NULL, ruta_archivo_cifrado TEXT NOT NULL, nombre_original TEXT NOT NULL, mime_type TEXT NOT NULL, tamano_bytes INTEGER NOT NULL, cargado_en TEXT NOT NULL DEFAULT (datetime('now')));`)
		database.Exec(`CREATE TABLE IF NOT EXISTS niveles_tinta (id TEXT PRIMARY KEY, dispositivo_id TEXT NOT NULL, color TEXT NOT NULL, nivel_porcentaje INTEGER NOT NULL CHECK (nivel_porcentaje BETWEEN 0 AND 100), codigo_tinta INTEGER, ultima_actualizacion TEXT NOT NULL DEFAULT (datetime('now')), FOREIGN KEY (dispositivo_id) REFERENCES dispositivos(id) ON DELETE CASCADE, UNIQUE(dispositivo_id, color));`)
		database.Exec(`CREATE TABLE IF NOT EXISTS mantenimientos (id TEXT PRIMARY KEY, dispositivo_id TEXT NOT NULL, realizado_por TEXT NOT NULL, descripcion TEXT NOT NULL, costo REAL DEFAULT 0.0, fecha_hora TEXT NOT NULL, url_pdf TEXT, creado_en TEXT NOT NULL DEFAULT (datetime('now')), FOREIGN KEY (dispositivo_id) REFERENCES dispositivos(id) ON DELETE CASCADE);`)
	}
	// Migraciones cautelosas para BD existentes (ADD COLUMN es idempotente si ya existe -> ignorar error)
	migrations := []string{
		`ALTER TABLE dispositivos ADD COLUMN parent_id TEXT REFERENCES dispositivos(id) ON DELETE SET NULL`,
		`ALTER TABLE dispositivos ADD COLUMN tipo_componente TEXT`,
		`ALTER TABLE dispositivos ADD COLUMN sede TEXT`,
		`ALTER TABLE dispositivos ADD COLUMN actualizado_en TEXT`,
		`ALTER TABLE niveles_tinta ADD COLUMN codigo_tinta INTEGER`,
	}
	for _, m := range migrations {
		database.Exec(m)
	}
	// Crear tablas nuevas si no existen (por si BD vieja sin ellas)
	database.Exec(`
		CREATE TABLE IF NOT EXISTS movimientos (
			id TEXT PRIMARY KEY,
			dispositivo_id TEXT NOT NULL REFERENCES dispositivos(id) ON DELETE CASCADE,
			origen_parent_id TEXT REFERENCES dispositivos(id) ON DELETE SET NULL,
			destino_parent_id TEXT REFERENCES dispositivos(id) ON DELETE SET NULL,
			tipo_movimiento TEXT NOT NULL CHECK (tipo_movimiento IN ('asignacion','desasignacion','traslado','canibalizacion','instalacion','retiro')),
			descripcion TEXT,
			realizado_por TEXT,
			fecha_movimiento TEXT NOT NULL DEFAULT (datetime('now')),
			creado_en TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`)
	database.Exec(`
		CREATE TABLE IF NOT EXISTS etiquetas (
			id TEXT PRIMARY KEY,
			version INTEGER NOT NULL CHECK (version IN (4,7)),
			estado TEXT NOT NULL CHECK (estado IN ('disponible','impreso','asignado','anulado')) DEFAULT 'disponible',
			dispositivo_id TEXT UNIQUE REFERENCES dispositivos(id) ON DELETE SET NULL,
			codigo_barras TEXT UNIQUE,
			generado_en TEXT NOT NULL DEFAULT (datetime('now')),
			impreso_en TEXT,
			asignado_en TEXT
		);
	`)
	database.Exec(`CREATE INDEX IF NOT EXISTS idx_dispositivos_estado ON dispositivos(estado)`)
	database.Exec(`CREATE INDEX IF NOT EXISTS idx_dispositivos_ubicacion ON dispositivos(ubicacion)`)
	database.Exec(`CREATE INDEX IF NOT EXISTS idx_dispositivos_parent ON dispositivos(parent_id)`)
	database.Exec(`CREATE INDEX IF NOT EXISTS idx_dispositivos_tipo_componente ON dispositivos(tipo_componente)`)
	database.Exec(`CREATE INDEX IF NOT EXISTS idx_fotos_entidad ON fotos_evidencia(entidad_tipo, entidad_id)`)
	database.Exec(`CREATE INDEX IF NOT EXISTS idx_mantenimientos_dispositivo ON mantenimientos(dispositivo_id)`)
	database.Exec(`CREATE INDEX IF NOT EXISTS idx_movimientos_dispositivo ON movimientos(dispositivo_id)`)
	database.Exec(`CREATE INDEX IF NOT EXISTS idx_movimientos_origen ON movimientos(origen_parent_id)`)
	database.Exec(`CREATE INDEX IF NOT EXISTS idx_movimientos_destino ON movimientos(destino_parent_id)`)
	database.Exec(`CREATE INDEX IF NOT EXISTS idx_etiquetas_estado ON etiquetas(estado)`)
	database.Exec(`CREATE INDEX IF NOT EXISTS idx_etiquetas_dispositivo ON etiquetas(dispositivo_id)`)

	// Asegurar foreign_keys activado por conexión
	if _, err := database.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		log.Printf("WARN: no se pudo activar foreign_keys: %v", err)
	}
	log.Println("✓ Esquema SQLite inicializado (6 tablas, índices y FKs)")
	return nil
}
