import React, { useEffect, useRef, useState } from 'react';

export default function Escaner({ onScan, onDeviceFound }) {
  const [scanning, setScanning] = useState(false);
  const [error, setError] = useState(null);
  const [manual, setManual] = useState('');
  const [lastResult, setLastResult] = useState(null);
  const scannerRef = useRef(null);
  const html5Ref = useRef(null);

  const startScan = async () => {
    setError(null);
    setScanning(true);
    // html5-qrcode carga dinámica para no romper SSR
    try {
      const { Html5Qrcode } = await import('html5-qrcode');
      const elId = 'qr-reader';
      // asegurar contenedor
      if (!document.getElementById(elId)) return;
      const html5Qr = new Html5Qrcode(elId);
      html5Ref.current = html5Qr;
      await html5Qr.start(
        { facingMode: 'environment' },
        { fps: 10, qrbox: { width: 250, height: 250 }, aspectRatio: 1.0 },
        async (decodedText) => {
          handleDetected(decodedText);
        },
        () => {}
      );
    } catch (e) {
      console.error(e);
      setError('No se pudo iniciar escáner: ' + (e.message || e));
      setScanning(false);
    }
  };

  const stopScan = async () => {
    try {
      if (html5Ref.current) {
        await html5Ref.current.stop();
        html5Ref.current.clear();
      }
    } catch {}
    setScanning(false);
  };

  useEffect(() => {
    return () => {
      stopScan();
    };
  }, []);

  const handleDetected = async (texto) => {
    const id = texto.trim().replace(/^.*\/([A-F0-9\-]+)$/i, '$1').replace(/^.*\/e\//, '').split('?')[0];
    // texto puede ser UUID, ET-XXXX o URL completa
    const uuid = id.includes('-') ? id : texto.trim();
    setLastResult(uuid);
    // vibrar si disponible
    if (navigator.vibrate) navigator.vibrate(120);
    await stopScan();
    if (onScan) onScan(uuid);
    if (onDeviceFound) {
      try {
        const res = await fetch(`/api/etiquetas/scan/${encodeURIComponent(uuid)}`);
        if (res.ok) {
          const data = await res.json();
          onDeviceFound(data);
        } else {
          onDeviceFound({ etiqueta: null, error: await res.text() });
        }
      } catch (e) {
        onDeviceFound({ error: e.message });
      }
    }
  };

  const handleManual = async (e) => {
    e.preventDefault();
    if (!manual.trim()) return;
    await handleDetected(manual.trim());
  };

  return (
    <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm space-y-4">
      <div className="flex justify-between items-center">
        <h3 className="font-bold text-slate-900">🔍 Escáner de Etiquetas</h3>
        {!scanning ? (
          <button onClick={startScan} className="bg-slate-900 text-white px-5 py-2 rounded-full text-sm font-medium hover:bg-slate-800">
            📷 Iniciar cámara
          </button>
        ) : (
          <button onClick={stopScan} className="bg-red-50 border border-red-200 text-red-700 px-5 py-2 rounded-full text-sm">
            ✕ Detener
          </button>
        )}
      </div>

      <div id="qr-reader" className={`${scanning ? 'block' : 'hidden'} w-full max-w-md mx-auto rounded-xl overflow-hidden border border-slate-200 bg-black`} />

      {!scanning && (
        <p className="text-xs text-slate-500 text-center">Apunta al QR de la etiqueta (UUID v4/v7 o ET-XXXX). Requiere HTTPS o localhost.</p>
      )}

      {error && <div className="bg-amber-50 border border-amber-200 text-amber-800 px-3 py-2 rounded-lg text-xs">⚠️ {error}</div>}

      {lastResult && (
        <div className="bg-emerald-50 border border-emerald-200 px-3 py-2 rounded-lg text-xs">
          Último escaneo: <span className="font-mono font-bold">{lastResult}</span> <button onClick={() => navigator.clipboard.writeText(lastResult)} className="ml-2 underline">Copiar</button>
        </div>
      )}

      <form onSubmit={handleManual} className="flex gap-2">
        <input value={manual} onChange={(e) => setManual(e.target.value)} placeholder="Pegar UUID o ET-XXXX manual" className="flex-1 border border-slate-200 rounded-full px-4 py-2 text-sm font-mono" />
        <button type="submit" className="bg-sky-600 text-white px-5 py-2 rounded-full text-sm hover:bg-sky-500">Buscar</button>
      </form>
    </div>
  );
}
