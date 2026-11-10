import React, { useEffect, useRef, useState } from 'react';
import {
  AlertTriangle,
  Camera,
  Check,
  Cpu,
  Loader2,
  Minus,
  Plus,
  ScanEye,
  Upload,
  X,
} from 'lucide-react';

interface Deteccion {
  box: [number, number, number, number]; // xmin, ymin, xmax, ymax (px o normalizado)
  score: number;
  label: string;
}

interface VLMConfig {
  vlm: { disponible: boolean; tipo: string; modelo: string };
  navegador: { disponible: boolean; modelo: string };
}

interface Props {
  conjuntoId: string;
  nombre: string;
  categoria?: string | null;
  cantidadActual: number;
  unidad: string;
  onGuardado?: () => void;
}

// traducción es→en para prompts (OWL-ViT se entrenó con texto en inglés)
const ES_EN: Record<string, string> = {
  tornillo: 'screw', tornillos: 'screw', tuerca: 'nut', tuercas: 'nut',
  arandela: 'washer', clavo: 'nail', resistencia: 'resistor', resistores: 'resistor',
  resistor: 'resistor', resistencias: 'resistor', capacitor: 'capacitor',
  condensador: 'capacitor', fusible: 'fuse', fusable: 'fuse', diodo: 'diode',
  led: 'led', transistor: 'transistor', cable: 'cable', cables: 'cable',
  jumper: 'jumper', jumpers: 'jumper', pila: 'battery', baterias: 'battery',
  bateria: 'battery', motor: 'motor', engranaje: 'gear', resortes: 'spring',
  resorte: 'spring', remache: 'rivet', remaches: 'rivet', tuercaallen: 'hex nut',
};

const MODELO_NAVEGADOR = 'owlvit-base-patch32';
const TRANSFORMERS_CDN = 'https://cdn.jsdelivr.net/npm/@huggingface/transformers@4.3.1';

function promptInicial(nombre: string, categoria?: string | null): string {
  const palabras = `${nombre} ${categoria || ''}`
    .toLowerCase()
    .split(/[^a-záéíóúñ]+/)
    .filter((w) => w.length > 2);
  const labels = new Set<string>();
  for (const p of palabras) {
    if (ES_EN[p]) labels.add(ES_EN[p]);
  }
  if (labels.size === 0) labels.add('screw');
  return Array.from(labels).slice(0, 4).join(', ');
}

export default function ContadorIA({ conjuntoId, nombre, categoria, cantidadActual, unidad, onGuardado }: Props): React.JSX.Element {
  const [imageDataUrl, setImageDataUrl] = useState<string | null>(null);
  const [imgDims, setImgDims] = useState<{ w: number; h: number } | null>(null);
  const [detecciones, setDetecciones] = useState<Deteccion[]>([]);
  const [threshold, setThreshold] = useState<number>(0.15);
  const [prompt, setPrompt] = useState<string>(promptInicial(nombre, categoria));
  const [status, setStatus] = useState<string>('');
  const [progreso, setProgreso] = useState<number | null>(null);
  const [confirmada, setConfirmada] = useState<number>(cantidadActual);
  const [modeloUsado, setModeloUsado] = useState<string>('manual');
  const [aplicarAjuste, setAplicarAjuste] = useState<boolean>(true);
  const [guardarFoto, setGuardarFoto] = useState<boolean>(true);
  const [vlm, setVlm] = useState<VLMConfig | null>(null);
  const [vlmResultado, setVlmResultado] = useState<{ descripcion?: string; cantidad_estimada?: number; items?: { nombre: string; cantidad: number }[]; confianza?: string } | null>(null);
  const [vlmTxt, setVlmTxt] = useState<string | null>(null);
  const [fotoId, setFotoId] = useState<string | null>(null);
  const [cameraOn, setCameraOn] = useState<boolean>(false);
  const [cameraError, setCameraError] = useState<string | null>(null);
  const [saving, setSaving] = useState<boolean>(false);
  const [saved, setSaved] = useState<boolean>(false);
  const videoRef = useRef<HTMLVideoElement>(null);
  const captureCanvasRef = useRef<HTMLCanvasElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const pipelineRef = useRef<unknown>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    fetch('/api/ia/config')
      .then((r) => r.json())
      .then((d: VLMConfig) => setVlm(d))
      .catch(() => setVlm(null));
  }, []);

  useEffect(() => {
    return () => {
      if (streamRef.current) streamRef.current.getTracks().forEach((t) => t.stop());
    };
  }, []);

  const startCamera = async (): Promise<void> => {
    setCameraError(null);
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      setCameraError('Tu navegador no soporta cámara. Usa "Subir imagen".');
      return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: 'environment', width: { ideal: 1280 }, height: { ideal: 720 } },
        audio: false,
      });
      streamRef.current = stream;
      setCameraOn(true);
      setTimeout(() => {
        if (videoRef.current) {
          videoRef.current.srcObject = stream;
          videoRef.current.play().catch(() => {});
        }
      }, 100);
    } catch (e) {
      const err = e as DOMException;
      let msg = 'No se pudo acceder a la cámara.';
      if (err.name === 'NotAllowedError') msg = 'Permiso de cámara denegado.';
      else if (err.name === 'NotFoundError') msg = 'No se encontró cámara.';
      else if (err.name === 'SecurityError') msg = 'Se requiere HTTPS o localhost.';
      setCameraError(msg + ' Puedes subir una imagen.');
    }
  };

  const stopCamera = (): void => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((t) => t.stop());
      streamRef.current = null;
    }
    if (videoRef.current) videoRef.current.srcObject = null;
    setCameraOn(false);
  };

  const usarImagen = (dataUrl: string): void => {
    setImageDataUrl(dataUrl);
    setDetecciones([]);
    setVlmResultado(null);
    setVlmTxt(null);
    setFotoId(null);
    setSaved(false);
    setConfirmada(cantidadActual);
    const img = new Image();
    img.onload = () => setImgDims({ w: img.naturalWidth, h: img.naturalHeight });
    img.src = dataUrl;
  };

  const capturar = (): void => {
    const video = videoRef.current;
    const canvas = captureCanvasRef.current;
    if (!video || !canvas) return;
    const w = video.videoWidth;
    const h = video.videoHeight;
    if (!w || !h) return;
    canvas.width = w;
    canvas.height = h;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    ctx.drawImage(video, 0, 0, w, h);
    usarImagen(canvas.toDataURL('image/jpeg', 0.85));
    stopCamera();
  };

  const handleFile = (e: React.ChangeEvent<HTMLInputElement>): void => {
    const f = e.target.files?.[0];
    if (!f) return;
    const reader = new FileReader();
    reader.onload = () => usarImagen(reader.result as string);
    reader.readAsDataURL(f);
    e.target.value = '';
  };

  // --- Vía navegador: transformers.js + OWL-ViT zero-shot (runs 100% local) ---
  const detectarNavegador = async (): Promise<void> => {
    if (!imageDataUrl) return;
    setStatus('Cargando IA local (primera vez descarga ~100MB, luego queda en caché)...');
    setProgreso(0);
    try {
      if (!pipelineRef.current) {
        const mod = (await import(/* @vite-ignore */ TRANSFORMERS_CDN)) as {
          pipeline: (task: string, model: string, opts?: Record<string, unknown>) => Promise<unknown>;
        };
        pipelineRef.current = await mod.pipeline('zero-shot-object-detection', `Xenova/${MODELO_NAVEGADOR}`, {
          progress_callback: (p: { status: string; progress?: number }) => {
            if (p.status === 'progress' && typeof p.progress === 'number') setProgreso(Math.round(p.progress));
          },
        });
      }
      const detector = pipelineRef.current as (
        img: string,
        labels: string[],
        opts?: Record<string, unknown>
      ) => Promise<{ score: number; label: string; box: { xmin: number; ymin: number; xmax: number; ymax: number } }[]>;
      setStatus('Detectando objetos (100% local, en tu dispositivo)...');
      setProgreso(null);
      const labels = prompt.split(',').map((l) => l.trim()).filter(Boolean);
      const raw = await detector(imageDataUrl, labels, { threshold: 0.01, percentage_boxes: false });
      const deteccionesFiltradas: Deteccion[] = raw
        .filter((d) => d.score >= threshold)
        .map((d) => ({ box: [d.box.xmin, d.box.ymin, d.box.xmax, d.box.ymax], score: d.score, label: d.label }));
      setDetecciones(deteccionesFiltradas);
      setModeloUsado(`browser:${MODELO_NAVEGADOR}`);
      if (deteccionesFiltradas.length > 0) setConfirmada(deteccionesFiltradas.length);
      setStatus(
        deteccionesFiltradas.length > 0
          ? `${deteccionesFiltradas.length} objetos detectados. Toca una caja para descartar falsos positivos y ajusta el número.`
          : 'Sin detecciones sobre el umbral. Prueba con otro texto de búsqueda (en inglés suele funcionar mejor), baja el umbral, o ajusta el número a mano.',
      );
    } catch (e) {
      setStatus('');
      setProgreso(null);
      alert('Error al ejecutar la IA local: ' + (e as Error).message);
    }
  };

  // --- Vía VLM local (Ollama/llama.cpp vía backend): mejor identificación ---
  const identificarVLM = async (): Promise<void> => {
    if (!imageDataUrl) return;
    setStatus('Subiendo foto cifrada y consultando el VLM local...');
    try {
      // subir foto cifrada primero (AES-256-GCM en el servidor)
      const blob = await (await fetch(imageDataUrl)).blob();
      const form = new FormData();
      form.append('entidad_tipo', 'conjunto');
      form.append('entidad_id', conjuntoId);
      form.append('fotos', new File([blob], `conteo_${conjuntoId.slice(0, 8)}.jpg`, { type: 'image/jpeg' }));
      const upRes = await fetch('/api/evidencia/upload', { method: 'POST', body: form });
      if (!upRes.ok) throw new Error(await upRes.text());
      const upData = (await upRes.json()) as { foto_ids: string[] };
      const fid = upData.foto_ids?.[0] || null;
      setFotoId(fid);
      const res = await fetch('/api/ia/identificar', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ foto_evidencia_id: fid, prompt: `Conjunto: ${nombre} ${categoria || ''}. ` }),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as { resultado: { descripcion?: string; cantidad_estimada?: number; items?: { nombre: string; cantidad: number }[]; confianza?: string }; texto: string; modelo: string };
      setVlmResultado(data.resultado);
      setVlmTxt(data.texto || null);
      setModeloUsado(`vlm:${data.modelo}`);
      if (typeof data.resultado?.cantidad_estimada === 'number') setConfirmada(Math.max(0, Math.round(data.resultado.cantidad_estimada)));
      setStatus('VLM respondió. Revisa la identificación y confirma el número.');
    } catch (e) {
      setStatus('');
      alert('Error con el VLM local: ' + (e as Error).message);
    }
  };

  const quitarDeteccion = (i: number): void => {
    setDetecciones((prev) => {
      const next = prev.filter((_, idx) => idx !== i);
      if (next.length > 0 && next.length !== prev.length) setConfirmada(next.length);
      return next;
    });
  };

  // cajas en coordenadas de imagen → % para overlay SVG
  const cajaStyle = (d: Deteccion): { left: string; top: string; width: string; height: string } | null => {
    if (!imgDims) return null;
    const [xmin, ymin, xmax, ymax] = d.box;
    const normalizado = Math.max(Math.abs(xmin), Math.abs(ymin), Math.abs(xmax), Math.abs(ymax)) <= 1.5;
    const left = normalizado ? xmin * 100 : (xmin / imgDims.w) * 100;
    const top = normalizado ? ymin * 100 : (ymin / imgDims.h) * 100;
    const w = normalizado ? (xmax - xmin) * 100 : ((xmax - xmin) / imgDims.w) * 100;
    const h = normalizado ? (ymax - ymin) * 100 : ((ymax - ymin) / imgDims.h) * 100;
    return { left: `${left}%`, top: `${top}%`, width: `${Math.max(w, 1)}%`, height: `${Math.max(h, 1)}%` };
  };

  const guardar = async (): Promise<void> => {
    if (!imageDataUrl) return alert('Toma o sube una foto primero');
    setSaving(true);
    try {
      let fid = fotoId;
      if (guardarFoto && !fid) {
        const blob = await (await fetch(imageDataUrl)).blob();
        const form = new FormData();
        form.append('entidad_tipo', 'conjunto');
        form.append('entidad_id', conjuntoId);
        form.append('fotos', new File([blob], `conteo_${conjuntoId.slice(0, 8)}.jpg`, { type: 'image/jpeg' }));
        const upRes = await fetch('/api/evidencia/upload', { method: 'POST', body: form });
        if (!upRes.ok) throw new Error(await upRes.text());
        const upData = (await upRes.json()) as { foto_ids: string[] };
        fid = upData.foto_ids?.[0] || null;
      }
      const res = await fetch(`/api/conjuntos/${conjuntoId}/conteos`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          cantidad_detectada: detecciones.length,
          cantidad_confirmada: confirmada,
          modelo: modeloUsado,
          detecciones: detecciones.map((d) => ({ box: d.box, score: Math.round(d.score * 100) / 100, label: d.label })),
          foto_evidencia_id: guardarFoto ? fid : null,
          aplicar_ajuste: aplicarAjuste,
        }),
      });
      if (!res.ok) throw new Error(await res.text());
      setSaved(true);
      onGuardado?.();
    } catch (e) {
      alert('Error al guardar el conteo: ' + (e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const delta = confirmada - cantidadActual;

  return (
    <div className="mt-3 border border-sky-200 rounded-xl p-3 bg-sky-50/50 space-y-3">
      <div className="flex items-start gap-2">
        <ScanEye className="w-4 h-4 text-sky-700 mt-0.5 shrink-0" />
        <p className="text-xs text-sky-900">
          <span className="font-bold">Contador IA local</span> — la foto se analiza en tu dispositivo (offline tras la primera descarga del modelo)
          {vlm?.vlm.disponible ? <> o con el VLM local <span className="font-mono">{vlm.vlm.modelo}</span> (mejor identificación)</> : null}.
          La IA <span className="font-bold">propone</span>, tú <span className="font-bold">confirmas</span>.
        </p>
      </div>

      {/* Captura */}
      {!imageDataUrl && (
        <div className="flex flex-col sm:flex-row gap-2">
          {!cameraOn ? (
            <button type="button" onClick={cameraOn ? stopCamera : startCamera} className="flex-1 bg-slate-900 text-white px-3 py-2.5 min-h-[44px] rounded-full text-xs font-semibold inline-flex items-center justify-center gap-1.5">
              <Camera className="w-4 h-4" /> Tomar foto
            </button>
          ) : (
            <>
              <button type="button" onClick={capturar} className="flex-1 bg-white text-slate-900 font-bold px-3 py-2.5 min-h-[44px] rounded-full text-xs inline-flex items-center justify-center gap-1.5 border border-slate-300">
                <Camera className="w-4 h-4" /> Capturar
              </button>
              <button type="button" onClick={stopCamera} className="px-3 py-2.5 min-h-[44px] rounded-full text-xs bg-red-50 border border-red-200 text-red-700 inline-flex items-center justify-center gap-1.5">
                <X className="w-4 h-4" /> Cancelar
              </button>
            </>
          )}
          <button type="button" onClick={() => fileRef.current?.click()} className="flex-1 bg-slate-800 text-white px-3 py-2.5 min-h-[44px] rounded-full text-xs font-semibold inline-flex items-center justify-center gap-1.5">
            <Upload className="w-4 h-4" /> Subir imagen
          </button>
          <input ref={fileRef} type="file" accept="image/*" capture="environment" className="hidden" onChange={handleFile} />
        </div>
      )}

      {cameraOn && (
        <div className="bg-black rounded-lg overflow-hidden relative aspect-[4/3] max-h-[60vh]">
          <video ref={videoRef} autoPlay playsInline muted className="w-full h-full object-cover" />
          <canvas ref={captureCanvasRef} className="hidden" />
        </div>
      )}
      {!cameraOn && <canvas ref={captureCanvasRef} className="hidden" />}

      {cameraError && (
        <div className="bg-amber-50 border border-amber-200 text-amber-800 px-3 py-2 rounded-lg text-xs flex items-start gap-2">
          <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" /> {cameraError}
        </div>
      )}

      {/* Imagen + overlay de detecciones */}
      {imageDataUrl && (
        <div className="relative rounded-lg overflow-hidden border border-slate-200 bg-black">
          <img src={imageDataUrl} alt="Conteo" className="w-full max-h-[50vh] object-contain" />
          {detecciones.map((d, i) => {
            const st = cajaStyle(d);
            if (!st) return null;
            return (
              <button
                key={i}
                type="button"
                onClick={() => quitarDeteccion(i)}
                title={`${d.label} ${(d.score * 100).toFixed(0)}% — tocar para descartar`}
                className="absolute border-2 border-emerald-400 bg-emerald-400/10 text-[10px] font-bold text-white"
                style={{ ...st, left: st.left, top: st.top, width: st.width, height: st.height }}
              >
                <span className="absolute -top-0.5 left-0 bg-emerald-500 text-black px-1 rounded-br">{(d.score * 100).toFixed(0)}%</span>
              </button>
            );
          })}
          {detecciones.length > 0 && (
            <p className="text-[10px] text-white/80 absolute bottom-1 left-2 bg-black/50 rounded px-1.5 py-0.5">
              {detecciones.length} cajas · toca una para descartar
            </p>
          )}
        </div>
      )}

      {/* Controles de detección */}
      {imageDataUrl && (
        <div className="space-y-2">
          <div className="flex flex-col sm:flex-row gap-2">
            <input value={prompt} onChange={(e) => setPrompt(e.target.value)} placeholder="Qué buscar (inglés funciona mejor: screw, resistor...)" className="flex-1 border border-slate-200 rounded-lg px-3 py-2 text-xs bg-white" />
            <div className="flex items-center gap-2 bg-white border border-slate-200 rounded-lg px-3 py-2 text-xs shrink-0">
              <span className="text-slate-500">umbral</span>
              <input type="range" min="0.05" max="0.6" step="0.05" value={threshold} onChange={(e) => setThreshold(parseFloat(e.target.value))} className="w-20 accent-sky-600" />
              <span className="font-mono font-bold">{threshold.toFixed(2)}</span>
            </div>
          </div>
          <div className="flex flex-col sm:flex-row gap-2">
            <button type="button" onClick={detectarNavegador} className="flex-1 bg-sky-600 text-white px-3 py-2.5 min-h-[44px] rounded-full text-xs font-semibold inline-flex items-center justify-center gap-1.5 disabled:opacity-50" disabled={status.includes('Detectando') || status.includes('Cargando')}>
              <Cpu className="w-4 h-4" /> Detectar en navegador (local)
            </button>
            {vlm?.vlm.disponible && (
              <button type="button" onClick={identificarVLM} className="flex-1 bg-violet-600 text-white px-3 py-2.5 min-h-[44px] rounded-full text-xs font-semibold inline-flex items-center justify-center gap-1.5">
                <ScanEye className="w-4 h-4" /> Identificar con VLM ({vlm.vlm.modelo})
              </button>
            )}
          </div>
          {!('gpu' in navigator) && (
            <p className="text-[10px] text-slate-500">⚠ Sin WebGPU: usará WASM (más lento, pero funciona para una foto).</p>
          )}
        </div>
      )}

      {(status || progreso !== null) && (
        <div className="bg-white border border-slate-200 rounded-lg px-3 py-2 text-xs text-slate-700 flex items-center gap-2">
          {(status.includes('Cargando') || status.includes('Detectando') || status.includes('VLM') || status.includes('Subiendo')) && <Loader2 className="w-4 h-4 animate-spin shrink-0" />}
          <span>{status}{progreso !== null ? ` ${progreso}%` : ''}</span>
        </div>
      )}

      {vlmResultado && (
        <div className="bg-violet-50 border border-violet-200 rounded-lg px-3 py-2 text-xs space-y-1">
          <p className="font-bold text-violet-900 inline-flex items-center gap-1.5"><ScanEye className="w-3.5 h-3.5" /> Identificación VLM</p>
          {vlmResultado.descripcion && <p className="text-violet-900">{vlmResultado.descripcion}</p>}
          {vlmResultado.items && vlmResultado.items.length > 0 && (
            <ul className="list-disc ml-4 text-violet-800">
              {vlmResultado.items.map((it, i) => (
                <li key={i}><span className="font-semibold">{it.nombre}</span>: {it.cantidad}</li>
              ))}
            </ul>
          )}
          {vlmResultado.confianza && <p className="text-violet-600 italic">confianza: {vlmResultado.confianza}</p>}
          {vlmTxt && !vlmResultado.descripcion && <pre className="text-[10px] text-violet-700 whitespace-pre-wrap max-h-24 overflow-auto">{vlmTxt}</pre>}
        </div>
      )}

      {/* Confirmación humana */}
      <div className="bg-white border border-slate-200 rounded-xl p-3 space-y-2">
        <p className="text-xs font-bold text-slate-800">Confirmar cantidad ({unidad})</p>
        <div className="flex items-center gap-2">
          <button type="button" onClick={() => setConfirmada((c) => Math.max(0, c - 1))} className="w-11 h-11 rounded-full bg-slate-100 border border-slate-200 grid place-items-center touch-manipulation">
            <Minus className="w-5 h-5" />
          </button>
          <input type="number" min="0" value={confirmada} onChange={(e) => setConfirmada(Math.max(0, parseInt(e.target.value, 10) || 0))} className="flex-1 text-center text-2xl font-black border border-slate-200 rounded-xl py-2" />
          <button type="button" onClick={() => setConfirmada((c) => c + 1)} className="w-11 h-11 rounded-full bg-slate-100 border border-slate-200 grid place-items-center touch-manipulation">
            <Plus className="w-5 h-5" />
          </button>
        </div>
        <p className="text-[11px] text-slate-500">
          Stock actual: <span className="font-bold">{cantidadActual} {unidad}</span> → confirmado: <span className="font-bold">{confirmada}</span>
          {delta !== 0 && <span className={delta < 0 ? 'text-rose-600 font-bold' : 'text-emerald-600 font-bold'}> ({delta > 0 ? '+' : ''}{delta})</span>}
        </p>
        <div className="flex flex-col gap-1.5 text-xs">
          <label className="inline-flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={aplicarAjuste} onChange={(e) => setAplicarAjuste(e.target.checked)} /> Ajustar el stock al valor confirmado (movimiento tipo <em>ajuste</em> en el libro)
          </label>
          <label className="inline-flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={guardarFoto} onChange={(e) => setGuardarFoto(e.target.checked)} /> Guardar la foto cifrada (AES-256-GCM) como evidencia del conteo
          </label>
        </div>
        <div className="flex gap-2">
          <button type="button" onClick={guardar} disabled={saving || saved} className="flex-1 bg-emerald-600 text-white px-3 py-2.5 min-h-[44px] rounded-full text-xs font-bold inline-flex items-center justify-center gap-1.5 disabled:opacity-60">
            {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : saved ? <Check className="w-4 h-4" /> : null}
            {saved ? 'Guardado' : 'Guardar conteo'}
          </button>
          <button type="button" onClick={() => { setImageDataUrl(null); setDetecciones([]); setVlmResultado(null); setStatus(''); setSaved(false); }} className="px-4 py-2.5 min-h-[44px] rounded-full text-xs border border-slate-200 bg-white">
            {saved ? 'Nueva foto' : 'Descartar'}
          </button>
        </div>
        <p className="text-[10px] text-slate-400">
          Modelo: <span className="font-mono">{modeloUsado}</span> · se guarda detectado ({detecciones.length}) vs confirmado ({confirmada}) para medir la precisión de la IA.
        </p>
      </div>
    </div>
  );
}
