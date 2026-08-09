//go:build js && wasm

// Motor de ejercicios "mc" (opción múltiple) compilado a WebAssembly.
//
// Milestone 1 del plan Go->WASM: reemplaza solo la lógica del tipo `mc` del
// motor de EXERCISE_QUEUES (ver prototype/index.html), y agrega algo que el
// prototipo HTML/JS no tiene: progreso persistido entre sesiones, vía
// localStorage. Los otros 3 tipos (gapfill, truefalse, ordering) siguen
// viviendo en JS hasta el próximo milestone — ver wasmapp/README.md.
//
// Expone en window.mcEngine: load, current, answer, next, progress.
// Todas las funciones reciben/devuelven JSON como string, no objetos JS
// directos, para mantener la interfaz simple y fácil de debuggear desde la
// consola del navegador.
package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"
)

// Question es el formato de entrada: un ítem `mc` ya aplanado (el índice de
// la opción correcta, no un booleano por opción como en EXERCISE_QUEUES, para
// no tener que reimplementar el parseo del formato legacy en esta primera
// versión).
type Question struct {
	ID      string   `json:"id"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
	Correct int      `json:"correct"`
}

// publicQuestion es lo que se expone a JS — sin el índice correcto, para no
// filtrar la respuesta por la consola del navegador.
type publicQuestion struct {
	ID      string   `json:"id"`
	Index   int      `json:"index"`
	Total   int      `json:"total"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
}

type answerResult struct {
	Error        string `json:"error,omitempty"`
	Correct      bool   `json:"correct"`
	CorrectIndex int    `json:"correctIndex"`
	Score        int    `json:"score"`
	Total        int    `json:"total"`
	Done         bool   `json:"done"`
}

// progressRecord es lo único que persiste entre sesiones (localStorage), por
// nivel. Fase 5 del roadmap (PWA) pedía justamente esto — hoy el prototipo
// HTML no guarda nada.
type progressRecord struct {
	Attempts   int    `json:"attempts"`
	BestScore  int    `json:"bestScore"`
	BestTotal  int    `json:"bestTotal"`
	LastPlayed string `json:"lastPlayedAt"`
}

type engine struct {
	levelKey  string
	questions []Question
	index     int
	score     int
	answered  bool
}

func storageKey(levelKey string) string {
	return "italianclub:progress:mc:" + levelKey
}

func loadProgress(levelKey string) progressRecord {
	var rec progressRecord
	raw := js.Global().Get("localStorage").Call("getItem", storageKey(levelKey))
	if raw.IsNull() || raw.IsUndefined() {
		return rec
	}
	_ = json.Unmarshal([]byte(raw.String()), &rec) // si está corrupto, arrancamos de cero
	return rec
}

func saveProgress(levelKey string, rec progressRecord) {
	data, err := json.Marshal(rec)
	if err != nil {
		return
	}
	js.Global().Get("localStorage").Call("setItem", storageKey(levelKey), string(data))
}

func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"error":"no se pudo serializar la respuesta: ` + err.Error() + `"}`
	}
	return string(b)
}

func errJSON(msg string) string {
	return toJSON(map[string]string{"error": msg})
}

var current *engine

// load(levelKey, questionsJSON) — reemplaza el engine activo. Devuelve
// cuántas preguntas cargó y el progreso histórico ya guardado de ese nivel.
func load(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return errJSON("uso: mcEngine.load(levelKey, questionsJSON)")
	}
	levelKey := args[0].String()

	var qs []Question
	if err := json.Unmarshal([]byte(args[1].String()), &qs); err != nil {
		return errJSON("JSON de preguntas inválido: " + err.Error())
	}
	if len(qs) == 0 {
		return errJSON("la lista de preguntas está vacía")
	}

	current = &engine{levelKey: levelKey, questions: qs}
	return toJSON(map[string]interface{}{
		"loaded":   len(qs),
		"progress": loadProgress(levelKey),
	})
}

// current() — la pregunta activa, o {"done":true} si ya se terminó la ronda.
func currentQuestion(this js.Value, args []js.Value) interface{} {
	if current == nil {
		return errJSON("no hay ejercicios cargados — llamá a load() primero")
	}
	if current.index >= len(current.questions) {
		return toJSON(map[string]bool{"done": true})
	}
	q := current.questions[current.index]
	return toJSON(publicQuestion{
		ID: q.ID, Index: current.index, Total: len(current.questions),
		Prompt: q.Prompt, Options: q.Options,
	})
}

// answer(selectedIndex) — corrige la pregunta activa. No avanza el índice:
// eso lo hace next(), para que la UI pueda mostrar el feedback antes de
// pasar a la siguiente (mismo patrón que showFeedback/nextExercise en el
// prototipo JS).
func answer(this js.Value, args []js.Value) interface{} {
	if current == nil {
		return errJSON("no hay ejercicios cargados — llamá a load() primero")
	}
	if current.index >= len(current.questions) {
		return toJSON(answerResult{Done: true})
	}
	if current.answered {
		return errJSON("esta pregunta ya fue respondida — llamá a next()")
	}
	if len(args) < 1 {
		return errJSON("uso: mcEngine.answer(selectedIndex)")
	}

	selected := args[0].Int()
	q := current.questions[current.index]
	correct := selected == q.Correct
	if correct {
		current.score++
	}
	current.answered = true

	return toJSON(answerResult{
		Correct: correct, CorrectIndex: q.Correct,
		Score: current.score, Total: len(current.questions),
	})
}

// next() — avanza a la siguiente pregunta. Cuando la ronda termina, persiste
// el resultado en localStorage (intentos totales + mejor puntaje).
func next(this js.Value, args []js.Value) interface{} {
	if current == nil {
		return errJSON("no hay ejercicios cargados — llamá a load() primero")
	}
	current.index++
	current.answered = false

	done := current.index >= len(current.questions)
	if done {
		rec := loadProgress(current.levelKey)
		isFirstAttempt := rec.Attempts == 0
		rec.Attempts++
		if isFirstAttempt || current.score > rec.BestScore {
			rec.BestScore = current.score
			rec.BestTotal = len(current.questions)
		}
		rec.LastPlayed = time.Now().UTC().Format(time.RFC3339)
		saveProgress(current.levelKey, rec)
	}
	return toJSON(map[string]bool{"done": done})
}

// progress(levelKey) — progreso histórico de un nivel, sin necesidad de
// tener un engine cargado (para mostrarlo en una pantalla de resumen).
func progress(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return errJSON("uso: mcEngine.progress(levelKey)")
	}
	return toJSON(loadProgress(args[0].String()))
}

func main() {
	api := js.Global().Get("Object").New()
	api.Set("load", js.FuncOf(load))
	api.Set("current", js.FuncOf(currentQuestion))
	api.Set("answer", js.FuncOf(answer))
	api.Set("next", js.FuncOf(next))
	api.Set("progress", js.FuncOf(progress))
	js.Global().Set("mcEngine", api)

	fmt.Println("italianclub mc-engine (wasm) listo — window.mcEngine disponible")

	select {} // mantiene vivo el programa para que los callbacks sigan respondiendo
}
