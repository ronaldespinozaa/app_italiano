# Italian Club App — migración de contenido a app móvil por nivel

Migración del contenido educativo de [onlineitalianclub.com](https://onlineitalianclub.com/) a
una experiencia de app móvil organizada por nivel CEFR (A1–C2), sin dependencia de WordPress
en producción. Ver `docs/architecture.md` para las decisiones de diseño.

## Estado del proyecto (actualizado en esta sesión)

| Fase | Estado |
|---|---|
| Arquitectura y motor de ejercicios (4 tipos: mc, gapfill, truefalse, ordering) | ✅ Completo, ver `prototype/index.html` |
| Inventario de contenido (6 niveles, 756 páginas catalogadas) | ✅ Completo, ver `content/master-content.json` |
| Vocabulario (89 listas) | ✅ Migrado |
| Lecciones (54 títulos — confirmado: son bundles de enlaces, no cuerpo propio) | ✅ Migrado |
| Gramática — texto real (82 explicaciones) | ✅ 82/82 (100%) verbatim del sitio (vía `grammar-scraper.go` + `merge_scraper_output.py`) |
| Listening (175 audios reales + transcripciones) | ✅ 175/175 migrado (vía `listening-scraper.go`) — audio real por streaming desde SoundCloud, no offline (ver `docs/architecture.md` decisión 7) |
| Ejercicios interactivos | 🟢 1174 ítems reales en `EXERCISE_QUEUES` (23 curados a mano + 1151 migrados de `/free_italian_exercises/*.html`, 216 páginas). 914 más migrados pero sin nivel confirmado (`content/exercises-sinnivel.json`). ~123 preguntas tipo "matching" + ~7 páginas de texto con huecos múltiples, sin convertir — ver detalle abajo |

### Cobertura real de gramática por nivel

*(actualizado el 2026-08-10 — se cerraron los últimos 4 ítems pendientes con acceso real a
internet: se corrigió una URL rota en `verbi-modali-passato-prossimo` y se confirmaron a mano
en el índice del sitio las 3 URLs de B1 que `discoverURL` no encontraba con su regex simple.
Ver `tools/grammar-scraper.go` y `migration-log/log.json`.)*

| Nivel | Verbatim del sitio | Pendiente |
|---|---|---|
| A1 | 14/14 | 0 |
| A2 | 16/16 | 0 |
| B1 | 12/12 | 0 |
| B2 | 13/13 | 0 |
| C1 | 14/14 | 0 |
| C2 | 13/13 | 0 |
| **Total** | **82/82** | **0** |

Ningún ítem en `content/grammar-*.json` conserva ya el campo `"status"` a nivel de ítem — todos
tienen contenido 100% verbatim del sitio. `tools/grammar-scraper.go` queda como herramienta
reproducible (con las URLs confirmadas) por si el sitio cambia contenido en el futuro y hay que
re-verificar algo.

### Ejercicios legacy: qué se migró y qué no (2026-08-12)

El "333" original era un conteo de **páginas** hecho al inventariar el sitio. En la práctica son
~243 páginas reales bajo `/free_italian_exercises/*.html` (190 en el índice principal + ~53
alcanzadas siguiendo 6 páginas "hub" que solo listan enlaces a sub-páginas de conjugación, ej.
`imperfetto.html` → `imperfetto_cantare.html`, `imperfetto_essere.html`...). Cada página tiene
varias preguntas — el total real de **preguntas individuales** es 2065.

`tools/exercise_scraper.py` detecta el tipo de ejercicio por la **forma de los datos** (no por
qué script `bits/*.js` los renderiza, para cubrir también variantes embebidas sin ese include) y
convierte lo que reconoce con certeza al formato de `EXERCISE_QUEUES`:

| Resultado | Cantidad | Detalle |
|---|---|---|
| Migrado y en `prototype/index.html` | 1151 preguntas (216 páginas) | tipos `mc` y `gapfill` (y algo de `ordering`) — el índice de respuesta correcta se verificó a mano contra el código fuente real de `bits/uncountable3.js` y `bits/dropdown.js` antes de confiar en la conversión masiva |
| Migrado pero **sin nivel CEFR confirmado** | 914 preguntas (82 páginas) | `content/exercises-sinnivel.json` — el nombre de archivo no indica nivel (ej. `Cantare.html`, `Essere.html`, drills de conjugación individuales) y no se quiso adivinar; no está embebido en el prototipo todavía |
| No convertido — falta un tipo de componente | ~123 preguntas | ejercicios "matching" (unir pares) — el motor actual solo tiene 4 tipos (ver `docs/architecture.md` punto 4), matching sería un 5º |
| No convertido — forma de datos distinta | ~7 páginas | textos con 10-15 huecos numerados en un solo párrafo (`gappedtext*.js`); no es una pregunta por ítem como el resto, requiere partir el párrafo por cada posición de hueco — no implementado |

El feedback de las 1151 preguntas migradas es genérico ("¡Correcto!" / la respuesta correcta),
no la explicación pedagógica en español que sí tienen los 23 ítems originales hechos a mano —
el sitio no provee esa explicación, así que no se inventó una. Detalle completo (página por
página, motivo exacto de cada omisión) en `tools/exercise-scraper-report.json`.

## Estructura del repositorio

```
content/
  level-*-index.json     Índice por nivel: conteo de lecciones, gramática, vocabulario,
                          listening y ejercicios (extraído de los índices del sitio)
  grammar-*.json         Contenido REAL de cada explicación de gramática (título + texto)
  listening-*.json       Contenido REAL de cada ejercicio de listening (transcripción +
                          `soundcloud_track_id` — el sitio no aloja mp3 propios, ver
                          docs/architecture.md decisión 7)
  exercises-*.json       Preguntas reales migradas de /free_italian_exercises/*.html, ya
                          en el formato de EXERCISE_QUEUES (uno por nivel + "sinnivel"
                          para las que no tienen nivel CEFR confirmado)
  master-content.json    Consolidado de los 6 niveles + resumen numérico

prototype/
  index.html             Prototipo funcional mobile-first (HTML/CSS/JS puro, sin build step).
                          Abrir directo en un navegador móvil. Usa los datos de content/
                          embebidos como constantes JS (ver sección "REAL_VOCAB",
                          "LESSON_CONTENT", "GRAMMAR_CONTENT", "LISTENING_CONTENT",
                          "EXERCISE_QUEUES" dentro del archivo).

tools/
  grammar-scraper.go      Descarga las páginas de gramática restantes con acceso directo
                           al HTML (no depende de un buscador). Incluye descubrimiento
                           automático de URL para los ítems de B1 sin URL confirmada.
                           USO: go run grammar-scraper.go > grammar-final.json
  listening-scraper.go    Descubre y descarga los ejercicios de listening: transcripción +
                           ID del track de SoundCloud embebido en cada página. Escribe
                           directo en content/listening-<nivel>.json (no requiere merge).
                           USO: go run tools/listening-scraper.go
  exercise_scraper.py     Descubre, descarga y convierte los ejercicios legacy (mc/gapfill/
                           ordering) al formato de EXERCISE_QUEUES. En Python, no Go —
                           el problema es parsear varias plantillas de un motor de quiz
                           JS viejo, no solo bajar texto (ver docstring del archivo).
                           Escribe content/exercises-<nivel>.json + un reporte de qué
                           se omitió y por qué. USO: python3 tools/exercise_scraper.py
  level-content-service.go  Microservicio HTTP de ejemplo (cachea contenido por nivel).
                             Su rol quedó obsoleto tras decidir migración estática — se
                             conserva como referencia de arquitectura alternativa.

migration-log/
  log.json                Registro de problemas encontrados durante la migración manual.

docs/
  architecture.md         Decisiones de UX/arquitectura tomadas durante el proyecto.
  roadmap.md               Plan de fases completo, incluyendo lo pendiente (listening, ejercicios).
```

## Cómo continuar

1. ~~Cerrar el 5% de gramática que falta~~ — **hecho el 2026-08-10**: gramática está 100%
   (82/82) verbatim del sitio. También se sincronizó `prototype/index.html` (el
   `GRAMMAR_CONTENT` embebido) para que refleje exactamente el contenido de
   `content/grammar-*.json` — antes tenía ~30 ítems con paráfrasis o placeholders propios,
   ahora es un espejo 1:1 del contenido migrado.
2. ~~Listening~~ — **hecho el 2026-08-10**: 175/175 ítems migrados (`tools/listening-scraper.go`
   + `content/listening-*.json`), con transcripción verbatim y audio real reproducible en
   `prototype/index.html` (`openListening()`, embed de SoundCloud con carga diferida por
   ítem). Limitación conocida y documentada: el audio requiere internet (streaming desde
   SoundCloud, no se descargó ningún binario) — ver `docs/architecture.md` decisión 7. El
   conteo real (175) difiere del estimado inicial (198); corregido en cada `level-*-index.json`.
3. ~~Ejercicios legacy~~ — **hecho en gran parte el 2026-08-12**: 1151 preguntas migradas y
   embebidas en `EXERCISE_QUEUES` (ver tabla arriba). Queda pendiente, en orden de impacto:
   (a) asignar nivel a mano a las 914 preguntas de `content/exercises-sinnivel.json` (82
   páginas, sobre todo drills de conjugación individuales) y embeberlas; (b) un 5º tipo de
   componente "matching" para ~123 preguntas de unir pares; (c) un parser para las ~7 páginas
   de texto con huecos múltiples (`gappedtext*.js`) que reparta cada hueco en un ítem `gapfill`.
4. **Sacar el prototipo de HTML plano** a una PWA real con Workbox (offline-first) cuando
   el contenido esté más completo — ver `docs/roadmap.md`, fase 6. El módulo de listening
   necesitará resolver su dependencia de internet (SoundCloud) antes de poder llamarse
   "100% offline" — ver decisión 7 en `docs/architecture.md`.

## Principio de honestidad de datos

Ningún archivo de este repo afirma tener contenido que no fue verificado. Si un ítem no se
pudo extraer del sitio original, su campo `status` lo dice explícitamente. Mantener esta
disciplina al añadir contenido nuevo — es lo que hace que el 56% ya migrado sea confiable
para publicar, en vez de tener que re-verificar todo desde cero más adelante.
