import React, { useState, useEffect } from 'react';
import EncryptedGallery from './EncryptedGallery';
import InkLevelGauge from './InkLevelGauge';
import {
  AlertTriangle,
  Building2,
  Calendar,
  Clock,
  DollarSign,
  FileText,
  Link2,
  MapPin,
  Pencil,
  Puzzle,
  Save,
  Store,
  Tag,
  Trash2,
  User,
  X,
} from 'lucide-react';

export default function DispositivoDetail({ id: propId }) {
  const getRealId = () => {
    if (propId && propId !== 'placeholder') return propId;
    if (typeof window !== 'undefined') {
      const parts = window.location.pathname.split('/').filter(Boolean);
      return parts[parts.length - 1] || propId;
    }
    return propId;
  };
  const id = getRealId();
  const [dispositivo, setDispositivo] = useState(null);
  const [mantenimientos, setMantenimientos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [editingId, setEditingId] = useState(null);
  const [editingDisp, setEditingDisp] = useState(false);
  const [savingDisp, setSavingDisp] = useState(false);
  const [movimientos, setMovimientos] = useState([]);

  const fetchAll = () => {
    setLoading(true);
    fetch(`/api/dispositivos/${id}`)
      .then((r) => {
        if (!r.ok) throw new Error(r.status === 404 ? 'No encontrado' : `Error ${r.status}`);
        return r.json();
      })
      .then((data) => {
        setDispositivo(data);
        setError(null);
        return fetch(`/api/mantenimientos?dispositivo_id=${id}`);
      })
      .then((r) => (r && r.ok ? r.json() : { mantenimientos: [] }))
      .then((d) => {
        setMantenimientos(d.mantenimientos || []);
        return fetch(`/api/movimientos?dispositivo_id=${id}`);
      })
      .then((r) => (r && r.ok ? r.json() : { movimientos: [] }))
      .then((d) => setMovimientos(d.movimientos || []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchAll();
  }, [id]);

  const handleBaja = async (e) => {
    e.preventDefault();
    const motivo = e.target.motivo_baja.value;
    if (!motivo) return alert('Motivo requerido');
    const res = await fetch(`/api/dispositivos/${id}/baja`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ motivo_baja: motivo }),
    });
    if (res.ok) fetchAll();
    else alert('Error al dar de baja');
  };

  const handleDispUpdate = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const payload = {
      codigo_activo: fd.get('codigo_activo'),
      numero_serie: fd.get('numero_serie'),
      tipo: fd.get('tipo'),
      marca: fd.get('marca'),
      modelo: fd.get('modelo'),
      especificaciones: fd.get('especificaciones'),
      valor_compra: parseFloat(fd.get('valor_compra')) || 0,
      proveedor: fd.get('proveedor'),
      ubicacion: fd.get('ubicacion'),
      sede: fd.get('sede') || null,
      asignado_a: fd.get('asignado_a') || null,
      fecha_compra: fd.get('fecha_compra'),
      estado: fd.get('estado'),
      tipo_componente: fd.get('tipo_componente') || null,
      parent_id: fd.get('parent_id') || null,
    };
    if (!payload.asignado_a) payload.asignado_a = "";
    if (!payload.sede) payload.sede = "";
    if (!payload.tipo_componente) payload.tipo_componente = "";
    if (!payload.parent_id) payload.parent_id = "";
    setSavingDisp(true);
    const res = await fetch(`/api/dispositivos/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    setSavingDisp(false);
    if (res.ok) {
      setEditingDisp(false);
      fetchAll();
    } else {
      const err = await res.text().catch(() => '');
      alert('Error al actualizar dispositivo: ' + err);
    }
  };

  const handleMoverComponente = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const destino = fd.get('destino_parent_id')?.toString().trim();
    const tipo = fd.get('tipo_movimiento')?.toString() || 'traslado';
    const body = {
      dispositivo_id: id,
      destino_parent_id: destino || null,
      tipo_movimiento: destino ? tipo : 'retiro',
      descripcion: fd.get('descripcion')?.toString() || null,
      realizado_por: fd.get('realizado_por')?.toString() || null,
    };
    if (!destino && tipo !== 'retiro' && tipo !== 'desasignacion') body.tipo_movimiento = 'retiro';
    const res = await fetch('/api/movimientos', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    if (res.ok) { e.target.reset(); fetchAll(); } else alert('Error movimiento: ' + await res.text());
  };

  const handleDispDelete = async () => {
    if (!confirm('¿Eliminar este dispositivo? Se borrarán también mantenimientos y tintas.')) return;
    const res = await fetch(`/api/dispositivos/${id}`, { method: 'DELETE' });
    if (res.ok) {
      window.location.href = '/';
    } else {
      const err = await res.text().catch(() => '');
      alert('Error al eliminar: ' + err);
    }
  };

  const handleMant = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const payload = {
      dispositivo_id: id,
      realizado_por: fd.get('realizado_por'),
      descripcion: fd.get('descripcion'),
      costo: parseFloat(fd.get('costo')) || 0,
      fecha_hora: new Date().toISOString().slice(0, 19).replace('T', ' '),
    };
    const pdfFile = fd.get('pdf');
    const urlPdf = (fd.get('url_pdf') || '').toString().trim();
    let res;
    if (pdfFile && pdfFile.size > 0) {
      const form = new FormData();
      Object.entries(payload).forEach(([k, v]) => form.append(k, String(v)));
      form.append('pdf', pdfFile);
      res = await fetch('/api/mantenimientos', { method: 'POST', body: form });
    } else if (urlPdf) {
      res = await fetch('/api/mantenimientos', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...payload, url_pdf: urlPdf }),
      });
    } else {
      res = await fetch('/api/mantenimientos', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
    }
    if (res.ok) {
      e.target.reset();
      fetchAll();
    } else {
      const err = await res.text().catch(() => '');
      alert('Error al guardar mantenimiento: ' + err);
    }
  };

  const handleDelete = async (mantId) => {
    if (!confirm('¿Eliminar este mantenimiento? Esta acción no se puede deshacer.')) return;
    const res = await fetch(`/api/mantenimientos/${mantId}`, { method: 'DELETE' });
    if (res.ok) {
      setMantenimientos((prev) => prev.filter((m) => m.id !== mantId));
    } else {
      const err = await res.text().catch(() => '');
      alert('Error al eliminar: ' + err);
    }
  };

  const handleUpdate = async (e, mantId) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const pdfFile = fd.get('pdf');
    const urlPdfValue = (fd.get('url_pdf') || '').toString().trim();
    const hasNewUrlField = fd.has('url_pdf');

    if (pdfFile && pdfFile.size > 0) {
      const form = new FormData();
      form.append('realizado_por', fd.get('realizado_por'));
      form.append('descripcion', fd.get('descripcion'));
      form.append('costo', fd.get('costo') || '0');
      form.append('pdf', pdfFile);
      const res = await fetch(`/api/mantenimientos/${mantId}`, { method: 'PUT', body: form });
      if (res.ok) {
        setEditingId(null);
        fetchAll();
      } else {
        const err = await res.text().catch(() => '');
        alert('Error al actualizar: ' + err);
      }
      return;
    }

    const body = {
      realizado_por: fd.get('realizado_por'),
      descripcion: fd.get('descripcion'),
      costo: parseFloat(fd.get('costo')) || 0,
    };
    if (hasNewUrlField) {
      body.url_pdf = urlPdfValue;
    }
    const res = await fetch(`/api/mantenimientos/${mantId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (res.ok) {
      setEditingId(null);
      fetchAll();
    } else {
      const err = await res.text().catch(() => '');
      alert('Error al actualizar: ' + err);
    }
  };

  if (loading) return <p className="text-center text-slate-400 py-12">Cargando dispositivo...</p>;
  if (error)
    return (
      <div className="bg-rose-50 border border-rose-200 text-rose-700 p-6 rounded-xl">
        <p className="inline-flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</p>
        <a href="/" className="text-sm underline mt-2 inline-block">
          ← Volver al inventario
        </a>
      </div>
    );
  if (!dispositivo) return null;

  return (
    <div className="space-y-8">
      {/* Ficha */}
      {!editingDisp ? (
        <header className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm flex flex-col md:flex-row justify-between gap-4">
          <div className="flex-1">
            <p className="text-xs font-mono text-slate-400">
              Código: {dispositivo.codigo_activo} | S/N: {dispositivo.numero_serie}
            </p>
            <h1 className="text-2xl font-bold text-slate-900 mt-1">
              {dispositivo.marca} {dispositivo.modelo}
            </h1>
            <p className="text-sm text-slate-500 mt-1">
              {dispositivo.tipo} · {dispositivo.especificaciones || 'Sin especificaciones'}
            </p>
            <div className="flex flex-wrap gap-4 mt-4 text-xs text-slate-600">
              <span className="inline-flex items-center gap-1"><DollarSign className="w-3.5 h-3.5" /> <strong>Valor:</strong> ${dispositivo.valor_compra}</span>
              <span className="inline-flex items-center gap-1"><Building2 className="w-3.5 h-3.5" /> <strong>Proveedor:</strong> {dispositivo.proveedor}</span>
              <span className="inline-flex items-center gap-1"><MapPin className="w-3.5 h-3.5" /> <strong>Ubicación:</strong> {dispositivo.ubicacion}</span>
              {dispositivo.sede && <span className="inline-flex items-center gap-1"><Store className="w-3.5 h-3.5" /> <strong>Sede:</strong> {dispositivo.sede}</span>}
              <span className="inline-flex items-center gap-1"><User className="w-3.5 h-3.5" /> <strong>Asignado:</strong> {dispositivo.asignado_a || '—'}</span>
              <span className="inline-flex items-center gap-1"><Calendar className="w-3.5 h-3.5" /> <strong>Compra:</strong> {dispositivo.fecha_compra}</span>
              {dispositivo.tipo_componente && <span className="inline-flex items-center gap-1"><Puzzle className="w-3.5 h-3.5" /> <strong>Componente:</strong> {dispositivo.tipo_componente}</span>}
              {dispositivo.parent_id && <span className="inline-flex items-center gap-1"><Link2 className="w-3.5 h-3.5" /> <strong>Padre:</strong> <a href={`/dispositivos/${dispositivo.parent_id}`} className="text-sky-600 underline">{dispositivo.parent_id.slice(0,8)}</a></span>}
            </div>
            {dispositivo.fecha_baja && (
              <div className="mt-4 bg-rose-50 border border-rose-200 text-rose-700 px-4 py-2 rounded-lg text-sm">
                <strong>Dado de baja:</strong> {dispositivo.fecha_baja} — {dispositivo.motivo_baja}
              </div>
            )}
          </div>
          <div className="flex flex-col gap-2 items-end shrink-0">
            <span
              className={`px-3 py-1.5 rounded-full text-xs font-bold uppercase tracking-wider
                ${dispositivo.estado === 'activo' ? 'bg-emerald-100 text-emerald-700' : ''}
                ${dispositivo.estado === 'disponible' ? 'bg-sky-100 text-sky-700' : ''}
                ${dispositivo.estado === 'reserva' ? 'bg-sky-100 text-sky-700' : ''}
                ${dispositivo.estado === 'en_mantenimiento' ? 'bg-amber-100 text-amber-700' : ''}
                ${dispositivo.estado === 'en_reparacion' ? 'bg-orange-100 text-orange-700' : ''}
                ${dispositivo.estado === 'averiado' ? 'bg-red-100 text-red-700' : ''}
                ${dispositivo.estado === 'obsoleto' ? 'bg-slate-200 text-slate-700' : ''}
                ${dispositivo.estado === 'dado_de_baja' ? 'bg-rose-100 text-rose-700' : ''}
                ${dispositivo.estado === 'extraviado' ? 'bg-purple-100 text-purple-700' : 'bg-slate-100 text-slate-700'}`}
            >
              {dispositivo.estado}
            </span>
            <div className="flex gap-2 mt-2">
              <button onClick={() => setEditingDisp(true)} className="text-xs bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 px-4 py-2 rounded-full font-medium inline-flex items-center gap-1.5">
                <Pencil className="w-3.5 h-3.5" /> Editar datos
              </button>
              <button onClick={handleDispDelete} className="text-xs bg-rose-50 border border-rose-200 hover:bg-rose-100 text-rose-700 px-4 py-2 rounded-full font-medium inline-flex items-center gap-1.5">
                <Trash2 className="w-3.5 h-3.5" /> Eliminar
              </button>
            </div>
            <a href="/" className="text-xs text-slate-500 hover:text-slate-700 mt-1">
              ← Volver
            </a>
          </div>
        </header>
      ) : (
        <form onSubmit={handleDispUpdate} className="bg-white p-6 rounded-2xl border border-sky-200 shadow-sm space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="text-lg font-bold text-slate-900 inline-flex items-center gap-2"><Pencil className="w-5 h-5" /> Editar dispositivo</h2>
            <button type="button" onClick={() => setEditingDisp(false)} className="text-sm text-slate-500 hover:text-slate-700 inline-flex items-center gap-1">
              Cancelar <X className="w-4 h-4" />
            </button>
          </div>
          <div className="grid sm:grid-cols-2 gap-4">
            <div>
              <label className="text-xs font-semibold text-slate-600">Código Activo *</label>
              <input name="codigo_activo" defaultValue={dispositivo.codigo_activo} required className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-600">Número Serie *</label>
              <input name="numero_serie" defaultValue={dispositivo.numero_serie} required className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            </div>
          </div>
          <div className="grid sm:grid-cols-3 gap-4">
            <div>
              <label className="text-xs font-semibold text-slate-600">Tipo *</label>
              <select name="tipo" defaultValue={dispositivo.tipo} className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm">
                <option>Desktop</option>
                <option>Laptop</option>
                <option>Servidor</option>
                <option>Tablet</option>
                <option>Monitor</option>
                <option>Proyector</option>
                <option>Impresora</option>
                <option>Scanner</option>
                <option>Router</option>
                <option>Switch</option>
                <option>Access Point</option>
                <option>Firewall</option>
                <option>NAS</option>
                <option>UPS</option>
                <option>Regulador</option>
                <option>Telefono IP</option>
                <option>Camara IP</option>
                <option>Disco Externo</option>
                <option>Mouse</option>
                <option>Teclado</option>
                <option>Parlante</option>
                <option>Microfono</option>
                <option>Camara</option>
                <option>Pantalla</option>
                <option>Audifono</option>
                <option>Auricular</option>
                <option>Cargador</option>
                <option>Otro</option>
              </select>
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-600">Marca *</label>
              <input name="marca" defaultValue={dispositivo.marca} required className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-600">Modelo *</label>
              <input name="modelo" defaultValue={dispositivo.modelo} required className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            </div>
          </div>
          <div>
            <label className="text-xs font-semibold text-slate-600">Especificaciones</label>
            <textarea name="especificaciones" defaultValue={dispositivo.especificaciones} rows="2" className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm"></textarea>
          </div>
          <div className="grid sm:grid-cols-3 gap-4">
            <div>
              <label className="text-xs font-semibold text-slate-600">Valor compra</label>
              <input name="valor_compra" type="number" step="0.01" defaultValue={dispositivo.valor_compra} className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-600">Proveedor</label>
              <input name="proveedor" defaultValue={dispositivo.proveedor} className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-600">Ubicación</label>
              <input name="ubicacion" defaultValue={dispositivo.ubicacion} className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            </div>
          </div>
          <div className="grid sm:grid-cols-3 gap-4">
            <div>
              <label className="text-xs font-semibold text-slate-600">Asignado a</label>
              <input name="asignado_a" defaultValue={dispositivo.asignado_a || ''} placeholder="correo@..." className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-600">Fecha compra</label>
              <input name="fecha_compra" type="date" defaultValue={dispositivo.fecha_compra} className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-600">Estado</label>
              <select name="estado" defaultValue={dispositivo.estado} className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm">
                <option value="activo">activo</option>
                <option value="disponible">disponible</option>
                <option value="reserva">reserva</option>
                <option value="en_mantenimiento">en_mantenimiento</option>
                <option value="en_reparacion">en_reparacion</option>
                <option value="averiado">averiado</option>
                <option value="obsoleto">obsoleto</option>
                <option value="dado_de_baja">dado_de_baja</option>
                <option value="extraviado">extraviado</option>
              </select>
            </div>
          </div>
          <div className="grid sm:grid-cols-3 gap-4">
            <div>
              <label className="text-xs font-semibold text-slate-600">Sede</label>
              <input name="sede" defaultValue={dispositivo.sede || ''} placeholder="Matriz, Icon..." className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-600">Tipo Componente</label>
              <select name="tipo_componente" defaultValue={dispositivo.tipo_componente || ''} className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm">
                <option value="">— No es componente —</option>
                <option>Ventilador</option>
                <option>SSD</option>
                <option>HDD</option>
                <option>NVMe</option>
                <option>RAM</option>
                <option>Bateria</option>
                <option>Fuente</option>
                <option>Placa</option>
                <option>CPU</option>
                <option>GPU</option>
                <option>Teclado</option>
                <option>Pantalla</option>
                <option>Cargador</option>
                <option>Disco</option>
                <option>Disco Externo</option>
                <option>Cable Poder</option>
                <option>Cable HDMI</option>
                <option>Cable VGA</option>
                <option>Cable DisplayPort</option>
                <option>Cable Ethernet</option>
                <option>Cable USB</option>
                <option>Cable SATA</option>
                <option>Patch Cord</option>
                <option>Adaptador</option>
                <option>Regulador</option>
                <option>UPS</option>
                <option>Bateria UPS</option>
                <option>Fusible</option>
                <option>Tarjeta Red</option>
                <option>Mouse</option>
                <option>Parlante</option>
                <option>Microfono</option>
                <option>Camara</option>
                <option>Disipador</option>
                <option>Otro</option>
              </select>
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-600">Parent ID (vacío = sin padre)</label>
              <input name="parent_id" defaultValue={dispositivo.parent_id || ''} placeholder="UUID padre" className="w-full mt-1 border border-slate-200 rounded-lg px-3 py-2 text-sm font-mono" />
            </div>
          </div>
          <button type="submit" disabled={savingDisp} className="w-full bg-sky-600 hover:bg-sky-500 disabled:opacity-50 text-white font-semibold py-3 rounded-xl transition inline-flex items-center justify-center gap-2">
            <Save className="w-4 h-4" /> {savingDisp ? 'Guardando...' : 'Guardar cambios'}
          </button>
        </form>
      )}

      {dispositivo.tipo === 'Impresora' && (
        <InkLevelGauge tintas={dispositivo.tintas || []} dispositivoId={id} editable={true} />
      )}

      <EncryptedGallery entidadTipo="dispositivo" entidadId={id} />

      {/* Etiqueta UUID */}
      <section className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm">
        <h2 className="text-lg font-bold text-slate-900 mb-3 inline-flex items-center gap-2"><Tag className="w-5 h-5" /> Etiqueta UUID</h2>
        {dispositivo.etiqueta ? (
          <div className="flex flex-col sm:flex-row gap-4 items-start">
            <div className="flex-1">
              <p className="font-mono text-sm font-bold">{dispositivo.etiqueta.id}</p>
              <p className="text-xs text-slate-500">{dispositivo.etiqueta.codigo_barras} · v{dispositivo.etiqueta.version} · {dispositivo.etiqueta.estado}</p>
              <p className="text-xs text-slate-400 mt-1">Generada {new Date(dispositivo.etiqueta.generado_en).toLocaleString('es-EC')} {dispositivo.etiqueta.asignado_en ? `· Asignada ${new Date(dispositivo.etiqueta.asignado_en).toLocaleString('es-EC')}` : ''}</p>
              <div className="flex gap-2 mt-3">
                <button
                  onClick={async () => {
                    if (!confirm('¿Desasociar etiqueta? Quedará disponible para reutilizar.')) return;
                    const res = await fetch(`/api/etiquetas/${dispositivo.etiqueta.id}/desasociar`, { method: 'POST' });
                    if (res.ok) fetchAll(); else alert(await res.text());
                  }}
                  className="text-xs bg-amber-50 border border-amber-200 px-3 py-1.5 rounded-full hover:bg-amber-100"
                >
                  Desasociar
                </button>
                <a href={`/etiquetas`} className="text-xs bg-slate-900 text-white px-3 py-1.5 rounded-full">
                  Gestionar etiquetas
                </a>
              </div>
            </div>
            <div className="bg-white border p-2 rounded-xl">
              {/* QR inline usando API pública o canvas; fallback texto */}
              <img src={`https://api.qrserver.com/v1/create-qr-code/?size=120x120&data=${encodeURIComponent(dispositivo.etiqueta.id)}`} alt="QR" className="w-[120px] h-[120px]" />
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            <p className="text-sm text-slate-500">Sin etiqueta asignada. Escanea una etiqueta disponible o pega su UUID.</p>
            <form
              onSubmit={async (e) => {
                e.preventDefault();
                const fd = new FormData(e.target);
                const etiquetaId = fd.get('etiqueta_id')?.toString().trim();
                if (!etiquetaId) return alert('Ingresa UUID');
                const res = await fetch(`/api/etiquetas/${etiquetaId}/asociar`, {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({ dispositivo_id: id }),
                });
                if (res.ok) { e.target.reset(); fetchAll(); } else alert(await res.text());
              }}
              className="flex gap-2"
            >
              <input name="etiqueta_id" placeholder="UUID v4/v7 o ET-XXXX escaneado" className="flex-1 border border-slate-200 rounded-lg px-3 py-2 text-sm font-mono" />
              <button type="submit" className="bg-sky-600 text-white px-5 py-2 rounded-full text-sm hover:bg-sky-500">
                Asociar
              </button>
            </form>
            <a href="/escaner" className="text-xs text-sky-600 underline">
              Abrir escáner →
            </a>
          </div>
        )}
      </section>

      {/* Componentes */}
      <section className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm">
        <h2 className="text-lg font-bold text-slate-900 mb-4 inline-flex items-center gap-2"><Puzzle className="w-5 h-5" /> Componentes ({dispositivo.componentes?.length || 0})</h2>
        {dispositivo.parent_id && <p className="text-xs text-slate-500 mb-2">Este equipo es componente de: <a href={`/dispositivos/${dispositivo.parent_id}`} className="text-sky-600 underline">{dispositivo.parent_id}</a> ({dispositivo.tipo_componente})</p>}
        {(!dispositivo.componentes || dispositivo.componentes.length === 0) ? (
          <p className="text-sm text-slate-400">Sin componentes asignados. Ej: ventilador, SSD, RAM dentro de este equipo.</p>
        ) : (
          <div className="grid sm:grid-cols-2 gap-3">
            {dispositivo.componentes.map((c) => (
              <a key={c.id} href={`/dispositivos/${c.id}`} className="border border-slate-200 rounded-xl p-3 hover:shadow flex justify-between">
                <div>
                  <p className="text-xs font-mono text-slate-400">{c.codigo_activo} · {c.tipo_componente || c.tipo}</p>
                  <p className="font-semibold text-sm">{c.marca} {c.modelo}</p>
                  <p className="text-xs text-slate-500">{c.estado}</p>
                </div>
                <span className="text-xs bg-slate-100 px-2 py-1 rounded-full h-fit">{c.tipo_componente || '—'}</span>
              </a>
            ))}
          </div>
        )}
      </section>

      {/* Historial Movimientos */}
      <section className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm">
        <h2 className="text-lg font-bold text-slate-900 mb-4">📦 Historial de Movimientos</h2>
        {movimientos.length === 0 ? (
          <p className="text-sm text-slate-400">Sin movimientos. Los cambios de parent_id se registran automáticamente.</p>
        ) : (
          <div className="space-y-2">
            {movimientos.map((m) => (
              <div key={m.id} className="border border-slate-100 rounded-xl p-3 bg-slate-50 text-sm">
                <div className="flex justify-between">
                  <span className="font-semibold">{m.tipo_movimiento}</span>
                  <span className="text-xs text-slate-400">{m.fecha_movimiento}</span>
                </div>
                <p className="text-xs text-slate-600 mt-1">Origen: {m.origen_parent_id || '—'} → Destino: {m.destino_parent_id || '—'}</p>
                {m.descripcion && <p className="text-slate-700 mt-1">{m.descripcion}</p>}
                {m.realizado_por && <p className="text-xs text-slate-500">Por: {m.realizado_por}</p>}
              </div>
            ))}
          </div>
        )}
        <details className="mt-4 border border-slate-200 rounded-xl p-3">
          <summary className="text-sm font-semibold cursor-pointer">+ Registrar movimiento manual</summary>
          <form onSubmit={handleMoverComponente} className="mt-3 space-y-2">
            <div className="grid sm:grid-cols-2 gap-2">
              <select name="tipo_movimiento" className="border border-slate-200 rounded-lg px-3 py-2 text-sm">
                <option value="traslado">traslado</option>
                <option value="instalacion">instalacion</option>
                <option value="retiro">retiro</option>
                <option value="canibalizacion">canibalizacion</option>
                <option value="asignacion">asignacion</option>
                <option value="desasignacion">desasignacion</option>
              </select>
              <input name="destino_parent_id" placeholder="UUID destino (vacío = retiro)" className="border border-slate-200 rounded-lg px-3 py-2 text-sm font-mono" />
            </div>
            <input name="descripcion" placeholder="Descripción" className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <input name="realizado_por" placeholder="Realizado por" className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <button type="submit" className="w-full bg-slate-900 text-white py-2 rounded-lg text-sm">Registrar</button>
          </form>
        </details>
      </section>

      <section className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm">
        <h2 className="text-lg font-bold text-slate-900 mb-4 inline-flex items-center gap-2"><FileText className="w-5 h-5" /> Historial de Mantenimientos</h2>
        {mantenimientos.length === 0 ? (
          <p className="text-sm text-slate-400">Sin mantenimientos registrados.</p>
        ) : (
          <div className="space-y-3">
            {mantenimientos.map((m) => (
              <div key={m.id} className="border border-slate-100 rounded-xl p-4 bg-slate-50">
                {editingId === m.id ? (
                  <form onSubmit={(e) => handleUpdate(e, m.id)} className="space-y-3">
                    <div className="grid sm:grid-cols-2 gap-3">
                      <input name="realizado_por" defaultValue={m.realizado_por} placeholder="Técnico / Proveedor" required className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
                      <input name="costo" type="number" step="0.01" defaultValue={m.costo} placeholder="Costo" className="border border-slate-200 rounded-lg px-3 py-2 text-sm" />
                    </div>
                    <textarea name="descripcion" defaultValue={m.descripcion} placeholder="Descripción" required rows="3" className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm"></textarea>
                    <input name="url_pdf" defaultValue={m.url_pdf || ''} placeholder="URL del informe (https://... ) — deja vacío para borrar, o sube nuevo PDF" className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm" />
                    <div className="flex items-center gap-3">
                      <input name="pdf" type="file" accept="application/pdf" className="flex-1 text-sm" />
                      <span className="text-xs text-slate-400">Si subes PDF, reemplaza la URL</span>
                    </div>
                    <div className="flex gap-2">
                      <button type="submit" className="bg-emerald-600 hover:bg-emerald-500 text-white px-4 py-2 rounded-lg text-sm font-semibold flex-1">
                        Guardar
                      </button>
                      <button type="button" onClick={() => setEditingId(null)} className="bg-slate-200 hover:bg-slate-300 text-slate-700 px-4 py-2 rounded-lg text-sm font-semibold">
                        Cancelar
                      </button>
                    </div>
                  </form>
                ) : (
                  <>
                    <div className="flex justify-between items-start gap-3">
                      <div className="flex-1">
                        <p className="font-semibold text-sm text-slate-800">{m.realizado_por}</p>
                        <p className="text-sm text-slate-600 mt-1 whitespace-pre-wrap">{m.descripcion}</p>
                        <p className="text-xs text-slate-400 mt-2 inline-flex items-center gap-1"><Clock className="w-3 h-3" /> {m.fecha_hora} · <DollarSign className="w-3 h-3" /> {m.costo}</p>
                      </div>
                      <div className="flex flex-col gap-2 items-end shrink-0">
                        {m.url_pdf && (
                          <a
                            href={m.url_pdf}
                            target="_blank"
                            rel="noreferrer"
                            className="text-xs bg-slate-900 text-white px-3 py-1.5 rounded-full hover:bg-slate-700 text-center inline-flex items-center gap-1"
                          >
                            <FileText className="w-3 h-3" /> Ver informe
                          </a>
                        )}
                        <div className="flex gap-2">
                          <button onClick={() => setEditingId(m.id)} className="text-xs bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 px-3 py-1.5 rounded-full inline-flex items-center gap-1">
                            <Pencil className="w-3 h-3" /> Editar
                          </button>
                          <button onClick={() => handleDelete(m.id)} className="text-xs bg-rose-50 border border-rose-200 hover:bg-rose-100 text-rose-700 px-3 py-1.5 rounded-full inline-flex items-center gap-1">
                            <Trash2 className="w-3 h-3" /> Borrar
                          </button>
                        </div>
                      </div>
                    </div>
                    <div className="mt-3">
                      <EncryptedGallery entidadTipo="mantenimiento" entidadId={m.id} />
                    </div>
                  </>
                )}
              </div>
            ))}
          </div>
        )}

        <details className="mt-6 border border-slate-200 rounded-xl p-4 bg-white">
          <summary className="font-semibold text-sm cursor-pointer text-slate-700">+ Registrar mantenimiento</summary>
          <form onSubmit={handleMant} className="mt-4 space-y-3">
            <input name="realizado_por" placeholder="Técnico / Proveedor" required className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <textarea name="descripcion" placeholder="Descripción del trabajo" required rows="3" className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm"></textarea>
            <div className="flex gap-3">
              <input name="costo" type="number" step="0.01" placeholder="Costo" className="flex-1 border border-slate-200 rounded-lg px-3 py-2 text-sm" />
              <input name="pdf" type="file" accept="application/pdf" className="flex-1 text-sm" />
            </div>
            <input name="url_pdf" type="url" placeholder="O pega URL del informe (https://drive.google.com/... )" className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm" />
            <p className="text-xs text-slate-400">Puedes subir un PDF <em>o</em> pegar una URL. Si subes PDF tiene prioridad.</p>
            <button type="submit" className="bg-sky-600 hover:bg-sky-500 text-white px-5 py-2 rounded-lg text-sm font-semibold w-full">
              Guardar mantenimiento
            </button>
          </form>
        </details>
      </section>

      {dispositivo.estado !== 'dado_de_baja' && (
        <details className="bg-rose-50 border border-rose-200 p-5 rounded-2xl">
          <summary className="font-semibold text-sm text-rose-700 cursor-pointer inline-flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> Dar de baja este equipo</summary>
          <form onSubmit={handleBaja} className="mt-4 flex gap-3">
            <input name="motivo_baja" placeholder="Motivo (ej: obsoleto, daño irreparable)" required className="flex-1 border border-rose-200 rounded-lg px-3 py-2 text-sm" />
            <button type="submit" className="bg-rose-600 hover:bg-rose-500 text-white px-6 py-2 rounded-lg text-sm font-semibold">
              Confirmar baja
            </button>
          </form>
        </details>
      )}
    </div>
  );
}
