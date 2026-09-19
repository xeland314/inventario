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
  const [filtro, setFiltro] = useState<string>(() => {
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      return params.get('estado') || 'todos';
    }
    return 'todos';
  });

  const fetchData = (estado: string): void => {
    setLoading(true);
    const url = estado && estado !== 'todos' ? `/api/dispositivos?estado=${estado}` : '/api/dispositivos';
    fetch(url)
      .then((r) => {
        if (!r.ok) throw new Error(`Backend ${r.status}`);
        return r.json() as Promise<{ dispositivos: Dispositivo[] }>;
      })
      .then((data) => {
        setDispositivos(data.dispositivos || []);
        setError(null);
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchData(filtro);
    const url = filtro === 'todos' ? '/' : `/?estado=${filtro}`;
    window.history.replaceState({}, '', url);
  }, [filtro]);

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
