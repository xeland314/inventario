import React, { useEffect, useMemo, useState } from 'react';

interface Mantenimiento {
  id: string;
  dispositivo_id: string;
  realizado_por: string;
  descripcion: string;
  costo: number;
  fecha_hora: string;
  url_pdf?: string | null;
  creado_en: string;
}

interface Plan {
  id: string;
  dispositivo_id?: string | null;
  tipo?: string | null;
  cada_dias: number;
  proxima_fecha: string;
  descripcion: string;
  activo: number;
}

type CalendarEvent = {
  date: string; // YYYY-MM-DD
  kind: 'mantenimiento' | 'plan';
  title: string;
  subtitle: string;
  color: string;
  raw: Mantenimiento | Plan;
};

function toDateOnly(s: string): string {
  // mantenimientos: "2026-09-15 15:15:39" or ISO; planes: "2026-09-16"
  if (s.includes('T')) return s.slice(0, 10);
  if (s.includes(' ')) return s.slice(0, 10);
  return s.slice(0, 10);
}

export default function CalendarioMantenimiento() {
  const [mants, setMants] = useState<Mantenimiento[]>([]);
  const [planes, setPlanes] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [show, setShow] = useState<'todos' | 'mantenimientos' | 'planes'>('todos');
  const [current, setCurrent] = useState(() => {
    const now = new Date();
    return new Date(now.getFullYear(), now.getMonth(), 1);
  });
  const [selectedDay, setSelectedDay] = useState<string | null>(null);

  const fetchAll = () => {
    setLoading(true);
    Promise.all([
      fetch('/api/mantenimientos').then((r) => (r.ok ? r.json() : { mantenimientos: [] })),
      fetch('/api/planes?activo=1').then((r) => (r.ok ? r.json() : { planes: [] })),
    ])
      .then(([dm, dp]) => {
        setMants(dm.mantenimientos || []);
        setPlanes(dp.planes || []);
        setError(null);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchAll(); }, []);

  const events: CalendarEvent[] = useMemo(() => {
    const evs: CalendarEvent[] = [];
    for (const m of mants) {
      evs.push({
        date: toDateOnly(m.fecha_hora),
        kind: 'mantenimiento',
        title: m.descripcion.slice(0, 60),
        subtitle: `${m.realizado_por} · $${m.costo}`,
        color: 'bg-emerald-600',
        raw: m,
      });
    }
    for (const p of planes) {
      evs.push({
        date: toDateOnly(p.proxima_fecha),
        kind: 'plan',
        title: p.descripcion.slice(0, 60),
        subtitle: `${p.tipo || 'Plan'} · cada ${p.cada_dias}d`,
        color: p.proxima_fecha < new Date().toISOString().slice(0, 10) ? 'bg-rose-600' : 'bg-sky-600',
        raw: p,
      });
    }
    return evs;
  }, [mants, planes]);

  const filtered = useMemo(() => {
    if (show === 'mantenimientos') return events.filter((e) => e.kind === 'mantenimiento');
    if (show === 'planes') return events.filter((e) => e.kind === 'plan');
    return events;
  }, [events, show]);

  const eventsByDate = useMemo(() => {
    const map = new Map<string, CalendarEvent[]>();
    for (const e of filtered) {
      if (!map.has(e.date)) map.set(e.date, []);
      map.get(e.date)!.push(e);
    }
    return map;
  }, [filtered]);

  const year = current.getFullYear();
  const month = current.getMonth();
  const firstDay = new Date(year, month, 1).getDay(); // 0 Sun
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const monthLabel = current.toLocaleDateString('es-EC', { month: 'long', year: 'numeric' });
  const todayStr = new Date().toISOString().slice(0, 10);

  const cells: (number | null)[] = [];
  for (let i = 0; i < firstDay; i++) cells.push(null);
  for (let d = 1; d <= daysInMonth; d++) cells.push(d);
  while (cells.length % 7 !== 0) cells.push(null);

  if (loading) return <div className="text-center py-12 text-slate-400">Cargando calendario...</div>;
  if (error) return <div className="bg-amber-50 border border-amber-200 text-amber-800 px-4 py-3 rounded-xl">⚠️ {error}</div>;

  const dayEvents = selectedDay ? eventsByDate.get(selectedDay) || [] : [];

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex flex-wrap gap-2 justify-between items-center bg-white border border-slate-200 rounded-2xl p-3">
        <div className="flex items-center gap-2">
          <button onClick={() => setCurrent(new Date(year, month - 1, 1))} className="w-9 h-9 grid place-items-center border border-slate-200 rounded-full hover:bg-slate-50">‹</button>
          <span className="font-black text-slate-900 capitalize min-w-[160px] text-center">{monthLabel}</span>
          <button onClick={() => setCurrent(new Date(year, month + 1, 1))} className="w-9 h-9 grid place-items-center border border-slate-200 rounded-full hover:bg-slate-50">›</button>
          <button onClick={() => { const n = new Date(); setCurrent(new Date(n.getFullYear(), n.getMonth(), 1)); setSelectedDay(n.toISOString().slice(0, 10)); }} className="ml-2 text-xs bg-slate-900 text-white px-3 py-1.5 rounded-full font-bold">Hoy</button>
        </div>
        <div className="flex items-center gap-2">
          <select value={show} onChange={(e) => setShow(e.target.value as any)} className="border border-slate-200 rounded-full px-3 py-1.5 text-xs font-semibold">
            <option value="todos">Todos</option>
            <option value="mantenimientos">Solo mantenimientos</option>
            <option value="planes">Solo planes</option>
          </select>
          <a href="/api/planes/calendario.ics" className="text-xs bg-white border border-slate-200 px-3 py-1.5 rounded-full hover:bg-slate-50">⬇ ICS</a>
        </div>
      </div>

      {/* Legend */}
      <div className="flex flex-wrap gap-2 text-xs">
        <span className="inline-flex items-center gap-1.5 bg-emerald-50 border border-emerald-200 px-2.5 py-1 rounded-full"><span className="w-2 h-2 bg-emerald-600 rounded-full" /> Mantenimiento realizado</span>
        <span className="inline-flex items-center gap-1.5 bg-sky-50 border border-sky-200 px-2.5 py-1 rounded-full"><span className="w-2 h-2 bg-sky-600 rounded-full" /> Plan próximo</span>
        <span className="inline-flex items-center gap-1.5 bg-rose-50 border border-rose-200 px-2.5 py-1 rounded-full"><span className="w-2 h-2 bg-rose-600 rounded-full" /> Plan vencido</span>
        <span className="text-slate-500">· Click en día para detalle · {filtered.length} eventos filtrados · {mants.length} mant + {planes.length} planes</span>
      </div>

      {/* Google Calendar style grid */}
      <div className="bg-white border border-slate-200 rounded-2xl overflow-hidden">
        <div className="grid grid-cols-7 bg-slate-50 border-b border-slate-200 text-[11px] font-bold tracking-widest text-slate-500 uppercase">
          {['Dom', 'Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb'].map((d) => (
            <div key={d} className="px-2 py-2 text-center border-r border-slate-200 last:border-r-0">{d}</div>
          ))}
        </div>
        <div className="grid grid-cols-7 auto-rows-[110px] sm:auto-rows-[130px]">
          {cells.map((d, idx) => {
            if (d === null) return <div key={idx} className="border-r border-b border-slate-100 bg-slate-50/50" />;
            const dateStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
            const evs = eventsByDate.get(dateStr) || [];
            const isToday = dateStr === todayStr;
            const isSelected = dateStr === selectedDay;
            return (
              <button
                key={idx}
                onClick={() => setSelectedDay(dateStr)}
                className={`text-left border-r border-b p-1.5 flex flex-col gap-1 hover:bg-sky-50/50 transition overflow-hidden ${isSelected ? 'ring-2 ring-sky-500 ring-inset bg-sky-50' : ''} ${isToday ? 'bg-amber-50' : 'bg-white'}`}
              >
                <span className={`text-xs font-bold w-6 h-6 grid place-items-center rounded-full ${isToday ? 'bg-slate-900 text-white' : isSelected ? 'bg-sky-600 text-white' : 'text-slate-700'}`}>{d}</span>
                <div className="space-y-1 overflow-hidden">
                  {evs.slice(0, 3).map((e, i) => (
                    <span key={i} className={`block text-[10px] leading-tight px-1.5 py-1 rounded-md text-white truncate ${e.color}`}>{e.title.slice(0, 28)}</span>
                  ))}
                  {evs.length > 3 && <span className="block text-[10px] text-slate-500 font-semibold">+{evs.length - 3} más</span>}
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Day detail drawer */}
      {selectedDay && (
        <div className="bg-white border border-slate-200 rounded-2xl p-5">
          <div className="flex justify-between items-start gap-3">
            <div>
              <h3 className="font-black text-slate-900">📅 {selectedDay}</h3>
              <p className="text-xs text-slate-500">{dayEvents.length} evento{dayEvents.length !== 1 ? 's' : ''} · click en equipo para ir a ficha</p>
            </div>
            <button onClick={() => setSelectedDay(null)} className="text-xs border border-slate-200 px-3 py-1.5 rounded-full hover:bg-slate-50">Cerrar ×</button>
          </div>
          {dayEvents.length === 0 ? (
            <p className="text-sm text-slate-400 mt-4">Sin mantenimientos ni planes ese día.</p>
          ) : (
            <div className="mt-4 space-y-2">
              {dayEvents.map((e, i) => {
                const isMant = e.kind === 'mantenimiento';
                const mant = e.raw as Mantenimiento;
                const plan = e.raw as Plan;
                return (
                  <div key={i} className={`border rounded-xl p-3 flex gap-3 ${isMant ? 'border-emerald-200 bg-emerald-50/50' : 'border-sky-200 bg-sky-50/50'}`}>
                    <span className={`mt-1 w-2 h-2 rounded-full shrink-0 ${e.color}`} />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-bold text-slate-900 truncate">{e.title}</p>
                      <p className="text-xs text-slate-600 truncate">{e.subtitle}</p>
                      <p className="text-xs text-slate-500 mt-1 line-clamp-2">{isMant ? (mant.descripcion) : plan.descripcion}</p>
                      {isMant && mant.url_pdf && <a href={mant.url_pdf} target="_blank" rel="noreferrer" className="text-xs text-sky-600 underline">📄 Ver PDF</a>}
                    </div>
                    <div className="shrink-0 flex flex-col gap-1">
                      <span className={`text-[10px] font-bold px-2 py-1 rounded-full text-white text-center ${e.color}`}>{isMant ? 'MANT' : 'PLAN'}</span>
                      <a href={`/dispositivos/${isMant ? mant.dispositivo_id : plan.dispositivo_id || ''}`} className={`text-xs px-2 py-1 rounded-full border text-center ${isMant ? 'bg-white border-emerald-200' : 'bg-white border-sky-200'} ${!isMant && !plan.dispositivo_id ? 'opacity-50 pointer-events-none' : ''}`}>
                        Ficha →
                      </a>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
          <div className="mt-4 flex gap-2 text-xs">
            <a href="/estadisticas" className="bg-slate-900 text-white px-3 py-1.5 rounded-full">📊 Estadísticas</a>
            <button onClick={fetchAll} className="bg-white border border-slate-200 px-3 py-1.5 rounded-full">↻ Recargar</button>
          </div>
        </div>
      )}

      {/* Lista cronológica opcional */}
      <div className="bg-white border border-slate-200 rounded-2xl p-5">
        <h3 className="font-bold text-slate-900">Próximos planes</h3>
        <p className="text-xs text-slate-500">Orden por proxima_fecha</p>
        <div className="mt-3 space-y-2 max-h-[320px] overflow-auto">
          {planes.slice(0, 20).map((p) => (
            <div key={p.id} className="flex justify-between items-center border border-slate-100 rounded-xl px-3 py-2 bg-slate-50">
              <div className="min-w-0 flex-1">
                <p className="text-xs font-bold truncate">{p.descripcion}</p>
                <p className="text-[11px] text-slate-500">{p.proxima_fecha} · {p.tipo || '—'} · cada {p.cada_dias}d</p>
              </div>
              <span className={`text-xs font-bold px-2 py-1 rounded-full text-white shrink-0 ml-2 ${p.proxima_fecha < todayStr ? 'bg-rose-600' : 'bg-sky-600'}`}>{p.proxima_fecha}</span>
            </div>
          ))}
          {planes.length === 0 && <p className="text-xs text-slate-400">Sin planes activos — crea uno en Dispositivo o vía POST /api/planes</p>}
        </div>
      </div>
    </div>
  );
}
