#!/usr/bin/env python3
# Loader inventario desde data\GGM-* YAMLs (ignora NORIO*) hacia API http://localhost:8080/api
import os, json, pathlib, requests, shutil, time

API = "http://localhost:8080/api"
BASE = pathlib.Path(r"C:\Users\ASUS\workspace\informes-tecnicos\informes-tecnicos")
OUTPUT = BASE / "output"
UPLOAD_DST = pathlib.Path(r"C:\Users\ASUS\workspace\inventory\backend\uploads\pdf")
UPLOAD_DST.mkdir(parents=True, exist_ok=True)

# Definicion dispositivos deseados (sin NORIO)
dispositivos = [
    {
        "codigo_activo": "GGM-COM-014",
        "numero_serie": "DESKTOP-JVJUAKJ",
        "tipo": "Desktop",
        "marca": "Biostar",
        "modelo": "i3-4160 8GB DDR3 - Hikvision HS-SSD-E100 128GB + Samsung HD322HJ 320GB",
        "especificaciones": "SSD Hikvision HS-SSD-E100 128GB falla controladora CM_PROB_PHANTOM (35->46 phantom). Kernel-Power 41, Event 6008, Kernel-Boot 29 0xC00000D4. Probado todos puertos SATA + cambio ramal alimentacion sin exito. En frio revive, warm reboot falla. Validacion cruzada otra placa exoneró Biostar. Dictamen: falla controladora definitiva. Usuario en PC respaldo.",
        "valor_compra": 0,
        "proveedor": "No especificado",
        "ubicacion": "GGM Matriz",
        "asignado_a": "GGM Matriz",
        "fecha_compra": "2026-09-10",
        "estado": "en_mantenimiento",
        "pdf": OUTPUT / "Informe_Tecnico_Final_GGM-COM-014_Dictamen_SSD_2026-09-15.pdf",
        "mant_desc": "Diagnóstico completo GGM-COM-014: desconexion bus SATA SSD Hikvision tras conectar HDD Samsung. Intervencion: todos puertos SATA con cables distintos + cambio alimentacion a otro ramal. Matriz verificacion + evidencia fotografica (5 fotos temp/assets/ssd). Dictamen: falla controladora SSD. Recomendacion: reemplazo 240-256GB con DRAM, recuperacion en frio (problema controlador).",
        "fecha_hora": "2026-09-15 15:15:39",
    },
    {
        "codigo_activo": "GGM-COM-025",
        "numero_serie": "DESKTOP-L3142NV",
        "tipo": "Desktop",
        "marca": "Intel",
        "modelo": "i5-14400 8GB DDR4 SSD 256GB - Realtek PCIe GbE #2 MAC 10:FF:E0:98:E2:95",
        "especificaciones": "Windows 11 Pro 64b. Ethernet Realtek 1168.27.50.920 sin enlace (LEDs apagados) por plug RJ45 funda gris desprendida y traccion pares. Tarjeta OK (test con cable bueno levanta enlace). Mitigacion Wi-Fi TP-Link MU-MIMO USB operativa. Pendiente re-ponchado T568B.",
        "valor_compra": 0,
        "proveedor": "No especificado",
        "ubicacion": "GGM Aula Híbrida",
        "asignado_a": "Innovacion Educativa",
        "fecha_compra": "2026-09-07",
        "estado": "activo",
        "pdf": OUTPUT / "Informe_Tecnico_GGM-COM-025_Ethernet.pdf",
        "mant_desc": "Falla enlace Ethernet GGM-COM-025: diagnostico plug RJ45 dañado, descarte tarjeta Realtek operativa, evidencia 5 fotos, plan re-ponchado pendiente por falta conector/ponchadora, mitigacion Wi-Fi TP-Link.",
        "fecha_hora": "2026-09-07 12:00:00",
    },
    {
        "codigo_activo": "GGM-LAP-020",
        "numero_serie": "3B004789LAP",
        "tipo": "Laptop",
        "marca": "Dell",
        "modelo": "Latitude 7390 2-in-1 i7-8650U 16GB NVMe 256GB",
        "especificaciones": "Bateria DELL N18GG65 60Wh: design 60002 mWh, actual 3792 mWh (6.3% salud, degradacion 93.7%). Apagado abrupto al 30%. Contingencia AC exclusivo + hibernacion al 35% + respaldo OneDrive. Requiere reemplazo bateria N18GG65 60Wh (no 42Wh).",
        "valor_compra": 0,
        "proveedor": "No especificado",
        "ubicacion": "GGM Matriz - Gerencia",
        "asignado_a": "Gerencia",
        "fecha_compra": "2026-09-04",
        "estado": "en_mantenimiento",
        "pdf": OUTPUT / "Informe_Tecnico_GGM-LAP-020_Contingencia.pdf",
        "mant_desc": "Contingencia bateria GGM-LAP-020: powercfg batteryreport 3792/60002 mWh, protocolo AC exclusivo, hibernacion 35%, respaldo nube. Requerimiento bateria N18GG65.",
        "fecha_hora": "2026-09-04 10:00:00",
    },
    {
        "codigo_activo": "DESKTOP-35RN9VH",
        "numero_serie": "DESKTOP-35RN9VH",
        "tipo": "Desktop",
        "marca": "Intel",
        "modelo": "i3-4170 12GB (4+8) HS-SSD-E100 128GB + Hitachi 931GB",
        "especificaciones": "Windows 10 Pro 10.0.19045. Incidencia corto 5V USB teclado AKL-520 YL PCB V1.2_CWJ_20181023 (OCP Kernel-Power 41 + 6008 09:38). 58 CM_PROB_PHANTOM HID. Tras limpieza/retoque y reemplazo teclado: 47min uptime estable CPU 6% RAM 30% SMART Healthy.",
        "valor_compra": 0,
        "proveedor": "No especificado",
        "ubicacion": "GGM Matriz",
        "asignado_a": "GGM Matriz",
        "fecha_compra": "2026-09-07",
        "estado": "activo",
        "pdf": OUTPUT / "Informe_Tecnico_GGM-Teclado-DESKTOP-35RN9VH.pdf",
        "mant_desc": "Incidencia teclado AKL-520 cortocircuito 5V en DESKTOP-35RN9VH: limpieza PCB, retoque soldaduras VCC/D-/D+/GND, descarte (solo funciona en ciertos equipos, etiquetado defectuoso), reemplazo teclado compatible, 47min estable.",
        "fecha_hora": "2026-09-07 09:38:00",
    },
    {
        "codigo_activo": "CARGADOR-COM-01",
        "numero_serie": "CARGADOR-01-COMUNICACION",
        "tipo": "Cargador",
        "marca": "Generico",
        "modelo": "USB-A 5V nominal (medido 1.6V DC)",
        "especificaciones": "Verificacion multimetro V-DC en 2 tomas 110V AC. Cargador #1 Comunicacion entrega 1.6V vs >=5.0V requerido (-68%). Falla regulacion no reparable. Dar de baja y etiquetar FUERA DE SERVICIO 1.6V. Evidencia Fig1.",
        "valor_compra": 0,
        "proveedor": "No especificado",
        "ubicacion": "Comunicacion",
        "asignado_a": "Comunicacion",
        "fecha_compra": "2026-09-10",
        "estado": "dado_de_baja",
        "motivo_baja": "Falla regulacion 1.6V DC (68% caida) - dar de baja",
        "pdf": OUTPUT / "Informe_Tecnico_Cargadores_Comunicacion.pdf",
        "mant_desc": "Verificacion voltaje Cargador #1 Comunicacion: 1.6V DC, no operativo, testigo >=5V valida instrumento. Dar de baja.",
        "fecha_hora": "2026-09-10 12:00:00",
    },
    {
        "codigo_activo": "CARGADOR-COM-02",
        "numero_serie": "CARGADOR-02-COMUNICACION",
        "tipo": "Cargador",
        "marca": "Generico",
        "modelo": "USB-A 5V nominal (medido 1.6V DC)",
        "especificaciones": "Cargador #2 mismo fallo 1.6V estable, no handshake. Requiere reemplazo 2 unidades >=5V/2A marca certificada. Evidencia Fig2.",
        "valor_compra": 0,
        "proveedor": "No especificado",
        "ubicacion": "Comunicacion",
        "asignado_a": "Comunicacion",
        "fecha_compra": "2026-09-10",
        "estado": "dado_de_baja",
        "motivo_baja": "Falla regulacion 1.6V DC - dar de baja",
        "pdf": OUTPUT / "Informe_Tecnico_Cargadores_Comunicacion.pdf",
        "mant_desc": "Verificacion Cargador #2: 1.6V DC, mismo fallo que #1, fuera de servicio.",
        "fecha_hora": "2026-09-10 12:00:00",
    },
    {
        "codigo_activo": "CARGADOR-TESTIGO-OK",
        "numero_serie": "CARGADOR-03-TESTIGO",
        "tipo": "Cargador",
        "marca": "Generico",
        "modelo": "USB-A >=5V DC control",
        "especificaciones": "Cargador testigo que si vale >=5V DC dentro spec 5.0-5.25V. Valida instrumento y toma AC. Conservar como respaldo temporal hasta reposicion 2 unidades. Evidencia Fig3 ampliada.",
        "valor_compra": 0,
        "proveedor": "No especificado",
        "ubicacion": "Comunicacion",
        "asignado_a": "Soporte Tecnico",
        "fecha_compra": "2026-09-10",
        "estado": "activo",
        "pdf": OUTPUT / "Informe_Tecnico_Cargadores_Comunicacion.pdf",
        "mant_desc": "Cargador testigo OK >=5V DC, control validacion instrumento/toma, conservar como respaldo.",
        "fecha_hora": "2026-09-10 12:00:00",
    },
    {
        "codigo_activo": "IMPRESORA-VICSAN",
        "numero_serie": "IMPRESORAVICSAN-SN",
        "tipo": "Impresora",
        "marca": "Epson",
        "modelo": "EcoTank L3250",
        "especificaciones": "Hostname IMPRESORAVICSAN, SSID Intertelnet1661, DHCP ping OK, pagina estado impresa. Renombrado via Epson Web Config, instalacion por hostname/WSD. Etiquetado fisico. Reserva DHCP recomendada.",
        "valor_compra": 0,
        "proveedor": "No especificado",
        "ubicacion": "Vicsan",
        "asignado_a": "Vicsan",
        "fecha_compra": "2026-09-10",
        "estado": "activo",
        "pdf": OUTPUT / "Informe_Tecnico_Impresoras_Renombrado_Red.pdf",
        "mant_desc": "Renombrado y reconfiguracion Wi-Fi IMPRESORAVICSAN L3250 a Intertelnet1661, validacion hostname/SSID/impresion, etiquetado.",
        "fecha_hora": "2026-09-10 10:00:00",
    },
    {
        "codigo_activo": "IMPRESORA-ICON",
        "numero_serie": "ICON-L3250-001",
        "tipo": "Impresora",
        "marca": "Epson",
        "modelo": "EcoTank L3250",
        "especificaciones": "Hostname IMPRESORAICON (Icon), SSID OPEN-VIXAN, MainPad previo 6354/6346 (ya reseteado), ahora operativa tras renombrado. Misma fisica que ICON-L3250-001 de mantenimiento almohadillas. Reserva DHCP recomendada.",
        "valor_compra": 0,
        "proveedor": "No especificado",
        "ubicacion": "Icon",
        "asignado_a": "Icon",
        "fecha_compra": "2026-09-04",
        "estado": "activo",
        "pdf": OUTPUT / "Informe_Tecnico_Impresoras_Renombrado_Red.pdf",
        "mant_desc": "Renombrado IMPRESORAICON L3250 a OPEN-VIXAN (ya con almohadillas reemplazadas y MainPad 6354->0). Operativa.",
        "fecha_hora": "2026-09-10 10:00:00",
    },
    {
        "codigo_activo": "DGIMPRESORA",
        "numero_serie": "DGIMPRESORA-L1250-NUEVA",
        "tipo": "Impresora",
        "marca": "Epson",
        "modelo": "EcoTank L1250",
        "especificaciones": "DGIMPRESORA L1250 nueva, hostname DGIMPRESORA, SSID OPEN-VIXAN, uso exclusivo administrativo (OU Administracion), no compartir con Icon/Norio/Vicsan. Alta nueva etiquetada.",
        "valor_compra": 0,
        "proveedor": "No especificado",
        "ubicacion": "Administracion DG",
        "asignado_a": "Direccion General",
        "fecha_compra": "2026-09-10",
        "estado": "activo",
        "pdf": OUTPUT / "Informe_Tecnico_Impresoras_Renombrado_Red.pdf",
        "mant_desc": "Alta nueva DGIMPRESORA L1250 en OPEN-VIXAN, uso exclusivo administrativo, etiquetado y registro inventario.",
        "fecha_hora": "2026-09-10 10:00:00",
    },
    # ISABEL opcional - personal, se carga si se desea inventario personal
    # {
    #     "codigo_activo": "ISABEL-3501",
    #     "numero_serie": "ISABEL-Dell-3501",
    #     "tipo": "Laptop",
    #     "marca": "Dell",
    #     "modelo": "Inspiron 3501 i3-1115G4 12GB SSD",
    #     "especificaciones": "Resource Hijacking FAHConsole.exe C:\\Program Files\\WinZip\\FAHConsole.exe SHA256 CD0AE... persistencia HKLM Run WinZip FAH. CPU 45% throttling 1.54GHz -> tras desinfeccion 3.28-4.10GHz. Remediacion Remove-ItemProperty + desinstal WinZip.",
    #     "valor_compra": 0,
    #     "proveedor": "No especificado",
    #     "ubicacion": "Personal - Tia Isabel",
    #     "asignado_a": "Tia Isabel",
    #     "fecha_compra": "2026-09-04",
    #     "estado": "activo",
    #     "pdf": OUTPUT / "Informe_Tecnico_TIA-ISABEL-3501_Seguridad.pdf",
    #     "mant_desc": "Desinfeccion ISABEL-3501: FAHConsole resource hijacking, purge HKLM Run, desinstalacion WinZip, CPU desbloqueada 1.54->3.28GHz.",
    #     "fecha_hora": "2026-09-04 12:00:00",
    # },
]

def get_dispositivos():
    r = requests.get(f"{API}/dispositivos", timeout=10)
    r.raise_for_status()
    return r.json().get("dispositivos", [])

def ensure_pdf_copied(src: pathlib.Path):
    if not src.exists():
        print(f"WARN pdf no existe: {src}")
        return None
    dst = UPLOAD_DST / src.name
    # copiar si no existe o tamaño difiere
    if not dst.exists() or dst.stat().st_size != src.stat().st_size:
        shutil.copy2(src, dst)
        print(f"  PDF copiado: {src.name} -> {dst}")
    else:
        print(f"  PDF ya existe: {dst}")
    # API lo sirve como /uploads/pdf/<name>
    return f"/uploads/pdf/{src.name}"

def main():
    existentes = get_dispositivos()
    idx = {d["codigo_activo"]: d for d in existentes}
    print(f"Existentes: {len(existentes)}")
    for c in existentes:
        print(f" - {c['codigo_activo']} id={c['id']} estado={c['estado']}")

    for disp in dispositivos:
        codigo = disp["codigo_activo"]
        pdf_path = disp.pop("pdf")
        mant_desc = disp.pop("mant_desc")
        fecha_hora = disp.pop("fecha_hora")
        motivo_baja = disp.pop("motivo_baja", None)

        print(f"\n=== {codigo} ===")
        disp_id = None
        if codigo in idx:
            disp_id = idx[codigo]["id"]
            print(f"Ya existe, id={disp_id}, se verificará mantenimiento")
            # actualizar especificaciones si hace falta (PATCH)
            try:
                # opcional PUT para sincronizar especificaciones
                pass
            except Exception as e:
                print(f"  update warn: {e}")
        else:
            # crear
            payload = {k: v for k, v in disp.items() if v is not None}
            # api espera asignado_a nullable
            print(f"Creando {codigo} ... payload={json.dumps(payload, ensure_ascii=False)}")
            r = requests.post(f"{API}/dispositivos", json=payload, timeout=10)
            if r.status_code != 201:
                print(f"ERROR crear {codigo}: {r.status_code} {r.text}")
                continue
            disp_id = r.json().get("id")
            print(f"Creado {codigo} -> {disp_id}")
            # si estado es dado_de_baja, aplicar baja
            if disp.get("estado") == "dado_de_baja" and motivo_baja:
                rb = requests.patch(f"{API}/dispositivos/{disp_id}/baja", json={"motivo_baja": motivo_baja}, timeout=10)
                print(f"  Baja aplicada: {rb.status_code} {rb.text}")

        # asegurar pdf copiado
        url_pdf = ensure_pdf_copied(pdf_path) if pdf_path else None

        # verificar mantenimientos existentes
        rm = requests.get(f"{API}/mantenimientos", params={"dispositivo_id": disp_id}, timeout=10)
        rm.raise_for_status()
        mantenimientos = rm.json().get("mantenimientos", [])
        # si ya existe uno con misma descripcion (contiene codigo), skip
        ya_tiene = any(codigo in (m.get("descripcion") or "") for m in mantenimientos)
        if ya_tiene and url_pdf and any(m.get("url_pdf")==url_pdf for m in mantenimientos):
            print(f"  Mantenimiento ya existe para {codigo}, skip")
            continue
        if ya_tiene and not url_pdf:
            print(f"  Mantenimiento ya existe (sin pdf) skip")
            continue

        # crear mantenimiento con PDF upload si hay pdf
        if pdf_path and pdf_path.exists() and url_pdf:
            # usar multipart con pdf
            print(f"  Creando mantenimiento con PDF {pdf_path.name}")
            with open(pdf_path, "rb") as f:
                files = {"pdf": (pdf_path.name, f, "application/pdf")}
                data = {
                    "dispositivo_id": disp_id,
                    "realizado_por": "Christopher Villamarin",
                    "descripcion": f"{codigo}: {mant_desc}",
                    "costo": "0",
                    "fecha_hora": fecha_hora,
                }
                r2 = requests.post(f"{API}/mantenimientos", data=data, files=files, timeout=30)
                print(f"    -> {r2.status_code} {r2.text[:500]}")
                if r2.status_code not in (201,200):
                    # fallback JSON con url_pdf
                    print("    Fallback JSON con url_pdf")
                    r2 = requests.post(f"{API}/mantenimientos", json={
                        "dispositivo_id": disp_id,
                        "realizado_por": "Christopher Villamarin",
                        "descripcion": f"{codigo}: {mant_desc}",
                        "costo": 0,
                        "fecha_hora": fecha_hora,
                        "url_pdf": url_pdf
                    }, timeout=10)
                    print(f"    fallback -> {r2.status_code} {r2.text[:500]}")
        else:
            print(f"  Creando mantenimiento JSON sin PDF")
            r2 = requests.post(f"{API}/mantenimientos", json={
                "dispositivo_id": disp_id,
                "realizado_por": "Christopher Villamarin",
                "descripcion": f"{codigo}: {mant_desc}",
                "costo": 0,
                "fecha_hora": fecha_hora,
                "url_pdf": url_pdf
            }, timeout=10)
            print(f"    -> {r2.status_code} {r2.text[:500]}")

        time.sleep(0.2)

    # resumen final
    print("\n=== RESUMEN FINAL ===")
    final = get_dispositivos()
    print(json.dumps(final, indent=2, ensure_ascii=False))
    for d in final:
        rm = requests.get(f"{API}/mantenimientos", params={"dispositivo_id": d["id"]}, timeout=10).json().get("mantenimientos", [])
        print(f"{d['codigo_activo']}: {len(rm)} mantenimientos")
        for m in rm:
            print(f"  - {m['fecha_hora']} | {m['descripcion'][:80]} | pdf={m['url_pdf']}")

if __name__ == "__main__":
    main()
