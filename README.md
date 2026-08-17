# Italian Club App — migración de contenido a app móvil por nivel

Migración del contenido educativo de [onlineitalianclub.com](https://onlineitalianclub.com/) a
una experiencia de app móvil organizada por nivel CEFR (A1–C2), sin dependencia de WordPress
en producción. Ver `docs/architecture.md` para las decisiones de diseño.

## Estado del proyecto (actualizado 2026-08-17)

| Fase | Estado |
|---|---|
| Arquitectura y motor de ejercicios (5 tipos: mc, gapfill, truefalse, ordering, matching) | ✅ Completo, ver `prototype/index.html` |
| Inventario de contenido (6 niveles, 756 páginas catalogadas) | ✅ Completo, ver `content/master-content.json` |
| Vocabulario (89 listas) | ✅ Migrado |
| Lecciones (54 títulos — confirmado: son bundles de enlaces, no cuerpo propio) | ✅ Migrado |
| Gramática — texto real (82 explicaciones) | ✅ 82/82 (100%) verbatim del sitio (vía `grammar-scraper.go` + `merge_scraper_output.py`) |
| Listening (175 audios reales + transcripciones) | ✅ 175/175 migrado (vía `listening-scraper.go`) — audio real por streaming desde SoundCloud, no offline (ver `docs/architecture.md` decisión 7) |
| Ejercicios interactivos | ✅ 2193 ítems reales en `EXERCISE_QUEUES` (23 curados a mano + 2170 migrados de `/free_italian_exercises/*.html`, 237/237 páginas convertibles). 100% con nivel CEFR asignado — ver detalle abajo |
| Apagar WordPress (Fase 4) | ✅ Completo — con las 4 fases de arriba al 100%, la app no depende de WordPress para nada. `tools/level-content-service.go` (arquitectura alternativa descartada) archivado en `tools/archive/` |
| PWA instalable + progreso persistido (Fase 5) | ✅ Completo — manifest + service worker, progreso granular por ítem en `localStorage` |
| Motor de ejercicios en Go/WASM (`wasmapp/`) | ✅ **Milestone 4 — es el motor real de la app**, no una demo aparte. El JS de corrección en `prototype/index.html` ya no existe: `renderExercise`/`answerMC`/`answerGapfill`/`answerTF`/`pickWord`/`pickMatchRight` llaman a `window.exerciseEngine` (`app.wasm`), cacheado offline por `prototype/sw.js`. Ver `wasmapp/README.md` |
| Repetición espaciada (SM-2) para vocabulario | ✅ **Milestone 5 de `wasmapp/`** — `window.vocabEngine` reemplaza el ciclo secuencial que tenían las flashcards. Algoritmo SM-2 puro y testeado en `wasmapp/sm2.go`/`sm2_test.go`, estado persistido en la misma clave de `localStorage` que ya usa `exerciseEngine` |
| CI del binario `.wasm` | ✅ `.github/workflows/wasm-build.yml` recompila y comitea `app.wasm`/`wasm_exec.js` en cada push que toque `wasmapp/*.go` |

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
varias preguntas — el total real de **preguntas individuales** es 2170, en **237/237 páginas
convertibles** (0 omitidas — ver debajo por qué no son 243: ~6 páginas hub no cuentan como
"convertibles" en sí mismas, solo aportan las sub-páginas que sí lo son).

`tools/exercise_scraper.py` detecta el tipo de ejercicio por la **forma de los datos** (no por
qué script `bits/*.js` los renderiza, para cubrir también variantes embebidas sin ese include):

| Tipo legacy → tipo del motor | Preguntas | Estado |
|---|---|---|
| Selección múltiple (`correctArray`) → `mc` | 749 | ✅ migrado |
| Rellenar hueco de texto libre → `gapfill` | 472 | ✅ migrado (incluye los huecos de texto tipo cloze, ver abajo) |
| Reordenar frase → `ordering` | 6 | ✅ migrado |
| Unir pares → `matching` | 5 grupos / 123 pares | ✅ migrado — **5º tipo de componente nuevo**, agregado en el motor junto con esta migración (ver `docs/architecture.md` punto 4) |

De esas 1355 preguntas (749+472+6+123, contando cada par de matching), 1232 tenían nivel CEFR
confirmable por el nombre de archivo; las otras 938 (82 páginas — sobre todo drills de
conjugación verbal individuales sin nivel en el nombre, ej. `Cantare.html`, `Essere.html`,
alcanzados desde páginas hub) no lo tenían.

**Clasificación manual de esas 938 (2026-08-12)**: se clasificaron una por una por página de
origen (no por título — varios títulos se repiten entre tiempos verbales distintos, ej. "Essere"
existe como drill de passato remoto Y de congiuntivo presente, con nivel distinto cada vez).
Criterio: match directo contra el tema ya clasificado en `content/grammar-*.json` cuando existe
(ej. "Aggettivi possessivi" → A2 porque así está en `grammar-a2.json`); para los ~53 drills de
conjugación sin tema propio, por tiempo verbal según la progresión CEFR estándar del italiano,
también anclada en `grammar-*.json` (presente indicativo → A1, imperfetto/condizionale/futuro
semplice → A2, congiuntivo presente → B1, passato remoto → B2). El criterio completo, página por
página, queda documentado en `migration-log/log.json`. **Las 2170 preguntas están embebidas en
`prototype/index.html`; `content/exercises-sinnivel.json` quedó vacío (0 pendientes).**

Los textos con 10-15 huecos numerados en un solo párrafo (`gappedtext*.js`, ~7 páginas) se
convirtieron partiendo el párrafo por cada posición de hueco: cada hueco es un ítem `gapfill`
independiente (antes/después = el texto real que lo rodea en la página), no una vista de
párrafo continuo — pragmático, sigue siendo 100% contenido real, solo que fragmentado.

El feedback de las preguntas migradas es genérico ("¡Correcto!" / la respuesta correcta), no la
explicación pedagógica en español que sí tienen los 23 ítems originales hechos a mano — el sitio
no provee esa explicación, así que no se inventó una. Detalle completo (página por página) en
`tools/exercise-scraper-report.json`.

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
                           ordering/matching, + huecos múltiples partidos en gapfill) al
                           formato de EXERCISE_QUEUES. En Python, no Go — el problema es
                           parsear varias plantillas de un motor de quiz JS viejo, no solo
                           bajar texto (ver docstring del archivo).
                           Escribe content/exercises-<nivel>.json + un reporte de qué
                           se omitió y por qué. USO: python3 tools/exercise_scraper.py
  exercise_level_classification.py  Registro del criterio usado para clasificar a mano las
                           938 preguntas de content/exercises-sinnivel.json que quedaron sin
                           nivel CEFR por nombre de archivo (drills de conjugación verbal
                           sobre todo). No es una herramienta para re-correr — es la
                           documentación de la decisión, ya aplicada.

  archive/
    level-content-service.go  Microservicio HTTP de ejemplo (cachea contenido por nivel de
                               la REST API de WordPress). Archivado en la Fase 4 del roadmap
                               — la app ya no depende de WordPress para nada; se conserva
                               como referencia de la arquitectura alternativa que se descartó
                               (ver docs/architecture.md decisión 3).

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
3. ~~Ejercicios legacy~~ — **hecho el 2026-08-12, en dos pasadas**. Primera: 1232 preguntas
   migradas de `/free_italian_exercises/*.html` (237/237 páginas convertibles, 0 omitidas), más
   **matching** agregado como 5º tipo de componente del motor (UI + lógica de corrección en
   `prototype/index.html`, `pickMatchLeft`/`pickMatchRight`) y un parser para los textos con
   huecos múltiples (`gappedtext*.js`, cada hueco → un ítem `gapfill`). Segunda: las 938
   preguntas que habían quedado sin nivel CEFR confirmable por el nombre de archivo se
   clasificaron a mano, página por página (criterio y detalle en `migration-log/log.json`), y se
   embebieron. Total: **2193 preguntas reales en `EXERCISE_QUEUES`** (23 curadas + 2170
   migradas), 100% con nivel asignado. `content/exercises-sinnivel.json` quedó vacío.
4. ~~Apagar WordPress~~ — **hecho el 2026-08-12**: con las Fases 1-3 al 100%, la app no
   depende de WordPress para nada. `tools/level-content-service.go` (arquitectura
   alternativa descartada) se archivó en `tools/archive/` con una nota explicando por qué.
   El sitio real (`onlineitalianclub.com`) sigue en línea de forma independiente — esto solo
   significa que la app no lo necesita.
5. ~~Sacar el prototipo de HTML plano a una PWA real~~ — **hecho el 2026-08-14** (Fase 5):
   manifest + service worker, progreso real persistido en `localStorage`. El módulo de
   listening sigue dependiendo de internet (streaming SoundCloud) — ver decisión 7 en
   `docs/architecture.md`, no cuenta como "100% offline" por eso.
6. ~~Motor Go→WASM (`wasmapp/`)~~ — **hecho el 2026-08-16** (Milestone 4): reemplazo real, ya
   no vive aparte en `wasmapp/demo.html`. `prototype/index.html` no tiene lógica de corrección
   propia; los 5 tipos de ejercicio pasan por `window.exerciseEngine` (`app.wasm`), con el
   mismo esquema de `localStorage` que usaba el JS (sin migración de datos). Detalle completo,
   incluido un bug real de doble-submit encontrado y corregido en la prueba manual, en
   `wasmapp/README.md` y `migration-log/log.json`.
7. ~~Repetición espaciada (SM-2) para vocabulario~~ — **hecho el 2026-08-17** (Milestone 5 de
   `wasmapp/`): `window.vocabEngine` reemplaza el ciclo secuencial que tenían las flashcards
   de `prototype/index.html` (los botones "Aún no la sé" / "La sé" ya existían en el markup,
   pero no estaban conectados a ningún algoritmo). Detalle completo, incluida la API expuesta,
   en `wasmapp/README.md`.
8. ~~CI que compile `wasmapp/main.go`~~ — **hecho el 2026-08-17**: `.github/workflows/wasm-build.yml`.
9. **Próximo bloque de trabajo natural**: no hay ningún pendiente documentado del roadmap
   original sin cerrar. Candidatos para lo que sigue: UI de vocabulario en `wasmapp/demo.html`
   (hoy `vocabEngine` no tiene harness de prueba standalone, a diferencia de `exerciseEngine`),
   o retomar `docs/roadmap.md` para decidir el próximo foco del producto.

## Principio de honestidad de datos

Ningún archivo de este repo afirma tener contenido que no fue verificado. Si un ítem no se
pudo extraer del sitio original, su campo `status` lo dice explícitamente. Mantener esta
disciplina al añadir contenido nuevo — es lo que hace que el 56% ya migrado sea confiable
para publicar, en vez de tener que re-verificar todo desde cero más adelante.
