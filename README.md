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
| Listening (198 audios + transcripciones) | ⏳ No iniciado |
| Ejercicios interactivos (333 legacy) | 🟡 23/333 reescritos en el motor nuevo |

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

## Estructura del repositorio

```
content/
  level-*-index.json     Índice por nivel: conteo de lecciones, gramática, vocabulario,
                          listening y ejercicios (extraído de los índices del sitio)
  grammar-*.json         Contenido REAL de cada explicación de gramática (título + texto)
  master-content.json    Consolidado de los 6 niveles + resumen numérico

prototype/
  index.html             Prototipo funcional mobile-first (HTML/CSS/JS puro, sin build step).
                          Abrir directo en un navegador móvil. Usa los datos de content/
                          embebidos como constantes JS (ver sección "REAL_VOCAB",
                          "LESSON_CONTENT", "EXERCISE_QUEUES" dentro del archivo).

tools/
  grammar-scraper.go      Descarga las páginas de gramática restantes con acceso directo
                           al HTML (no depende de un buscador). Incluye descubrimiento
                           automático de URL para los ítems de B1 sin URL confirmada.
                           USO: go run grammar-scraper.go > grammar-final.json
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
2. **Listening**: las URLs de los 198 audios están identificadas en los
   `level-*-index.json` (`modules.listening`), pero no descargadas. Requiere un scraper
   nuevo que también baje el archivo de audio, no solo el texto. Con acceso real a internet
   ya confirmado en este entorno (ver `tools/grammar-scraper.go` como referencia de patrón),
   este es el siguiente bloque de trabajo más natural.
3. **Ejercicios**: los 333 ejercicios legacy (formato HTML/JS antiguo, ver
   `/free_italian_exercises/*.html` en el sitio) deben convertirse al formato de datos
   que ya consume el motor en `prototype/index.html` (busca `EXERCISE_QUEUES`) — son 4-5
   tipos de componente reutilizables, no 333 piezas de código distintas.
4. **Sacar el prototipo de HTML plano** a una PWA real con Workbox (offline-first) cuando
   el contenido esté más completo — ver `docs/roadmap.md`, fase 6.

## Principio de honestidad de datos

Ningún archivo de este repo afirma tener contenido que no fue verificado. Si un ítem no se
pudo extraer del sitio original, su campo `status` lo dice explícitamente. Mantener esta
disciplina al añadir contenido nuevo — es lo que hace que el 56% ya migrado sea confiable
para publicar, en vez de tener que re-verificar todo desde cero más adelante.
