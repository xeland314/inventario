import React, { useState, useEffect, useRef } from 'react';
import {
  Camera,
  Images,
  Upload,
  X,
  RefreshCw,
  Trash2,
  Clock,
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
  Loader2,
  LayoutGrid,
  Image as ImageIcon,
  Rows3,
  Printer,
} from 'lucide-react';

interface Foto {
  id: string;
  nombre_original: string;
  mime_type: string;
  cargado_en: string;
}

interface Props {
  entidadTipo: string;
  entidadId: string;
  compact?: boolean;
}

export default function EncryptedGallery({ entidadTipo, entidadId, compact = false }: Props): React.JSX.Element {
  const [fotos, setFotos] = useState<Foto[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [uploading, setUploading] = useState<boolean>(false);
  const [cameraOn, setCameraOn] = useState<boolean>(false);
  const [cameraError, setCameraError] = useState<string | null>(null);
  const [facingMode, setFacingMode] = useState<'environment' | 'user'>('environment');
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [previewIndex, setPreviewIndex] = useState<number | null>(null);
  const [viewMode, setViewMode] = useState<'cover' | 'contain' | 'masonry'>('cover');
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') || '' : '';

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

  const previewFoto = previewIndex !== null ? fotos[previewIndex] : null;
  const rawUrl = (fotoId: string): string =>
    token ? `/api/evidencia/${fotoId}/raw?token=${encodeURIComponent(token)}` : `/api/evidencia/${fotoId}/raw`;

  const openPreview = (idx: number): void => setPreviewIndex(idx);
  const closePreview = (): void => setPreviewIndex(null);
  const goPrev = (): void => setPreviewIndex((i) => (i !== null ? (i > 0 ? i - 1 : fotos.length - 1) : null));
  const goNext = (): void => setPreviewIndex((i) => (i !== null ? (i < fotos.length - 1 ? i + 1 : 0) : null));

  useEffect(() => {
    if (previewIndex === null) return;
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    const onKey = (e: KeyboardEvent): void => {
      if (e.key === 'Escape') closePreview();
      else if (e.key === 'ArrowLeft') goPrev();
      else if (e.key === 'ArrowRight') goNext();
    };
    window.addEventListener('keydown', onKey);
    return () => {
      document.body.style.overflow = prevOverflow;
      window.removeEventListener('keydown', onKey);
    };
  }, [previewIndex, fotos.length]);

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

  const handleDelete = async (fotoId: string): Promise<void> => {
    if (!confirm('¿Borrar esta foto? Esta acción no se puede deshacer.')) return;
    setDeletingId(fotoId);
    try {
      const res = await fetch(`/api/evidencia/${fotoId}`, { method: 'DELETE' });
      if (res.ok) {
        setFotos((prev) => prev.filter((f) => f.id !== fotoId));
        setPreviewIndex((idx) => {
          if (idx === null) return null;
          if (fotos[idx]?.id === fotoId) {
            const remaining = fotos.length - 1;
            if (remaining <= 0) return null;
            return idx >= remaining ? remaining - 1 : idx;
          }
          const deletedIdx = fotos.findIndex((f) => f.id === fotoId);
          if (deletedIdx !== -1 && deletedIdx < idx) return idx - 1;
          return idx;
        });
      } else {
        const txt = await res.text();
        if (res.status === 403) alert('No tienes permiso para borrar fotos (solo técnico).');
        else alert('Error al borrar foto: ' + txt);
      }
    } catch {
      alert('Error de red al borrar foto');
    } finally {
      setDeletingId(null);
    }
  };

  if (loading) return <div className="p-3 text-slate-500 italic text-sm">Cargando galería cifrada...</div>;

  return (
    <div className={`encrypted-gallery ${compact ? 'bg-white p-3 rounded-lg shadow-sm border border-slate-200 print:shadow-none print:border print:p-2' : 'bg-white p-4 sm:p-5 rounded-xl shadow-sm border border-slate-200 print:shadow-none print:border print:p-2'} min-w-0 max-w-full overflow-hidden`}>
      <div className={`${compact ? 'flex flex-col gap-2 mb-3' : 'flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3 mb-4'} min-w-0`}>
        <h3 className={`${compact ? 'text-sm' : 'text-base sm:text-lg'} font-bold text-slate-800 flex items-center gap-2 shrink-0 print:text-sm`}>
          <Images className={`${compact ? 'w-4 h-4' : 'w-5 h-5'} text-slate-700 shrink-0 print:hidden`} /> Galería de Evidencias
        </h3>
        <div className={`flex gap-1.5 min-w-0 ${compact ? 'w-full' : 'w-full sm:w-auto'} print:hidden`}>
          <button
            type="button"
            onClick={cameraOn ? stopCamera : startCamera}
            disabled={uploading}
            className={`flex-1 text-xs font-medium transition border touch-manipulation inline-flex items-center justify-center gap-1.5 min-w-0 ${compact ? 'px-3 py-2 rounded-full' : 'px-3 sm:px-5 py-2.5 sm:py-3 min-h-[36px] sm:min-h-[44px] text-sm rounded-full'} ${
              cameraOn
                ? 'bg-red-50 border-red-200 text-red-700 hover:bg-red-100'
                : 'bg-sky-600 border-sky-600 text-white hover:bg-sky-500'
            } disabled:opacity-50`}
          >
            {cameraOn ? (
              <>
                <X className="w-3.5 h-3.5 shrink-0" /> Cerrar
              </>
            ) : (
              <>
                <Camera className="w-3.5 h-3.5 shrink-0" /> Tomar foto
              </>
            )}
          </button>
          <label className={`flex-1 bg-slate-900 hover:bg-slate-800 text-white cursor-pointer transition flex items-center justify-center gap-1.5 touch-manipulation min-w-0 ${compact ? 'text-xs px-3 py-2 rounded-full' : 'text-xs sm:text-sm px-3 sm:px-5 py-2.5 sm:py-3 min-h-[36px] sm:min-h-[44px] rounded-full'}`}>
            {uploading ? (
              <>
                <Loader2 className="w-3.5 h-3.5 animate-spin shrink-0" /> Cifrando...
              </>
            ) : (
              <>
                <Upload className="w-3.5 h-3.5 shrink-0" /> Subir
              </>
            )}
            <input type="file" multiple accept="image/*" capture="environment" className="hidden" onChange={handleUpload} disabled={uploading} />
          </label>
        </div>
      </div>

      {cameraOn && (
        <div className={`${compact ? 'mb-3 p-2' : 'mb-4 p-3'} bg-slate-900 rounded-xl border border-slate-800 min-w-0 print:hidden`}>
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
              <div className="absolute inset-0 bg-black/60 flex items-center justify-center text-white text-xs sm:text-sm gap-2">
                <Loader2 className="w-5 h-5 animate-spin" /> Cifrando y subiendo...
              </div>
            )}
          </div>
          <div className="flex flex-col sm:flex-row gap-2 mt-3 min-w-0">
            <button
              type="button"
              onClick={handleCapture}
              disabled={uploading}
              className="w-full sm:flex-1 bg-white text-slate-900 font-bold py-2.5 sm:py-3.5 min-h-[40px] sm:min-h-[48px] rounded-full hover:bg-slate-100 disabled:opacity-50 flex items-center justify-center gap-2 touch-manipulation text-sm min-w-0"
            >
              {uploading ? <Loader2 className="w-4 h-4 animate-spin shrink-0" /> : <Camera className="w-4 h-4 shrink-0" />} <span className="truncate">{uploading ? 'Subiendo...' : 'Capturar y subir'}</span>
            </button>
            <div className="flex gap-2 min-w-0">
              <button
                type="button"
                onClick={toggleFacing}
                disabled={uploading}
                title="Cambiar cámara frontal/trasera"
                className="flex-1 sm:flex-none px-3 sm:px-5 py-2.5 min-h-[40px] sm:min-h-[48px] bg-slate-800 text-white rounded-full hover:bg-slate-700 disabled:opacity-50 text-xs sm:text-sm border border-slate-700 touch-manipulation inline-flex items-center justify-center gap-1.5 min-w-0"
              >
                <RefreshCw className="w-3.5 h-3.5 shrink-0" /> Girar
              </button>
              <button
                type="button"
                onClick={stopCamera}
                disabled={uploading}
                className="flex-1 sm:flex-none px-3 sm:px-5 py-2.5 min-h-[40px] sm:min-h-[48px] bg-slate-800 text-white rounded-full hover:bg-slate-700 disabled:opacity-50 text-xs sm:text-sm border border-slate-700 touch-manipulation inline-flex items-center justify-center gap-1.5 min-w-0"
              >
                <X className="w-3.5 h-3.5 shrink-0" /> Cerrar
              </button>
            </div>
          </div>
          <p className="text-[11px] text-slate-400 mt-2 text-center break-words">
            La foto se cifra con AES-256-GCM en el servidor antes de guardarse. Queda asociada a {entidadTipo} {entidadId.slice(0, 8)}…
          </p>
        </div>
      )}

      {cameraError && (
        <div className="mb-3 bg-amber-50 border border-amber-200 text-amber-800 px-3 py-2 rounded-lg text-xs flex items-start gap-2 min-w-0">
          <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" /> <span className="break-words min-w-0">{cameraError}</span>
        </div>
      )}

      {fotos.length === 0 ? (
        <p className={`${compact ? 'text-xs py-4' : 'text-sm py-6'} text-slate-400 text-center border-2 border-dashed border-slate-200 rounded-lg break-words print:hidden`}>
          No hay fotos de evidencia cargadas. Las imágenes se cifran con AES-256-GCM antes de guardarse en disco.
        </p>
      ) : (
        <>
          <div className="flex flex-wrap items-center justify-between gap-2 mb-3 print:hidden">
            <span className="text-xs text-slate-500">{fotos.length} foto{fotos.length !== 1 ? 's' : ''} · ideal para verticales usa &quot;Completa&quot;</span>
            <div className="flex items-center gap-1 bg-slate-100 rounded-full p-1">
              <button
                type="button"
                onClick={() => setViewMode('cover')}
                className={`px-2.5 py-1 rounded-full text-xs font-medium inline-flex items-center gap-1 transition ${viewMode === 'cover' ? 'bg-white shadow text-slate-900' : 'text-slate-600 hover:text-slate-800'}`}
                title="Recortar para rellenar (cubre)"
              >
                <LayoutGrid className="w-3 h-3" /> Recortado
              </button>
              <button
                type="button"
                onClick={() => setViewMode('contain')}
                className={`px-2.5 py-1 rounded-full text-xs font-medium inline-flex items-center gap-1 transition ${viewMode === 'contain' ? 'bg-white shadow text-slate-900' : 'text-slate-600 hover:text-slate-800'}`}
                title="Ver foto completa sin recorte (ideal vertical)"
              >
                <ImageIcon className="w-3 h-3" /> Completa
              </button>
              <button
                type="button"
                onClick={() => setViewMode('masonry')}
                className={`px-2.5 py-1 rounded-full text-xs font-medium inline-flex items-center gap-1 transition ${viewMode === 'masonry' ? 'bg-white shadow text-slate-900' : 'text-slate-600 hover:text-slate-800'}`}
                title="Mosaico altura automática"
              >
                <Rows3 className="w-3 h-3" /> Mosaico
              </button>
            </div>
          </div>

          {viewMode === 'masonry' ? (
            <div className={`gallery-masonry ${compact ? 'columns-2 gap-2 space-y-2' : 'columns-2 sm:columns-3 md:columns-4 gap-3 space-y-3'} print:columns-3 print:gap-3`}>
              {fotos.map((foto, idx) => (
                <div key={foto.id} className="gallery-card group border rounded-lg overflow-hidden bg-white relative shadow-sm min-w-0 break-inside-avoid mb-2 sm:mb-3 print:break-inside-avoid print:shadow-none print:border">
                  <img
                    src={rawUrl(foto.id)}
                    alt={foto.nombre_original}
                    onClick={() => openPreview(idx)}
                    className="gallery-img w-full h-auto object-contain bg-white cursor-zoom-in"
                    loading="lazy"
                  />
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDelete(foto.id);
                    }}
                    disabled={deletingId === foto.id}
                    title="Borrar foto errónea"
                    aria-label={`Borrar ${foto.nombre_original}`}
                    className="absolute top-1.5 right-1.5 bg-white/95 hover:bg-rose-600 hover:text-white text-slate-700 border border-slate-200 hover:border-rose-600 rounded-full w-7 h-7 sm:w-8 sm:h-8 flex items-center justify-center shadow-md transition opacity-100 sm:opacity-0 sm:group-hover:opacity-100 focus:opacity-100 disabled:opacity-50 disabled:cursor-not-allowed print:hidden"
                  >
                    {deletingId === foto.id ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Trash2 className="w-3.5 h-3.5" />}
                  </button>
                  <div className="p-2 bg-white border-t border-slate-100 min-w-0">
                    <p className="text-xs font-semibold text-slate-700 truncate">{foto.nombre_original}</p>
                    <p className="text-[10px] text-slate-400 mt-0.5 flex items-center gap-1 min-w-0">
                      <Clock className="w-3 h-3 shrink-0" /> <span className="truncate">{new Date(foto.cargado_en).toLocaleString('es-EC')}</span>
                    </p>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className={`gallery-grid ${compact ? 'grid grid-cols-2 gap-2' : 'grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3 sm:gap-4'} print:grid-cols-3 print:gap-3`}>
              {fotos.map((foto, idx) => (
                <div key={foto.id} className="gallery-card group border rounded-lg overflow-hidden bg-slate-50 relative shadow-sm min-w-0 break-inside-avoid print:break-inside-avoid print:shadow-none print:border">
                  <div className={`gallery-img-wrap ${viewMode === 'contain' ? 'bg-white p-1' : 'bg-slate-50'} w-full ${compact ? (viewMode === 'contain' ? 'h-32' : 'h-24') : viewMode === 'contain' ? 'h-40' : 'h-36 sm:h-40'} flex items-center justify-center overflow-hidden`}>
                    <img
                      src={rawUrl(foto.id)}
                      alt={foto.nombre_original}
                      onClick={() => openPreview(idx)}
                      className={`gallery-img w-full h-full cursor-zoom-in transition-transform duration-300 group-hover:scale-105 ${viewMode === 'contain' ? 'object-contain' : 'object-cover'}`}
                      loading="lazy"
                    />
                  </div>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDelete(foto.id);
                    }}
                    disabled={deletingId === foto.id}
                    title="Borrar foto errónea"
                    aria-label={`Borrar ${foto.nombre_original}`}
                    className="absolute top-1.5 right-1.5 bg-white/95 hover:bg-rose-600 hover:text-white text-slate-700 border border-slate-200 hover:border-rose-600 rounded-full w-7 h-7 sm:w-8 sm:h-8 flex items-center justify-center shadow-md transition opacity-100 sm:opacity-0 sm:group-hover:opacity-100 focus:opacity-100 disabled:opacity-50 disabled:cursor-not-allowed print:hidden"
                  >
                    {deletingId === foto.id ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Trash2 className="w-3.5 h-3.5" />}
                  </button>
                  <div className="p-2 bg-white border-t border-slate-100 min-w-0">
                    <p className="text-xs font-semibold text-slate-700 truncate">{foto.nombre_original}</p>
                    <p className="text-[10px] text-slate-400 mt-0.5 flex items-center gap-1 min-w-0">
                      <Clock className="w-3 h-3 shrink-0" /> <span className="truncate">{new Date(foto.cargado_en).toLocaleString('es-EC')}</span>
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}

      {previewFoto !== null && previewIndex !== null && (
        <div
          className="fixed inset-0 z-50 bg-black/85 backdrop-blur-sm flex flex-col print:hidden"
          role="dialog"
          aria-modal="true"
          aria-label={`Previsualización ${previewFoto.nombre_original}`}
          onClick={closePreview}
        >
          <div className="flex items-center justify-between gap-3 px-4 py-3 bg-black/60 text-white border-b border-white/10 shrink-0">
            <div className="min-w-0 flex-1">
              <p className="text-sm font-semibold truncate">{previewFoto.nombre_original}</p>
              <p className="text-xs text-white/60 truncate flex items-center gap-1.5">
                <Clock className="w-3.5 h-3.5" /> {new Date(previewFoto.cargado_en).toLocaleString('es-EC')} · {previewIndex + 1} / {fotos.length}
              </p>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(previewFoto.id);
                }}
                disabled={deletingId === previewFoto.id}
                className="hidden sm:inline-flex items-center gap-1.5 text-xs bg-white/10 hover:bg-rose-600 border border-white/20 hover:border-rose-600 px-3 py-2 rounded-full transition disabled:opacity-50"
                title="Borrar esta foto"
              >
                {deletingId === previewFoto.id ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />} Borrar
              </button>
              <button
                type="button"
                onClick={closePreview}
                className="w-10 h-10 rounded-full bg-white text-slate-900 hover:bg-slate-100 flex items-center justify-center shrink-0"
                aria-label="Cerrar previsualización"
                title="Cerrar (Esc)"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
          </div>

          <div className="flex-1 relative flex items-center justify-center p-4 sm:p-8 overflow-hidden" onClick={(e) => e.stopPropagation()}>
            {fotos.length > 1 && (
              <>
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    goPrev();
                  }}
                  className="absolute left-2 sm:left-4 z-10 w-11 h-11 sm:w-12 sm:h-12 rounded-full bg-white/95 hover:bg-white text-slate-900 shadow-lg flex items-center justify-center transition active:scale-95"
                  aria-label="Anterior"
                  title="Anterior (←)"
                >
                  <ChevronLeft className="w-6 h-6" />
                </button>
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    goNext();
                  }}
                  className="absolute right-2 sm:right-4 z-10 w-11 h-11 sm:w-12 sm:h-12 rounded-full bg-white/95 hover:bg-white text-slate-900 shadow-lg flex items-center justify-center transition active:scale-95"
                  aria-label="Siguiente"
                  title="Siguiente (→)"
                >
                  <ChevronRight className="w-6 h-6" />
                </button>
              </>
            )}

            <img
              src={rawUrl(previewFoto.id)}
              alt={previewFoto.nombre_original}
              className="max-w-[92vw] sm:max-w-[88vw] max-h-[72vh] sm:max-h-[78vh] object-contain rounded-lg shadow-2xl bg-black"
              onClick={(e) => e.stopPropagation()}
            />
          </div>

          <div className="px-4 py-3 bg-black/60 border-t border-white/10 flex justify-center gap-2 shrink-0" onClick={(e) => e.stopPropagation()}>
            <span className="text-xs text-white/70">Usa ← → para navegar · Esc para cerrar · Click fuera para cerrar</span>
          </div>
        </div>
      )}
    </div>
  );
}
