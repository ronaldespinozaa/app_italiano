// Sin build tag a propósito (a diferencia de main.go, que es js && wasm): la
// lógica de SM-2 no toca syscall/js ni localStorage, así que puede compilarse
// y testearse en cualquier plataforma con `go test ./wasmapp/...` — no hace
// falta un runner de WebAssembly para probar el algoritmo en sí. Ver
// sm2_test.go y wasmapp/README.md.
package main

import (
	"fmt"
	"math"
	"time"
)

// VocabItem es un par palabra/traducción, mismo shape que vocabSetForLevel()
// en prototype/index.html (word/back), indexado por posición dentro del
// nivel — igual convención que EXERCISE_QUEUES (sin ID propio, la posición
// ES el identificador).
type VocabItem struct {
	Word string `json:"word"`
	Back string `json:"back"`
}

// SRSCard es el estado SM-2 (SuperMemo 2, Piotr Wozniak, 1987 — el algoritmo
// estándar, no una reinvención) de un ítem de vocabulario para el usuario.
// EF (easiness factor) arranca en 2.5, el valor de referencia del algoritmo
// original. Due queda en "" mientras el ítem nunca se calificó.
type SRSCard struct {
	EF       float64 `json:"ef"`
	Interval int     `json:"interval"` // días hasta el próximo repaso
	Reps     int     `json:"reps"`     // repeticiones correctas consecutivas
	Due      string  `json:"due"`      // fecha ISO (YYYY-MM-DD); "" = nunca calificada
}

func newSRSCard() SRSCard {
	return SRSCard{EF: 2.5}
}

const dateLayout = "2006-01-02"

// today() usa time.Now(), que en Go/js-wasm sale del reloj real del
// navegador (Date.now()) — no hace falta syscall/js explícito para esto, por
// eso esta función puede vivir acá sin el build tag de main.go.
func today() string {
	return time.Now().Format(dateLayout)
}

// applySM2 aplica el algoritmo SM-2 a una card según la calidad de la
// respuesta (escala 0-5 del algoritmo original). La UI de prototype/
// index.html manda una versión simplificada de 2 botones ("Aún no la sé" /
// "La sé"), mapeada en main.go a quality 1 / 4 — ver comentario ahí.
//
//   - quality < 3: no se acordó. Resetea la racha (reps=0) y vuelve a
//     empezar la progresión de intervalos desde el día 1.
//   - quality >= 3: se acordó, con distinto grado de esfuerzo. El intervalo
//     sigue la progresión estándar 1 -> 6 -> anterior*EF.
//
// todayStr se recibe como parámetro (no se llama a today() acá adentro)
// para que el test pueda fijar una fecha determinística sin tocar el reloj
// del sistema.
func applySM2(card SRSCard, quality int, todayStr string) SRSCard {
	if quality < 0 {
		quality = 0
	}
	if quality > 5 {
		quality = 5
	}

	if quality < 3 {
		card.Reps = 0
		card.Interval = 1
	} else {
		switch card.Reps {
		case 0:
			card.Interval = 1
		case 1:
			card.Interval = 6
		default:
			card.Interval = int(math.Round(float64(card.Interval) * card.EF))
		}
		card.Reps++
	}

	q := float64(quality)
	card.EF = card.EF + (0.1 - (5-q)*(0.08+(5-q)*0.02))
	if card.EF < 1.3 {
		card.EF = 1.3
	}

	base, err := time.Parse(dateLayout, todayStr)
	if err != nil {
		base = time.Now()
	}
	card.Due = base.AddDate(0, 0, card.Interval).Format(dateLayout)
	return card
}

// pickVocabIndex decide qué ítem mostrar de una lista de `n` ítems (0..n-1),
// dado el estado SM-2 guardado (`srs`, clave = índice como string) y la
// fecha de hoy. Prioridad, en este orden:
//  1. ítems ya vencidos (due <= hoy), el más atrasado primero (due más
//     antiguo).
//  2. ítems nunca calificados (no están en `srs`), en orden original.
//  3. si no hay ninguno de los dos casos anteriores (todo calificado y
//     programado para el futuro), el de due más próximo.
//
// Así la cola nunca queda "vacía" — mismo espíritu que la cola infinita de
// EXERCISE_QUEUES en exerciseEngine (current.index % len(items)), adaptado
// a que acá el orden lo decide el algoritmo, no una posición secuencial.
func pickVocabIndex(srs map[string]SRSCard, n int, todayStr string) int {
	bestDueIdx, bestDueVal := -1, ""
	bestNewIdx := -1
	bestFutureIdx, bestFutureVal := -1, ""

	for i := 0; i < n; i++ {
		card, seen := srs[fmt.Sprintf("%d", i)]
		if !seen || card.Due == "" {
			if bestNewIdx == -1 {
				bestNewIdx = i
			}
			continue
		}
		if card.Due <= todayStr {
			if bestDueIdx == -1 || card.Due < bestDueVal {
				bestDueIdx, bestDueVal = i, card.Due
			}
		} else {
			if bestFutureIdx == -1 || card.Due < bestFutureVal {
				bestFutureIdx, bestFutureVal = i, card.Due
			}
		}
	}

	if bestDueIdx != -1 {
		return bestDueIdx
	}
	if bestNewIdx != -1 {
		return bestNewIdx
	}
	if bestFutureIdx != -1 {
		return bestFutureIdx
	}
	return 0
}

// vocabProgressCounts calcula seen/due/mastered sobre `total` ítems según el
// estado SM-2 guardado. `mastered` es un umbral práctico (3 repasos
// correctos consecutivos) para dar una señal de "ya lo sabés bien" en la UI
// — no es parte del algoritmo SM-2 en sí, que no define un concepto de
// "dominado".
func vocabProgressCounts(srs map[string]SRSCard, total int, todayStr string) (seen, due, mastered int) {
	for i := 0; i < total; i++ {
		card, ok := srs[fmt.Sprintf("%d", i)]
		if !ok {
			continue
		}
		seen++
		if card.Due <= todayStr {
			due++
		}
		if card.Reps >= 3 {
			mastered++
		}
	}
	return seen, due, mastered
}
