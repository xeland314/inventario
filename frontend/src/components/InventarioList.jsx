import React, { useState, useEffect } from 'react';

export default function InventarioList() {
  const [dispositivos, setDispositivos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filtro, setFiltro] = useState(() => {
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      return params.get('estado') || 'todos';
    }
    return 'todos';
  });

  const fetchData = (estado) => {
    setLoading(true);
    const url = estado && estado !== 'todos' ? `/api/dispositivos?estado=${estado}` : '/api/dispositivos';
    fetch(url)
      .then((r) => {
        if (!r.ok) throw new Error(`Backend ${r.status}`);
        return r.json();
      })
      .then((data) => {
        setDispositivos(data.dispositivos || []);
        setError(null);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchData(filtro);
    // sincronizar URL sin recargar
    const url = filtro === 'todos' ? '/' : `/?estado=${filtro}`;
    window.history.replaceState({}, '', url);
  }, [filtro]);

  const FiltroBtn = ({ value, label, activeClass }) => (
    <button
      onClick={() => setFiltro(value)}
      className={`px-4 py-2 rounded-full text-sm font-medium transition ${
        filtro === value ? activeClass : 'bg-white border border-slate-200 text-slate-600 hover:bg-slate-50'
      }`}
    >
      {label}
    </button>
  );

  return (
    <div className="space-y-6">
      <div className="flex gap-2 border-b border-slate-200 pb-2 flex-wrap">
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
        <div className="bg-white border-2 border-dashed border-slate-200 rounded-2xl p-12 text-center">
          <p className="text-slate-400">No hay dispositivos registrados.</p>
          <a href="/dispositivos/nuevo" className="text-sky-600 text-sm font-semibold hover:underline mt-2 inline-block">
            Crear el primero →
          </a>
        </div>
      ) : (
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {dispositivos.map((d) => (
            <a
              key={d.id}
              href={`/dispositivos/${d.id}`}
              className="bg-white border border-slate-200 rounded-2xl p-5 hover:shadow-lg hover:border-slate-300 transition group block"
            >
              <div className="flex justify-between items-start">
                <div>
                  <p className="text-xs font-mono text-slate-400">
                    {d.codigo_activo} · S/N {d.numero_serie}
                  </p>
                  <h3 className="font-bold text-slate-900 mt-1 group-hover:text-sky-600 transition">
                    {d.marca} {d.modelo}
                  </h3>
                  <p className="text-xs text-slate-500 mt-1">
                    {d.tipo} · {d.ubicacion}
                  </p>
                </div>
                <span
                  className={`text-[10px] font-bold px-2.5 py-1 rounded-full uppercase tracking-wider
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
