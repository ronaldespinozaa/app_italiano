# Roadmap

## Fase 0 — Prototipo y motor (✅ completo)
Prototipo HTML mobile-first con selector de nivel, 6 contenedores independientes, motor de
4 tipos de ejercicio, flashcards de vocabulario real, vista de lectura de gramática.
→ `prototype/index.html`

## Fase 1 — Migración de texto (✅ completo)
Lecciones, gramática y vocabulario de los 6 niveles. Gramática cerrada al 100% (82/82
verbatim del sitio) el 2026-08-10 — ver tabla de cobertura real en el README.
→ `content/level-*-index.json`, `content/grammar-*.json`

## Fase 2 — Listening (✅ completo, con una limitación documentada)
175 ítems migrados el 2026-08-10 (`tools/listening-scraper.go` → `content/listening-*.json`):
transcripción verbatim + ID de track de SoundCloud por ítem. El conteo real (175) reemplaza
al estimado inicial (198) una vez confirmado contra el índice real del sitio.

Cómo se resolvieron los 3 puntos pendientes originales:
1. ~~Extender el scraper para bajar también el audio~~ — resuelto distinto de lo planeado:
   el sitio no aloja `.mp3` propios, así que no hay binario que descargar. Cada página embebe
   un iframe de SoundCloud; `listening-scraper.go` extrae el `track_id` de ese embed.
2. ~~Decidir dónde alojar los archivos de audio~~ — resuelto no alojándolos: se reproduce por
   streaming desde SoundCloud (mismo origen que el sitio actual), cero costo de hosting propio.
   Ver `docs/architecture.md`, decisión 7, incluyendo el trade-off que esto implica para offline.
3. ~~Adaptar la UI para reproducir audio real~~ — hecho: `prototype/index.html` tiene
   `openListening()` con reproductor de SoundCloud real (carga diferida por ítem) y
   transcripción — visible por defecto en A1/A2, oculta tras "Mostrar transcripción" en B1+
   (mismo patrón de props-por-nivel que gramática, ver `docs/architecture.md` decisión 2).

Pendiente real que queda (no estaba en el plan original): el módulo de listening es el único
que **no puede** funcionar offline hoy, porque depende de que SoundCloud esté disponible. Si
la Fase 5 (PWA offline-first) se toma en serio el "100% offline", este módulo necesita una
solución aparte (¿negociar con el autor una copia de los audios? ¿aceptar que listening quede
como la única excepción online?) — decisión de producto, no técnica, que todavía no se tomó.

## Fase 3 — Ejercicios interactivos (🟡 13/333 = 4%)
Los 13 ejercicios ya migrados demuestran que el motor de 4 tipos funciona en los 6 niveles.
Falta convertir los ~320 restantes del formato legacy (`/free_italian_exercises/*.html`) al
formato de datos que consume `EXERCISE_QUEUES`.

Cada tipo de ejercicio legacy debe mapearse a uno de los 4 tipos del motor:
- Selección múltiple / verdadero-falso → ya soportado directamente
- Rellenar huecos (conjugación, preposiciones, etc.) → tipo `gapfill`
- Reordenar frase/diálogo → tipo `ordering`
- Emparejar (no soportado aún) → requiere un 5º tipo de componente nuevo

## Fase 4 — Apagar WordPress
Una vez Fases 1-3 estén al 100%, WordPress deja de ser necesario como fuente de contenido.
`tools/level-content-service.go` (el microservicio que consultaba la REST API) queda obsoleto
y puede archivarse — la decisión de arquitectura fue migración estática, no proxy en vivo.

## Fase 5 — PWA instalable
Convertir `prototype/index.html` en una PWA real:
- `manifest.json` + iconos para instalación en pantalla de inicio
- Service worker (Workbox recomendado) para funcionamiento 100% offline
- `IndexedDB` para progreso persistente del usuario (hoy el prototipo no persiste nada
  entre sesiones — es solo demo de interacción)

## Fase 6 — Monetización y contenido editorial
Integrar las tarjetas de CTA externas (ebooks en easyreaders.org, clases en
nativespeakerteachers.com, escuela en Bolonia) de forma contextual, y trasladar el contenido
editorial del autor (newsletter, "Best of") a su propia sección ("Club"/"Más") sin competir
por atención con el contenido de estudio. Ver `docs/architecture.md` para el razonamiento
de por qué esto no debe eliminarse.
