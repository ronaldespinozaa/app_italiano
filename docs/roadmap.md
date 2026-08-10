# Roadmap

## Fase 0 — Prototipo y motor (✅ completo)
Prototipo HTML mobile-first con selector de nivel, 6 contenedores independientes, motor de
4 tipos de ejercicio, flashcards de vocabulario real, vista de lectura de gramática.
→ `prototype/index.html`

## Fase 1 — Migración de texto (✅ completo)
Lecciones, gramática y vocabulario de los 6 niveles. Gramática cerrada al 100% (82/82
verbatim del sitio) el 2026-08-10 — ver tabla de cobertura real en el README.
→ `content/level-*-index.json`, `content/grammar-*.json`

## Fase 2 — Listening (⏳ no iniciado)
198 audios + transcripciones identificados por URL (ver campo `modules.listening` en cada
`level-*-index.json`), pero ningún archivo de audio descargado todavía (salvo el ejemplo de
"Fare prenotazioni" usado como contenido de muestra en el motor de ejercicios).

Trabajo pendiente:
1. Extender `tools/grammar-scraper.go` (o escribir uno nuevo) para además descargar el
   archivo de audio de cada página de listening, no solo el texto.
2. Decidir dónde alojar los archivos de audio (no caben en JSON) — candidatos: CDN estático,
   o empaquetados dentro del asset bundle de la app si el tamaño total lo permite.
3. Adaptar la UI de "Listening" del prototipo (hoy usa un botón ▶ decorativo) para reproducir
   audio real.

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
