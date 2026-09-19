import React, { useState, useEffect } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import Escaner from './Escaner.jsx';

export default function EtiquetasManager() {
  const [etiquetas, setEtiquetas] = useState([]);
  const [filtro, setFiltro] = useState('disponible');
  const [cantidad, setCantidad] = useState(10);
  const [version, setVersion] = useState(7);
  const [loading, setLoading] = useState(false);
  const [scanned, setScanned] = useState(null);
  const [asociarId, setAsociarId] = useState('');
  const [dispositivoId, setDispositivoId] = useState('');

  const fetchEtiquetas = async (estado = filtro) => {
    setLoading(true);
    const res = await fetch(`/api/etiquetas?estado=${estado}`);
    const data = await res.json().catch(() => ({ etiquetas: [] }));
    setEtiquetas(data.etiquetas || []);
    setLoading(false);
  };

  useEffect(() => {
    fetchEtiquetas(filtro);
  }, [filtro]);

  const handleGenerar = async (e) => {
    e.preventDefault();
    setLoading(true);
    const res = await fetch('/api/etiquetas/generar', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cantidad: parseInt(cantidad), version: parseInt(version) }),
    });
    if (res.ok) {
      await fetchEtiquetas('disponible');
      setFiltro('disponible');
    } else {
      alert('Error al generar: ' + (await res.text()));
    }
    setLoading(false);
  };

  const handleAsociar = async (e) => {
    e.preventDefault();
    if (!asociarId || !dispositivoId) return alert('Faltan IDs');
    const res = await fetch(`/api/etiquetas/${asociarId}/asociar`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ dispositivo_id: dispositivoId }),
    });
    if (res.ok) {
      alert('✓ Etiqueta asociada');
      fetchEtiquetas(filtro);
    } else alert('Error: ' + (await res.text()));
  };

  const handleImpreso = async (id) => {
    const res = await fetch(`/api/etiquetas/${id}/impreso`, { method: 'POST' });
    if (res.ok) fetchEtiquetas(filtro);
  };

  const handleAnular = async (id) => {
    if (!confirm('¿Anular etiqueta?')) return;
    const res = await fetch(`/api/etiquetas/${id}/anular`, { method: 'POST' });
    if (res.ok) fetchEtiquetas(filtro);
  };

  const handleDesasociar = async (id) => {
    if (!confirm('¿Desasociar etiqueta del dispositivo?')) return;
    const res = await fetch(`/api/etiquetas/${id}/desasociar`, { method: 'POST' });
    if (res.ok) fetchEtiquetas(filtro);
  };

  const imprimir = () => {
    window.print();
  };

  return (
    <div className="space-y-6">
      {/* Generar lote */}
      <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm">
        <h2 className="font-bold text-lg mb-4">🏷️ Generar Etiquetas UUID (pool imprimible)</h2>
        <p className="text-xs text-slate-500 mb-4">Genera UUIDs sin asignar, imprímelos y pégalos. Luego escanea para asociar al item. Soporta v4 y v7 (v7 ordenable por tiempo, ideal para inventario grande).</p>
        <form onSubmit={handleGenerar} className="flex flex-wrap gap-3 items-end">
          <div>
            <label className="text-xs font-semibold">Cantidad (1-500)</label>
            <input type="number" min="1" max="500" value={cantidad} onChange={(e) => setCantidad(e.target.value)} className="block border border-slate-200 rounded-lg px-3 py-2 text-sm w-32" />
          </div>
          <div>
            <label className="text-xs font-semibold">Versión</label>
            <select value={version} onChange={(e) => setVersion(e.target.value)} className="block border border-slate-200 rounded-lg px-3 py-2 text-sm">
              <option value={7}>UUIDv7 (recomendado, time-ordered)</option>
              <option value={4}>UUIDv4 (random)</option>
            </select>
          </div>
          <button type="submit" disabled={loading} className="bg-slate-900 text-white px-6 py-2 rounded-full text-sm hover:bg-slate-800 disabled:opacity-50">
            {loading ? 'Generando...' : '+ Generar lote'}
          </button>
          <button type="button" onClick={imprimir} className="border border-slate-200 px-6 py-2 rounded-full text-sm hover:bg-slate-50">
            🖨️ Imprimir visibles
          </button>
        </form>
      </div>

      {/* Filtros */}
      <div className="flex gap-2 flex-wrap">
        {['disponible', 'impreso', 'asignado', 'anulado', ''].map((e) => (
          <button key={e || 'todos'} onClick={() => setFiltro(e)} className={`px-4 py-2 rounded-full text-xs font-medium border ${filtro === e ? 'bg-slate-900 text-white border-slate-900' : 'bg-white border-slate-200 hover:bg-slate-50'}`}>
            {e || 'todos'}
          </button>
        ))}
        <span className="text-xs text-slate-400 self-center ml-2">{etiquetas.length} etiquetas</span>
      </div>

      {/* Escáner */}
      <Escaner
        onScan={(uuid) => setAsociarId(uuid)}
        onDeviceFound={(data) => setScanned(data)}
      />
      {scanned && (
        <div className="bg-white p-4 rounded-xl border border-slate-200">
          <h4 className="font-bold text-sm mb-2">Resultado escaneo</h4>
          <pre className="text-xs bg-slate-50 p-3 rounded-lg overflow-auto">{JSON.stringify(scanned, null, 2)}</pre>
          {scanned.dispositivo && (
            <a href={`/dispositivos/${scanned.dispositivo.id}`} className="text-sky-600 underline text-sm mt-2 inline-block">
              Ver dispositivo {scanned.dispositivo.codigo_activo} →
            </a>
          )}
        </div>
      )}

      {/* Asociar manual */}
      <div className="bg-white p-5 rounded-2xl border border-slate-200">
        <h3 className="font-bold mb-3">🔗 Asociar etiqueta a dispositivo</h3>
        <form onSubmit={handleAsociar} className="grid sm:grid-cols-3 gap-3">
          <input value={asociarId} onChange={(e) => setAsociarId(e.target.value)} placeholder="UUID etiqueta (escaneado)" className="border border-slate-200 rounded-lg px-3 py-2 text-sm font-mono" />
          <input value={dispositivoId} onChange={(e) => setDispositivoId(e.target.value)} placeholder="ID dispositivo (o código)" className="border border-slate-200 rounded-lg px-3 py-2 text-sm font-mono" />
          <button type="submit" className="bg-sky-600 text-white rounded-full px-5 py-2 text-sm hover:bg-sky-500">
            Asociar
          </button>
        </form>
        <p className="text-xs text-slate-400 mt-2">Pista: escanea la etiqueta arriba — se auto-rellena. Luego pega el ID del dispositivo o escanea su etiqueta actual.</p>
      </div>

      {/* Lista imprimible */}
      <div className="bg-white p-6 rounded-2xl border border-slate-200">
        <h3 className="font-bold mb-4">Etiquetas {filtro || 'todas'} — lista para imprimir y pegar</h3>
        {loading ? (
          <p className="text-sm text-slate-400">Cargando...</p>
        ) : etiquetas.length === 0 ? (
          <p className="text-sm text-slate-400">Sin etiquetas en este estado. Genera un lote.</p>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4 print:grid-cols-3 print:gap-6">
            {etiquetas.map((e) => (
              <div key={e.id} className="border border-slate-200 rounded-xl p-3 bg-white flex flex-col items-center text-center break-all print:border-black">
                <QRCodeSVG value={e.id} size={100} />
                <p className="font-mono text-[10px] mt-2 font-bold">{e.codigo_barras}</p>
                <p className="font-mono text-[9px] text-slate-500">{e.id}</p>
                <p className="text-[10px] mt-1">
                  <span className={`px-2 py-0.5 rounded-full text-[10px] ${e.estado === 'disponible' ? 'bg-emerald-100 text-emerald-700' : e.estado === 'asignado' ? 'bg-sky-100 text-sky-700' : 'bg-slate-100'}`}>{e.estado} v{e.version}</span>
                </p>
                {e.dispositivo_id && <p className="text-[10px] text-slate-600 mt-1">→ {e.dispositivo_id.slice(0, 8)}</p>}
                <div className="flex gap-1 mt-2 print:hidden">
                  {e.estado === 'disponible' && (
                    <button onClick={() => handleImpreso(e.id)} className="text-[11px] bg-slate-900 text-white px-2 py-1 rounded-full">
                      Marcar impreso
                    </button>
                  )}
                  {e.estado === 'asignado' && (
                    <button onClick={() => handleDesasociar(e.id)} className="text-[11px] bg-amber-50 border border-amber-200 px-2 py-1 rounded-full">
                      Desasociar
                    </button>
                  )}
                  {(e.estado === 'disponible' || e.estado === 'impreso') && (
                    <button onClick={() => handleAnular(e.id)} className="text-[11px] bg-red-50 border border-red-200 px-2 py-1 rounded-full">
                      Anular
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <style>{`@media print { nav, footer, button.print\\:hidden { display: none !important; } .print\\:grid-cols-3 { grid-template-columns: repeat(3, minmax(0,1fr)); } }`}</style>
    </div>
  );
}
