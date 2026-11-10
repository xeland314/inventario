import React, { useEffect, useState } from 'react';
import {
  AlertTriangle,
  ArrowUpDown,
  Boxes,
  Building2,
  Calendar,
  Check,
  Clock,
  DollarSign,
  Edit2,
  MapPin,
  Package,
  Plus,
  ScanLine,
  Search,
  Tag,
  Trash2,
  X,
} from 'lucide-react';
import ContadorIA from './ContadorIA';
import EncryptedGallery from './EncryptedGallery';

interface Conjunto {
  id: string;
  codigo: string;
  nombre: string;
  categoria?: string | null;
  especificaciones?: string | null;
  unidad: string;
  cantidad_actual: number;
  cantidad_minima: number;
  estado: string;
  ubicacion?: string | null;
  sede?: string | null;
  proveedor?: string | null;
  costo_unitario: number;
  etiqueta_id?: string | null;
  etiqueta_codigo?: string | null;
}

interface ConjuntoMov {
  id: string;
  tipo: string;
  cantidad: number;
  cantidad_previa: number;
  cantidad_nueva: number;
  nota?: string | null;
  fecha: string;
}

interface Conteo {
  id: string;
  cantidad_detectada: number;
  cantidad_confirmada: number;
  modelo: string;
  creado_en: string;
}

const unidades = ['pieza', 'gr', 'ml', 'm', 'paquete'] as const;

export default function ConjuntosManager(): React.JSX.Element {
  const [conjuntos, setConjuntos] = useState<Conjunto[]>([]);
  const [loading, setLoading] = useState(true);
  const [q, setQ] = useState('');
  const [categoriaFilter, setCategoriaFilter] = useState('');
  const [sedeFilter, setSedeFilter] = useState('');
  const [soloBajos, setSoloBajos] = useState(false);
  const [showNuevo, setShowNuevo] = useState(false);
  const [editing, setEditing] = useState<string | null>(null);
  const [movId, setMovId] = useState<string | null>(null);
  const [movTipo, setMovTipo] = useState<'ingreso' | 'consumo' | 'ajuste'>('consumo');
  const [movCantidad, setMovCantidad] = useState(1);
  const [movNota, setMovNota] = useState('');
  const [historyId, setHistoryId] = useState<string | null>(null);
  const [history, setHistory] = useState<{ movimientos: ConjuntoMov[]; conteos: Conteo[] }>({ movimientos: [], conteos: [] });
  const [contadorId, setContadorId] = useState<string | null>(null);
  const [etiId, setEtiId] = useState<string | null>(null);
  const [etiInput, setEtiInput] = useState('');

  const fetchConjuntos = () => {
    setLoading(true);
    const p = new URLSearchParams();
    if (q) p.set('q', q);
    if (categoriaFilter) p.set('categoria', categoriaFilter);
    if (sedeFilter) p.set('sede', sedeFilter);
    if (soloBajos) p.set('bajo', '1');
    fetch(`/api/conjuntos?${p.toString()}`)
      .then((r) => r.json())
      .then((d) => setConjuntos(d.conjuntos || []))
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchConjuntos();
  }, []);

  useEffect(() => {
    const t = setTimeout(fetchConjuntos, 300);
    return () => clearTimeout(t);
  }, [q, categoriaFilter, sedeFilter, soloBajos]);

  const handleCrear = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const payload: Record<string, unknown> = {
      nombre: fd.get('nombre'),
      categoria: (fd.get('categoria') as string) || null,
      especificaciones: (fd.get('especificaciones') as string) || null,
      unidad: fd.get('unidad') || 'pieza',
      cantidad_actual: parseInt((fd.get('cantidad_actual') as string) || '0', 10),
      cantidad_minima: parseInt((fd.get('cantidad_minima') as string) || '0', 10),
      ubicacion: (fd.get('ubicacion') as string) || null,
      sede: (fd.get('sede') as string) || null,
      proveedor: (fd.get('proveedor') as string) || null,
      costo_unitario: fd.get('costo_unitario') ? parseFloat(fd.get('costo_unitario') as string) : 0,
      etiqueta_id: (fd.get('etiqueta_id') as string) || null,
    };
    const res = await fetch('/api/conjuntos', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
    if (res.ok) {
      (e.target as HTMLFormElement).reset();
      setShowNuevo(false);
      fetchConjuntos();
    } else alert(await res.text());
  };

  const handleUpdate = async (id: string, data: Record<string, unknown>) => {
    const res = await fetch(`/api/conjuntos/${id}`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) });
    if (res.ok) {
      setEditing(null);
      fetchConjuntos();
    } else alert(await res.text());
  };

  const handleDelete = async (id: string) => {
    if (!confirm('¿Eliminar conjunto? Se borrará su historial. La etiqueta de la caja queda libre.')) return;
    const res = await fetch(`/api/conjuntos/${id}`, { method: 'DELETE' });
    if (res.ok) fetchConjuntos();
    else alert(await res.text());
  };

  const handleMov = async (id: string) => {
    if (!movCantidad || movCantidad === 0) return alert('Cantidad requerida');
    const res = await fetch(`/api/conjuntos/${id}/movimiento`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tipo: movTipo, cantidad: movCantidad, nota: movNota || null }),
    });
    if (res.ok) {
      setMovId(null);
      setMovNota('');
      setMovCantidad(1);
      fetchConjuntos();
      if (historyId === id) fetchHistory(id);
    } else alert(await res.text());
  };

  const fetchHistory = async (id: string) => {
    const [rm, rc] = await Promise.all([fetch(`/api/conjuntos/${id}/movimientos`), fetch(`/api/conjuntos/${id}/conteos`)]);
    const dm = await rm.json();
    const dc = await rc.json();
    setHistory({ movimientos: dm.movimientos || [], conteos: dc.conteos || [] });
    setHistoryId(id);
  };

  const handleAsociarEtiqueta = async (conjuntoId: string) => {
    const input = etiInput.trim();
    if (!input) return alert('Pega el UUID o ET-XXXX de la etiqueta (del pool impreso)');
    const res = await fetch(`/api/etiquetas/${encodeURIComponent(input)}/asociar`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ conjunto_id: conjuntoId }),
    });
    if (res.ok) {
      setEtiId(null);
      setEtiInput('');
      fetchConjuntos();
    } else alert(await res.text());
  };

  const handleDesasociarEtiqueta = async (etiquetaId: string, conjuntoId: string) => {
    if (!confirm('¿Quitar la etiqueta de esta caja? Queda libre para reutilizar.')) return;
    const res = await fetch(`/api/etiquetas/${etiquetaId}/desasociar`, { method: 'POST' });
    if (res.ok) fetchConjuntos();
    else alert(await res.text());
    void conjuntoId;
  };

  return (
    <div className="space-y-4">
      <div className="bg-sky-50 border border-sky-200 rounded-xl px-4 py-3 text-xs text-sky-800 flex gap-2">
        <Boxes className="w-4 h-4 shrink-0 mt-0.5" />
        <div>
          <span className="font-bold">Conjunto = caja con piezas idénticas</span> (tornillos, resistores...). Se etiqueta <strong>la caja</strong> con una etiqueta del pool (QR): escanéala y verás qué hay y cuánto queda. No es un <span className="font-semibold">activo único</span> (1 QR por unidad) — es stock contable como frascos, con libro de movimientos y mínimo de alerta.
        </div>
      </div>

      <div className="bg-white p-4 sm:p-5 rounded-2xl border border-slate-200 shadow-sm">
        <div className="flex flex-col lg:flex-row gap-3 justify-between">
          <div className="flex-1 flex flex-col sm:flex-row gap-2">
            <div className="relative flex-1">
              <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
              <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Buscar nombre, código, categoría, ubicación..." className="w-full border border-slate-200 rounded-full pl-9 pr-3 py-2.5 text-sm focus:border-sky-500 focus:outline-none" />
            </div>
            <input value={categoriaFilter} onChange={(e) => setCategoriaFilter(e.target.value)} placeholder="Categoría" className="border border-slate-200 rounded-full px-3 py-2.5 text-sm w-32" />
            <input value={sedeFilter} onChange={(e) => setSedeFilter(e.target.value)} placeholder="Sede" className="border border-slate-200 rounded-full px-3 py-2.5 text-sm w-28" />
            <label className="inline-flex items-center gap-1.5 text-xs font-semibold cursor-pointer border border-amber-200 bg-amber-50 px-3 py-2 rounded-full">
              <input type="checkbox" checked={soloBajos} onChange={(e) => setSoloBajos(e.target.checked)} /> <AlertTriangle className="w-3.5 h-3.5" /> Solo bajos
            </label>
          </div>
          <div className="flex gap-2">
            <button onClick={() => setShowNuevo((v) => !v)} className="bg-slate-900 text-white px-5 py-2.5 rounded-full text-sm font-semibold inline-flex items-center gap-2 justify-center">
              {showNuevo ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />} {showNuevo ? 'Cerrar' : 'Nuevo conjunto'}
            </button>
          </div>
        </div>

        {showNuevo && (
          <form onSubmit={handleCrear} className="mt-4 border border-slate-200 rounded-xl p-4 grid grid-cols-1 sm:grid-cols-3 gap-3 bg-slate-50">
            <input name="nombre" placeholder="Nombre * (Tornillos M3x10...)" required className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="categoria" placeholder="Categoría (tornillos, resistores...)" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <select name="unidad" defaultValue="pieza" className="border border-slate-200 rounded-lg px-3 py-2 text-sm">
              {unidades.map((u) => (
                <option key={u} value={u}>{u}</option>
              ))}
            </select>
            <input name="especificaciones" placeholder="Especificaciones (acero inox, 1/4W...)" className="border border-slate-200 rounded-lg px-3 py-2 text-sm sm:col-span-2" />
            <input name="cantidad_actual" type="number" min="0" defaultValue={0} placeholder="Stock inicial" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="cantidad_minima" type="number" min="0" defaultValue={0} placeholder="Mínimo alerta" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="ubicacion" placeholder="Ubicación (Caja A, cajón 3...)" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="sede" placeholder="Sede (Matriz, Icon...)" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="proveedor" placeholder="Proveedor" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="costo_unitario" type="number" step="0.001" placeholder="Costo unitario" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="etiqueta_id" placeholder="Etiqueta QR opcional (UUID o ET-XXXX)" className="border border-slate-200 rounded-lg px-3 py-2 text-sm font-mono sm:col-span-2" />
            <div className="sm:col-span-3 flex gap-2 mt-1">
              <button type="submit" className="flex-1 bg-sky-600 text-white py-2.5 rounded-full font-semibold inline-flex items-center justify-center gap-2">
                <Package className="w-4 h-4" /> Crear conjunto
              </button>
              <button type="button" onClick={() => setShowNuevo(false)} className="px-5 py-2.5 border border-slate-200 rounded-full">Cancelar</button>
            </div>
          </form>
        )}

        <div className="mt-4 flex items-center justify-between text-xs text-slate-500">
          <span>{conjuntos.length} conjuntos · {conjuntos.filter((c) => c.estado !== 'disponible').length} en alerta</span>
          <button onClick={fetchConjuntos} className="border border-slate-200 px-3 py-1.5 rounded-full inline-flex items-center gap-1">
            <ArrowUpDown className="w-3 h-3" /> Actualizar
          </button>
        </div>
      </div>

      {loading ? (
        <p className="text-center text-slate-400 py-8">Cargando conjuntos...</p>
      ) : conjuntos.length === 0 ? (
        <div className="bg-white border-2 border-dashed border-slate-200 rounded-2xl p-8 text-center">
          <Boxes className="w-8 h-8 mx-auto text-slate-300" />
          <p className="text-sm text-slate-500 mt-2">Sin conjuntos. Crea la primera caja (ej: tornillos M3x10) con su stock.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 min-w-0">
          {conjuntos.map((c) => (
            <div key={c.id} className={`bg-white border rounded-2xl p-4 shadow-sm min-w-0 max-w-full overflow-hidden ${c.estado === 'agotado' ? 'border-rose-200 bg-rose-50/30' : c.estado === 'bajo' ? 'border-amber-200 bg-amber-50/30' : 'border-slate-200'}`}>
              <div className="flex justify-between items-start gap-2">
                <div className="min-w-0 flex-1">
                  <p className="font-mono text-xs text-slate-500 truncate">{c.codigo}{c.categoria ? ` · ${c.categoria}` : ''}</p>
                  <h3 className="font-bold text-slate-900 truncate inline-flex items-center gap-1.5">
                    <Boxes className="w-4 h-4 text-slate-600" /> {c.nombre}
                  </h3>
                  <div className="flex items-center gap-2 mt-1 flex-wrap">
                    <span className={`text-xs font-bold px-2 py-1 rounded-full border ${c.estado === 'agotado' ? 'bg-rose-600 text-white border-rose-600' : c.estado === 'bajo' ? 'bg-amber-500 text-white border-amber-500' : 'bg-emerald-100 text-emerald-700 border-emerald-200'}`}>
                      stock {c.cantidad_actual} {c.unidad} / mín {c.cantidad_minima} · {c.estado}
                    </span>
                  </div>
                </div>
                <div className="flex flex-col gap-1 shrink-0">
                  <button onClick={() => setEditing(editing === c.id ? null : c.id)} className="w-8 h-8 grid place-items-center border border-slate-200 rounded-full hover:bg-slate-50">
                    <Edit2 className="w-4 h-4" />
                  </button>
                  <button onClick={() => handleDelete(c.id)} className="w-8 h-8 grid place-items-center border border-rose-200 text-rose-600 rounded-full hover:bg-rose-50">
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>

              {c.especificaciones && <p className="text-xs text-slate-600 mt-2 truncate">{c.especificaciones}</p>}

              <div className="mt-3 grid grid-cols-2 gap-2 text-xs text-slate-600 border-t border-slate-100 pt-3">
                <span className="inline-flex items-center gap-1"><MapPin className="w-3 h-3" /> {c.ubicacion || '—'}</span>
                <span className="inline-flex items-center gap-1"><Tag className="w-3 h-3" /> Sede: {c.sede || '—'}</span>
                <span className="inline-flex items-center gap-1"><Building2 className="w-3 h-3" /> {c.proveedor || '—'}</span>
                <span className="inline-flex items-center gap-1"><DollarSign className="w-3 h-3" /> ${c.costo_unitario}</span>
              </div>

              {/* Etiqueta QR de la caja */}
              <div className="mt-3 border-t border-slate-100 pt-3 text-xs">
                {c.etiqueta_id ? (
                  <div className="flex items-center justify-between gap-2">
                    <span className="inline-flex items-center gap-1.5 text-emerald-700 font-bold">
                      <ScanLine className="w-3.5 h-3.5" /> QR en caja: <span className="font-mono">{c.etiqueta_codigo || c.etiqueta_id.slice(0, 13)}</span>
                    </span>
                    <button onClick={() => handleDesasociarEtiqueta(c.etiqueta_id!, c.id)} className="text-[11px] text-rose-600 underline shrink-0">
                      Quitar
                    </button>
                  </div>
                ) : etiId === c.id ? (
                  <div className="space-y-1.5">
                    <input value={etiInput} onChange={(e) => setEtiInput(e.target.value)} placeholder="Pega UUID o ET-XXXX de la etiqueta" className="w-full border border-slate-200 rounded-lg px-2 py-2 text-xs font-mono" />
                    <div className="flex gap-1.5">
                      <button onClick={() => handleAsociarEtiqueta(c.id)} className="flex-1 bg-emerald-600 text-white px-2 py-2 rounded-full text-xs font-bold">Pegar en caja</button>
                      <button onClick={() => { setEtiId(null); setEtiInput(''); }} className="px-3 py-2 border border-slate-200 rounded-full text-xs">Cancelar</button>
                    </div>
                  </div>
                ) : (
                  <button onClick={() => { setEtiId(etiId === c.id ? null : c.id); setEtiInput(''); }} className="w-full border border-dashed border-slate-300 text-slate-500 px-2 py-2 rounded-full text-xs inline-flex items-center justify-center gap-1.5 hover:border-sky-400 hover:text-sky-700">
                    <ScanLine className="w-3.5 h-3.5" /> Asociar etiqueta QR a esta caja
                  </button>
                )}
              </div>

              {editing === c.id && (
                <form
                  onSubmit={(e) => {
                    e.preventDefault();
                    const fd = new FormData(e.currentTarget as HTMLFormElement);
                    handleUpdate(c.id, {
                      nombre: fd.get('nombre'),
                      categoria: (fd.get('categoria') as string) || null,
                      especificaciones: (fd.get('especificaciones') as string) || null,
                      unidad: fd.get('unidad'),
                      cantidad_minima: parseInt((fd.get('cantidad_minima') as string) || '0', 10),
                      ubicacion: (fd.get('ubicacion') as string) || null,
                      sede: (fd.get('sede') as string) || null,
                      proveedor: (fd.get('proveedor') as string) || null,
                      costo_unitario: parseFloat((fd.get('costo_unitario') as string) || '0'),
                    });
                  }}
                  className="mt-3 border border-sky-200 rounded-xl p-3 bg-sky-50/50 grid grid-cols-2 gap-2"
                >
                  <input name="nombre" defaultValue={c.nombre} className="border border-slate-200 rounded-lg px-2 py-2 text-xs col-span-2" placeholder="Nombre" />
                  <input name="categoria" defaultValue={c.categoria || ''} placeholder="Categoría" className="border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <select name="unidad" defaultValue={c.unidad} className="border border-slate-200 rounded-lg px-2 py-2 text-xs">
                    {unidades.map((u) => (
                      <option key={u} value={u}>{u}</option>
                    ))}
                  </select>
                  <input name="especificaciones" defaultValue={c.especificaciones || ''} placeholder="Especificaciones" className="border border-slate-200 rounded-lg px-2 py-2 text-xs col-span-2" />
                  <input name="cantidad_minima" type="number" min="0" defaultValue={c.cantidad_minima} placeholder="Mín stock" className="border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <input name="costo_unitario" type="number" step="0.001" defaultValue={c.costo_unitario} placeholder="Costo" className="border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <input name="ubicacion" defaultValue={c.ubicacion || ''} placeholder="Ubicación" className="border border-slate-200 rounded-lg px-2 py-2 text-xs col-span-2" />
                  <input name="sede" defaultValue={c.sede || ''} placeholder="Sede" className="border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <input name="proveedor" defaultValue={c.proveedor || ''} placeholder="Proveedor" className="border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <button type="submit" className="col-span-2 bg-sky-600 text-white py-2 rounded-full text-xs font-semibold inline-flex items-center justify-center gap-1">
                    <Check className="w-3 h-3" /> Guardar
                  </button>
                </form>
              )}

              <div className="mt-3 flex flex-wrap gap-1.5">
                <button onClick={() => { setMovId(movId === c.id ? null : c.id); setMovTipo('ingreso'); }} className="flex-1 bg-emerald-600 text-white px-3 py-2 rounded-full text-xs font-semibold inline-flex items-center justify-center gap-1">
                  <Plus className="w-3 h-3" /> Ingreso
                </button>
                <button onClick={() => { setMovId(movId === c.id ? null : c.id); setMovTipo('consumo'); }} className="flex-1 bg-slate-900 text-white px-3 py-2 rounded-full text-xs font-semibold inline-flex items-center justify-center gap-1">
                  <Trash2 className="w-3 h-3" /> Consumir
                </button>
                <button onClick={() => (historyId === c.id ? setHistoryId(null) : fetchHistory(c.id))} className="px-3 py-2 border border-slate-200 rounded-full text-xs inline-flex items-center gap-1">
                  <Clock className="w-3 h-3" /> Historial
                </button>
              </div>

              <div className="mt-2 flex flex-wrap gap-1.5">
                <button
                  onClick={() => setContadorId(contadorId === c.id ? null : c.id)}
                  className={`flex-1 px-3 py-2 rounded-full text-xs font-semibold inline-flex items-center justify-center gap-1 ${contadorId === c.id ? 'bg-sky-700 text-white' : 'bg-sky-50 border border-sky-200 text-sky-800 hover:bg-sky-100'}`}
                >
                  <Calendar className="w-3 h-3" /> {contadorId === c.id ? 'Cerrar contador' : 'Contar con IA'}
                </button>
              </div>

              {contadorId === c.id && (
                <ContadorIA
                  conjuntoId={c.id}
                  nombre={c.nombre}
                  categoria={c.categoria}
                  cantidadActual={c.cantidad_actual}
                  unidad={c.unidad}
                  onGuardado={fetchConjuntos}
                />
              )}

              {movId === c.id && (
                <div className="mt-3 border border-slate-200 rounded-xl p-3 bg-white space-y-2">
                  <div className="flex gap-2">
                    <select value={movTipo} onChange={(e) => setMovTipo(e.target.value as 'ingreso' | 'consumo' | 'ajuste')} className="flex-1 border border-slate-200 rounded-lg px-2 py-2 text-xs">
                      <option value="ingreso">Ingreso (+)</option>
                      <option value="consumo">Consumo (−)</option>
                      <option value="ajuste">Ajuste (±)</option>
                    </select>
                    <input type="number" value={movCantidad} onChange={(e) => setMovCantidad(parseInt(e.target.value, 10) || 0)} className="w-24 border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  </div>
                  <input value={movNota} onChange={(e) => setMovNota(e.target.value)} placeholder="Nota (ej: armado PC GGM-001)" className="w-full border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <button onClick={() => handleMov(c.id)} className="w-full bg-sky-600 text-white py-2 rounded-full text-xs font-semibold">
                    Confirmar {movTipo}
                  </button>
                </div>
              )}

              {historyId === c.id && (
                <div className="mt-3 border border-slate-200 rounded-xl p-3 bg-slate-50 max-h-56 overflow-auto space-y-3">
                  <div>
                    <p className="text-xs font-bold mb-2">Movimientos</p>
                    {history.movimientos.length === 0 ? (
                      <p className="text-xs text-slate-400">Sin movimientos</p>
                    ) : (
                      <div className="space-y-1">
                        {history.movimientos.map((m) => (
                          <div key={m.id} className="text-xs border border-slate-200 bg-white rounded-lg px-2 py-1.5 flex justify-between gap-2">
                            <span className={`font-bold ${m.cantidad > 0 ? 'text-emerald-600' : 'text-rose-600'}`}>
                              {m.tipo} {m.cantidad > 0 ? `+${m.cantidad}` : m.cantidad}
                            </span>
                            <span className="text-slate-500 truncate">{m.nota || ''}</span>
                            <span className="text-slate-400 shrink-0">{new Date(m.fecha).toLocaleString('es-EC')}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                  <div>
                    <p className="text-xs font-bold mb-2">Conteos IA</p>
                    {history.conteos.length === 0 ? (
                      <p className="text-xs text-slate-400">Sin conteos</p>
                    ) : (
                      <div className="space-y-1">
                        {history.conteos.map((ct) => (
                          <div key={ct.id} className="text-xs border border-slate-200 bg-white rounded-lg px-2 py-1.5 flex justify-between gap-2">
                            <span className="font-bold text-sky-700">{ct.cantidad_confirmada}</span>
                            <span className="text-slate-500 truncate">IA: {ct.cantidad_detectada} · {ct.modelo}</span>
                            <span className="text-slate-400 shrink-0">{new Date(ct.creado_en).toLocaleString('es-EC')}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              )}

              <div className="mt-4 min-w-0 max-w-full overflow-hidden">
                <EncryptedGallery entidadTipo="conjunto" entidadId={c.id} compact />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
