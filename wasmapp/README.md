# wasmapp — motor de ejercicios en Go, compilado a WebAssembly

Plan "Go → WASM" (ver conversación / `docs/roadmap.md` Fase 5): que Go corra en el navegador
como el "cerebro" del motor de ejercicios, sin agregar ningún servidor — sigue siendo 100%
estático, coherente con la decisión de arquitectura del proyecto (`docs/architecture.md`,
punto 3).

- **Milestone 1** (commit `bf2a076`): solo el tipo `mc`, expuesto en `window.mcEngine`.
- **Milestone 2** (este estado): generalizado a los 4 tipos de `EXERCISE_QUEUES` —
  `mc`, `gapfill`, `truefalse`, `ordering` — expuestos en `window.exerciseEngine`
  (`mcEngine` ya no existe, se renombró porque dejó de ser solo mc).

## Alcance actual (no es el motor completo todavía)

- Los 4 tipos de ejercicio están soportados, con la MISMA lógica de corrección que
  `prototype/index.html` (mismo trim+lowercase en `gapfill`, mismo join con espacio simple
  en `ordering`, misma comparación estricta en `truefalse`) — no es una reinterpretación,
  es el mismo contrato de datos.
- Probado con las **5 preguntas reales de A1** que ya existen en el prototipo
  (`wasmapp/demo.html` las reutiliza tal cual — 2 mc, 2 gapfill, 1 ordering — no son datos
  inventados).
- Agrega algo que el prototipo HTML no tiene: **progreso persistido** entre sesiones, vía
  `localStorage` (intentos totales + mejor puntaje por nivel, sobre toda la cola mixta de
  ejercicios). Esto es un adelanto de la Fase 5 del roadmap (PWA), que pedía justamente esto.
- Todavía **no incluye** repetición espaciada (SM-2) para vocabulario — es una pieza
  separada del motor de ejercicios, pendiente.
- Todavía **no está integrado** en `prototype/index.html` — vive aparte en `demo.html` a
  propósito, para no arriesgar el prototipo que ya funciona mientras se valida el toolchain.
  Integrarlo es el siguiente paso, no este.

## Cómo compilar

Requiere Go instalado (probado con go1.26; el `go.mod` declara `go 1.21` como mínimo).

```powershell
pwsh wasmapp/build.ps1
```

Genera `wasmapp/dist/app.wasm` y `wasmapp/dist/wasm_exec.js` (el runtime glue que trae el
propio Go — la ubicación de este archivo cambió entre versiones, el script busca en las dos
rutas conocidas).

## Cómo probar

`fetch()` de un `.wasm` no funciona sobre `file://` en la mayoría de los navegadores (CORS),
así que hace falta un servidor estático mínimo:

```powershell
cd wasmapp
python -m http.server 8000
# abrir http://localhost:8000/demo.html
```

Contestá las 2 preguntas, recargá la página: el mensaje de progreso de abajo debe mostrar el
intento anterior guardado en `localStorage` — esa persistencia es justamente lo nuevo que
aporta esta pieza respecto al prototipo HTML/JS puro.

## API expuesta en `window.exerciseEngine`

Todas las funciones reciben/devuelven JSON como string (no objetos JS), para poder
inspeccionarlas fácilmente desde la consola del navegador:

| Función | Uso |
|---|---|
| `load(levelKey, itemsJSON)` | Carga una cola de ejercicios mixta `[{id,type,...},...]` para ese nivel (ver campos por tipo abajo). Devuelve `{loaded, progress}`. |
| `current()` | Ejercicio activo, con solo los campos públicos de su tipo (sin la respuesta correcta), o `{done:true}`. |
| `answer(payloadJSON)` | Corrige el ejercicio activo. El payload depende del tipo (ver tabla abajo). Devuelve `{correct, correctAnswer, score, total}`. |
| `next()` | Avanza al siguiente. Al terminar la cola, persiste el resultado. Devuelve `{done}`. |
| `progress(levelKey)` | Progreso histórico guardado de ese nivel, sin necesitar un `load()` previo. |

Campos de entrada y payload de respuesta por tipo:

| Tipo | Campos en `load()` | Payload de `answer()` |
|---|---|---|
| `mc` | `prompt`, `options[]`, `correct` (índice) | `{"selectedIndex": 0}` |
| `gapfill` | `before`, `after`, `answer`, `hint` | `{"text": "ho"}` |
| `truefalse` | `statement`, `boolAnswer` | `{"selected": true}` |
| `ordering` | `words[]`, `correctSentence` | `{"words": ["Io","sono","italiano"]}` |

## Próximos pasos (no incluidos en este milestone)

1. Agregar un scheduler de repetición espaciada (SM-2) para vocabulario — no existe hoy en
   ningún lado del proyecto. Es una pieza separada del motor de ejercicios (opera sobre
   `REAL_VOCAB`, no sobre `EXERCISE_QUEUES`).
2. Reemplazar el bloque de JS correspondiente en `prototype/index.html` por llamadas a
   `exerciseEngine`, en vez de vivir en un `demo.html` aparte.
3. Envolver todo en PWA (manifest + service worker) y desplegar a GitHub Pages / Cloudflare
   Pages (hosting gratis, sin backend) — Fase 5 completa del roadmap.
