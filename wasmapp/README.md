# wasmapp — motor de ejercicios en Go, compilado a WebAssembly

Milestone 1 del plan "Go → WASM" (ver conversación / `docs/roadmap.md` Fase 5): probar de
punta a punta que Go puede correr en el navegador como el "cerebro" del motor de ejercicios,
sin agregar ningún servidor — sigue siendo 100% estático, coherente con la decisión de
arquitectura del proyecto (`docs/architecture.md`, punto 3).

## Alcance actual (no es el motor completo todavía)

- Solo el tipo `mc` (opción múltiple) de los 4 tipos que soporta `EXERCISE_QUEUES` en
  `prototype/index.html`. `gapfill`, `truefalse` y `ordering` siguen en JS por ahora.
- Solo probado con las 2 preguntas `mc` reales de A1 que ya existen en el prototipo
  (`wasmapp/demo.html` las reutiliza tal cual, no son datos inventados).
- Agrega algo que el prototipo HTML no tiene: **progreso persistido** entre sesiones, vía
  `localStorage` (intentos totales + mejor puntaje por nivel). Esto es un adelanto de la
  Fase 5 del roadmap (PWA), que pedía justamente esto.
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

## API expuesta en `window.mcEngine`

Todas las funciones reciben/devuelven JSON como string (no objetos JS), para poder
inspeccionarlas fácilmente desde la consola del navegador:

| Función | Uso |
|---|---|
| `load(levelKey, questionsJSON)` | Carga una lista de preguntas `[{id,prompt,options[],correct},...]` para ese nivel. Devuelve `{loaded, progress}`. |
| `current()` | Pregunta activa (sin revelar la respuesta correcta), o `{done:true}`. |
| `answer(selectedIndex)` | Corrige la pregunta activa. Devuelve `{correct, correctIndex, score, total}`. |
| `next()` | Avanza a la siguiente. Al terminar la ronda, persiste el resultado. Devuelve `{done}`. |
| `progress(levelKey)` | Progreso histórico guardado de ese nivel, sin necesitar un `load()` previo. |

## Próximos pasos (no incluidos en este milestone)

1. Portar `gapfill`, `truefalse`, `ordering` al mismo patrón.
2. Agregar un scheduler de repetición espaciada (SM-2) para vocabulario — no existe hoy en
   ningún lado del proyecto.
3. Reemplazar el bloque de JS correspondiente en `prototype/index.html` por llamadas a
   `mcEngine`, en vez de vivir en un `demo.html` aparte.
4. Envolver todo en PWA (manifest + service worker) y desplegar a GitHub Pages / Cloudflare
   Pages (hosting gratis, sin backend) — Fase 5 completa del roadmap.
