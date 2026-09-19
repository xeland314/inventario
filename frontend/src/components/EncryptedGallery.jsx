import React, { useState, useEffect, useRef } from 'react';

export default function EncryptedGallery({ entidadTipo, entidadId }) {
  const [fotos, setFotos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [cameraOn, setCameraOn] = useState(false);
  const [cameraError, setCameraError] = useState(null);
  const [facingMode, setFacingMode] = useState('environment');
  const videoRef = useRef(null);
  const canvasRef = useRef(null);
  const streamRef = useRef(null);

  const fetchGaleria = () => {
    fetch(`/api/evidencia/galeria?tipo=${entidadTipo}&id=${entidadId}`)
      .then((res) => res.json())
      .then((data) => {
        setFotos(data.fotos || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  };

  useEffect(() => {
    fetchGaleria();
  }, [entidadTipo, entidadId]);

  // Limpieza: apagar cámara al desmontar o cambiar entidad
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

  const startCamera = async () => {
    setCameraError(null);
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      setCameraError('Tu navegador no soporta cámara en tiempo real. Usa "Subir fotos".');
      return;
    }
    try {
      // Requiere contexto seguro (https o localhost). localhost está permitido.
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: facingMode, width: { ideal: 1280 }, height: { ideal: 720 } },
        audio: false,
      });
      streamRef.current = stream;
      setCameraOn(true);
      // Asignar srcObject en siguiente tick cuando videoRef esté montado
      setTimeout(() => {
        if (videoRef.current) {
          videoRef.current.srcObject = stream;
          videoRef.current.play().catch(() => {});
        }
      }, 100);
    } catch (err) {
      console.error(err);
      let msg = 'No se pudo acceder a la cámara.';
      if (err.name === 'NotAllowedError') msg = 'Permiso de cámara denegado. Actívalo en el navegador.';
      else if (err.name === 'NotFoundError') msg = 'No se encontró cámara en el dispositivo.';
      else if (err.name === 'NotReadableError') msg = 'Cámara ocupada por otra app.';
      else if (err.name === 'SecurityError') msg = 'Se requiere HTTPS o localhost para usar la cámara.';
      setCameraError(msg + ' Puedes usar "Subir fotos" como alternativa.');
    }
  };

  const stopCamera = () => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((t) => t.stop());
      streamRef.current = null;
    }
    if (videoRef.current) videoRef.current.srcObject = null;
    setCameraOn(false);
  };

  const toggleFacing = async () => {
    const newMode = facingMode === 'environment' ? 'user' : 'environment';
    setFacingMode(newMode);
    if (cameraOn) {
      stopCamera();
      // reiniciar con nuevo facingMode tras breve delay
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
        } catch (e) {
          setCameraError('No se pudo cambiar de cámara.');
        }
      }, 200);
    }
  };

  const handleCapture = async () => {
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
    // Para cámara frontal, espejar
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
        } catch (e) {
          alert('Error de red al subir captura');
        } finally {
          setUploading(false);
        }
      },
      'image/jpeg',
      0.85
    );
  };

  const handleUpload = async (e) => {
    const files = e.target.files;
    if (!files.length) return;
    setUploading(true);
    const form = new FormData();
    form.append('entidad_tipo', entidadTipo);
    form.append('entidad_id', entidadId);
    for (const f of files) form.append('fotos', f);

    const res = await fetch('/api/evidencia/upload', { method: 'POST', body: form });
    if (res.ok) {
      fetchGaleria();
    }
    setUploading(false);
    e.target.value = '';
  };

  if (loading) return <div className="p-4 text-slate-500 italic">Cargando galería cifrada...</div>;

  return (
    <div className="bg-white p-5 rounded-xl shadow-sm border border-slate-200">
      <div className="flex justify-between items-center mb-4 flex-wrap gap-2">
        <h3 className="text-lg font-bold text-slate-800 flex items-center gap-2">
          <span>📸</span> Galería de Evidencias (Almacenamiento Cifrado)
        </h3>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={cameraOn ? stopCamera : startCamera}
            disabled={uploading}
            className={`text-xs px-4 py-2 rounded-full font-medium transition border ${
              cameraOn
                ? 'bg-red-50 border-red-200 text-red-700 hover:bg-red-100'
                : 'bg-sky-600 border-sky-600 text-white hover:bg-sky-500'
            } disabled:opacity-50`}
          >
            {cameraOn ? '✕ Cerrar cámara' : '📷 Tomar foto'}
          </button>
          <label className="bg-slate-900 hover:bg-slate-800 text-white text-xs px-4 py-2 rounded-full cursor-pointer transition">
            {uploading ? 'Cifrando...' : '+ Subir fotos'}
            <input type="file" multiple accept="image/*" capture="environment" className="hidden" onChange={handleUpload} disabled={uploading} />
          </label>
        </div>
      </div>

      {/* Panel cámara en tiempo real */}
      {cameraOn && (
        <div className="mb-4 p-3 bg-slate-900 rounded-xl border border-slate-800">
          <div className="relative bg-black rounded-lg overflow-hidden aspect-video max-h-[360px] flex items-center justify-center">
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
          <div className="flex gap-2 mt-3">
            <button
              onClick={handleCapture}
              disabled={uploading}
              className="flex-1 bg-white text-slate-900 font-bold py-2.5 rounded-full hover:bg-slate-100 disabled:opacity-50 flex items-center justify-center gap-2"
            >
              <span>📸</span> {uploading ? 'Subiendo...' : 'Capturar y subir'}
            </button>
            <button
              onClick={toggleFacing}
              disabled={uploading}
              type="button"
              title="Cambiar cámara frontal/trasera"
              className="px-4 py-2.5 bg-slate-800 text-white rounded-full hover:bg-slate-700 disabled:opacity-50 text-xs border border-slate-700"
            >
              🔄 Girar
            </button>
            <button
              onClick={stopCamera}
              disabled={uploading}
              type="button"
              className="px-4 py-2.5 bg-slate-800 text-white rounded-full hover:bg-slate-700 disabled:opacity-50 text-xs border border-slate-700"
            >
              Cerrar
            </button>
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
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
          {fotos.map((foto) => (
            <div key={foto.id} className="group border rounded-lg overflow-hidden bg-slate-50 relative shadow-sm">
              <img
                src={`/api/evidencia/${foto.id}/raw`}
                alt={foto.nombre_original}
                className="w-full h-36 object-cover transition-transform duration-300 group-hover:scale-105"
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
