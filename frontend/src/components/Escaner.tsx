import React, { useEffect, useRef, useState } from 'react';
import {
  AlertCircle,
  AlertTriangle,
  Barcode as BarcodeIcon,
  Box,
  Camera,
  CheckCircle2,
  Clock,
  Copy,
  ExternalLink,
  Laptop,
  Maximize2,
  Printer,
  RefreshCw,
  ScanLine,
  Search,
  Tag,
  Trash2,
  Usb,
  Volume2,
  VolumeX,
  X,
  Zap,
} from 'lucide-react';
import { scannerSounds } from '../lib/scannerAudio';

interface EscanerProps {
  onScan?: (uuid: string) => void;
  onDeviceFound?: (data: Record<string, unknown>) => void;
}

interface ScanRecord {
  id: string;
  code: string;
  timestamp: string;
  duration_ms: number;
  source: string;
  is_scanner: boolean;
  matched_type: string;
  matched_entity?: Record<string, unknown>;
  product?: {
    code: string;
    format: string;
    name: string;
    category: string;
    price: number;
    stock: number;
    description: string;
    badge_color: string;
  };
}

interface DeviceHardwareInfo {
  connected: boolean;
  vendor_id: string;
  product_id: string;
  vendor_name: string;
  product_name: string;
  serial_number: string;
  device_node: string;
  is_readable: boolean;
  driver: string;
  operation_mode: string;
  evdev_active: boolean;
  web_wedge_active: boolean;
}

interface CatalogProduct {
  code: string;
  format: string;
  name: string;
  category: string;
  price: number;
  stock: number;
  description: string;
  badge_color: string;
}

export default function Escaner({ onScan, onDeviceFound }: EscanerProps): React.JSX.Element {
  const [activeTab, setActiveTab] = useState<'usb' | 'camera' | 'catalog'>('usb');
  const [manual, setManual] = useState<string>('');
  const [error, setError] = useState<string | null>(null);
  const [soundEnabled, setSoundEnabled] = useState<boolean>(scannerSounds.enabled);
  const [wsConnected, setWsConnected] = useState<boolean>(false);
  const [isScanningActive, setIsScanningActive] = useState<boolean>(false);

  // Hardware y estado
  const [hardwareInfo, setHardwareInfo] = useState<DeviceHardwareInfo | null>(null);
  const [scans, setScans] = useState<ScanRecord[]>([]);
  const [latestScan, setLatestScan] = useState<ScanRecord | null>(null);

  // Catálogo demo de prueba
  const [catalog, setCatalog] = useState<CatalogProduct[]>([]);
  const [zoomModal, setZoomModal] = useState<{ code: string; format: string; name: string } | null>(null);

  // Cámara html5-qrcode
  const [cameraScanning, setCameraScanning] = useState<boolean>(false);
  const html5Ref = useRef<unknown>(null);

  // Wedge keyboard buffers
  const wedgeBufferRef = useRef<string[]>([]);
  const lastKeyTimeRef = useRef<number>(0);
  const keyIntervalsRef = useRef<number[]>([]);
  const wsRef = useRef<WebSocket | null>(null);

  // 1. Cargar info de hardware e historial inicial
  const refreshHardware = async () => {
    try {
      const res = await fetch('/api/scanner/device');
      if (res.ok) {
        const data = (await res.json()) as DeviceHardwareInfo;
        setHardwareInfo(data);
      }
    } catch (err) {
      console.warn('Error obteniendo estado de hardware:', err);
    }
  };

  const loadHistory = async () => {
    try {
      const res = await fetch('/api/scanner/scans');
      if (res.ok) {
        const data = (await res.json()) as ScanRecord[];
        setScans(data);
        if (data.length > 0 && !latestScan) {
          setLatestScan(data[0]);
        }
      }
    } catch {}
  };

  const loadCatalog = async () => {
    try {
      const res = await fetch('/api/scanner/catalog');
      if (res.ok) {
        const data = (await res.json()) as CatalogProduct[];
        setCatalog(data);
      }
    } catch {}
  };

  // 2. Conectar WebSocket
  useEffect(() => {
    void refreshHardware();
    void loadHistory();
    void loadCatalog();

    let reconnectTimer: NodeJS.Timeout;
    const connectWS = () => {
      const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${proto}//${window.location.host}/ws`;

      try {
        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.onopen = () => {
          setWsConnected(true);
        };

        ws.onmessage = (event) => {
          try {
            const msg = JSON.parse(event.data);
            if (msg.type === 'scan_event' && msg.data) {
              const rec = msg.data as ScanRecord;
              handleIncomingScan(rec);
            } else if (msg.type === 'history_cleared') {
              setScans([]);
              setLatestScan(null);
            } else if (msg.type === 'init' && msg.device) {
              setHardwareInfo(msg.device);
              if (msg.scans && Array.isArray(msg.scans)) {
                setScans(msg.scans);
                if (msg.scans.length > 0) setLatestScan(msg.scans[0]);
              }
            }
          } catch {}
        };

        ws.onclose = () => {
          setWsConnected(false);
          reconnectTimer = setTimeout(connectWS, 2500);
        };

        ws.onerror = () => {
          setWsConnected(false);
        };
      } catch {
        setWsConnected(false);
      }
    };

    connectWS();

    return () => {
      clearTimeout(reconnectTimer);
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  // 3. Manejador unificado cuando llega un escaneo (por WS, wedge o API)
  const handleIncomingScan = (rec: ScanRecord) => {
    setLatestScan(rec);
    setScans((prev) => {
      const exists = prev.some((s) => s.id === rec.id);
      if (exists) return prev;
      return [rec, ...prev.slice(0, 99)];
    });

    // Efecto visual flash en el target
    setIsScanningActive(true);
    setTimeout(() => setIsScanningActive(false), 500);

    // Sonido
    if (rec.matched_type === 'dispositivo' || rec.matched_type === 'conjunto' || rec.matched_type === 'frasco' || rec.product) {
      scannerSounds.playMatch();
    } else {
      scannerSounds.playSuccess();
    }

    // Vibración táctil si el dispositivo la soporta
    if (navigator.vibrate) navigator.vibrate(80);

    // Despachar eventos
    window.dispatchEvent(new CustomEvent('escaner:scan', { detail: { uuid: rec.code, record: rec } }));
    if (onScan) onScan(rec.code);
    if (onDeviceFound && rec.matched_entity) {
      onDeviceFound(rec.matched_entity);
    }
  };

  // 4. Enviar escaneo a la API
  const submitScan = async (code: string, durationMs: number, source: string) => {
    const clean = code.trim();
    if (!clean) return;

    try {
      const res = await fetch('/api/scanner/scan', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          code: clean,
          duration_ms: durationMs,
          source: source,
        }),
      });

      if (res.ok) {
        const json = await res.json();
        if (json.data) {
          handleIncomingScan(json.data as ScanRecord);
        }
      }
    } catch (e) {
      const err = e as Error;
      setError('Error al procesar escaneo: ' + err.message);
    }
  };

  // 5. Global Keyboard Wedge Listener
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Si el foco está en un input normal (que no sea el de escáner manual),
      // solo capturamos si se detecta una ráfaga muy rápida de escáner (<45ms)
      const target = e.target as HTMLElement | null;
      const isInput = target && ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName);
      const isOurManualInput = target?.id === 'scanner-manual-input';

      if (isInput && !isOurManualInput && e.key !== 'Enter') {
        const now = performance.now();
        const interval = now - lastKeyTimeRef.current;
        // Si no es ráfaga ultra-rápida, dejar que el usuario escriba normalmente
        if (interval > 45 && wedgeBufferRef.current.length === 0) {
          return;
        }
      }

      const now = performance.now();
      const timeSinceLastKey = now - lastKeyTimeRef.current;

      // Si pasaron más de 350ms sin teclas, reiniciar buffer
      if (timeSinceLastKey > 350) {
        wedgeBufferRef.current = [];
        keyIntervalsRef.current = [];
      }

      if (e.key === 'Enter') {
        if (wedgeBufferRef.current.length > 0) {
          e.preventDefault();
          const code = wedgeBufferRef.current.join('');
          const totalDuration = keyIntervalsRef.current.reduce((a, b) => a + b, 0);
          const avgInterval = keyIntervalsRef.current.length > 0 ? totalDuration / keyIntervalsRef.current.length : 25;
          const isScannerSpeed = avgInterval < 55;

          void submitScan(
            code,
            Math.round(totalDuration || 25),
            isScannerSpeed ? 'usb_scanner_wedge' : 'manual_keyboard'
          );

          wedgeBufferRef.current = [];
          keyIntervalsRef.current = [];
        }
        return;
      }

      // Capturar solo caracteres imprimibles
      if (e.key.length === 1) {
        if (wedgeBufferRef.current.length > 0) {
          keyIntervalsRef.current.push(timeSinceLastKey);
        }
        wedgeBufferRef.current.push(e.key);
        lastKeyTimeRef.current = now;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  // 6. Cámara html5-qrcode
  const startCamera = async (): Promise<void> => {
    setError(null);
    setCameraScanning(true);
    try {
      const { Html5Qrcode } = (await import('html5-qrcode')) as {
        Html5Qrcode: new (id: string) => {
          start: (a: unknown, b: unknown, c: (t: string) => void, d: () => void) => Promise<void>;
          stop: () => Promise<void>;
          clear: () => void;
        };
      };
      const elId = 'qr-reader';
      if (!document.getElementById(elId)) return;
      const html5Qr = new Html5Qrcode(elId);
      (html5Ref as React.MutableRefObject<unknown>).current = html5Qr;
      const isMobile = window.innerWidth < 640;
      await html5Qr.start(
        { facingMode: 'environment' },
        { fps: 10, qrbox: { width: isMobile ? 220 : 250, height: isMobile ? 220 : 250 }, aspectRatio: 1.0 },
        async (decodedText: string) => {
          await stopCamera();
          await submitScan(decodedText, 45, 'camera');
        },
        () => {}
      );
    } catch (e) {
      const err = e as Error;
      setError('No se pudo iniciar cámara: ' + (err.message || String(e)));
      setCameraScanning(false);
    }
  };

  const stopCamera = async (): Promise<void> => {
    try {
      const ref = html5Ref.current as { stop: () => Promise<void>; clear: () => void } | null;
      if (ref) {
        await ref.stop();
        ref.clear();
      }
    } catch {}
    setCameraScanning(false);
  };

  useEffect(() => {
    return () => {
      void stopCamera();
    };
  }, []);

  // Limpiar historial
  const handleClearHistory = async () => {
    try {
      await fetch('/api/scanner/scans', { method: 'DELETE' });
      setScans([]);
      setLatestScan(null);
    } catch {}
  };

  const handleManualSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!manual.trim()) return;
    void submitScan(manual.trim(), 10, 'manual');
    setManual('');
  };

  const copyToClipboard = (text: string) => {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(text);
    }
  };

  return (
    <div className="space-y-6">
      {/* 1. Barra de Estado de Hardware y Conexión */}
      <div className="bg-slate-900 text-white rounded-2xl p-4 sm:p-5 shadow-lg border border-slate-800 flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex items-center gap-3.5">
          <div
            className={`w-11 h-11 rounded-xl flex items-center justify-center shrink-0 ${
              hardwareInfo?.connected ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30' : 'bg-slate-800 text-slate-400'
            }`}
          >
            <Usb className="w-6 h-6" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-semibold text-sm sm:text-base">
                {hardwareInfo?.connected ? (
                  <span className="text-emerald-400 inline-flex items-center gap-1.5">
                    <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
                    {hardwareInfo.product_name} ({hardwareInfo.vendor_id}:{hardwareInfo.product_id})
                  </span>
                ) : (
                  <span className="text-slate-300">Escáner Físico USB Listo (Modo Wedge Web)</span>
                )}
              </span>
              <span
                className={`text-[11px] font-mono px-2 py-0.5 rounded-full ${
                  wsConnected ? 'bg-emerald-950 text-emerald-300 border border-emerald-800' : 'bg-amber-950 text-amber-300 border border-amber-800'
                }`}
                title={wsConnected ? 'Canal en tiempo real activo' : 'Reconectando canal WebSocket'}
              >
                {wsConnected ? 'En vivo' : 'Offline'}
              </span>
            </div>
            <p className="text-xs text-slate-400 mt-0.5">
              {hardwareInfo?.connected
                ? `Nodo: ${hardwareInfo.device_node || '/dev/input/event13'} • Driver: ${hardwareInfo.driver} • evdev ${
                    hardwareInfo.evdev_active ? 'activo' : 'wedge'
                  }`
                : 'Apunta tu lector de código de barras a la pantalla o presiona el gatillo en cualquier momento.'}
            </p>
          </div>
        </div>

        {/* Acciones de la barra: sonido y recargar */}
        <div className="flex items-center gap-2 self-end md:self-auto">
          <button
            type="button"
            onClick={() => setSoundEnabled(scannerSounds.toggle())}
            className={`px-3 py-2 rounded-xl text-xs font-medium inline-flex items-center gap-2 transition-colors ${
              soundEnabled
                ? 'bg-sky-500/20 text-sky-300 border border-sky-500/30 hover:bg-sky-500/30'
                : 'bg-slate-800 text-slate-400 border border-slate-700 hover:bg-slate-700'
            }`}
            title="Activar o desactivar sonido de confirmación POS (1950 Hz)"
          >
            {soundEnabled ? <Volume2 className="w-4 h-4 text-sky-400" /> : <VolumeX className="w-4 h-4" />}
            <span>Sonido POS {soundEnabled ? 'ON' : 'OFF'}</span>
          </button>
          <button
            type="button"
            onClick={() => refreshHardware()}
            className="p-2 bg-slate-800 text-slate-300 hover:text-white rounded-xl border border-slate-700 hover:bg-slate-700 text-xs"
            title="Actualizar estado de conexión USB"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* 2. Selector de Pestañas */}
      <div className="flex border-b border-slate-200 gap-2">
        <button
          type="button"
          onClick={() => {
            setActiveTab('usb');
            void stopCamera();
          }}
          className={`px-4 py-2.5 text-sm font-medium border-b-2 inline-flex items-center gap-2 transition-all ${
            activeTab === 'usb'
              ? 'border-sky-600 text-sky-600 font-semibold'
              : 'border-transparent text-slate-500 hover:text-slate-800 hover:border-slate-300'
          }`}
        >
          <ScanLine className="w-4 h-4" /> Escáner USB / Láser
        </button>
        <button
          type="button"
          onClick={() => setActiveTab('camera')}
          className={`px-4 py-2.5 text-sm font-medium border-b-2 inline-flex items-center gap-2 transition-all ${
            activeTab === 'camera'
              ? 'border-sky-600 text-sky-600 font-semibold'
              : 'border-transparent text-slate-500 hover:text-slate-800 hover:border-slate-300'
          }`}
        >
          <Camera className="w-4 h-4" /> Cámara Móvil
        </button>
        <button
          type="button"
          onClick={() => {
            setActiveTab('catalog');
            void stopCamera();
          }}
          className={`px-4 py-2.5 text-sm font-medium border-b-2 inline-flex items-center gap-2 transition-all ${
            activeTab === 'catalog'
              ? 'border-sky-600 text-sky-600 font-semibold'
              : 'border-transparent text-slate-500 hover:text-slate-800 hover:border-slate-300'
          }`}
        >
          <BarcodeIcon className="w-4 h-4" /> Catálogo de Pruebas
        </button>
      </div>

      {error && (
        <div className="bg-amber-50 border border-amber-200 text-amber-800 p-4 rounded-xl text-xs flex items-center justify-between">
          <div className="flex items-center gap-2">
            <AlertTriangle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
          <button type="button" onClick={() => setError(null)} className="text-amber-800 hover:opacity-80">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* 3. Contenido de Pestaña USB / Láser */}
      {activeTab === 'usb' && (
        <div className="space-y-6">
          {/* Caja Objetivo de Escaneo */}
          <div
            className={`relative overflow-hidden rounded-2xl p-6 sm:p-8 text-center border-2 transition-all duration-300 ${
              isScanningActive
                ? 'bg-sky-500/10 border-sky-500 shadow-xl scale-[1.01]'
                : 'bg-gradient-to-b from-slate-50 to-white border-slate-200/80 shadow-sm'
            }`}
          >
            {/* Animación del haz láser de fondo */}
            <div className="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-transparent via-red-500 to-transparent opacity-75 animate-pulse pointer-events-none"></div>

            <div className="max-w-md mx-auto space-y-3">
              <div className="w-14 h-14 mx-auto rounded-2xl bg-sky-50 text-sky-600 border border-sky-100 flex items-center justify-center">
                <ScanLine className={`w-7 h-7 ${isScanningActive ? 'animate-bounce text-sky-500' : ''}`} />
              </div>
              <h2 className="text-lg font-bold text-slate-900">Listo para escanear</h2>
              <p className="text-xs sm:text-sm text-slate-500">
                Apunta tu lector láser Symbol a cualquier código de barras físico o pantalla y presiona el gatillo.
                No es necesario hacer clic en ningún campo.
              </p>

              {/* Formulario para ingreso manual o con pistola configurada */}
              <form onSubmit={handleManualSubmit} className="pt-2 flex gap-2">
                <input
                  id="scanner-manual-input"
                  type="text"
                  value={manual}
                  onChange={(e) => setManual(e.target.value)}
                  placeholder="Pegar o escribir código manual (GGM-..., CJ-..., ET-...)"
                  className="flex-1 bg-white border border-slate-200 rounded-xl px-4 py-2.5 text-sm font-mono focus:border-sky-500 focus:outline-none shadow-inner"
                />
                <button
                  type="submit"
                  className="bg-slate-900 hover:bg-slate-800 text-white px-5 py-2.5 rounded-xl text-sm font-medium transition-colors inline-flex items-center gap-1.5"
                >
                  <Search className="w-4 h-4" /> Buscar
                </button>
              </form>
            </div>
          </div>

          {/* Tarjeta de Último Escaneo Detectado */}
          {latestScan && (
            <div className="bg-white rounded-2xl border border-slate-200 p-5 shadow-sm space-y-4">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-slate-100 pb-3">
                <div className="flex items-center gap-2">
                  <span className="text-xs uppercase tracking-wider font-bold text-slate-400">Último Código Leído</span>
                  <span
                    className={`text-[11px] px-2 py-0.5 rounded-full font-medium ${
                      latestScan.is_scanner
                        ? 'bg-emerald-100 text-emerald-800'
                        : 'bg-slate-100 text-slate-700'
                    }`}
                  >
                    {latestScan.is_scanner ? 'Escáner Óptico Hardware' : 'Entrada Manual'}
                  </span>
                </div>
                <div className="text-xs text-slate-400 flex items-center gap-3">
                  <span className="inline-flex items-center gap-1">
                    <Clock className="w-3.5 h-3.5" /> {latestScan.timestamp}
                  </span>
                  <span className="inline-flex items-center gap-1">
                    <Zap className="w-3.5 h-3.5 text-amber-500" /> {latestScan.duration_ms} ms
                  </span>
                </div>
              </div>

              {/* Código grande con botón copiar */}
              <div className="flex items-center justify-between bg-slate-50 border border-slate-200 rounded-xl px-4 py-3">
                <span className="text-xl sm:text-2xl font-mono font-bold text-slate-900 break-all">
                  {latestScan.code}
                </span>
                <button
                  type="button"
                  onClick={() => copyToClipboard(latestScan.code)}
                  className="ml-3 p-2 text-slate-500 hover:text-slate-900 hover:bg-white rounded-lg transition-colors shrink-0"
                  title="Copiar código"
                >
                  <Copy className="w-4 h-4" />
                </button>
              </div>

              {/* Entidad Encontrada / Coincidencia de Inventario */}
              {latestScan.matched_type === 'dispositivo' && latestScan.matched_entity && (
                <div className="bg-emerald-50/60 border border-emerald-200 rounded-xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div className="space-y-1">
                    <div className="inline-flex items-center gap-1.5 text-emerald-700 font-bold text-sm">
                      <CheckCircle2 className="w-4 h-4" /> Dispositivo Encontrado
                    </div>
                    <div className="font-semibold text-slate-900 text-base">
                      {String(latestScan.matched_entity.marca || '')} {String(latestScan.matched_entity.modelo || '')}
                    </div>
                    <div className="text-xs text-slate-600 font-mono flex flex-wrap gap-x-3 gap-y-1">
                      <span>Activo: {String(latestScan.matched_entity.codigo_activo || '')}</span>
                      {latestScan.matched_entity.numero_serie && <span>Serie: {String(latestScan.matched_entity.numero_serie)}</span>}
                      <span>Estado: {String(latestScan.matched_entity.estado || '')}</span>
                      {latestScan.matched_entity.sede && <span>Sede: {String(latestScan.matched_entity.sede)}</span>}
                    </div>
                  </div>
                  <a
                    href={`/dispositivos/${latestScan.matched_entity.id}`}
                    className="bg-emerald-600 hover:bg-emerald-500 text-white px-4 py-2.5 rounded-xl text-xs font-semibold inline-flex items-center justify-center gap-1.5 shrink-0 transition-colors"
                  >
                    Ver Ficha Completa <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                </div>
              )}

              {latestScan.matched_type === 'conjunto' && latestScan.matched_entity && (
                <div className="bg-sky-50/60 border border-sky-200 rounded-xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div className="space-y-1">
                    <div className="inline-flex items-center gap-1.5 text-sky-700 font-bold text-sm">
                      <Box className="w-4 h-4" /> Conjunto (Caja de Stock a Granel)
                    </div>
                    <div className="font-semibold text-slate-900 text-base">
                      {String(latestScan.matched_entity.nombre || '')}
                    </div>
                    <div className="text-xs text-slate-600 font-mono flex flex-wrap gap-x-3 gap-y-1">
                      <span>Código: {String(latestScan.matched_entity.codigo || '')}</span>
                      <span>
                        Stock: {String(latestScan.matched_entity.cantidad_actual || 0)} {String(latestScan.matched_entity.unidad || '')}
                      </span>
                      {latestScan.matched_entity.ubicacion && <span>Ubicación: {String(latestScan.matched_entity.ubicacion)}</span>}
                    </div>
                  </div>
                  <a
                    href="/conjuntos"
                    className="bg-sky-600 hover:bg-sky-500 text-white px-4 py-2.5 rounded-xl text-xs font-semibold inline-flex items-center justify-center gap-1.5 shrink-0 transition-colors"
                  >
                    Abrir Conjuntos <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                </div>
              )}

              {latestScan.matched_type === 'frasco' && latestScan.matched_entity && (
                <div className="bg-purple-50/60 border border-purple-200 rounded-xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div className="space-y-1">
                    <div className="inline-flex items-center gap-1.5 text-purple-700 font-bold text-sm">
                      <CheckCircle2 className="w-4 h-4" /> Frasco de Tinta
                    </div>
                    <div className="font-semibold text-slate-900 text-base">
                      Tinta {String(latestScan.matched_entity.color || '')} {String(latestScan.matched_entity.marca || '')}
                    </div>
                    <div className="text-xs text-slate-600 font-mono">
                      Código: {String(latestScan.matched_entity.codigo || '')} • Cantidad: {String(latestScan.matched_entity.cantidad_actual ?? 0)}
                    </div>
                  </div>
                  <a
                    href="/frascos"
                    className="bg-purple-600 hover:bg-purple-500 text-white px-4 py-2.5 rounded-xl text-xs font-semibold inline-flex items-center justify-center gap-1.5 shrink-0 transition-colors"
                  >
                    Ver Frascos <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                </div>
              )}

              {latestScan.matched_type === 'etiqueta' && latestScan.matched_entity && (
                <div className="bg-amber-50/60 border border-amber-200 rounded-xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div className="space-y-1">
                    <div className="inline-flex items-center gap-1.5 text-amber-700 font-bold text-sm">
                      <Tag className="w-4 h-4" /> Etiqueta Disponible sin Asignar
                    </div>
                    <div className="font-mono text-sm text-slate-800">
                      ID: {String(latestScan.matched_entity.id || '')}
                    </div>
                    <div className="text-xs text-slate-600">
                      Estado: {String(latestScan.matched_entity.estado || 'disponible')}
                    </div>
                  </div>
                  <a
                    href="/etiquetas"
                    className="bg-amber-600 hover:bg-amber-500 text-white px-4 py-2.5 rounded-xl text-xs font-semibold inline-flex items-center justify-center gap-1.5 shrink-0 transition-colors"
                  >
                    Asociar Etiqueta <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                </div>
              )}

              {latestScan.product && (
                <div className="bg-indigo-50/60 border border-indigo-200 rounded-xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div className="space-y-1">
                    <div className="inline-flex items-center gap-1.5 text-indigo-700 font-bold text-sm">
                      <CheckCircle2 className="w-4 h-4" /> Producto del Catálogo Demo
                    </div>
                    <div className="font-semibold text-slate-900 text-base">{latestScan.product.name}</div>
                    <div className="text-xs text-slate-600 font-mono">
                      Categoría: {latestScan.product.category} • Formato: {latestScan.product.format.toUpperCase()} • Stock: {latestScan.product.stock}
                    </div>
                    <p className="text-xs text-slate-500">{latestScan.product.description}</p>
                  </div>
                  <div className="text-right shrink-0">
                    <span className="text-lg font-bold text-slate-900">${latestScan.product.price.toFixed(2)}</span>
                  </div>
                </div>
              )}

              {latestScan.matched_type === 'unknown' && !latestScan.product && (
                <div className="bg-slate-50 border border-slate-200 rounded-xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div className="space-y-1">
                    <div className="inline-flex items-center gap-1.5 text-slate-600 font-bold text-sm">
                      <AlertCircle className="w-4 h-4 text-sky-600" /> Código Decodificado Exitosamente
                    </div>
                    <p className="text-xs text-slate-500">
                      Longitud: {latestScan.code.length} caracteres. El escáner Symbol capturó el código nítidamente, pero no está registrado aún en la base de datos de producción.
                    </p>
                  </div>
                  <div className="flex gap-2">
                    <a
                      href="/dispositivos/nuevo"
                      className="bg-slate-900 hover:bg-slate-800 text-white px-3 py-2 rounded-lg text-xs font-medium"
                    >
                      Nuevo Dispositivo
                    </a>
                    <a
                      href="/etiquetas"
                      className="bg-white border border-slate-300 hover:bg-slate-50 text-slate-700 px-3 py-2 rounded-lg text-xs font-medium"
                    >
                      Crear Etiqueta
                    </a>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Historial de Escaneos de la Sesión */}
          <div className="bg-white rounded-2xl border border-slate-200 overflow-hidden shadow-sm">
            <div className="px-5 py-4 border-b border-slate-100 flex items-center justify-between">
              <div>
                <h3 className="font-bold text-slate-900 text-sm sm:text-base">Historial de Lectura de la Sesión</h3>
                <p className="text-xs text-slate-500">Escaneos capturados en tiempo real en esta estación.</p>
              </div>
              {scans.length > 0 && (
                <button
                  type="button"
                  onClick={handleClearHistory}
                  className="text-xs text-red-600 hover:text-red-700 font-medium inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-red-200 hover:bg-red-50 transition-colors"
                >
                  <Trash2 className="w-3.5 h-3.5" /> Limpiar
                </button>
              )}
            </div>

            {scans.length === 0 ? (
              <div className="text-center py-12 px-4 text-slate-400 space-y-2">
                <BarcodeIcon className="w-10 h-10 mx-auto opacity-40" />
                <p className="text-sm font-medium">No hay lecturas registradas en esta sesión.</p>
                <p className="text-xs text-slate-400">
                  Usa tu escáner láser Symbol o escribe un código para comenzar.
                </p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs sm:text-sm">
                  <thead className="bg-slate-50/75 text-slate-500 border-b border-slate-200">
                    <tr>
                      <th className="py-2.5 px-4 font-semibold">Hora</th>
                      <th className="py-2.5 px-4 font-semibold font-mono">Código</th>
                      <th className="py-2.5 px-4 font-semibold">Coincidencia</th>
                      <th className="py-2.5 px-4 font-semibold">Tiempo</th>
                      <th className="py-2.5 px-4 font-semibold">Origen</th>
                      <th className="py-2.5 px-4 font-semibold text-right">Acción</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {scans.map((s) => (
                      <tr key={s.id} className="hover:bg-slate-50/50 transition-colors">
                        <td className="py-3 px-4 font-mono text-slate-500 text-xs">{s.timestamp}</td>
                        <td className="py-3 px-4 font-mono font-semibold text-slate-900">{s.code}</td>
                        <td className="py-3 px-4">
                          {s.matched_type === 'dispositivo' && (
                            <span className="text-emerald-700 font-semibold inline-flex items-center gap-1">
                              <Laptop className="w-3.5 h-3.5" /> Dispositivo
                            </span>
                          )}
                          {s.matched_type === 'conjunto' && (
                            <span className="text-sky-700 font-semibold inline-flex items-center gap-1">
                              <Box className="w-3.5 h-3.5" /> Conjunto Caja
                            </span>
                          )}
                          {s.matched_type === 'frasco' && (
                            <span className="text-purple-700 font-semibold">Frasco Tinta</span>
                          )}
                          {s.matched_type === 'etiqueta' && (
                            <span className="text-amber-700 font-semibold">Etiqueta Pool</span>
                          )}
                          {s.product && (
                            <span className="text-indigo-700 font-medium">{s.product.name}</span>
                          )}
                          {s.matched_type === 'unknown' && !s.product && (
                            <span className="text-slate-400 italic">No registrado</span>
                          )}
                        </td>
                        <td className="py-3 px-4 text-xs font-mono text-slate-500">{s.duration_ms} ms</td>
                        <td className="py-3 px-4">
                          <span
                            className={`text-[11px] px-2 py-0.5 rounded-full font-medium ${
                              s.is_scanner
                                ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                                : 'bg-slate-100 text-slate-600'
                            }`}
                          >
                            {s.is_scanner ? 'Láser Óptico' : 'Manual'}
                          </span>
                        </td>
                        <td className="py-3 px-4 text-right">
                          <button
                            type="button"
                            onClick={() => copyToClipboard(s.code)}
                            className="text-slate-500 hover:text-slate-900 p-1"
                            title="Copiar código"
                          >
                            <Copy className="w-3.5 h-3.5" />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}

      {/* 4. Contenido de Pestaña Cámara Móvil */}
      {activeTab === 'camera' && (
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm space-y-4">
          <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3">
            <div>
              <h3 className="font-bold text-slate-900 text-base inline-flex items-center gap-2">
                <Camera className="w-5 h-5 text-sky-600" /> Escáner por Cámara
              </h3>
              <p className="text-xs text-slate-500">Útil en teléfonos inteligentes o laptops sin lector USB.</p>
            </div>
            {!cameraScanning ? (
              <button
                type="button"
                onClick={startCamera}
                className="bg-slate-900 text-white px-5 py-2.5 rounded-full text-sm font-medium hover:bg-slate-800 inline-flex items-center justify-center gap-2"
              >
                <Camera className="w-4 h-4" /> Iniciar Cámara
              </button>
            ) : (
              <button
                type="button"
                onClick={stopCamera}
                className="bg-red-50 border border-red-200 text-red-700 px-5 py-2.5 rounded-full text-sm hover:bg-red-100 inline-flex items-center justify-center gap-2"
              >
                <X className="w-4 h-4" /> Detener Cámara
              </button>
            )}
          </div>

          <div
            id="qr-reader"
            className={`${cameraScanning ? 'block' : 'hidden'} w-full max-w-md mx-auto rounded-xl overflow-hidden border border-slate-200 bg-black min-h-[280px] sm:min-h-[300px]`}
          />

          {!cameraScanning && (
            <div className="text-center py-10 bg-slate-50 rounded-xl border border-dashed border-slate-300">
              <Camera className="w-10 h-10 mx-auto text-slate-400 mb-2" />
              <p className="text-xs text-slate-500">Presiona "Iniciar Cámara" para activar el lector web.</p>
            </div>
          )}
        </div>
      )}

      {/* 5. Contenido de Catálogo de Pruebas */}
      {activeTab === 'catalog' && (
        <div className="space-y-6">
          <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
            <div>
              <h3 className="font-bold text-slate-900 text-base">Catálogo de Pruebas y Calibración</h3>
              <p className="text-xs text-slate-500">
                Códigos de prueba en formatos estándar (EAN-13, Code 128, Code 39, UPC-A, QR). Haz clic en "Ampliar" para escanearlos directamente desde la pantalla.
              </p>
            </div>
            <a
              href="/escaner/imprimible"
              target="_blank"
              rel="noreferrer"
              className="bg-sky-600 hover:bg-sky-500 text-white px-5 py-2.5 rounded-xl text-xs font-semibold inline-flex items-center gap-2 shrink-0 transition-colors shadow-sm"
            >
              <Printer className="w-4 h-4" /> Hoja Imprimible (Ctrl + P)
            </a>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {catalog.map((item) => (
              <div
                key={item.code}
                className="bg-white border border-slate-200 rounded-2xl p-4 shadow-sm flex flex-col justify-between hover:border-sky-300 transition-colors"
              >
                <div>
                  <div className="flex items-center justify-between gap-2 mb-2">
                    <span className="text-[11px] font-mono px-2 py-0.5 rounded-full bg-slate-100 text-slate-700 font-semibold uppercase">
                      {item.format}
                    </span>
                    <span className="text-xs font-medium text-slate-400">{item.category}</span>
                  </div>
                  <h4 className="font-bold text-slate-900 text-sm mb-1">{item.name}</h4>
                  <p className="text-xs text-slate-500 mb-3 line-clamp-2">{item.description}</p>
                </div>

                <div className="space-y-3">
                  {/* Imagen SVG del código de barras */}
                  <div className="bg-white border border-slate-100 rounded-xl p-3 flex items-center justify-center min-h-[90px]">
                    <img
                      src={`/api/barcode/${item.format}/${encodeURIComponent(item.code)}.svg`}
                      alt={item.name}
                      className="max-h-20 w-auto object-contain"
                      loading="lazy"
                    />
                  </div>

                  <div className="flex items-center justify-between pt-1">
                    <span className="font-mono text-xs font-bold text-slate-700">{item.code}</span>
                    <button
                      type="button"
                      onClick={() => setZoomModal({ code: item.code, format: item.format, name: item.name })}
                      className="bg-slate-100 hover:bg-slate-200 text-slate-700 px-3 py-1.5 rounded-lg text-xs font-medium inline-flex items-center gap-1 transition-colors"
                    >
                      <Maximize2 className="w-3.5 h-3.5" /> Ampliar
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Modal de Zoom en Pantalla para Escaneo Óptico Cómodo */}
      {zoomModal && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-white rounded-3xl p-6 sm:p-8 max-w-lg w-full text-center space-y-4 shadow-2xl relative">
            <button
              type="button"
              onClick={() => setZoomModal(null)}
              className="absolute top-4 right-4 p-2 text-slate-400 hover:text-slate-700 rounded-full hover:bg-slate-100"
            >
              <X className="w-5 h-5" />
            </button>

            <div>
              <span className="text-xs font-mono font-bold uppercase tracking-wider text-sky-600 bg-sky-50 px-3 py-1 rounded-full">
                {zoomModal.format}
              </span>
              <h3 className="font-bold text-slate-900 text-lg mt-2">{zoomModal.name}</h3>
              <p className="text-xs text-slate-500 font-mono mt-0.5">{zoomModal.code}</p>
            </div>

            {/* Código en gran formato con fondo blanco de alto contraste */}
            <div className="bg-white border-2 border-slate-200 rounded-2xl p-6 shadow-inner flex items-center justify-center min-h-[160px]">
              <img
                src={`/api/barcode/${zoomModal.format}/${encodeURIComponent(zoomModal.code)}.svg`}
                alt={zoomModal.name}
                className="max-h-40 w-auto object-contain"
              />
            </div>

            <p className="text-xs text-slate-400">
              💡 Coloca tu lector láser Symbol a una distancia de 10 a 20 cm de la pantalla con brillo moderado.
            </p>

            <button
              type="button"
              onClick={() => setZoomModal(null)}
              className="w-full bg-slate-900 hover:bg-slate-800 text-white py-3 rounded-xl text-sm font-semibold transition-colors"
            >
              Cerrar
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
