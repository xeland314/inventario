import React, { useEffect, useState } from 'react';
import {
  AlertTriangle,
  Building2,
  Calendar,
  Check,
  Droplet,
  Edit2,
  MapPin,
  Package,
  Plus,
  Search,
  Trash2,
  Minus,
  ArrowUpDown,
  Clock,
  DollarSign,
  Tag,
  X,
  Printer,
  Image as ImageIcon,
} from 'lucide-react';
import DeviceSearchCombobox from './DeviceSearchCombobox';
import EncryptedGallery from './EncryptedGallery';

interface Frasco {
  id: string;
  codigo: string;
  marca: string;
  modelo_compat: string | null;
  color: string;
  nivel_porcentaje: number;
  cantidad_actual: number;
  cantidad_minima: number;
  estado: string;
  sede: string | null;
  ubicacion: string | null;
  proveedor: string | null;
  lote: string | null;
  vencimiento: string | null;
  costo_unitario: number;
  creado_en: string;
  actualizado_en: string;
}

const colores = ['negro', 'cian', 'magenta', 'amarillo', 'negro_foto', 'gris', 'otro'] as const;
const colorClasses: Record<string, string> = {
  negro: 'bg-slate-900 text-white',
  cian: 'bg-cyan-500 text-white',
  magenta: 'bg-fuchsia-500 text-white',
  amarillo: 'bg-amber-400 text-slate-900',
  negro_foto: 'bg-zinc-800 text-white',
  gris: 'bg-zinc-400 text-white',
  otro: 'bg-slate-500 text-white',
};

export default function FrascosManager(): React.JSX.Element {
  const [frascos, setFrascos] = useState<Frasco[]>([]);
  const [loading, setLoading] = useState(true);
  const [q, setQ] = useState('');
  const [sedeFilter, setSedeFilter] = useState('');
  const [colorFilter, setColorFilter] = useState('');
  const [soloBajos, setSoloBajos] = useState(false);
  const [showNuevo, setShowNuevo] = useState(false);
  const [editing, setEditing] = useState<string | null>(null);
  const [movId, setMovId] = useState<string | null>(null);
  const [movTipo, setMovTipo] = useState<'ingreso' | 'consumo'>('consumo');
  const [movCantidad, setMovCantidad] = useState(1);
  const [movDisp, setMovDisp] = useState('');
  const [movNota, setMovNota] = useState('');
  const [historyId, setHistoryId] = useState<string | null>(null);
  const [history, setHistory] = useState<any[]>([]);

  const fetchFrascos = () => {
    setLoading(true);
    const p = new URLSearchParams();
    if (q) p.set('q', q);
    if (sedeFilter) p.set('sede', sedeFilter);
    if (colorFilter) p.set('color', colorFilter);
    if (soloBajos) p.set('bajo', '1');
    fetch(`/api/frascos?${p.toString()}`)
      .then((r) => r.json())
      .then((d) => setFrascos(d.frascos || []))
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchFrascos();
  }, []);

  useEffect(() => {
    const t = setTimeout(fetchFrascos, 300);
    return () => clearTimeout(t);
  }, [q, sedeFilter, colorFilter, soloBajos]);

  const handleCrear = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const payload: any = {
      codigo: (fd.get('codigo') as string) || undefined,
      marca: fd.get('marca'),
      modelo_compat: (fd.get('modelo_compat') as string) || null,
      color: fd.get('color'),
      nivel_porcentaje: fd.get('nivel_porcentaje') ? parseInt(fd.get('nivel_porcentaje') as string, 10) : 100,
      cantidad_actual: parseInt((fd.get('cantidad_actual') as string) || '0', 10),
      cantidad_minima: parseInt((fd.get('cantidad_minima') as string) || '2', 10),
      sede: (fd.get('sede') as string) || null,
      ubicacion: (fd.get('ubicacion') as string) || null,
      proveedor: (fd.get('proveedor') as string) || null,
      lote: (fd.get('lote') as string) || null,
      vencimiento: (fd.get('vencimiento') as string) || null,
      costo_unitario: fd.get('costo_unitario') ? parseFloat(fd.get('costo_unitario') as string) : 0,
    };
    const res = await fetch('/api/frascos', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
    if (res.ok) {
      (e.target as HTMLFormElement).reset();
      setShowNuevo(false);
      fetchFrascos();
    } else alert(await res.text());
  };

  const handleUpdate = async (id: string, data: any) => {
    const res = await fetch(`/api/frascos/${id}`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) });
    if (res.ok) {
      setEditing(null);
      fetchFrascos();
    } else alert(await res.text());
  };

  const handleDelete = async (id: string) => {
    if (!confirm('¿Eliminar frasco? Se borrará su historial.')) return;
    const res = await fetch(`/api/frascos/${id}`, { method: 'DELETE' });
    if (res.ok) fetchFrascos();
    else alert(await res.text());
  };

  const handleMov = async (id: string) => {
    if (!movCantidad || movCantidad <= 0) return alert('Cantidad requerida');
    const res = await fetch(`/api/frascos/${id}/movimiento`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        tipo: movTipo,
        cantidad: movCantidad,
        dispositivo_id: movDisp || null,
        nota: movNota || null,
      }),
    });
    if (res.ok) {
      setMovId(null);
      setMovDisp('');
      setMovNota('');
      setMovCantidad(1);
      fetchFrascos();
      if (historyId === id) fetchHistory(id);
    } else alert(await res.text());
  };

  const fetchHistory = async (id: string) => {
    const r = await fetch(`/api/frascos/${id}/movimientos`);
    const d = await r.json();
    setHistory(d.movimientos || []);
    setHistoryId(id);
  };

  return (
    <div className="space-y-4">
      <style>{`
        @media print {
          nav, footer { display: none !important; }
          body { background: white !important; -webkit-print-color-adjust: exact; print-color-adjust: exact; }
          @page { margin: 10mm; size: A4; }
          .frascos-grid { grid-template-columns: repeat(2, minmax(0,1fr)) !important; gap: 14px !important; }
          .frasco-card { break-inside: avoid !important; page-break-inside: avoid !important; box-shadow: none !important; border: 1px solid #cbd5e1 !important; }
          .frasco-card .print-hidden { display: none !important; }
          .encrypted-gallery { border: 1px solid #e2e8f0 !important; box-shadow: none !important; padding: 8px !important; }
          .encrypted-gallery .gallery-grid { display: grid !important; grid-template-columns: repeat(2, minmax(0,1fr)) !important; gap: 8px !important; columns: unset !important; }
          .encrypted-gallery .gallery-masonry { columns: 2 !important; gap: 8px !important; display: block !important; }
          .encrypted-gallery .gallery-img-wrap { height: 150px !important; min-height: 150px !important; max-height: 150px !important; background: white !important; display: flex !important; align-items: center !important; justify-content: center !important; overflow: hidden !important; }
          .encrypted-gallery .gallery-img { object-fit: contain !important; height: 100% !important; max-height: 150px !important; width: 100% !important; background: white !important; }
          .encrypted-gallery .gallery-card { break-inside: avoid !important; border: 1px solid #e2e8f0 !important; }
          /* enlarge cards */
          .frasco-card { font-size: 11px !important; }
        }
      `}</style>
      {/* Print header solo en impresión */}
      <div className="hidden print:block bg-white border border-slate-200 rounded-xl p-4">
        <h1 className="text-xl font-black text-slate-900 inline-flex items-center gap-2"><Droplet className="w-6 h-6" /> Informe Inventario de Frascos de Tinta</h1>
        <p className="text-xs text-slate-500 mt-1">Fecha: {new Date().toLocaleString('es-EC')} · {frascos.length} frascos · {frascos.filter((f) => f.estado !== 'disponible').length} en alerta (bajo/agotado) · Sede: {sedeFilter || 'todas'} · Color: {colorFilter || 'todos'}</p>
        <p className="text-[11px] text-slate-400 mt-1">Stock físico consumible — distinto de tinta en impresora (niveles_tinta %). Imágenes: vista &quot;Completa&quot; sin recorte para verticales, optimizada para impresión.</p>
      </div>

      {/* Diferencia clara con tintas de impresora */}
      <div className="bg-sky-50 border border-sky-200 rounded-xl px-4 py-3 text-xs text-sky-800 flex gap-2 print:hidden">
        <Droplet className="w-4 h-4 shrink-0 mt-0.5" />
        <div>
          <span className="font-bold">Frascos = stock físico (consumible)</span> — no es <span className="font-semibold">tinta en impresora</span> (<span className="inline-flex items-center gap-1"><Package className="w-3 h-3" /> niveles_tinta</span> por `dispositivo_id+color %`). Los frascos no llevan QR: se gestionan por <strong>código lote</strong> + <strong>cantidad</strong>. Al recargar una impresora, usa <em>Consumir</em> y vincula la impresora (opcional) — descuenta del stock.
        </div>
      </div>

      <div className="bg-white p-4 sm:p-5 rounded-2xl border border-slate-200 shadow-sm print:shadow-none print:border print:p-2">
        <div className="flex flex-col lg:flex-row gap-3 justify-between print:hidden">
          <div className="flex-1 flex flex-col sm:flex-row gap-2">
            <div className="relative flex-1">
              <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
              <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Buscar código, marca, lote, proveedor..." className="w-full border border-slate-200 rounded-full pl-9 pr-3 py-2.5 text-sm focus:border-sky-500 focus:outline-none" />
            </div>
            <select value={colorFilter} onChange={(e) => setColorFilter(e.target.value)} className="border border-slate-200 rounded-full px-3 py-2.5 text-sm">
              <option value="">Todos colores</option>
              {colores.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
            <input value={sedeFilter} onChange={(e) => setSedeFilter(e.target.value)} placeholder="Sede" className="border border-slate-200 rounded-full px-3 py-2.5 text-sm w-32" />
            <label className="inline-flex items-center gap-1.5 text-xs font-semibold cursor-pointer border border-amber-200 bg-amber-50 px-3 py-2 rounded-full">
              <input type="checkbox" checked={soloBajos} onChange={(e) => setSoloBajos(e.target.checked)} /> <AlertTriangle className="w-3.5 h-3.5" /> Solo bajos
            </label>
          </div>
          <div className="flex gap-2">
            <button onClick={() => window.print()} className="bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 px-5 py-2.5 rounded-full text-sm font-semibold inline-flex items-center gap-2 justify-center print:hidden">
              <Printer className="w-4 h-4" /> Imprimir informe
            </button>
            <button onClick={() => setShowNuevo((v) => !v)} className="bg-slate-900 text-white px-5 py-2.5 rounded-full text-sm font-semibold inline-flex items-center gap-2 justify-center print:hidden">
              {showNuevo ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />} {showNuevo ? 'Cerrar' : 'Nuevo frasco'}
            </button>
          </div>
        </div>

        {showNuevo && (
          <form onSubmit={handleCrear} className="mt-4 border border-slate-200 rounded-xl p-4 grid grid-cols-1 sm:grid-cols-3 gap-3 bg-slate-50 print:hidden">
            <input name="codigo" placeholder="Código (auto FR-XXXX si vacío)" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="marca" placeholder="Marca * (Epson, Canon...)" required className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <select name="color" required className="border border-slate-200 rounded-lg px-3 py-2 text-sm">
              <option value="">Color *</option>
              {colores.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
            <input name="modelo_compat" placeholder="Modelo compat (L3250...)" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <div className="border border-slate-200 rounded-lg px-3 py-2 bg-white flex flex-col justify-center">
              <label className="text-[11px] font-semibold text-slate-600 flex items-center gap-1"><Droplet className="w-3 h-3" /> Nivel % (visual, como tinta)</label>
              <input name="nivel_porcentaje" type="range" min="0" max="100" defaultValue={100} className="w-full mt-1 accent-sky-600" />
              <div className="flex justify-between text-[10px] text-slate-400"><span>0%</span><span>50%</span><span>100%</span></div>
            </div>
            <input name="cantidad_actual" type="number" required defaultValue={1} placeholder="Stock unidades *" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="cantidad_minima" type="number" defaultValue={2} placeholder="Mínimo alerta" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="sede" placeholder="Sede (Matriz, Icon...)" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="ubicacion" placeholder="Ubicación (Bodega...)" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="proveedor" placeholder="Proveedor" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="lote" placeholder="Lote" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="vencimiento" type="date" placeholder="Vencimiento" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="costo_unitario" type="number" step="0.01" placeholder="Costo unitario" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <div className="sm:col-span-3 flex gap-2 mt-2">
              <button type="submit" className="flex-1 bg-sky-600 text-white py-2.5 rounded-full font-semibold inline-flex items-center justify-center gap-2">
                <Package className="w-4 h-4" /> Crear frasco
              </button>
              <button type="button" onClick={() => setShowNuevo(false)} className="px-5 py-2.5 border border-slate-200 rounded-full">
                Cancelar
              </button>
            </div>
          </form>
        )}

        <div className="mt-4 flex items-center justify-between text-xs text-slate-500">
          <span>{frascos.length} frascos · {frascos.filter((f) => f.estado !== 'disponible').length} en alerta</span>
          <button onClick={fetchFrascos} className="border border-slate-200 px-3 py-1.5 rounded-full inline-flex items-center gap-1">
            <ArrowUpDown className="w-3 h-3" /> Actualizar
          </button>
        </div>
      </div>

      {loading ? (
        <p className="text-center text-slate-400 py-8">Cargando frascos...</p>
      ) : frascos.length === 0 ? (
        <div className="bg-white border-2 border-dashed border-slate-200 rounded-2xl p-8 text-center">
          <Droplet className="w-8 h-8 mx-auto text-slate-300" />
          <p className="text-sm text-slate-500 mt-2">Sin frascos. Crea el primero con el stock inicial.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 min-w-0 frascos-grid print:grid-cols-2">
          {frascos.map((f) => (
            <div key={f.id} className={`bg-white border rounded-2xl p-4 shadow-sm min-w-0 max-w-full overflow-hidden frasco-card print:break-inside-avoid ${f.estado === 'agotado' ? 'border-rose-200 bg-rose-50/30' : f.estado === 'bajo' ? 'border-amber-200 bg-amber-50/30' : 'border-slate-200'}`}>
              <div className="flex justify-between items-start gap-2">
                <div className="min-w-0 flex-1">
                  <p className="font-mono text-xs text-slate-500 truncate">{f.codigo} · {f.lote ? `Lote ${f.lote}` : 'Sin lote'}</p>
                  <h3 className="font-bold text-slate-900 truncate inline-flex items-center gap-1.5">
                    <Package className="w-4 h-4 text-slate-600" /> {f.marca} {f.modelo_compat ? `· ${f.modelo_compat}` : ''}
                  </h3>
                  <div className="flex gap-3 items-start">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mt-1 flex-wrap">
                        <span className={`text-[10px] font-bold px-2 py-1 rounded-full uppercase ${colorClasses[f.color] || 'bg-slate-500 text-white'}`}>{f.color}</span>
                        <span className={`text-xs font-bold px-2 py-1 rounded-full border ${f.estado === 'agotado' ? 'bg-rose-600 text-white border-rose-600' : f.estado === 'bajo' ? 'bg-amber-500 text-white border-amber-500' : 'bg-emerald-100 text-emerald-700 border-emerald-200'}`}>
                          stock {f.cantidad_actual} / mín {f.cantidad_minima} · {f.estado}
                        </span>
                      </div>
                      <div className="mt-2 flex items-center gap-2 text-xs text-slate-600">
                        <Droplet className="w-3.5 h-3.5 text-slate-400" /> Nivel visual {f.nivel_porcentaje}% {f.nivel_porcentaje <= 20 && <span className="text-amber-600 font-bold">· ¡bajo!</span>}
                      </div>
                    </div>
                    <div className="shrink-0 flex flex-col items-center gap-1">
                      <div className="w-12 h-20 border-2 border-slate-300 rounded-b-xl rounded-t-sm bg-slate-50 relative overflow-hidden shadow-inner">
                        <div
                          className={`absolute bottom-0 w-full transition-all duration-500 ${f.color === 'negro' ? 'bg-slate-900' : f.color === 'cian' ? 'bg-cyan-500' : f.color === 'magenta' ? 'bg-fuchsia-500' : f.color === 'amarillo' ? 'bg-amber-400' : f.color === 'negro_foto' ? 'bg-zinc-800' : f.color === 'gris' ? 'bg-zinc-400' : 'bg-slate-600'}`}
                          style={{ height: `${Math.max(0, Math.min(100, f.nivel_porcentaje))}%` }}
                        />
                        <div className="absolute inset-0 border border-white/20 rounded-b-xl pointer-events-none" />
                      </div>
                      <span className={`text-xs font-black px-2 py-0.5 rounded-full ${f.nivel_porcentaje <= 20 ? 'bg-amber-100 text-amber-700' : f.nivel_porcentaje <= 50 ? 'bg-sky-100 text-sky-700' : 'bg-emerald-100 text-emerald-700'}`}>{f.nivel_porcentaje}%</span>
                    </div>
                  </div>
                </div>
                <div className="flex flex-col gap-1 shrink-0 print:hidden">
                  <button onClick={() => setEditing(editing === f.id ? null : f.id)} className="w-8 h-8 grid place-items-center border border-slate-200 rounded-full hover:bg-slate-50">
                    <Edit2 className="w-4 h-4" />
                  </button>
                  <button onClick={() => handleDelete(f.id)} className="w-8 h-8 grid place-items-center border border-rose-200 text-rose-600 rounded-full hover:bg-rose-50">
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>

              <div className="mt-3 grid grid-cols-2 gap-2 text-xs text-slate-600 border-t border-slate-100 pt-3">
                <span className="inline-flex items-center gap-1"><Building2 className="w-3 h-3" /> {f.proveedor || '—'}</span>
                <span className="inline-flex items-center gap-1"><MapPin className="w-3 h-3" /> {f.ubicacion || '—'}</span>
                <span className="inline-flex items-center gap-1"><Tag className="w-3 h-3" /> Sede: {f.sede || '—'}</span>
                <span className="inline-flex items-center gap-1"><DollarSign className="w-3 h-3" /> ${f.costo_unitario}</span>
                {f.vencimiento && <span className="inline-flex items-center gap-1 col-span-2"><Calendar className="w-3 h-3" /> Vence: {f.vencimiento}</span>}
              </div>

              {editing === f.id && (
                <form
                  onSubmit={(e) => {
                    e.preventDefault();
                    const fd = new FormData(e.currentTarget as HTMLFormElement);
                    handleUpdate(f.id, {
                      marca: fd.get('marca'),
                      modelo_compat: fd.get('modelo_compat') || null,
                      color: fd.get('color'),
                      nivel_porcentaje: parseInt((fd.get('nivel_porcentaje') as string) || '0', 10),
                      cantidad_minima: parseInt((fd.get('cantidad_minima') as string) || '0', 10),
                      sede: fd.get('sede') || null,
                      ubicacion: fd.get('ubicacion') || null,
                      proveedor: fd.get('proveedor') || null,
                      lote: fd.get('lote') || null,
                      vencimiento: fd.get('vencimiento') || null,
                      costo_unitario: parseFloat((fd.get('costo_unitario') as string) || '0'),
                    });
                  }}
                  className="mt-3 border border-sky-200 rounded-xl p-3 bg-sky-50/50 grid grid-cols-2 gap-2 print:hidden"
                >
                  <input name="marca" defaultValue={f.marca} className="border border-slate-200 rounded-lg px-2 py-2 text-xs" placeholder="Marca" />
                  <select name="color" defaultValue={f.color} className="border border-slate-200 rounded-lg px-2 py-2 text-xs">
                    {colores.map((c) => (
                      <option key={c} value={c}>
                        {c}
                      </option>
                    ))}
                  </select>
                  <input name="modelo_compat" defaultValue={f.modelo_compat || ''} placeholder="Modelo" className="border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <div className="border border-slate-200 rounded-lg px-2 py-2 bg-white flex flex-col">
                    <label className="text-[10px] font-semibold text-slate-600 flex items-center gap-1"><Droplet className="w-3 h-3" /> Nivel %</label>
                    <input name="nivel_porcentaje" type="range" min="0" max="100" defaultValue={f.nivel_porcentaje ?? 100} className="w-full accent-sky-600" />
                    <div className="flex justify-between text-[10px] text-slate-400"><span>0</span><span>{f.nivel_porcentaje}%</span><span>100</span></div>
                  </div>
                  <input name="cantidad_minima" type="number" defaultValue={f.cantidad_minima} placeholder="Mín stock" className="border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <input name="sede" defaultValue={f.sede || ''} placeholder="Sede" className="border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <input name="ubicacion" defaultValue={f.ubicacion || ''} placeholder="Ubicación" className="border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <input name="proveedor" defaultValue={f.proveedor || ''} placeholder="Proveedor" className="border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <input name="lote" defaultValue={f.lote || ''} placeholder="Lote" className="border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <input name="vencimiento" type="date" defaultValue={f.vencimiento || ''} className="border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <input name="costo_unitario" type="number" step="0.01" defaultValue={f.costo_unitario} className="col-span-2 border border-slate-200 rounded-lg px-2 py-2 text-xs" placeholder="Costo" />
                  <button type="submit" className="col-span-2 bg-sky-600 text-white py-2 rounded-full text-xs font-semibold inline-flex items-center justify-center gap-1">
                    <Check className="w-3 h-3" /> Guardar
                  </button>
                </form>
              )}

              <div className="mt-3 flex flex-wrap gap-1.5 print:hidden">
                <button onClick={() => { setMovId(movId === f.id ? null : f.id); setMovTipo('ingreso'); }} className="flex-1 bg-emerald-600 text-white px-3 py-2 rounded-full text-xs font-semibold inline-flex items-center justify-center gap-1">
                  <Plus className="w-3 h-3" /> Ingreso
                </button>
                <button onClick={() => { setMovId(movId === f.id ? null : f.id); setMovTipo('consumo'); }} className="flex-1 bg-slate-900 text-white px-3 py-2 rounded-full text-xs font-semibold inline-flex items-center justify-center gap-1">
                  <Minus className="w-3 h-3" /> Consumir
                </button>
                <button onClick={() => (historyId === f.id ? setHistoryId(null) : fetchHistory(f.id))} className="px-3 py-2 border border-slate-200 rounded-full text-xs inline-flex items-center gap-1">
                  <Clock className="w-3 h-3" /> Historial
                </button>
              </div>

              {movId === f.id && (
                <div className="mt-3 border border-slate-200 rounded-xl p-3 bg-white space-y-2 print:hidden">
                  <div className="flex gap-2">
                    <select value={movTipo} onChange={(e) => setMovTipo(e.target.value as any)} className="flex-1 border border-slate-200 rounded-lg px-2 py-2 text-xs">
                      <option value="ingreso">Ingreso (+)</option>
                      <option value="consumo">Consumo (-)</option>
                      <option value="ajuste">Ajuste</option>
                    </select>
                    <input type="number" min="1" value={movCantidad} onChange={(e) => setMovCantidad(parseInt(e.target.value, 10) || 1)} className="w-20 border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  </div>
                  <DeviceSearchCombobox value={movDisp} onChange={(id) => setMovDisp(id)} placeholder="Impresora destino (opcional)" />
                  <input value={movNota} onChange={(e) => setMovNota(e.target.value)} placeholder="Nota (ej: recarga Epson L3250)" className="w-full border border-slate-200 rounded-lg px-2 py-2 text-xs" />
                  <button onClick={() => handleMov(f.id)} className="w-full bg-sky-600 text-white py-2 rounded-full text-xs font-semibold">
                    Confirmar {movTipo}
                  </button>
                </div>
              )}

              {historyId === f.id && (
                <div className="mt-3 border border-slate-200 rounded-xl p-3 bg-slate-50 max-h-48 overflow-auto">
                  <p className="text-xs font-bold mb-2">Movimientos</p>
                  {history.length === 0 ? (
                    <p className="text-xs text-slate-400">Sin movimientos</p>
                  ) : (
                    <div className="space-y-1">
                      {history.map((m) => (
                        <div key={m.id} className="text-xs border border-slate-200 bg-white rounded-lg px-2 py-1.5 flex justify-between gap-2">
                          <span className={`font-bold ${m.cantidad > 0 ? 'text-emerald-600' : 'text-rose-600'}`}>
                            {m.tipo} {m.cantidad > 0 ? `+${m.cantidad}` : m.cantidad}
                          </span>
                          <span className="text-slate-500 truncate">{m.nota || ''} {m.dispositivo_id ? `· ${m.dispositivo_id.slice(0, 8)}` : ''}</span>
                          <span className="text-slate-400 shrink-0">{new Date(m.fecha).toLocaleString('es-EC')}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              <div className="mt-4 min-w-0 max-w-full overflow-hidden">
                <EncryptedGallery entidadTipo="frasco" entidadId={f.id} compact />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
