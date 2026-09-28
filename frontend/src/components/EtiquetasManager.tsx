import React, { useState, useEffect } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import Escaner from './Escaner';
import DeviceSearchCombobox from './DeviceSearchCombobox';
import { FileText, Link2, Printer, Tag } from 'lucide-react';

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
  const [pdfGenerando, setPdfGenerando] = useState<boolean>(false);
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

  const handleImprimirNavegador = (): void => {
    window.print();
  };

  // PDF con pdf-lib + qrcode (mismo stack que xeland314.github.io MapMaker.tsx:1156)
  // A4 595x842pt, margen 1cm = 28.35pt, cards cuadradas sin gap, QR solo negro
  const handleGenerarPDF = async (): Promise<void> => {
    if (!etiquetas.length) return alert('No hay etiquetas para exportar');
    setPdfGenerando(true);
    try {
      const { PDFDocument, StandardFonts, rgb } = await import('pdf-lib');
      const QRCode = (await import('qrcode')).default;

      const pdf = await PDFDocument.create();
      const font = await pdf.embedFont(StandardFonts.Helvetica);
      const fontBold = await pdf.embedFont(StandardFonts.HelveticaBold);

      // A4 en puntos (72 DPI)
      const pageW = 595.28;
      const pageH = 841.89;
      const margin = 28.35; // 1cm
      const cols = 4;
      const rows = 7; // 28 etiquetas por hoja (4x7), ~39mm alto c/u
      const cardW = (pageW - margin * 2) / cols; // ~134.6pt (~47.5mm)
      const cardH = (pageH - margin * 2) / rows; // ~112.2pt (~39.6mm)
      // Al límite: QR ~88pt (~31mm) aprovechando sobrante al máximo, misma 4x7
      const qrSize = Math.min(cardW - 8, cardH - 24); // ~88pt ~31mm límite
      const qrMarginInCard = 4;

      let page = pdf.addPage([pageW, pageH]);
      let col = 0;
      let row = 0;

      for (let idx = 0; idx < etiquetas.length; idx++) {
        const et = etiquetas[idx];
        if (row >= rows) {
          page = pdf.addPage([pageW, pageH]);
          col = 0;
          row = 0;
        }
        const x = margin + col * cardW;
        const y = pageH - margin - (row + 1) * cardH; // origen inferior

        // Borde cuadrangular negro, sin radius, solo tinta negra
        page.drawRectangle({
          x,
          y,
          width: cardW,
          height: cardH,
          borderWidth: 0.5,
          borderColor: rgb(0, 0, 0),
        });

        // QR solo negro sobre blanco, errorCorrection M, margin 1 (quiet zone)
        const qrDataUrl = await QRCode.toDataURL(et.id, {
          width: 400,
          margin: 1,
          errorCorrectionLevel: 'M',
          color: { dark: '#000000', light: '#FFFFFF' },
        } as unknown as Record<string, unknown>) as string;
        const qrBytes = Uint8Array.from(atob(qrDataUrl.split(',')[1]), (c) => c.charCodeAt(0));
        const qrImage = await pdf.embedPng(qrBytes);
        const qrX = x + (cardW - qrSize) / 2;
        const qrY = y + cardH - qrSize - 6; // tope 6pt, deja 16pt abajo para texto
        page.drawImage(qrImage, { x: qrX, y: qrY, width: qrSize, height: qrSize });

        // Texto solo ET y UUID, centrado, negro
        const etText = et.codigo_barras;
        const uuidText = et.id;
        const etWidth = fontBold.widthOfTextAtSize(etText, 7);
        page.drawText(etText, {
          x: x + (cardW - etWidth) / 2,
          y: qrY - 8,
          size: 7,
          font: fontBold,
          color: rgb(0, 0, 0),
        });
        const uuidFontSize = 5;
        const uuidW = font.widthOfTextAtSize(uuidText, uuidFontSize);
        let fs = uuidFontSize;
        let w = uuidW;
        while (w > cardW - 6 && fs > 4) {
          fs -= 0.2;
          w = font.widthOfTextAtSize(uuidText, fs);
        }
        page.drawText(uuidText, {
          x: x + (cardW - w) / 2,
          y: qrY - 16,
          size: fs,
          font,
          color: rgb(0, 0, 0),
        });

        col++;
        if (col >= cols) {
          col = 0;
          row++;
        }
      }

      const pdfBytes = await pdf.save();
      const blob = new Blob([pdfBytes as unknown as BlobPart], { type: 'application/pdf' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `etiquetas-${filtro || 'todas'}-${new Date().toISOString().slice(0, 10)}.pdf`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      console.error(e);
      alert('Error generando PDF: ' + (e as Error).message);
    } finally {
      setPdfGenerando(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="bg-white p-4 sm:p-6 rounded-2xl border border-slate-200 shadow-sm print:hidden">
        <h2 className="font-bold text-base sm:text-lg mb-2 inline-flex items-center gap-2"><Tag className="w-5 h-5" /> Generar Etiquetas UUID</h2>
        <p className="text-xs sm:text-sm text-slate-500 mb-4">Genera UUIDs sin asignar, imprímelos y pégalos. Luego escanea para asociar al item. v7 ordenable por tiempo. QR mínimo legible ~15mm (recomendado 20-25mm a 300 DPI).</p>
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
            <button type="button" onClick={handleImprimirNavegador} className="flex-1 sm:flex-none border border-slate-200 px-4 py-3 min-h-[44px] rounded-full text-sm hover:bg-slate-50 touch-manipulation inline-flex items-center justify-center gap-1.5">
              <Printer className="w-4 h-4" /> Print
            </button>
            <button type="button" onClick={handleGenerarPDF} disabled={pdfGenerando || !etiquetas.length} className="flex-1 sm:flex-none bg-black text-white px-4 py-3 min-h-[44px] rounded-full text-sm hover:bg-zinc-800 disabled:opacity-50 touch-manipulation">
              {pdfGenerando ? 'Generando PDF...' : <span className="inline-flex items-center gap-1"><FileText className="w-4 h-4" /> PDF 1cm</span>}
            </button>
          </div>
        </form>
        <p className="text-[11px] text-slate-400 mt-2">PDF: A4 1cm, 4×7 (28/hoja) borde 0.5pt, QR ~31mm al límite (antes 22mm). Mín 15mm M, rec. 20-25mm.</p>
      </div>

      <div className="flex gap-2 flex-wrap overflow-x-auto pb-1 -mx-4 px-4 sm:mx-0 sm:px-0 print:hidden">
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
        <div className="bg-white p-4 rounded-xl border border-slate-200 print:hidden">
          <h4 className="font-bold text-sm mb-2">Resultado escaneo</h4>
          <pre className="text-xs bg-slate-50 p-3 rounded-lg overflow-auto max-h-64">{JSON.stringify(scanned, null, 2)}</pre>
          {(scanned as { dispositivo?: { id: string; codigo_activo: string } }).dispositivo && (
            <a href={`/dispositivos/${(scanned as { dispositivo: { id: string } }).dispositivo.id}`} className="text-sky-600 underline text-sm mt-2 inline-block">
              Ver dispositivo {(scanned as { dispositivo: { codigo_activo: string } }).dispositivo.codigo_activo} →
            </a>
          )}
        </div>
      )}

      <div className="bg-white p-4 sm:p-5 rounded-2xl border border-slate-200 print:hidden">
        <h3 className="font-bold mb-3 text-sm sm:text-base inline-flex items-center gap-2"><Link2 className="w-4 h-4" /> Asociar etiqueta a dispositivo</h3>
        <form onSubmit={handleAsociar} className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <input value={asociarId} onChange={(e) => setAsociarId(e.target.value)} placeholder="UUID etiqueta (escaneado)" className="border border-slate-200 rounded-xl px-3 py-3 min-h-[44px] text-sm font-mono focus:border-sky-500 focus:outline-none" />
          <DeviceSearchCombobox value={dispositivoId} onChange={(id) => setDispositivoId(id)} placeholder="Buscar compu, disco, GGM-... o pegar UUID" label="" />
          <button type="submit" className="bg-sky-600 text-white rounded-full px-5 py-3 min-h-[44px] text-sm hover:bg-sky-500 touch-manipulation">
            Asociar
          </button>
        </form>
        <p className="text-xs text-slate-400 mt-2">Busca escribiendo “compu”, “disco”, “GGM-COM” o pega el UUID. También funciona con ID completo.</p>
      </div>

      {/* Vista pantalla */}
      <div className="bg-white p-4 sm:p-6 rounded-2xl border border-slate-200 print:hidden">
        <h3 className="font-bold mb-4 text-sm sm:text-base">Etiquetas {filtro || 'todas'} — vista previa (pantalla)</h3>
        {loading ? (
          <p className="text-sm text-slate-400">Cargando...</p>
        ) : etiquetas.length === 0 ? (
          <p className="text-sm text-slate-400">Sin etiquetas en este estado. Genera un lote.</p>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
            {etiquetas.map((e) => (
              <div key={e.id} className="border border-slate-200 rounded-xl p-3 sm:p-4 bg-white flex flex-col items-center text-center break-all">
                <QRCodeSVG value={e.id} size={112} bgColor="#FFFFFF" fgColor="#000000" level="M" includeMargin={false} className="w-28 h-28 sm:w-[112px] sm:h-[112px]" />
                <p className="font-mono text-[11px] mt-2 font-bold">{e.codigo_barras}</p>
                <p className="font-mono text-[9px] text-slate-500 break-all px-1">{e.id}</p>
                <div className="flex flex-wrap justify-center gap-1.5 mt-3">
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

      {/* Vista impresión: cuadrada, sin gaps, solo negro, UUID+ET */}
      <div className="hidden print:block">
        <div className="grid" style={{ gridTemplateColumns: 'repeat(4, 1fr)', gap: 0 }}>
          {etiquetas.map((e) => (
            <div key={e.id} className="border border-black flex flex-col items-center justify-center text-center p-2" style={{ aspectRatio: '1 / 1.18', borderWidth: '0.5pt' }}>
              <QRCodeSVG value={e.id} size={96} bgColor="#FFFFFF" fgColor="#000000" level="M" includeMargin={false} />
              <p style={{ fontFamily: 'monospace', fontSize: '7pt', fontWeight: 700, marginTop: '4pt' }}>{e.codigo_barras}</p>
              <p style={{ fontFamily: 'monospace', fontSize: '5pt', color: '#000', wordBreak: 'break-all', lineHeight: 1.1 }}>{e.id}</p>
            </div>
          ))}
        </div>
      </div>

      <style>{`@page { margin: 1cm; size: A4; } @media print { nav, footer { display: none !important; } .print\\:hidden { display: none !important; } }`}</style>
    </div>
  );
}
