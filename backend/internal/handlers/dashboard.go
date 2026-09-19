package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type DashboardHandler struct {
	DB *sql.DB
}

func (h *DashboardHandler) Resumen(w http.ResponseWriter, r *http.Request) {
	umbralStr := r.URL.Query().Get("tinta_umbral")
	if umbralStr == "" {
		umbralStr = r.URL.Query().Get("umbral")
	}
	umbral := 20
	if umbralStr != "" {
		if v, err := strconv.Atoi(umbralStr); err == nil {
			umbral = v
		}
	}
	hoy := time.Now().Format("2006-01-02")
	limite30 := time.Now().AddDate(0, 0, 30).Format("2006-01-02")

	// conteo estado
	conteoEstado := map[string]int{}
	rows, _ := h.DB.Query(`SELECT estado, COUNT(*) FROM dispositivos GROUP BY estado`)
	if rows != nil {
		for rows.Next() {
			var estado string
			var cnt int
			rows.Scan(&estado, &cnt)
			conteoEstado[estado] = cnt
		}
		rows.Close()
	}
	// conteo sede
	conteoSede := map[string]int{}
	rows, _ = h.DB.Query(`SELECT COALESCE(sede,'Sin sede'), COUNT(*) FROM dispositivos GROUP BY COALESCE(sede,'Sin sede')`)
	if rows != nil {
		for rows.Next() {
			var sede string
			var cnt int
			rows.Scan(&sede, &cnt)
			conteoSede[sede] = cnt
		}
		rows.Close()
	}
	// conteo tipo
	conteoTipo := map[string]int{}
	rows, _ = h.DB.Query(`SELECT tipo, COUNT(*) FROM dispositivos GROUP BY tipo`)
	if rows != nil {
		for rows.Next() {
			var tipo string
			var cnt int
			rows.Scan(&tipo, &cnt)
			conteoTipo[tipo] = cnt
		}
		rows.Close()
	}
	// valor total y por sede
	var valorTotal sql.NullFloat64
	h.DB.QueryRow(`SELECT SUM(valor_compra) FROM dispositivos`).Scan(&valorTotal)
	valorPorSede := map[string]float64{}
	rows, _ = h.DB.Query(`SELECT COALESCE(sede,'Sin sede'), SUM(valor_compra) FROM dispositivos GROUP BY COALESCE(sede,'Sin sede')`)
	if rows != nil {
		for rows.Next() {
			var sede string
			var sum sql.NullFloat64
			rows.Scan(&sede, &sum)
			if sum.Valid {
				valorPorSede[sede] = sum.Float64
			}
		}
		rows.Close()
	}
	// costo por mes
	type mesCosto struct {
		Mes   string  `json:"mes"`
		Costo float64 `json:"costo"`
	}
	var costoPorMes []mesCosto
	rows, _ = h.DB.Query(`SELECT substr(fecha_hora,1,7) as mes, SUM(costo) FROM mantenimientos GROUP BY mes ORDER BY mes`)
	if rows != nil {
		for rows.Next() {
			var mes string
			var costo sql.NullFloat64
			rows.Scan(&mes, &costo)
			if costo.Valid {
				costoPorMes = append(costoPorMes, mesCosto{mes, costo.Float64})
			}
		}
		rows.Close()
	}
	if costoPorMes == nil {
		costoPorMes = []mesCosto{}
	}
	// costo por tipo
	costoPorTipo := map[string]float64{}
	rows, _ = h.DB.Query(`SELECT d.tipo, SUM(m.costo) FROM mantenimientos m JOIN dispositivos d ON m.dispositivo_id = d.id GROUP BY d.tipo`)
	if rows != nil {
		for rows.Next() {
			var tipo string
			var sum sql.NullFloat64
			rows.Scan(&tipo, &sum)
			if sum.Valid {
				costoPorTipo[tipo] = sum.Float64
			}
		}
		rows.Close()
	}
	// costo por dispositivo
	type dispCosto struct {
		DispositivoID string  `json:"dispositivo_id"`
		CodigoActivo  string  `json:"codigo_activo"`
		Costo         float64 `json:"costo"`
	}
	var costoPorDisp []dispCosto
	rows, _ = h.DB.Query(`SELECT m.dispositivo_id, d.codigo_activo, SUM(m.costo) as total FROM mantenimientos m JOIN dispositivos d ON m.dispositivo_id = d.id GROUP BY m.dispositivo_id ORDER BY total DESC`)
	if rows != nil {
		for rows.Next() {
			var dc dispCosto
			rows.Scan(&dc.DispositivoID, &dc.CodigoActivo, &dc.Costo)
			costoPorDisp = append(costoPorDisp, dc)
		}
		rows.Close()
	}
	if costoPorDisp == nil {
		costoPorDisp = []dispCosto{}
	}
	topCostosos := costoPorDisp
	if len(topCostosos) > 5 {
		topCostosos = topCostosos[:5]
	}
	// antiguedad promedio dias
	var avgDias sql.NullFloat64
	h.DB.QueryRow(`SELECT AVG(julianday('now') - julianday(fecha_compra)) FROM dispositivos`).Scan(&avgDias)
	// candidatos obsolescencia
	type cand struct {
		ID           string `json:"id"`
		CodigoActivo string `json:"codigo_activo"`
		Tipo         string `json:"tipo"`
		FechaCompra  string `json:"fecha_compra"`
		Dias         int    `json:"dias"`
	}
	var candidatos []cand
	rows, _ = h.DB.Query(`SELECT id, codigo_activo, tipo, fecha_compra, CAST(julianday('now') - julianday(fecha_compra) AS INTEGER) as dias FROM dispositivos WHERE estado='obsoleto' OR julianday('now') - julianday(fecha_compra) > 1825 ORDER BY dias DESC LIMIT 20`)
	if rows != nil {
		for rows.Next() {
			var c cand
			rows.Scan(&c.ID, &c.CodigoActivo, &c.Tipo, &c.FechaCompra, &c.Dias)
			candidatos = append(candidatos, c)
		}
		rows.Close()
	}
	if candidatos == nil {
		candidatos = []cand{}
	}
	// tiempo medio en reparacion (horas) via auditoria
	var totalHoras sql.NullFloat64
	var countRepar sql.NullInt64
	// query ordered
	rows, _ = h.DB.Query(`SELECT entidad_id, valor_anterior, valor_nuevo, fecha FROM auditoria WHERE entidad='dispositivo' AND campo='estado' ORDER BY entidad_id, fecha`)
	var horasSum float64
	var horasCount int
	if rows != nil {
		// map per device start time
		starts := map[string]time.Time{}
		for rows.Next() {
			var eid string
			var ant, nuevo *string
			var fechaStr string
			rows.Scan(&eid, &ant, &nuevo, &fechaStr)
			fecha, err := time.Parse(time.RFC3339, fechaStr)
			if err != nil {
				// try YYYY-MM-DD
				fecha, err = time.Parse("2006-01-02", fechaStr)
				if err != nil {
					continue
				}
			}
			valAnt := ""
			if ant != nil {
				valAnt = *ant
			}
			valNuevo := ""
			if nuevo != nil {
				valNuevo = *nuevo
			}
			if valNuevo == "en_reparacion" {
				starts[eid] = fecha
			} else if valAnt == "en_reparacion" && valNuevo != "en_reparacion" {
				if st, ok := starts[eid]; ok {
					diff := fecha.Sub(st).Hours()
					if diff >= 0 {
						horasSum += diff
						horasCount++
					}
					delete(starts, eid)
				}
			}
		}
		rows.Close()
	}
	if horasCount > 0 {
		totalHoras = sql.NullFloat64{Float64: horasSum / float64(horasCount), Valid: true}
		countRepar = sql.NullInt64{Int64: int64(horasCount), Valid: true}
	}
	// frecuencia por marca y modelo
	type freq struct {
		Marca  string `json:"marca,omitempty"`
		Modelo string `json:"modelo,omitempty"`
		Fallas int    `json:"fallas"`
	}
	var freqMarca []freq
	rows, _ = h.DB.Query(`SELECT d.marca, COUNT(*) as fallas FROM auditoria a JOIN dispositivos d ON a.entidad_id = d.id WHERE a.entidad='dispositivo' AND a.campo='estado' AND a.valor_nuevo IN ('averiado','en_reparacion','en_mantenimiento') GROUP BY d.marca ORDER BY fallas DESC`)
	if rows != nil {
		for rows.Next() {
			var f freq
			rows.Scan(&f.Marca, &f.Fallas)
			freqMarca = append(freqMarca, f)
		}
		rows.Close()
	}
	if freqMarca == nil {
		freqMarca = []freq{}
	}
	var freqModelo []freq
	rows, _ = h.DB.Query(`SELECT d.modelo, COUNT(*) as fallas FROM auditoria a JOIN dispositivos d ON a.entidad_id = d.id WHERE a.entidad='dispositivo' AND a.campo='estado' AND a.valor_nuevo IN ('averiado','en_reparacion','en_mantenimiento') GROUP BY d.modelo ORDER BY fallas DESC`)
	if rows != nil {
		for rows.Next() {
			var m string
			var c int
			rows.Scan(&m, &c)
			freqModelo = append(freqModelo, freq{Modelo: m, Fallas: c})
		}
		rows.Close()
	}
	if freqModelo == nil {
		freqModelo = []freq{}
	}
	// tintas bajo umbral
	type tintaLow struct {
		DispositivoID string `json:"dispositivo_id"`
		Color         string `json:"color"`
		Nivel         int    `json:"nivel"`
		CodigoTinta   *int   `json:"codigo_tinta"`
	}
	var tintas []tintaLow
	rows, _ = h.DB.Query(`SELECT dispositivo_id, color, nivel_porcentaje, codigo_tinta FROM niveles_tinta WHERE nivel_porcentaje <= ? ORDER BY nivel_porcentaje ASC`, umbral)
	if rows != nil {
		for rows.Next() {
			var t tintaLow
			rows.Scan(&t.DispositivoID, &t.Color, &t.Nivel, &t.CodigoTinta)
			tintas = append(tintas, t)
		}
		rows.Close()
	}
	if tintas == nil {
		tintas = []tintaLow{}
	}
	// planes vencidos/proximos
	var vencidos, proximos30 int
	h.DB.QueryRow(`SELECT COUNT(*) FROM planes_mantenimiento WHERE activo=1 AND proxima_fecha < ?`, hoy).Scan(&vencidos)
	h.DB.QueryRow(`SELECT COUNT(*) FROM planes_mantenimiento WHERE activo=1 AND proxima_fecha >= ? AND proxima_fecha <= ?`, hoy, limite30).Scan(&proximos30)

	valTotalF := 0.0
	if valorTotal.Valid {
		valTotalF = valorTotal.Float64
	}
	avgDiasF := 0.0
	if avgDias.Valid {
		avgDiasF = avgDias.Float64
	}
	tiempoMedio := 0.0
	if totalHoras.Valid {
		tiempoMedio = totalHoras.Float64
	}
	resp := map[string]interface{}{
		"conteo_estado":                    conteoEstado,
		"conteo_sede":                      conteoSede,
		"conteo_tipo":                      conteoTipo,
		"valor_total":                      valTotalF,
		"valor_por_sede":                   valorPorSede,
		"costo_por_mes":                    costoPorMes,
		"costo_por_tipo":                   costoPorTipo,
		"costo_por_dispositivo":            costoPorDisp,
		"top_costosos":                     topCostosos,
		"antiguedad_promedio_dias":         avgDiasF,
		"candidatos_obsolescencia":         candidatos,
		"tiempo_medio_en_reparacion_horas": tiempoMedio,
		"tiempo_medio_conteo":              countRepar.Int64,
		"frecuencia_por_marca":             freqMarca,
		"frecuencia_por_modelo":            freqModelo,
		"tintas_bajo_umbral":               tintas,
		"tinta_umbral":                     umbral,
		"planes_vencidos":                  vencidos,
		"planes_proximos_30":               proximos30,
		"hoy":                              hoy,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
