import React, { useEffect, useState } from 'react';

interface DashboardData {
  conteo_estado: Record<string, number>;
  conteo_sede: Record<string, number>;
  conteo_tipo: Record<string, number>;
  valor_total: number;
  valor_por_sede: Record<string, number>;
  costo_por_mes: { mes: string; costo: number }[];
  costo_por_tipo: Record<string, number>;
  costo_por_dispositivo: { dispositivo_id: string; codigo_activo: string; costo: number }[];
  top_costosos: { dispositivo_id: string; codigo_activo: string; costo: number }[];
  antiguedad_promedio_dias: number;
  candidatos_obsolescencia: { id: string; codigo_activo: string; tipo: string; fecha_compra: string; dias: number }[];
  tiempo_medio_en_reparacion_horas: number;
  tiempo_medio_conteo: number;
  frecuencia_por_marca: { marca: string; fallas: number }[];
  frecuencia_por_modelo: { modelo: string; fallas: number }[];
  tintas_bajo_umbral: { dispositivo_id: string; color: string; nivel: number; codigo_tinta: number | null }[];
  tinta_umbral: number;
  planes_vencidos: number;
  planes_proximos_30: number;
  hoy: string;
}

const estadoColors: Record<string, string> = {
  activo: 'bg-emerald-500',
  disponible: 'bg-sky-500',
  reserva: 'bg-sky-400',
  en_mantenimiento: 'bg-amber-500',
  en_reparacion: 'bg-orange-600',
  averiado: 'bg-red-600',
  obsoleto: 'bg-slate-600',
  dado_de_baja: 'bg-rose-600',
  extraviado: 'bg-purple-600',
};

export default function EstadisticasPanel() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [umbral, setUmbral] = useState(20);

  const fetchData = (u: number) => {
    setLoading(true);
    fetch(`/api/estadisticas?tinta_umbral=${u}`)
      .then((r) => {
        if (!r.ok) throw new Error(`Backend ${r.status}`);
        return r.json();
      })
      .then((d) => { setData(d); setError(null); })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchData(umbral); }, []);

  if (loading) return <div className="text-center py-12 text-slate-400">Cargando estadísticas...</div>;
  if (error) return <div className="bg-amber-50 border border-amber-200 text-amber-800 px-4 py-3 rounded-xl">⚠️ {error} — verifica :8080</div>;
  if (!data) return null;

  const totalDispositivos = Object.values(data.conteo_estado).reduce((a, b) => a + b, 0);
  const maxCostoMes = Math.max(...data.costo_por_mes.map((c) => c.costo), 1);
  const maxTinta = data.tintas_bajo_umbral.length;

  return (
    <div className="space-y-6">
      {/* Header KPIs */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-white border border-slate-200 rounded-2xl p-5">
          <p className="text-xs font-bold tracking-widest text-slate-400 uppercase">Total activos</p>
          <p className="text-3xl font-black text-slate-900 mt-1">{totalDispositivos}</p>
          <p className="text-xs text-slate-500 mt-1">{data.conteo_estado['activo'] || 0} activos · {data.conteo_estado['disponible'] || 0} disponibles</p>
        </div>
        <div className="bg-white border border-slate-200 rounded-2xl p-5">
          <p className="text-xs font-bold tracking-widest text-slate-400 uppercase">Valor inventario</p>
          <p className="text-3xl font-black text-slate-900 mt-1">${data.valor_total.toFixed(2)}</p>
          <p className="text-xs text-slate-500 mt-1">Prom. antigüedad {(data.antiguedad_promedio_dias / 365).toFixed(1)} años</p>
        </div>
        <div className="bg-white border border-slate-200 rounded-2xl p-5">
          <p className="text-xs font-bold tracking-widest text-slate-400 uppercase">Mantenimientos</p>
          <p className="text-3xl font-black text-slate-900 mt-1">{data.costo_por_mes.reduce((a, c) => a + c.costo, 0).toFixed(0)} $</p>
          <p className="text-xs text-slate-500 mt-1">{data.costo_por_dispositivo.length} equipos con costo · {data.top_costosos.length} top</p>
        </div>
        <div className={`border rounded-2xl p-5 ${data.planes_vencidos > 0 ? 'bg-rose-50 border-rose-200' : 'bg-white border-slate-200'}`}>
          <p className="text-xs font-bold tracking-widest uppercase" style={{ color: data.planes_vencidos > 0 ? '#be123c' : '#94a3b8' }}>Planes</p>
          <p className="text-3xl font-black mt-1" style={{ color: data.planes_vencidos > 0 ? '#be123c' : '#0f172a' }}>{data.planes_vencidos} vencidos</p>
          <p className="text-xs mt-1" style={{ color: data.planes_vencidos > 0 ? '#e11d48' : '#64748b' }}>{data.planes_proximos_30} próximos 30d · hoy {data.hoy}</p>
        </div>
      </div>

      {/* Conteo por estado - barras */}
      <div className="bg-white border border-slate-200 rounded-2xl p-6">
        <h3 className="font-bold text-slate-900">Distribución por estado</h3>
        <p className="text-xs text-slate-500 mb-4">Total {totalDispositivos} · tiempo medio reparación {data.tiempo_medio_en_reparacion_horas.toFixed(1)}h ({data.tiempo_medio_conteo} casos)</p>
        <div className="space-y-2">
          {Object.entries(data.conteo_estado).sort((a, b) => b[1] - a[1]).map(([estado, cnt]) => {
            const pct = totalDispositivos ? (cnt / totalDispositivos) * 100 : 0;
            return (
              <div key={estado} className="flex items-center gap-3">
                <span className="w-28 text-xs font-semibold text-slate-600 truncate">{estado}</span>
                <div className="flex-1 h-3 bg-slate-100 rounded-full overflow-hidden">
                  <div className={`h-full rounded-full ${estadoColors[estado] || 'bg-slate-400'}`} style={{ width: `${pct}%` }} />
                </div>
                <span className="w-12 text-right text-xs font-mono font-bold">{cnt}</span>
              </div>
            );
          })}
        </div>
      </div>

      <div className="grid lg:grid-cols-2 gap-6">
        {/* Sedes */}
        <div className="bg-white border border-slate-200 rounded-2xl p-6">
          <h3 className="font-bold text-slate-900">Por sede</h3>
          <p className="text-xs text-slate-500 mb-4">Valor por sede y conteo</p>
          <div className="space-y-3">
            {Object.entries(data.conteo_sede).sort((a, b) => b[1] - a[1]).map(([sede, cnt]) => (
              <div key={sede} className="flex justify-between items-center border border-slate-100 rounded-xl px-3 py-2.5 bg-slate-50">
                <div>
                  <p className="text-sm font-bold text-slate-800 truncate max-w-[160px]">{sede}</p>
                  <p className="text-xs text-slate-500">{cnt} equipos · ${(data.valor_por_sede[sede] || 0).toFixed(2)}</p>
                </div>
                <span className="text-xs bg-slate-900 text-white px-2.5 py-1 rounded-full font-bold">{cnt}</span>
              </div>
            ))}
            {Object.keys(data.conteo_sede).length === 0 && <p className="text-xs text-slate-400">Sin datos sede</p>}
          </div>
        </div>

        {/* Tipos */}
        <div className="bg-white border border-slate-200 rounded-2xl p-6">
          <h3 className="font-bold text-slate-900">Por tipo</h3>
          <p className="text-xs text-slate-500 mb-4">Tipos más frecuentes</p>
          <div className="flex flex-wrap gap-2">
            {Object.entries(data.conteo_tipo).sort((a, b) => b[1] - a[1]).slice(0, 20).map(([tipo, cnt]) => (
              <span key={tipo} className="inline-flex items-center gap-1.5 bg-slate-900 text-white text-xs px-3 py-1.5 rounded-full">
                {tipo} <strong className="bg-white text-slate-900 px-1.5 py-0.5 rounded-full text-[10px]">{cnt}</strong>
              </span>
            ))}
            {Object.keys(data.conteo_tipo).length === 0 && <p className="text-xs text-slate-400">Sin tipos</p>}
          </div>
          <div className="mt-4 border-t border-slate-100 pt-4">
            <p className="text-xs font-semibold text-slate-600 mb-2">Costo mant. por tipo</p>
            {Object.entries(data.costo_por_tipo).length === 0 ? <p className="text-xs text-slate-400">Sin costos</p> :
              Object.entries(data.costo_por_tipo).sort((a, b) => b[1] - a[1]).map(([tipo, costo]) => (
                <div key={tipo} className="flex justify-between text-xs py-1">
                  <span className="text-slate-600">{tipo}</span><span className="font-mono font-bold">${costo.toFixed(2)}</span>
                </div>
              ))}
          </div>
        </div>
      </div>

      {/* Costo por mes - barra simple sin librería */}
      <div className="bg-white border border-slate-200 rounded-2xl p-6">
        <h3 className="font-bold text-slate-900">Costo de mantenimientos por mes</h3>
        <p className="text-xs text-slate-500 mb-4">{data.costo_por_mes.length} meses registrados</p>
        {data.costo_por_mes.length === 0 ? <p className="text-sm text-slate-400">Sin mantenimientos con costo</p> : (
          <div className="space-y-2">
            {data.costo_por_mes.map((m) => (
              <div key={m.mes} className="flex items-center gap-3">
                <span className="w-20 text-xs font-mono text-slate-600">{m.mes}</span>
                <div className="flex-1 h-6 bg-slate-100 rounded-lg overflow-hidden flex items-center">
                  <div className="h-full bg-gradient-to-r from-sky-600 to-indigo-600 rounded-lg flex items-center justify-end pr-2" style={{ width: `${(m.costo / maxCostoMes) * 100}%`, minWidth: m.costo > 0 ? '40px' : '0' }}>
                    <span className="text-[10px] font-bold text-white">${m.costo.toFixed(0)}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
        {data.top_costosos.length > 0 && (
          <div className="mt-6">
            <p className="text-xs font-bold text-slate-700 mb-2">Top 5 más costosos</p>
            <div className="grid sm:grid-cols-2 lg:grid-cols-5 gap-2">
              {data.top_costosos.map((t) => (
                <a key={t.dispositivo_id} href={`/dispositivos/${t.dispositivo_id}`} className="border border-slate-200 rounded-xl p-3 hover:shadow bg-slate-50">
                  <p className="text-xs font-mono text-slate-500 truncate">{t.codigo_activo}</p>
                  <p className="text-sm font-bold text-rose-600">${t.costo.toFixed(2)}</p>
                </a>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Tintas y obsolescencia + frecuencia */}
      <div className="grid lg:grid-cols-2 gap-6">
        <div className="bg-white border border-slate-200 rounded-2xl p-6">
          <div className="flex justify-between items-start gap-3">
            <div>
              <h3 className="font-bold text-slate-900">Tintas bajo umbral</h3>
              <p className="text-xs text-slate-500">Umbral actual {umbral}% · {maxTinta} alertas</p>
            </div>
            <div className="flex items-center gap-2">
              <input type="range" min={5} max={50} value={umbral} onChange={(e) => setUmbral(parseInt(e.target.value))} className="w-20" />
              <button onClick={() => fetchData(umbral)} className="text-xs bg-slate-900 text-white px-3 py-1.5 rounded-full">Actualizar</button>
            </div>
          </div>
          <div className="mt-4 space-y-2 max-h-[300px] overflow-auto">
            {data.tintas_bajo_umbral.length === 0 ? <p className="text-sm text-emerald-600">✓ Ninguna tinta bajo umbral</p> :
              data.tintas_bajo_umbral.map((t, i) => (
                <div key={i} className="flex justify-between items-center border border-amber-200 bg-amber-50 rounded-xl px-3 py-2">
                  <div>
                    <p className="text-xs font-bold">{t.color} · {t.nivel}%</p>
                    <p className="text-[11px] font-mono text-slate-500">{t.dispositivo_id.slice(0, 8)} {t.codigo_tinta ? `· tinta ${t.codigo_tinta}` : ''}</p>
                  </div>
                  <span className={`text-xs font-black px-2 py-1 rounded-full ${t.nivel <= 10 ? 'bg-red-600 text-white' : 'bg-amber-500 text-white'}`}>{t.nivel}%</span>
                </div>
              ))}
          </div>
        </div>

        <div className="bg-white border border-slate-200 rounded-2xl p-6">
          <h3 className="font-bold text-slate-900">Candidatos obsolescencia</h3>
          <p className="text-xs text-slate-500 mb-3">&gt;5 años o estado obsoleto · {data.candidatos_obsolescencia.length} equipos</p>
          {data.candidatos_obsolescencia.length === 0 ? <p className="text-sm text-slate-400">Sin candidatos</p> :
            <div className="space-y-2 max-h-[300px] overflow-auto">
              {data.candidatos_obsolescencia.map((c) => (
                <a key={c.id} href={`/dispositivos/${c.id}`} className="flex justify-between items-center border border-slate-100 rounded-xl px-3 py-2 hover:bg-slate-50">
                  <div>
                    <p className="text-xs font-bold font-mono">{c.codigo_activo} · {c.tipo}</p>
                    <p className="text-[11px] text-slate-500">{c.fecha_compra} · {c.dias} días ({(c.dias / 365).toFixed(1)}a)</p>
                  </div>
                  <span className="text-xs bg-slate-200 text-slate-700 px-2 py-1 rounded-full font-bold">{c.dias}d</span>
                </a>
              ))}
            </div>}
        </div>
      </div>

      {/* Frecuencia por marca/modelo */}
      <div className="grid lg:grid-cols-2 gap-6">
        <div className="bg-white border border-slate-200 rounded-2xl p-6">
          <h3 className="font-bold text-slate-900">Fallas por marca</h3>
          <p className="text-xs text-slate-500 mb-3">Conteo auditoría en_reparacion/averiado/en_mantenimiento</p>
          {data.frecuencia_por_marca.length === 0 ? <p className="text-xs text-slate-400">Sin datos</p> :
            data.frecuencia_por_marca.slice(0, 10).map((f) => (
              <div key={f.marca} className="flex justify-between text-sm py-1.5 border-b border-slate-50">
                <span className="font-medium">{f.marca || '—'}</span><span className="font-mono bg-red-50 text-red-700 px-2 py-0.5 rounded-full text-xs font-bold">{f.fallas} fallas</span>
              </div>
            ))}
        </div>
        <div className="bg-white border border-slate-200 rounded-2xl p-6">
          <h3 className="font-bold text-slate-900">Fallas por modelo</h3>
          <p className="text-xs text-slate-500 mb-3">Top modelos con incidentes</p>
          {data.frecuencia_por_modelo.length === 0 ? <p className="text-xs text-slate-400">Sin datos</p> :
            data.frecuencia_por_modelo.slice(0, 10).map((f) => (
              <div key={f.modelo} className="flex justify-between text-sm py-1.5 border-b border-slate-50">
                <span className="font-medium truncate max-w-[200px]">{f.modelo || '—'}</span><span className="font-mono bg-orange-50 text-orange-700 px-2 py-0.5 rounded-full text-xs font-bold">{f.fallas}</span>
              </div>
            ))}
        </div>
      </div>

      <div className="flex gap-2 text-xs">
        <a href="/calendario" className="bg-slate-900 text-white px-4 py-2 rounded-full font-semibold">📅 Ver calendario mantenimientos →</a>
        <a href="/" className="bg-white border border-slate-200 px-4 py-2 rounded-full">← Inventario</a>
      </div>
    </div>
  );
}
