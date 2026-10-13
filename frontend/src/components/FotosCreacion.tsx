import React, { useEffect, useRef, useState } from 'react';
import { AlertTriangle, Camera, RefreshCw, Trash2, Upload, X } from 'lucide-react';

interface FotoPendiente {
  blob: Blob;
  url: string;
}

export default function FotosCreacion(): React.JSX.Element {
  const [fotos, setFotos] = useState<FotoPendiente[]>([]);
  const [cameraOn, setCameraOn] = useState<boolean>(false);
  const [facingMode, setFacingMode] = useState<'environment' | 'user'>('environment');
  const [error, setError] = useState<string | null>(null);
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const streamRef = useRef<MediaStream | null>(null);

  useEffect(() => {
    (window as unknown as { __fotosCreacion?: Blob[] }).__fotosCreacion = fotos.map((f) => f.blob);
  }, [fotos]);

  useEffect(() => {
    return () => {
      streamRef.current?.getTracks().forEach((t) => t.stop());
      fotos.forEach((f) => URL.revokeObjectURL(f.url));
    };
  }, []);

  useEffect(() => {
    if (cameraOn && videoRef.current && streamRef.current) {
      videoRef.current.srcObject = streamRef.current;
      videoRef.current.play().catch(() => {});
    }
  }, [cameraOn]);

  const startCamera = async (): Promise<void> => {
    setError(null);
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      setError('Tu navegador no soporta cámara. Usa "Subir" como alternativa.');
      return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode, width: { ideal: 1280 }, height: { ideal: 720 } },
        audio: false,
      });
      streamRef.current = stream;
      setCameraOn(true);
      setTimeout(() => {
        if (videoRef.current) videoRef.current.srcObject = stream;
      }, 100);
    } catch (e) {
      const err = e as DOMException;
      let msg = 'No se pudo acceder a la cámara.';
      if (err.name === 'NotAllowedError') msg = 'Permiso de cámara denegado.';
      else if (err.name === 'NotFoundError') msg = 'No se encontró cámara.';
      setError(msg);
    }
  };

  const stopCamera = (): void => {
    streamRef.current?.getTracks().forEach((t) => t.stop());
    streamRef.current = null;
    if (videoRef.current) videoRef.current.srcObject = null;
    setCameraOn(false);
  };

  const capturar = (): void => {
    const video = videoRef.current;
    const canvas = canvasRef.current;
    if (!video || !canvas || !video.videoWidth) return;
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    if (facingMode === 'user') {
      ctx.translate(canvas.width, 0);
      ctx.scale(-1, 1);
    }
    ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
    canvas.toBlob(
      (blob) => {
        if (!blob) return;
        setFotos((prev) => [...prev, { blob, url: URL.createObjectURL(blob) }]);
      },
      'image/jpeg',
      0.85
    );
  };

  const toggleFacing = (): void => {
    const nuevo: 'environment' | 'user' = facingMode === 'environment' ? 'user' : 'environment';
    setFacingMode(nuevo);
    if (cameraOn) {
      stopCamera();
      setTimeout(() => void startCamera(), 200);
    }
  };

  const quitar = (idx: number): void => {
    setFotos((prev) => {
      URL.revokeObjectURL(prev[idx].url);
      return prev.filter((_, i) => i !== idx);
    });
  };

  const agregarArchivos = (e: React.ChangeEvent<HTMLInputElement>): void => {
    const files = e.target.files;
    if (!files) return;
    const nuevas: FotoPendiente[] = [];
    for (const f of Array.from(files)) {
      nuevas.push({ blob: f, url: URL.createObjectURL(f) });
    }
    setFotos((prev) => [...prev, ...nuevas]);
    e.target.value = '';
  };

  return (
    <div id="fotos-creacion" className="bg-white border border-slate-200 rounded-2xl p-5 shadow-sm space-y-3">
      <div className="flex justify-between items-center gap-3">
        <h2 className="font-bold text-slate-900 text-base inline-flex items-center gap-2">
          <Camera className="w-5 h-5" /> Fotos de evidencia ({fotos.length > 0 ? fotos.length + ' listas' : 'opcional'})
        </h2>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={cameraOn ? stopCamera : startCamera}
            className={`text-xs px-4 py-2 min-h-[40px] rounded-full font-medium inline-flex items-center gap-1.5 touch-manipulation ${cameraOn ? 'bg-red-50 border border-red-200 text-red-700' : 'bg-sky-600 text-white hover:bg-sky-500'}`}
          >
            {cameraOn ? <><X className="w-3.5 h-3.5" /> Cerrar cámara</> : <><Camera className="w-3.5 h-3.5" /> Tomar foto</>}
          </button>
          <label className="text-xs px-4 py-2 min-h-[40px] rounded-full bg-slate-900 text-white hover:bg-slate-800 cursor-pointer inline-flex items-center gap-1.5 touch-manipulation">
            <Upload className="w-3.5 h-3.5" /> Subir
            <input type="file" multiple accept="image/*" capture="environment" className="hidden" onChange={agregarArchivos} />
          </label>
        </div>
      </div>
      <p className="text-xs text-slate-500">Captura inicial del equipo; al crear el dispositivo se suben cifradas (AES-256-GCM) a su galería.</p>

      {cameraOn && (
        <div className="bg-slate-900 rounded-xl p-3">
          <div className="relative bg-black rounded-lg overflow-hidden aspect-[4/3] sm:aspect-video max-h-[60vh] flex items-center justify-center">
            <video ref={videoRef} autoPlay playsInline muted className={`w-full h-full object-cover ${facingMode === 'user' ? 'scale-x-[-1]' : ''}`} />
            <canvas ref={canvasRef} className="hidden" />
          </div>
          <div className="flex gap-2 mt-3">
            <button type="button" onClick={capturar} className="flex-1 bg-white text-slate-900 font-bold py-3 min-h-[48px] rounded-full text-sm inline-flex items-center justify-center gap-2 touch-manipulation">
              <Camera className="w-4 h-4" /> Capturar
            </button>
            <button type="button" onClick={toggleFacing} className="px-5 py-3 min-h-[48px] bg-slate-800 border border-slate-700 text-white rounded-full text-xs inline-flex items-center justify-center gap-1.5 touch-manipulation">
              <RefreshCw className="w-3.5 h-3.5" /> Girar
            </button>
          </div>
        </div>
      )}

      {error && (
        <div className="bg-amber-50 border border-amber-200 text-amber-800 px-3 py-2 rounded-lg text-xs inline-flex items-start gap-2">
          <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" /> <span>{error}</span>
        </div>
      )}

      {fotos.length > 0 && (
        <div className="grid grid-cols-4 sm:grid-cols-6 gap-2">
          {fotos.map((f, i) => (
            <div key={f.url} className="relative border border-slate-200 rounded-lg overflow-hidden aspect-square">
              <img src={f.url} alt={'foto ' + (i + 1)} className="w-full h-full object-cover" />
              <button
                type="button"
                onClick={() => quitar(i)}
                className="absolute top-1 right-1 bg-white/90 hover:bg-rose-600 hover:text-white text-slate-700 rounded-full w-6 h-6 flex items-center justify-center shadow"
                aria-label="Quitar foto"
              >
                <Trash2 className="w-3 h-3" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
