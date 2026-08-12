# Decisiones de arquitectura

## 1. El nivel CEFR es el contenedor raíz, no un filtro

La app no tiene una pantalla que mezcle los 6 niveles. Elegir un nivel reemplaza por completo
el contenido visible (gramática, listening, vocabulario, diálogos, verbos, lectura de ESE
nivel), no aplica un filtro sobre una lista compartida. Cambiar de nivel es una transición de
contexto completo, reforzada visualmente en `prototype/index.html` con la recarga total de
las tarjetas de módulo.

## 2. La experiencia se diferencia por nivel, no solo el contenido

| | A1 | B2+ |
|---|---|---|
| Idioma de interfaz | Español/inglés | Italiano |
| Densidad de pantalla | 1 concepto por pantalla | Vistas tipo artículo, más densas |
| Feedback | Inmediato, con refuerzo positivo | Diferido/analítico |
| Audio | Lento, con transcripción visible | Velocidad nativa, sin transcripción |
| Gramática | Oculta, solo como "tip" | Consultable como referencia |

Un mismo componente (ej. el módulo de listening) recibe distintas props según el nivel
(`velocidad_audio`, `mostrar_transcripcion`, etc.) — no son componentes duplicados.

## 3. Migración estática, sin WordPress en producción

Decisión tomada a mitad de proyecto: en vez de un microservicio que consulta la REST API de
WordPress en cada request (`tools/level-content-service.go`, ahora legado), todo el contenido
se migra UNA VEZ a archivos JSON estáticos que viajan embebidos en la app. Consecuencias:

- Cero coste de hosting de backend
- La app funciona 100% offline una vez instalada/cacheada
- WordPress puede apagarse sin romper nada
- El "costo" es que el contenido no se actualiza automáticamente si cambia en el sitio
  original — cualquier cambio futuro requiere re-migrar manualmente ese contenido

## 4. Motor de ejercicios: 5 tipos genéricos, no 333 piezas de código

Los ~333 ejercicios legacy del sitio (formato HTML/JS antiguo con un motor de quiz propio de
hace más de una década) se mapean a 5 tipos de componente reutilizable:

- `mc` — opción múltiple
- `gapfill` — rellenar hueco
- `truefalse` — verdadero/falso
- `ordering` — ordenar palabras en una frase
- `matching` — unir pares (agregado el 2026-08-12, ver abajo)

Cada ejercicio migrado es un objeto de datos con un campo `type`; el componente que lo
renderiza es el mismo sin importar el nivel o el tema gramatical. Ver `EXERCISE_QUEUES` en
`prototype/index.html` para el formato exacto de cada tipo.

Confirmado en la migración real del 2026-08-12 (`tools/exercise_scraper.py`, ver
`docs/roadmap.md` Fase 3): de 2170 preguntas legacy reales descubiertas, 749 mc + 472 gapfill
+ 6 ordering mapearon limpio a los tipos que ya existían. El "emparejar" (`newmatching*.js` en
el sitio) resultó ser real y no marginal — 123 preguntas en 5 páginas — así que se agregó como
**5º tipo de componente** en vez de forzarlo a otro tipo o descartarlo: `{type:'matching',
pairs:[{left,right},...]}`, con UI de dos columnas (izquierda fija, derecha barajada) donde se
toca un término de cada lado para intentar la pareja — igual que `ordering`, el conjunto
completo de pares de una página es UN ítem de la cola, no una fila por par. Ver
`pickMatchLeft`/`pickMatchRight` en `prototype/index.html`.

Las páginas de texto con huecos múltiples en un solo párrafo (`gappedtext*.js`) resultaron ser
un caso aparte: no son "una pregunta con 4 campos" como las demás, son un párrafo con 10-15
huecos numerados. Se resolvieron *sin* agregar un 6º tipo — cada hueco se convierte en su propio
ítem `gapfill` independiente (antes/después = el texto real alrededor de ese hueco), fragmentando
la vista continua del párrafo en varios ítems sueltos en vez de mantenerla como una sola
experiencia de lectura. Es una decisión pragmática, no la más fiel a la experiencia original del
sitio — si en el futuro importa esa continuidad, ahí sí haría falta un tipo de dato nuevo
("párrafo con lista de huecos", no un array plano de ejercicios).

## 5. Vocabulario decreciente por nivel (hallazgo real, no decisión de producto)

El sitio original reduce el vocabulario estructurado a medida que sube el nivel: A1 (32
listas) → A2 (18) → B1 (18) → B2 (18) → C1 (3) → C2 (0). La app hereda este patrón
fielmente — C2 no muestra vocabulario inventado para rellenar la pantalla; muestra un mensaje
honesto explicando que ese nivel se centra 100% en gramática y matices.

## 6. Priorización híbrida de migración (no vertical ni horizontal puras)

- **Vertical pura** (completar un nivel al 100% antes de tocar el siguiente) retrasa a todos
  los usuarios de niveles altos.
- **Horizontal pura** (un poco de cada tipo de contenido en los 6 niveles a la vez) deja a
  todos con una app a medias.
- **Híbrido elegido**: priorizar A1/A2 (mayoría de usuarios reales, según la pirámide típica
  de aprendizaje de idiomas) y, dentro de cada nivel, priorizar texto barato (lecciones,
  gramática, vocabulario) antes que contenido caro (audio, ejercicios reescritos).

## 7. Audio de listening: streaming desde SoundCloud, no archivos propios

Hallazgo real (no decisión de producto): el sitio original no aloja archivos `.mp3` propios
para los 175 ejercicios de listening — cada página embebe un reproductor de SoundCloud
(`<iframe src="https://w.soundcloud.com/player/?url=.../tracks/<id>...">`) apuntando a un
track alojado en la cuenta de SoundCloud del autor.

Decisión: la app migra el **ID del track de SoundCloud** de cada ítem (`content/listening-*.json`,
campo `soundcloud_track_id`), no el binario de audio. `prototype/index.html` reproduce el audio
real incrustando el mismo iframe de SoundCloud (carga diferida: solo al abrir cada ítem, no los
~40 de golpe al abrir el nivel). Consecuencias:

- **A favor**: cero costo de hosting/ancho de banda propio para 175 audios; siempre la fuente
  original, sin re-subir contenido de terceros a una cuenta propia sin permiso.
- **En contra**: rompe el objetivo de "100% offline" de la Fase 5 (PWA) para este módulo
  específico — el audio de listening SIEMPRE requiere internet, a diferencia de gramática,
  lecciones y vocabulario, que sí quedan embebidos y offline. Esto queda documentado como
  limitación conocida, no oculta — ver README y `docs/roadmap.md` Fase 2.
- Coherente con el principio de honestidad de datos (README): el conteo real de items de
  listening descubiertos en el índice del sitio (175) es distinto del estimado inicial (198)
  registrado en la primera pasada de inventario; se corrigió en `content/level-*-index.json`
  en vez de mantener la cifra vieja.
- Por diseño, y en línea con la decisión 2 de este documento (mismo componente, distintas
  props por nivel): A1/A2 muestran la transcripción abierta por defecto (apoyo de texto);
  B1 en adelante la esconden detrás de un `<details>` ("intenta primero sin leerla"),
  reforzando el objetivo de comprensión auditiva a velocidad nativa sin depender del texto.
