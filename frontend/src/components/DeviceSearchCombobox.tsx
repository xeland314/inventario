import React, { useState, useEffect, useRef } from 'react';

interface Dispositivo {
  id: string;
  codigo_activo: string;
  numero_serie: string;
  tipo: string;
  marca: string;
  modelo: string;
  estado: string;
  ubicacion: string;
}

interface Props {
  value: string;
  onChange: (id: string, dispositivo?: Dispositivo) => void;
  placeholder?: string;
  label?: string;
  required?: boolean;
}

export default function DeviceSearchCombobox({ value, onChange, placeholder = 'Buscar compu, disco, GGM-... o pegar UUID', label, required }: Props): React.JSX.Element {
  const [query, setQuery] = useState<string>(value || '');
  const [open, setOpen] = useState<boolean>(false);
  const [results, setResults] = useState<Dispositivo[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [selected, setSelected] = useState<Dispositivo | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<number | null>(null);

  // Sincroniza value externo con query y carga label si es UUID
  useEffect(() => {
    setQuery(value || '');
    if (value && !selected) {
      // intenta cargar dispositivo por ID exacto para mostrar label
      fetch(`/api/dispositivos/${value}`)
        .then((r) => (r.ok ? r.json() : null))
        .then((d) => {
          if (d && d.id) setSelected(d as Dispositivo);
        })
        .catch(() => {});
    } else if (!value) {
      setSelected(null);
    }
  }, [value]);

  useEffect(() => {
    const handler = (e: MouseEvent): void => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const fetchSearch = async (q: string): Promise<void> => {
    if (!q || q.trim().length < 2) {
      setResults([]);
      return;
    }
    setLoading(true);
    try {
      const res = await fetch(`/api/dispositivos?q=${encodeURIComponent(q)}&limit=20`);
      const data = (await res.json()) as { dispositivos: Dispositivo[] };
      setResults(data.dispositivos || []);
    } catch {
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const handleInput = (v: string): void => {
    setQuery(v);
    setOpen(true);
    // si es UUID completo, avisa al padre aunque no esté en lista
    onChange(v, undefined);
    setSelected(null);
    if (debounceRef.current) window.clearTimeout(debounceRef.current);
    debounceRef.current = window.setTimeout(() => {
      void fetchSearch(v);
    }, 250);
  };

  const handleSelect = (d: Dispositivo): void => {
    setSelected(d);
    setQuery(d.id); // guarda ID real, pero muestra label
    onChange(d.id, d);
    setOpen(false);
  };

  const displayValue = selected ? `${selected.codigo_activo} — ${selected.tipo} ${selected.marca} ${selected.modelo}` : query;

  // Si hay query pero no selected, muestra query tal cual (permite pegar UUID)
  // Si hay selected, muestra label pero mantiene ID en value
  const inputValue = selected ? displayValue : query;

  return (
    <div ref={containerRef} className="relative">
      {label && <label className="text-xs font-semibold text-slate-600 block mb-1">{label}{required && ' *'}</label>}
      <div className="relative">
        <input
          value={inputValue}
          onChange={(e) => handleInput(e.target.value)}
          onFocus={() => {
            setOpen(true);
            if (query) void fetchSearch(query);
          }}
          placeholder={placeholder}
          className="w-full border border-slate-200 rounded-xl px-3 py-3 min-h-[44px] text-sm font-mono focus:border-sky-500 focus:outline-none pr-10"
          required={required}
        />
        <span className="absolute right-3 top-3.5 text-slate-400 text-xs">⌕</span>
        {selected && (
          <button
            type="button"
            onClick={() => {
              setSelected(null);
              setQuery('');
              onChange('');
            }}
            className="absolute right-8 top-3 text-slate-400 hover:text-slate-600 text-xs"
            aria-label="Limpiar"
          >
            ✕
          </button>
        )}
      </div>

      {open && (query.length >= 2 || results.length > 0) && (
        <div className="absolute z-50 mt-1 w-full bg-white border border-slate-200 rounded-xl shadow-lg max-h-72 overflow-auto">
          {loading && <div className="p-3 text-xs text-slate-400">Buscando “{query}”...</div>}
          {!loading && results.length === 0 && query.length >= 2 && (
            <div className="p-3 text-xs text-slate-500">
              Sin resultados para “{query}”. Puedes pegar el UUID completo y usarlo igual.
              <div className="mt-2 font-mono text-[11px] bg-slate-50 p-2 rounded border break-all">{query}</div>
            </div>
          )}
          {!loading &&
            results.map((d) => (
              <button
                key={d.id}
                type="button"
                onClick={() => handleSelect(d)}
                className="w-full text-left px-3 py-2.5 hover:bg-slate-50 border-b border-slate-100 last:border-0 flex flex-col gap-0.5"
              >
                <span className="text-sm font-semibold text-slate-900 truncate">
                  {d.codigo_activo} <span className="font-normal text-slate-500">· {d.tipo}</span>
                </span>
                <span className="text-xs text-slate-600 truncate">
                  {d.marca} {d.modelo} — {d.ubicacion}
                </span>
                <span className="text-[11px] font-mono text-slate-400 truncate">{d.id} · {d.estado}</span>
              </button>
            ))}
          <div className="p-2 bg-slate-50 border-t text-[11px] text-slate-400 text-center">
            Escribe “compu”, “disco”, “GGM” o pega el UUID. Funciona también con ID completo.
          </div>
        </div>
      )}

      {selected && (
        <p className="text-[11px] text-emerald-700 mt-1">✓ Seleccionado: {selected.codigo_activo} ({selected.id.slice(0, 8)}…)</p>
      )}
    </div>
  );
}
