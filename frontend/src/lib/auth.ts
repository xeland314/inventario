export type Rol = "tecnico" | "lector";
export interface UsuarioSession { id: string; username: string; rol: Rol; }

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}
export function getUser(): UsuarioSession | null {
  if (typeof window === "undefined") return null;
  const raw = localStorage.getItem("usuario");
  if (!raw) return null;
  try { return JSON.parse(raw) as UsuarioSession; } catch { return null; }
}
export function setSession(token: string, usuario: UsuarioSession) {
  localStorage.setItem("token", token);
  localStorage.setItem("usuario", JSON.stringify(usuario));
}
export function clearSession() {
  localStorage.removeItem("token");
  localStorage.removeItem("usuario");
}
export async function logout() {
  const token = getToken();
  if (token) {
    try {
      await fetch("/api/auth/logout", { method: "POST", headers: { Authorization: `Bearer ${token}` } });
    } catch {}
  }
  clearSession();
  location.href = "/login";
}
export function authHeaders(): Record<string,string> {
  const token = getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}
export async function authFetch(input: RequestInfo, init: RequestInit = {}) {
  const headers = new Headers(init.headers as Record<string,string> || {});
  const token = getToken();
  if (token) headers.set("Authorization", `Bearer ${token}`);
  // nunca enviar token por query; solo header
  const res = await fetch(input, { ...init, headers });
  if (res.status === 401 && location.pathname !== "/login") {
    clearSession();
    location.href = "/login";
  }
  return res;
}
export function isTecnico(): boolean {
  const u = getUser();
  return u?.rol === "tecnico";
}
