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
| Gramática — texto real (82 explicaciones) | 🟡 46/82 (56%) verbatim del sitio, 13 verificados de fuente externa, 23 pendientes — ver tabla abajo |
| Listening (198 audios + transcripciones) | ⏳ No iniciado |
| Ejercicios interactivos (333 legacy) | ⏳ 13 reescritos como muestra en el motor nuevo |

### Cobertura real de gramática por nivel

| Nivel | Literal del sitio | Fuente externa verificada | Pendiente (requiere scraper) |
|---|---|---|---|
| A1 | 13/14 | 0 | 1 |
| A2 | 15/16 | 0 | 1 |
| B1 | 5/12  | 7 | 0 |
| B2 | 10/13 | 3 | 0 |
| C1 | 2/14  | 3 | 9 |
| C2 | 1/13  | 0 | 12 |

Cada ítem en `content/grammar-*.json` que no tiene contenido 100% verbatim lleva un campo
`"status"` explicando por qué (fuente externa, contenido sin verificar, etc.). No se oculta
ninguna limitación — usar ese campo como checklist de QA antes de publicar contenido en producción.

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

1. **Cerrar el 100% de gramática**: correr `tools/grammar-scraper.go` con internet real
   (no funciona en entornos con red restringida). Reemplaza los `grammar-*.json` con las
   URLs pendientes resueltas.
2. **Listening**: mismo patrón — las URLs de los 198 audios están identificadas en los
   `level-*-index.json` (`modules.listening`), pero no descargadas. Requiere un scraper
   nuevo que también baje el archivo de audio, no solo el texto.
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
