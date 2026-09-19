import React, { useState, useEffect, useRef } from 'react';

interface Foto {
  id: string;
  nombre_original: string;
  mime_type: string;
  cargado_en: string;
}

interface Props {
  entidadTipo: string;
  entidadId: string;
}

export default function EncryptedGallery({ entidadTipo, entidadId }: Props): React.JSX.Element {
  const [fotos, setFotos] = useState<Foto[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [uploading, setUploading] = useState<boolean>(false);
  const [cameraOn, setCameraOn] = useState<boolean>(false);
  const [cameraError, setCameraError] = useState<string | null>(null);
  const [facingMode, setFacingMode] = useState<'environment' | 'user'>('environment');
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const streamRef = useRef<MediaStream | null>(null);

  const fetchGaleria = (): void => {
    fetch(`/api/evidencia/galeria?tipo=${entidadTipo}&id=${entidadId}`)
      .then((res) => res.json() as Promise<{ fotos: Foto[] }>)
      .then((data) => {
        setFotos(data.fotos || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  };

  useEffect(() => {
    fetchGaleria();
  }, [entidadTipo, entidadId]);

  useEffect(() => {
    return () => {
      if (streamRef.current) {
        streamRef.current.getTracks().forEach((t) => t.stop());
      }
    };
  }, []);

  useEffect(() => {
    if (cameraOn && videoRef.current && streamRef.current) {
      videoRef.current.srcObject = streamRef.current;
    }
  }, [cameraOn]);

  const startCamera = async (): Promise<void> => {
    setCameraError(null);
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      setCameraError('Tu navegador no soporta cámara en tiempo real. Usa "Subir fotos".');
      return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: facingMode, width: { ideal: 1280 }, height: { ideal: 720 } },
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
    } catch (err) {
      const e = err as DOMException;
      let msg = 'No se pudo acceder a la cámara.';
      if (e.name === 'NotAllowedError') msg = 'Permiso de cámara denegado. Actívalo en el navegador.';
      else if (e.name === 'NotFoundError') msg = 'No se encontró cámara en el dispositivo.';
      else if (e.name === 'NotReadableError') msg = 'Cámara ocupada por otra app.';
      else if (e.name === 'SecurityError') msg = 'Se requiere HTTPS o localhost para usar la cámara.';
      setCameraError(msg + ' Puedes usar "Subir fotos" como alternativa.');
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

  const toggleFacing = async (): Promise<void> => {
    const newMode: 'environment' | 'user' = facingMode === 'environment' ? 'user' : 'environment';
    setFacingMode(newMode);
    if (cameraOn) {
      stopCamera();
      setTimeout(async () => {
        try {
          const stream = await navigator.mediaDevices.getUserMedia({
            video: { facingMode: newMode },
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
        } catch {
          setCameraError('No se pudo cambiar de cámara.');
        }
      }, 200);
    }
  };

  const handleCapture = async (): Promise<void> => {
    const video = videoRef.current;
    const canvas = canvasRef.current;
    if (!video || !canvas) return;
    const w = video.videoWidth;
    const h = video.videoHeight;
    if (!w || !h) {
      alert('Cámara aún no lista, espera un segundo.');
      return;
    }
    canvas.width = w;
    canvas.height = h;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    if (facingMode === 'user') {
      ctx.translate(w, 0);
      ctx.scale(-1, 1);
    }
    ctx.drawImage(video, 0, 0, w, h);

    setUploading(true);
    canvas.toBlob(
      async (blob) => {
        if (!blob) {
          setUploading(false);
          alert('Error al capturar imagen');
          return;
        }
        const fileName = `captura_${entidadTipo}_${Date.now()}.jpg`;
        const file = new File([blob], fileName, { type: 'image/jpeg' });
        const form = new FormData();
        form.append('entidad_tipo', entidadTipo);
        form.append('entidad_id', entidadId);
        form.append('fotos', file);

        try {
          const res = await fetch('/api/evidencia/upload', { method: 'POST', body: form });
          if (res.ok) {
            fetchGaleria();
          } else {
            const txt = await res.text();
            alert('Error al subir captura: ' + txt);
          }
        } catch {
          alert('Error de red al subir captura');
        } finally {
          setUploading(false);
        }
      },
      'image/jpeg',
      0.85
    );
  };

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>): Promise<void> => {
    const files = e.target.files;
    if (!files || !files.length) return;
    setUploading(true);
    const form = new FormData();
    form.append('entidad_tipo', entidadTipo);
    form.append('entidad_id', entidadId);
    for (const f of Array.from(files)) form.append('fotos', f);

    const res = await fetch('/api/evidencia/upload', { method: 'POST', body: form });
    if (res.ok) {
      fetchGaleria();
    }
    setUploading(false);
    e.target.value = '';
  };

  if (loading) return <div className="p-4 text-slate-500 italic">Cargando galería cifrada...</div>;

  return (
    <div className="bg-white p-4 sm:p-5 rounded-xl shadow-sm border border-slate-200">
      <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3 mb-4">
        <h3 className="text-base sm:text-lg font-bold text-slate-800 flex items-center gap-2">
          <span>📸</span> Galería de Evidencias
        </h3>
        <div className="flex gap-2 w-full sm:w-auto">
          <button
            type="button"
            onClick={cameraOn ? stopCamera : startCamera}
            disabled={uploading}
            className={`flex-1 sm:flex-none text-sm px-5 py-3 min-h-[44px] rounded-full font-medium transition border touch-manipulation ${
              cameraOn
                ? 'bg-red-50 border-red-200 text-red-700 hover:bg-red-100'
                : 'bg-sky-600 border-sky-600 text-white hover:bg-sky-500'
            } disabled:opacity-50`}
          >
            {cameraOn ? '✕ Cerrar' : '📷 Tomar foto'}
          </button>
          <label className="flex-1 sm:flex-none bg-slate-900 hover:bg-slate-800 text-white text-sm px-5 py-3 min-h-[44px] rounded-full cursor-pointer transition flex items-center justify-center touch-manipulation">
            {uploading ? 'Cifrando...' : '+ Subir'}
            <input type="file" multiple accept="image/*" capture="environment" className="hidden" onChange={handleUpload} disabled={uploading} />
          </label>
        </div>
      </div>

      {cameraOn && (
        <div className="mb-4 p-3 bg-slate-900 rounded-xl border border-slate-800">
          <div className="relative bg-black rounded-lg overflow-hidden aspect-[4/3] sm:aspect-video max-h-[60vh] sm:max-h-[360px] flex items-center justify-center">
            <video
              ref={videoRef}
              autoPlay
              playsInline
              muted
              className={`w-full h-full object-cover ${facingMode === 'user' ? 'scale-x-[-1]' : ''}`}
            />
            <canvas ref={canvasRef} className="hidden" />
            {uploading && (
              <div className="absolute inset-0 bg-black/60 flex items-center justify-center text-white text-sm">
                Cifrando y subiendo...
              </div>
            )}
          </div>
          <div className="flex flex-col sm:flex-row gap-2 mt-3">
            <button
              type="button"
              onClick={handleCapture}
              disabled={uploading}
              className="w-full sm:flex-1 bg-white text-slate-900 font-bold py-3.5 min-h-[48px] rounded-full hover:bg-slate-100 disabled:opacity-50 flex items-center justify-center gap-2 touch-manipulation text-base"
            >
              <span>📸</span> {uploading ? 'Subiendo...' : 'Capturar y subir'}
            </button>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={toggleFacing}
                disabled={uploading}
                title="Cambiar cámara frontal/trasera"
                className="flex-1 sm:flex-none px-5 py-3 min-h-[48px] bg-slate-800 text-white rounded-full hover:bg-slate-700 disabled:opacity-50 text-sm border border-slate-700 touch-manipulation"
              >
                🔄 Girar
              </button>
              <button
                type="button"
                onClick={stopCamera}
                disabled={uploading}
                className="flex-1 sm:flex-none px-5 py-3 min-h-[48px] bg-slate-800 text-white rounded-full hover:bg-slate-700 disabled:opacity-50 text-sm border border-slate-700 touch-manipulation"
              >
                Cerrar
              </button>
            </div>
          </div>
          <p className="text-[11px] text-slate-400 mt-2 text-center">
            La foto se cifra con AES-256-GCM en el servidor antes de guardarse. Queda asociada a {entidadTipo} {entidadId.slice(0, 8)}…
          </p>
        </div>
      )}

      {cameraError && (
        <div className="mb-4 bg-amber-50 border border-amber-200 text-amber-800 px-3 py-2 rounded-lg text-xs">
          ⚠️ {cameraError}
        </div>
      )}

      {fotos.length === 0 ? (
        <p className="text-sm text-slate-400 py-6 text-center border-2 border-dashed border-slate-200 rounded-lg">
          No hay fotos de evidencia cargadas. Las imágenes se cifran con AES-256-GCM antes de guardarse en disco.
        </p>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3 sm:gap-4">
          {fotos.map((foto) => (
            <div key={foto.id} className="group border rounded-lg overflow-hidden bg-slate-50 relative shadow-sm">
              <img
                src={`/api/evidencia/${foto.id}/raw`}
                alt={foto.nombre_original}
                className="w-full h-36 sm:h-40 object-cover transition-transform duration-300 group-hover:scale-105"
                loading="lazy"
              />
              <div className="p-2.5 bg-white border-t border-slate-100">
                <p className="text-xs font-semibold text-slate-700 truncate">{foto.nombre_original}</p>
                <p className="text-[10px] text-slate-400 mt-0.5">🕒 {new Date(foto.cargado_en).toLocaleString('es-EC')}</p>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
