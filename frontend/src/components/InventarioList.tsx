import React, { useState, useEffect } from 'react';

interface Dispositivo {
  id: string;
  codigo_activo: string;
  numero_serie: string;
  tipo: string;
  marca: string;
  modelo: string;
  valor_compra: number;
  proveedor: string;
  estado: string;
  ubicacion: string;
  fecha_baja?: string | null;
  motivo_baja?: string | null;
}

export default function InventarioList(): React.JSX.Element {
  const [dispositivos, setDispositivos] = useState<Dispositivo[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [etiquetaDevice, setEtiquetaDevice] = useState<Record<string, unknown> | null>(null);
  const [filtro, setFiltro] = useState<string>(() => {
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      return params.get('estado') || 'todos';
    }
    return 'todos';
  });
  const [busqueda, setBusqueda] = useState<string>(() => {
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      return params.get('q') || '';
    }
    return '';
  });

  const fetchData = (estado: string, q: string): void => {
    setLoading(true);
    setEtiquetaDevice(null);
    const params = new URLSearchParams();
    if (estado && estado !== 'todos') params.set('estado', estado);
    if (q && q.trim()) params.set('q', q.trim());
    const url = params.toString() ? `/api/dispositivos?${params.toString()}` : '/api/dispositivos';
    fetch(url)
      .then((r) => {
        if (!r.ok) throw new Error(`Backend ${r.status}`);
        return r.json() as Promise<{ dispositivos: Dispositivo[] }>;
      })
      .then((data) => {
        setDispositivos(data.dispositivos || []);
        setError(null);
        // Si no hay dispositivos y q parece UUID/ET, intenta resolver via etiqueta
        const trimmed = (q || '').trim();
        const isUuid = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/.test(trimmed);
        const isEt = /^ET-[A-F0-9]{8}$/i.test(trimmed);
        if ((data.dispositivos || []).length === 0 && (isUuid || isEt)) {
          fetch(`/api/etiquetas/scan/${encodeURIComponent(trimmed)}`)
            .then((r2) => (r2.ok ? r2.json() : null))
            .then((scan: Record<string, unknown> | null) => {
              if (scan && (scan as { dispositivo?: unknown }).dispositivo) {
                setEtiquetaDevice(scan as Record<string, unknown>);
              } else if (scan && (scan as { etiqueta?: unknown }).etiqueta) {
                setEtiquetaDevice(scan as Record<string, unknown>);
              }
            })
            .catch(() => {});
        }
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchData(filtro, busqueda);
    const params = new URLSearchParams();
    if (filtro !== 'todos') params.set('estado', filtro);
    if (busqueda) params.set('q', busqueda);
    const url = params.toString() ? `/?${params.toString()}` : '/';
    window.history.replaceState({}, '', url);
  }, [filtro, busqueda]);

  const FiltroBtn: React.FC<{ value: string; label: string; activeClass: string }> = ({ value, label, activeClass }) => (
    <button
      type="button"
      onClick={() => setFiltro(value)}
      className={`px-3 sm:px-4 py-2.5 min-h-[44px] rounded-full text-xs sm:text-sm font-medium transition touch-manipulation ${
        filtro === value ? activeClass : 'bg-white border border-slate-200 text-slate-600 hover:bg-slate-50'
      }`}
    >
      {label}
    </button>
  );

  return (
    <div className="space-y-6">
      <div className="relative">
        <input
          value={busqueda}
          onChange={(e) => setBusqueda(e.target.value)}
          placeholder="Buscar compu, disco, GGM-..., marca, ID o UUID..."
          className="w-full border border-slate-200 rounded-xl px-4 py-3 min-h-[44px] text-sm focus:border-sky-500 focus:outline-none pr-10"
        />
        <span className="absolute right-3 top-3 text-slate-400 text-sm">⌕</span>
        {busqueda && (
          <button
            type="button"
            onClick={() => setBusqueda('')}
            className="absolute right-10 top-3 text-slate-400 hover:text-slate-600 text-sm"
          >
            ✕
          </button>
        )}
      </div>
      <div className="flex gap-2 border-b border-slate-200 pb-3 flex-wrap overflow-x-auto -mx-4 px-4 sm:mx-0 sm:px-0 scrollbar-hide">
        <FiltroBtn value="todos" label="Todos" activeClass="bg-slate-900 text-white" />
        <FiltroBtn value="activo" label="Activos" activeClass="bg-emerald-600 text-white" />
        <FiltroBtn value="disponible" label="Disponible" activeClass="bg-sky-600 text-white" />
        <FiltroBtn value="reserva" label="Reserva" activeClass="bg-sky-500 text-white" />
        <FiltroBtn value="en_mantenimiento" label="En Mant." activeClass="bg-amber-500 text-white" />
        <FiltroBtn value="en_reparacion" label="En Reparación" activeClass="bg-orange-600 text-white" />
        <FiltroBtn value="averiado" label="Averiado" activeClass="bg-red-600 text-white" />
        <FiltroBtn value="obsoleto" label="Obsoleto" activeClass="bg-slate-600 text-white" />
        <FiltroBtn value="dado_de_baja" label="Baja" activeClass="bg-rose-600 text-white" />
        <FiltroBtn value="extraviado" label="Extraviado" activeClass="bg-purple-600 text-white" />
      </div>
      {busqueda && !loading && (
        <p className="text-xs text-slate-500">
          {dispositivos.length} resultado{dispositivos.length !== 1 ? 's' : ''} para “{busqueda}” {filtro !== 'todos' ? `· filtro ${filtro}` : ''} — también funciona con UUID/ID completo.
        </p>
      )}

      {etiquetaDevice && (
        <div className="bg-sky-50 border border-sky-200 rounded-2xl p-4">
          {(etiquetaDevice as { dispositivo?: { id: string; codigo_activo: string; marca: string; modelo: string } }).dispositivo ? (
            <div>
              <p className="text-xs font-bold text-sky-800">🔗 Etiqueta encontrada — dispositivo asociado</p>
              <a href={`/dispositivos/${(etiquetaDevice as { dispositivo: { id: string } }).dispositivo.id}`} className="block mt-2 bg-white border border-sky-200 rounded-xl p-3 hover:bg-sky-50">
                <span className="font-bold text-slate-900">{(etiquetaDevice as { dispositivo: { marca: string; modelo: string } }).dispositivo.marca} {(etiquetaDevice as { dispositivo: { modelo: string } }).dispositivo.modelo}</span>
                <span className="text-xs font-mono text-slate-500 block">{(etiquetaDevice as { dispositivo: { codigo_activo: string; id: string } }).dispositivo.codigo_activo} · {(etiquetaDevice as { dispositivo: { id: string } }).dispositivo.id}</span>
                <span className="text-sky-600 text-xs underline mt-1 inline-block">Ver ficha →</span>
              </a>
            </div>
          ) : (
            <div>
              <p className="text-xs font-bold text-amber-800">🏷️ Etiqueta encontrada pero sin asignar</p>
              <p className="font-mono text-xs text-slate-600 break-all mt-1">{(etiquetaDevice as { etiqueta?: { id: string; codigo_barras: string } }).etiqueta?.id || busqueda} {(etiquetaDevice as { etiqueta?: { codigo_barras: string } }).etiqueta?.codigo_barras ? `· ${(etiquetaDevice as { etiqueta: { codigo_barras: string } }).etiqueta.codigo_barras}` : ''}</p>
              <a href="/etiquetas" className="text-sky-600 text-xs underline mt-2 inline-block">Asociar en Etiquetas →</a>
            </div>
          )}
        </div>
      )}

      {error && (
        <div className="bg-amber-50 border border-amber-200 text-amber-800 px-4 py-3 rounded-xl text-sm">
          ⚠️ No se pudo conectar al backend ({error}) — verifica que Go corre en :8080
        </div>
      )}

      {loading ? (
        <p className="text-center text-slate-400 py-12">Cargando inventario...</p>
      ) : dispositivos.length === 0 && !error ? (
        <div className="bg-white border-2 border-dashed border-slate-200 rounded-2xl p-8 sm:p-12 text-center">
          <p className="text-slate-400">No hay dispositivos registrados.</p>
          <a href="/dispositivos/nuevo" className="text-sky-600 text-sm font-semibold hover:underline mt-2 inline-block">
            Crear el primero →
          </a>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-5">
          {dispositivos.map((d) => (
            <a
              key={d.id}
              href={`/dispositivos/${d.id}`}
              className="bg-white border border-slate-200 rounded-2xl p-5 hover:shadow-lg hover:border-slate-300 transition group block active:scale-[0.98]"
            >
              <div className="flex justify-between items-start gap-3">
                <div className="min-w-0 flex-1">
                  <p className="text-xs font-mono text-slate-400 truncate">
                    {d.codigo_activo} · S/N {d.numero_serie}
                  </p>
                  <h3 className="font-bold text-slate-900 mt-1 group-hover:text-sky-600 transition truncate">
                    {d.marca} {d.modelo}
                  </h3>
                  <p className="text-xs text-slate-500 mt-1 truncate">
                    {d.tipo} · {d.ubicacion}
                  </p>
                </div>
                <span
                  className={`shrink-0 text-[10px] font-bold px-2.5 py-1 rounded-full uppercase tracking-wider
                    ${d.estado === 'activo' ? 'bg-emerald-100 text-emerald-700' : ''}
                    ${d.estado === 'disponible' ? 'bg-sky-100 text-sky-700' : ''}
                    ${d.estado === 'reserva' ? 'bg-sky-100 text-sky-700' : ''}
                    ${d.estado === 'en_mantenimiento' ? 'bg-amber-100 text-amber-700' : ''}
                    ${d.estado === 'en_reparacion' ? 'bg-orange-100 text-orange-700' : ''}
                    ${d.estado === 'averiado' ? 'bg-red-100 text-red-700' : ''}
                    ${d.estado === 'obsoleto' ? 'bg-slate-200 text-slate-700' : ''}
                    ${d.estado === 'dado_de_baja' ? 'bg-rose-100 text-rose-700' : ''}
                    ${d.estado === 'extraviado' ? 'bg-purple-100 text-purple-700' : ''}
                    ${!['activo','disponible','reserva','en_mantenimiento','en_reparacion','averiado','obsoleto','dado_de_baja','extraviado'].includes(d.estado) ? 'bg-slate-100 text-slate-700' : ''}`}
                >
                  {d.estado}
                </span>
              </div>
              <div className="flex gap-3 mt-4 text-xs text-slate-500 border-t border-slate-100 pt-3">
                <span>💰 ${d.valor_compra}</span>
                <span>🏢 {d.proveedor}</span>
              </div>
              {d.fecha_baja && (
                <p className="text-xs text-rose-600 mt-2">
                  📅 Baja: {d.fecha_baja} — {d.motivo_baja}
                </p>
              )}
            </a>
          ))}
        </div>
      )}
    </div>
  );
}
