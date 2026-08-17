# wasmapp — motor de ejercicios en Go, compilado a WebAssembly

Plan "Go → WASM" (ver conversación / `docs/roadmap.md` Fase 5): que Go corra en el navegador
como el "cerebro" del motor de ejercicios, sin agregar ningún servidor — sigue siendo 100%
estático, coherente con la decisión de arquitectura del proyecto (`docs/architecture.md`,
punto 3).

- **Milestone 1** (commit `bf2a076`): solo el tipo `mc`, expuesto en `window.mcEngine`.
- **Milestone 2** (commit `65eba71`): generalizado a 4 tipos (`mc`, `gapfill`, `truefalse`,
  `ordering`), expuestos en `window.exerciseEngine`.
- **Milestone 3** (este estado): mientras tanto, el motor JS de `prototype/index.html`
  avanzó en paralelo mucho más rápido — 82/82 gramática, 175/175 listening, **2193
  ejercicios reales** (5º tipo `matching` incluido), PWA completa (manifest + service
  worker) y progreso persistido granular por ítem. Este milestone pone a `wasmapp` a la
  par de esa realidad, en vez de agregar features nuevas que el JS no tiene todavía:
  - Se agregó el tipo `matching` (5º tipo).
  - La cola pasó a ser **infinita** (`current.index % len(items)`), igual que
    `exQueueIndex % queue.length` en el JS — ya no hay concepto de "ronda terminada".
  - El progreso ahora se persiste en la **misma clave de localStorage que ya usa el JS**
    (`italianClubProgress_v1`, tocando solo las sub-claves `ex`/`exCorrect`), no en una
    clave propia — así no hay migración de datos cuando se reemplace el motor JS.
  - `tools/extract_exercise_queues.js` (Node — hace falta un parser JS real, no es JSON
    válido) extrae `EXERCISE_QUEUES` de `prototype/index.html` a
    `wasmapp/data/exercises-<nivel>.json` (gitignored, se regenera; **no** es
    `content/exercises-<nivel>.json` — ese es el contenido migrado canónico en otro shape,
    no lo toca). **2193/2193 ítems extraídos y validados** (conteos por tipo cuadran con el
    README del proyecto).
  - `demo.html` ahora carga el **dataset real completo** de cualquier nivel (selector
    A1–C2), no una muestra de 5 ítems.
- **Milestone 4** (commit `f1bd86a`, este estado): **reemplazo real** — el motor JS de
  corrección en `prototype/index.html` ya no existe. `renderExercise`/`answerMC`/
  `answerGapfill`/`answerTF`/`pickWord`/`pickMatchRight` llaman a `window.exerciseEngine`
  (`app.wasm`); `prototype/sw.js` cachea el binario para uso offline (`CACHE_NAME` subido a
  v2). Bug real encontrado en la prueba manual y corregido antes del commit: ninguna de las
  5 funciones de respuesta revisaba `res.error` de `exerciseEngine.answer()` — un doble-tap
  en "Verifica" (gapfill) disparaba una segunda llamada que Go rechaza correctamente, pero
  como nadie la revisaba, se mostraba "Respuesta correcta: undefined" en el feedback. Ver
  `migration-log/log.json` para el detalle completo, incluida la reproducción contra el
  `.wasm` real en Node antes de dar el fix por bueno.

## Alcance actual

- Los 5 tipos de ejercicio están soportados, con la MISMA lógica de corrección que tenía el
  JS viejo (mismo trim+lowercase en `gapfill`, mismo join con espacio simple en `ordering`,
  misma comparación estricta en `truefalse`, mismo "sin estado incorrecto final" en
  `matching`) — no fue una reinterpretación, fue el mismo contrato de datos.
- **Es el motor real de la app** — no una demo aparte. `prototype/index.html` no tiene
  lógica de corrección propia; toda pasa por acá.
- El progreso usa la misma clave/esquema de `localStorage` que usaba el JS
  (`italianClubProgress_v1`) — sin migración de datos para los usuarios existentes.
- `demo.html` sigue existiendo como harness de prueba standalone (dataset real completo,
  útil para probar cambios en `main.go` sin cargar todo el prototipo).
- Todavía **no incluye** repetición espaciada (SM-2) para vocabulario — es una pieza
  separada del motor de ejercicios, pendiente, y tampoco existe en el JS.

## Cómo compilar

Requiere Go instalado (probado con go1.26; el `go.mod` declara `go 1.21` como mínimo).

```powershell
pwsh wasmapp/build.ps1
```

Genera `wasmapp/dist/app.wasm` y `wasmapp/dist/wasm_exec.js` (el runtime glue que trae el
propio Go — la ubicación de este archivo cambió entre versiones, el script busca en las dos
rutas conocidas).

## Cómo probar

Primero generá el dataset real (una sola vez, o cada vez que cambie `EXERCISE_QUEUES`):

```powershell
cd italian-club-app
node tools/extract_exercise_queues.js
```

`fetch()` de un `.wasm`/`.json` no funciona sobre `file://` en la mayoría de los
navegadores (CORS), así que hace falta un servidor estático:

```powershell
cd wasmapp
python -m http.server 8000
# abrir http://localhost:8000/demo.html
```

Elegí un nivel del selector, contestá algunos ejercicios (probá los 5 tipos), recargá la
página: el mensaje de progreso de abajo debe reflejar lo ya visto — es el mismo
`localStorage` (`italianClubProgress_v1`) que ya usa `prototype/index.html`, así que si
tenés progreso guardado ahí desde el prototipo real, este demo lo va a leer también.

## API expuesta en `window.exerciseEngine`

Todas las funciones reciben/devuelven JSON como string (no objetos JS), para poder
inspeccionarlas fácilmente desde la consola del navegador:

| Función | Uso |
|---|---|
| `load(levelKey, itemsJSON)` | Carga la cola de ejercicios de un nivel `[{id,type,...},...]` (ver campos por tipo abajo). Devuelve `{loaded, progress}`. |
| `current()` | Ejercicio activo (`queueIndex` + campos públicos de su tipo, sin la respuesta correcta). La cola es infinita — no hay `{done:true}`. |
| `answer(payloadJSON)` | Corrige el ejercicio activo y persiste el intento de inmediato. El payload depende del tipo (ver tabla abajo). Devuelve `{correct, correctAnswer}`. |
| `next()` | Avanza al siguiente ítem de la cola (con wraparound). |
| `progress(levelKey)` | `{seen, correct, total, pct}` de ese nivel — mismo cálculo que la parte `ex` de `progressSummary()` en el JS. `total` solo es preciso si ese nivel está cargado con `load()`. |

Campos de entrada y payload de respuesta por tipo:

| Tipo | Campos en `load()` | Payload de `answer()` |
|---|---|---|
| `mc` | `prompt`, `options[]`, `correct` (índice) | `{"selectedIndex": 0}` |
| `gapfill` | `before`, `after`, `answer`, `hint` | `{"text": "ho"}` |
| `truefalse` | `statement`, `boolAnswer` | `{"selected": true}` |
| `ordering` | `words[]`, `correctSentence` | `{"words": ["Io","sono","italiano"]}` |
| `matching` | `pairs[]` (`{left,right}`) | `{"matchesComplete": true}` — el emparejamiento en sí se valida del lado del cliente (`current()` ya expone `rightIdx` por opción shuffleada), Go solo registra que se completó |

`current()` para `matching` devuelve `lefts[]` (orden original) y `rightsShuffled[]`
(`{text, rightIdx}`, shuffleado en Go con `math/rand`) — el par correcto de `lefts[i]` es el
elemento de `rightsShuffled` cuyo `rightIdx === i`.

## Próximos pasos

1. Repetición espaciada (SM-2) para vocabulario — no existe hoy en ningún lado del proyecto.
   Es una pieza separada del motor de ejercicios (opera sobre `REAL_VOCAB`, no sobre
   `EXERCISE_QUEUES`).
2. ~~CI que compile `wasmapp/main.go` y actualice `prototype/app.wasm` automáticamente~~ —
   **hecho el 2026-08-17**: `.github/workflows/wasm-build.yml` recompila `app.wasm` +
   `wasm_exec.js` y los comitea a `prototype/` en cada push que toque `wasmapp/*.go` o
   `go.mod` (también invocable a mano vía `workflow_dispatch`). `build.ps1` sigue siendo útil
   para compilar y probar en local antes de pushear.
