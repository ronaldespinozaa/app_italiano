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

## 4. Motor de ejercicios: 4 tipos genéricos, no 333 piezas de código

Los ~333 ejercicios legacy del sitio (formato HTML/JS antiguo con un motor de quiz propio de
hace más de una década) se mapean a 4 tipos de componente reutilizable:

- `mc` — opción múltiple
- `gapfill` — rellenar hueco
- `truefalse` — verdadero/falso
- `ordering` — ordenar palabras en una frase

Cada ejercicio migrado es un objeto de datos con un campo `type`; el componente que lo
renderiza es el mismo sin importar el nivel o el tema gramatical. Ver `EXERCISE_QUEUES` en
`prototype/index.html` para el formato exacto de cada tipo.

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
