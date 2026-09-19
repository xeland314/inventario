import React, { useState, useEffect } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import Escaner from './Escaner';

interface Etiqueta {
  id: string;
  version: number;
  estado: string;
  dispositivo_id?: string | null;
  codigo_barras: string;
  generado_en: string;
}

export default function EtiquetasManager(): React.JSX.Element {
  const [etiquetas, setEtiquetas] = useState<Etiqueta[]>([]);
  const [filtro, setFiltro] = useState<string>('disponible');
  const [cantidad, setCantidad] = useState<number>(10);
  const [version, setVersion] = useState<number>(7);
  const [loading, setLoading] = useState<boolean>(false);
  const [scanned, setScanned] = useState<Record<string, unknown> | null>(null);
  const [asociarId, setAsociarId] = useState<string>('');
  const [dispositivoId, setDispositivoId] = useState<string>('');

  const fetchEtiquetas = async (estado: string = filtro): Promise<void> => {
    setLoading(true);
    const res = await fetch(`/api/etiquetas?estado=${estado}`);
    const data = (await res.json().catch(() => ({ etiquetas: [] }))) as { etiquetas: Etiqueta[] };
    setEtiquetas(data.etiquetas || []);
    setLoading(false);
  };

  useEffect(() => {
    void fetchEtiquetas(filtro);
  }, [filtro]);

  const handleGenerar = async (e: React.FormEvent): Promise<void> => {
    e.preventDefault();
    setLoading(true);
    const res = await fetch('/api/etiquetas/generar', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cantidad: cantidad, version: version }),
    });
    if (res.ok) {
      await fetchEtiquetas('disponible');
      setFiltro('disponible');
    } else {
      alert('Error al generar: ' + (await res.text()));
    }
    setLoading(false);
  };

  const handleAsociar = async (e: React.FormEvent): Promise<void> => {
    e.preventDefault();
    if (!asociarId || !dispositivoId) return alert('Faltan IDs');
    const res = await fetch(`/api/etiquetas/${asociarId}/asociar`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ dispositivo_id: dispositivoId }),
    });
    if (res.ok) {
      alert('✓ Etiqueta asociada');
      void fetchEtiquetas(filtro);
    } else alert('Error: ' + (await res.text()));
  };

  const handleImpreso = async (id: string): Promise<void> => {
    const res = await fetch(`/api/etiquetas/${id}/impreso`, { method: 'POST' });
    if (res.ok) void fetchEtiquetas(filtro);
  };

  const handleAnular = async (id: string): Promise<void> => {
    if (!confirm('¿Anular etiqueta?')) return;
    const res = await fetch(`/api/etiquetas/${id}/anular`, { method: 'POST' });
    if (res.ok) void fetchEtiquetas(filtro);
  };

  const handleDesasociar = async (id: string): Promise<void> => {
    if (!confirm('¿Desasociar etiqueta del dispositivo?')) return;
    const res = await fetch(`/api/etiquetas/${id}/desasociar`, { method: 'POST' });
    if (res.ok) void fetchEtiquetas(filtro);
  };

  const imprimir = (): void => {
    window.print();
  };

  return (
    <div className="space-y-6">
      <div className="bg-white p-4 sm:p-6 rounded-2xl border border-slate-200 shadow-sm">
        <h2 className="font-bold text-base sm:text-lg mb-2">🏷️ Generar Etiquetas UUID</h2>
        <p className="text-xs sm:text-sm text-slate-500 mb-4">Genera UUIDs sin asignar, imprímelos y pégalos. Luego escanea para asociar al item. v7 ordenable por tiempo.</p>
        <form onSubmit={handleGenerar} className="flex flex-col sm:flex-row flex-wrap gap-3 sm:items-end">
          <div className="flex gap-3">
            <div className="flex-1">
              <label className="text-xs font-semibold">Cantidad (1-500)</label>
              <input type="number" min="1" max="500" value={cantidad} onChange={(e) => setCantidad(parseInt(e.target.value, 10) || 1)} className="block w-full border border-slate-200 rounded-xl px-3 py-3 min-h-[44px] text-sm mt-1" />
            </div>
            <div className="flex-1">
              <label className="text-xs font-semibold">Versión</label>
              <select value={version} onChange={(e) => setVersion(parseInt(e.target.value, 10))} className="block w-full border border-slate-200 rounded-xl px-3 py-3 min-h-[44px] text-sm mt-1">
                <option value={7}>UUIDv7 (recomendado)</option>
                <option value={4}>UUIDv4</option>
              </select>
            </div>
          </div>
          <div className="flex gap-2">
            <button type="submit" disabled={loading} className="flex-1 sm:flex-none bg-slate-900 text-white px-6 py-3 min-h-[44px] rounded-full text-sm hover:bg-slate-800 disabled:opacity-50 touch-manipulation">
              {loading ? 'Generando...' : '+ Generar'}
            </button>
            <button type="button" onClick={imprimir} className="flex-1 sm:flex-none border border-slate-200 px-6 py-3 min-h-[44px] rounded-full text-sm hover:bg-slate-50 touch-manipulation">
              🖨️ Imprimir
            </button>
          </div>
        </form>
      </div>

      <div className="flex gap-2 flex-wrap overflow-x-auto pb-1 -mx-4 px-4 sm:mx-0 sm:px-0">
        {['disponible', 'impreso', 'asignado', 'anulado', ''].map((e) => (
          <button key={e || 'todos'} type="button" onClick={() => setFiltro(e)} className={`px-4 py-2.5 min-h-[44px] rounded-full text-xs font-medium border touch-manipulation ${filtro === e ? 'bg-slate-900 text-white border-slate-900' : 'bg-white border-slate-200 hover:bg-slate-50'}`}>
            {e || 'todos'}
          </button>
        ))}
        <span className="text-xs text-slate-400 self-center ml-2">{etiquetas.length} etiquetas</span>
      </div>

      <Escaner
        onScan={(uuid: string) => setAsociarId(uuid)}
        onDeviceFound={(data: Record<string, unknown>) => setScanned(data)}
      />
      {scanned && (
        <div className="bg-white p-4 rounded-xl border border-slate-200">
          <h4 className="font-bold text-sm mb-2">Resultado escaneo</h4>
          <pre className="text-xs bg-slate-50 p-3 rounded-lg overflow-auto max-h-64">{JSON.stringify(scanned, null, 2)}</pre>
          {(scanned as { dispositivo?: { id: string; codigo_activo: string } }).dispositivo && (
            <a href={`/dispositivos/${(scanned as { dispositivo: { id: string } }).dispositivo.id}`} className="text-sky-600 underline text-sm mt-2 inline-block">
              Ver dispositivo {(scanned as { dispositivo: { codigo_activo: string } }).dispositivo.codigo_activo} →
            </a>
          )}
        </div>
      )}

      <div className="bg-white p-4 sm:p-5 rounded-2xl border border-slate-200">
        <h3 className="font-bold mb-3 text-sm sm:text-base">🔗 Asociar etiqueta a dispositivo</h3>
        <form onSubmit={handleAsociar} className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <input value={asociarId} onChange={(e) => setAsociarId(e.target.value)} placeholder="UUID etiqueta (escaneado)" className="border border-slate-200 rounded-xl px-3 py-3 min-h-[44px] text-sm font-mono focus:border-sky-500 focus:outline-none" />
          <input value={dispositivoId} onChange={(e) => setDispositivoId(e.target.value)} placeholder="ID dispositivo" className="border border-slate-200 rounded-xl px-3 py-3 min-h-[44px] text-sm font-mono focus:border-sky-500 focus:outline-none" />
          <button type="submit" className="bg-sky-600 text-white rounded-full px-5 py-3 min-h-[44px] text-sm hover:bg-sky-500 touch-manipulation">
            Asociar
          </button>
        </form>
        <p className="text-xs text-slate-400 mt-2">Escanea arriba — se auto-rellena.</p>
      </div>

      <div className="bg-white p-4 sm:p-6 rounded-2xl border border-slate-200">
        <h3 className="font-bold mb-4 text-sm sm:text-base">Etiquetas {filtro || 'todas'} — lista para pegar</h3>
        {loading ? (
          <p className="text-sm text-slate-400">Cargando...</p>
        ) : etiquetas.length === 0 ? (
          <p className="text-sm text-slate-400">Sin etiquetas en este estado. Genera un lote.</p>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4 print:grid-cols-3 print:gap-6">
            {etiquetas.map((e) => (
              <div key={e.id} className="border border-slate-200 rounded-xl p-3 sm:p-4 bg-white flex flex-col items-center text-center break-all print:border-black">
                <QRCodeSVG value={e.id} size={112} className="w-28 h-28 sm:w-[112px] sm:h-[112px]" />
                <p className="font-mono text-[11px] mt-2 font-bold">{e.codigo_barras}</p>
                <p className="font-mono text-[9px] text-slate-500 break-all px-1">{e.id}</p>
                <p className="text-[10px] mt-2">
                  <span className={`px-2 py-1 rounded-full text-[10px] ${e.estado === 'disponible' ? 'bg-emerald-100 text-emerald-700' : e.estado === 'asignado' ? 'bg-sky-100 text-sky-700' : 'bg-slate-100'}`}>{e.estado} v{e.version}</span>
                </p>
                {e.dispositivo_id && <p className="text-[10px] text-slate-600 mt-1">→ {e.dispositivo_id.slice(0, 8)}</p>}
                <div className="flex flex-wrap justify-center gap-1.5 mt-3 print:hidden">
                  {e.estado === 'disponible' && (
                    <button type="button" onClick={() => handleImpreso(e.id)} className="text-xs bg-slate-900 text-white px-3 py-2 min-h-[36px] rounded-full touch-manipulation">
                      Impreso
                    </button>
                  )}
                  {e.estado === 'asignado' && (
                    <button type="button" onClick={() => handleDesasociar(e.id)} className="text-xs bg-amber-50 border border-amber-200 px-3 py-2 min-h-[36px] rounded-full touch-manipulation">
                      Desasociar
                    </button>
                  )}
                  {(e.estado === 'disponible' || e.estado === 'impreso') && (
                    <button type="button" onClick={() => handleAnular(e.id)} className="text-xs bg-red-50 border border-red-200 px-3 py-2 min-h-[36px] rounded-full touch-manipulation">
                      Anular
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <style>{`@media print { nav, footer, button { display: none !important; } }`}</style>
    </div>
  );
}
