import React, { useState } from 'react';

const COLOR_MAP = {
  negro: { bg: 'bg-slate-900', text: 'text-slate-200', label: 'Negro' },
  cian: { bg: 'bg-cyan-500', text: 'text-cyan-400', label: 'Cian' },
  magenta: { bg: 'bg-fuchsia-500', text: 'text-fuchsia-400', label: 'Magenta' },
  amarillo: { bg: 'bg-amber-400', text: 'text-amber-400', label: 'Amarillo' },
};

export default function InkLevelGauge({ tintas = [], dispositivoId, editable = false }) {
  const [localTintas, setLocalTintas] = useState(tintas);
  const [saving, setSaving] = useState(false);

  React.useEffect(() => setLocalTintas(tintas), [JSON.stringify(tintas)]);

  const handleChange = (color, nivel) => {
    setLocalTintas((prev) => prev.map((t) => (t.color === color ? { ...t, nivel_porcentaje: nivel } : t)));
  };
  const handleCodigoChange = (color, codigo) => {
    setLocalTintas((prev) =>
      prev.map((t) => (t.color === color ? { ...t, codigo_tinta: codigo === '' ? null : parseInt(codigo, 10) } : t))
    );
  };

  const handleSave = async () => {
    setSaving(true);
    await fetch(`/api/impresoras/${dispositivoId}/tintas`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        tintas: localTintas.map((t) => ({
          color: t.color,
          nivel_porcentaje: t.nivel_porcentaje,
          codigo_tinta: t.codigo_tinta ?? null,
        })),
      }),
    });
    setSaving(false);
  };

  const tintasToRender = editable ? localTintas : tintas;

  if (!tintasToRender.length) {
    return (
      <div className="bg-slate-900 text-white p-5 rounded-2xl shadow-xl border border-slate-800">
        <p className="text-sm text-slate-400">No hay datos de tinta para esta impresora.</p>
      </div>
    );
  }

  return (
    <div className="bg-slate-900 text-white p-5 rounded-2xl shadow-xl border border-slate-800">
      <div className="flex justify-between items-center mb-5">
        <h4 className="text-xs font-bold uppercase tracking-wider text-slate-400">Niveles de Tinta en Tiempo Real</h4>
        <span className="text-xs bg-slate-800 text-slate-300 px-2.5 py-1 rounded-full">Actualizado en tiempo real</span>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        {tintasToRender.map((item) => {
          const config = COLOR_MAP[item.color.toLowerCase()] || COLOR_MAP.negro;
          return (
            <div
              key={item.id || item.color}
              className="flex flex-col items-center bg-slate-800/50 p-3.5 rounded-xl border border-slate-700/60"
            >
              <span className={`text-xs font-bold uppercase mb-3 ${config.text}`}>{item.color}</span>
              <div className="w-10 h-28 bg-slate-950 rounded-full p-1 flex flex-col justify-end border border-slate-700 relative overflow-hidden">
                <div
                  className={`w-full rounded-full transition-all duration-700 ${config.bg}`}
                  style={{ height: `${item.nivel_porcentaje}%` }}
                />
              </div>
              <span className="text-sm font-extrabold mt-3 text-slate-100">{item.nivel_porcentaje}%</span>
              <span className="text-[11px] font-mono mt-1 px-2 py-0.5 rounded bg-slate-900 border border-slate-700 text-slate-300">
                Cód: {item.codigo_tinta ?? '—'}
              </span>
              {editable && (
                <>
                  <input
                    type="range"
                    min="0"
                    max="100"
                    value={item.nivel_porcentaje}
                    onChange={(e) => handleChange(item.color, parseInt(e.target.value))}
                    className="w-full mt-2 h-1 accent-sky-500"
                  />
                  <input
                    type="number"
                    inputMode="numeric"
                    placeholder="Código"
                    value={item.codigo_tinta ?? ''}
                    onChange={(e) => handleCodigoChange(item.color, e.target.value)}
                    className="w-full mt-2 bg-slate-900 border border-slate-600 rounded-lg px-2 py-1 text-xs text-center text-white placeholder:text-slate-500 focus:border-sky-500 focus:outline-none"
                  />
                </>
              )}
            </div>
          );
        })}
      </div>

      {editable && (
        <button
          onClick={handleSave}
          disabled={saving}
          className="mt-5 w-full bg-sky-600 hover:bg-sky-500 disabled:opacity-50 text-white font-semibold py-2.5 rounded-xl transition text-sm"
        >
          {saving ? 'Guardando...' : '💾 Guardar niveles'}
        </button>
      )}
    </div>
  );
}
