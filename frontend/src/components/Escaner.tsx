import React, { useEffect, useRef, useState } from 'react';

interface EscanerProps {
  onScan?: (uuid: string) => void;
  onDeviceFound?: (data: Record<string, unknown>) => void;
}

export default function Escaner({ onScan, onDeviceFound }: EscanerProps): React.JSX.Element {
  const [scanning, setScanning] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [manual, setManual] = useState<string>('');
  const [lastResult, setLastResult] = useState<string | null>(null);
  const html5Ref = useRef<unknown>(null);

  const startScan = async (): Promise<void> => {
    setError(null);
    setScanning(true);
    try {
      const { Html5Qrcode } = await import('html5-qrcode') as { Html5Qrcode: new (id: string) => { start: (a: unknown, b: unknown, c: (t: string) => void, d: () => void) => Promise<void>; stop: () => Promise<void>; clear: () => void } };
      const elId = 'qr-reader';
      if (!document.getElementById(elId)) return;
      const html5Qr = new Html5Qrcode(elId);
      (html5Ref as React.MutableRefObject<unknown>).current = html5Qr;
      const isMobile = window.innerWidth < 640;
      await html5Qr.start(
        { facingMode: 'environment' },
        { fps: 10, qrbox: { width: isMobile ? 220 : 250, height: isMobile ? 220 : 250 }, aspectRatio: 1.0 },
        async (decodedText: string) => {
          await handleDetected(decodedText);
        },
        () => {}
      );
    } catch (e) {
      const err = e as Error;
      console.error(err);
      setError('No se pudo iniciar escáner: ' + (err.message || String(e)));
      setScanning(false);
    }
  };

  const stopScan = async (): Promise<void> => {
    try {
      const ref = html5Ref.current as { stop: () => Promise<void>; clear: () => void } | null;
      if (ref) {
        await ref.stop();
        ref.clear();
      }
    } catch {}
    setScanning(false);
  };

  useEffect(() => {
    return () => {
      void stopScan();
    };
  }, []);

  const handleDetected = async (texto: string): Promise<void> => {
    const id = texto.trim().replace(/^.*\/([A-F0-9\-]+)$/i, '$1').replace(/^.*\/e\//, '').split('?')[0];
    const uuid = id.includes('-') ? id : texto.trim();
    setLastResult(uuid);
    if (navigator.vibrate) navigator.vibrate(120);
    await stopScan();
    if (onScan) onScan(uuid);
    if (onDeviceFound) {
      try {
        const res = await fetch(`/api/etiquetas/scan/${encodeURIComponent(uuid)}`);
        if (res.ok) {
          const data = (await res.json()) as Record<string, unknown>;
          onDeviceFound(data);
        } else {
          onDeviceFound({ etiqueta: null, error: await res.text() });
        }
      } catch (e) {
        const err = e as Error;
        onDeviceFound({ error: err.message });
      }
    }
  };

  const handleManual = async (e: React.FormEvent): Promise<void> => {
    e.preventDefault();
    if (!manual.trim()) return;
    await handleDetected(manual.trim());
  };

  return (
    <div className="bg-white p-4 sm:p-5 rounded-2xl border border-slate-200 shadow-sm space-y-4">
      <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3">
        <h3 className="font-bold text-slate-900 text-base">🔍 Escáner de Etiquetas</h3>
        {!scanning ? (
          <button
            type="button"
            onClick={startScan}
            className="w-full sm:w-auto bg-slate-900 text-white px-5 py-3 min-h-[44px] rounded-full text-sm font-medium hover:bg-slate-800 touch-manipulation"
          >
            📷 Iniciar cámara
          </button>
        ) : (
          <button
            type="button"
            onClick={stopScan}
            className="w-full sm:w-auto bg-red-50 border border-red-200 text-red-700 px-5 py-3 min-h-[44px] rounded-full text-sm touch-manipulation"
          >
            ✕ Detener
          </button>
        )}
      </div>

      <div
        id="qr-reader"
        className={`${scanning ? 'block' : 'hidden'} w-full max-w-md mx-auto rounded-xl overflow-hidden border border-slate-200 bg-black min-h-[280px] sm:min-h-[300px]`}
      />

      {!scanning && (
        <p className="text-xs text-slate-500 text-center px-2">Apunta al QR de la etiqueta (UUID v4/v7 o ET-XXXX). Requiere HTTPS o localhost. En móvil, mantén el QR centrado y a 15-20 cm.</p>
      )}

      {error && <div className="bg-amber-50 border border-amber-200 text-amber-800 px-3 py-2 rounded-lg text-xs">⚠️ {error}</div>}

      {lastResult && (
        <div className="bg-emerald-50 border border-emerald-200 px-3 py-3 rounded-lg text-xs break-all">
          Último escaneo: <span className="font-mono font-bold">{lastResult}</span>{' '}
          <button type="button" onClick={() => navigator.clipboard.writeText(lastResult)} className="ml-2 underline text-sky-700">
            Copiar
          </button>
        </div>
      )}

      <form onSubmit={handleManual} className="flex flex-col sm:flex-row gap-2">
        <input
          value={manual}
          onChange={(e) => setManual(e.target.value)}
          placeholder="Pegar UUID o ET-XXXX manual"
          className="flex-1 border border-slate-200 rounded-xl sm:rounded-full px-4 py-3 min-h-[44px] text-sm font-mono focus:border-sky-500 focus:outline-none"
        />
        <button type="submit" className="bg-sky-600 text-white px-6 py-3 min-h-[44px] rounded-xl sm:rounded-full text-sm hover:bg-sky-500 touch-manipulation whitespace-nowrap">
          Buscar
        </button>
      </form>
    </div>
  );
}
