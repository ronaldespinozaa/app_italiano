//go:build js && wasm

// Motor de ejercicios compilado a WebAssembly.
//
// Milestone 2 del plan "Go -> WASM": generaliza el Milestone 1 (que solo
// soportaba `mc`) a los 4 tipos que ya soporta `EXERCISE_QUEUES` en
// prototype/index.html: mc, gapfill, truefalse, ordering. La lógica de
// corrección de cada tipo replica EXACTO el comportamiento de
// answerMC/answerGapfill/answerTF/pickWord en prototype/index.html (mismo
// trim/lowercase, mismo join con espacio simple para ordering) — no es una
// reinterpretación, es el mismo contrato de datos.
//
// Expone en window.exerciseEngine: load, current, answer, next, progress.
// (Antes de este milestone se llamaba window.mcEngine — se renombró porque
// ya no es solo mc.)
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"
	"time"
)

// ExerciseItem es el superset de los 4 tipos, con el mismo shape que ya
// usa EXERCISE_QUEUES en el prototipo (adaptado a JSON con nombres en
// minúscula). Cada tipo solo llena los campos que le corresponden.
type ExerciseItem struct {
	Type string `json:"type"` // mc | gapfill | truefalse | ordering
	ID   string `json:"id"`

	// mc
	Prompt  string   `json:"prompt,omitempty"`
	Options []string `json:"options,omitempty"`
	Correct int      `json:"correct,omitempty"`

	// gapfill
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
	Answer string `json:"answer,omitempty"`
	Hint   string `json:"hint,omitempty"`

	// truefalse
	Statement  string `json:"statement,omitempty"`
	BoolAnswer bool   `json:"boolAnswer,omitempty"`

	// ordering
	Words           []string `json:"words,omitempty"`
	CorrectSentence string   `json:"correctSentence,omitempty"`
}

// answerPayload es lo que manda la UI al contestar. Solo el campo que
// corresponde al tipo de la pregunta activa debe venir seteado.
type answerPayload struct {
	SelectedIndex *int     `json:"selectedIndex,omitempty"` // mc
	Text          *string  `json:"text,omitempty"`          // gapfill
	Selected      *bool    `json:"selected,omitempty"`      // truefalse
	Words         []string `json:"words,omitempty"`         // ordering
}

type answerResult struct {
	Error           string `json:"error,omitempty"`
	Correct         bool   `json:"correct"`
	CorrectAnswer   string `json:"correctAnswer,omitempty"` // texto legible, para el feedback
	Score           int    `json:"score"`
	Total           int    `json:"total"`
	Done            bool   `json:"done"`
}

type progressRecord struct {
	Attempts   int    `json:"attempts"`
	BestScore  int    `json:"bestScore"`
	BestTotal  int    `json:"bestTotal"`
	LastPlayed string `json:"lastPlayedAt"`
}

type engineState struct {
	levelKey string
	items    []ExerciseItem
	index    int
	score    int
	answered bool
}

func storageKey(levelKey string) string {
	return "italianclub:progress:" + levelKey
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

var current *engineState

// load(levelKey, itemsJSON) — reemplaza la cola de ejercicios activa.
func load(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return errJSON("uso: exerciseEngine.load(levelKey, itemsJSON)")
	}
	levelKey := args[0].String()

	var items []ExerciseItem
	if err := json.Unmarshal([]byte(args[1].String()), &items); err != nil {
		return errJSON("JSON de ejercicios inválido: " + err.Error())
	}
	if len(items) == 0 {
		return errJSON("la lista de ejercicios está vacía")
	}
	for i, it := range items {
		if it.Type != "mc" && it.Type != "gapfill" && it.Type != "truefalse" && it.Type != "ordering" {
			return errJSON(fmt.Sprintf("ítem %d: tipo desconocido %q", i, it.Type))
		}
	}

	current = &engineState{levelKey: levelKey, items: items}
	return toJSON(map[string]interface{}{
		"loaded":   len(items),
		"progress": loadProgress(levelKey),
	})
}

// current() — el ejercicio activo, sin revelar la respuesta correcta, o
// {"done":true} si ya se terminó la cola.
func currentItem(this js.Value, args []js.Value) interface{} {
	if current == nil {
		return errJSON("no hay ejercicios cargados — llamá a load() primero")
	}
	if current.index >= len(current.items) {
		return toJSON(map[string]bool{"done": true})
	}
	it := current.items[current.index]

	public := map[string]interface{}{
		"id": it.ID, "type": it.Type,
		"index": current.index, "total": len(current.items),
	}
	switch it.Type {
	case "mc":
		public["prompt"] = it.Prompt
		public["options"] = it.Options
	case "gapfill":
		public["before"] = it.Before
		public["after"] = it.After
		public["hint"] = it.Hint
	case "truefalse":
		public["statement"] = it.Statement
	case "ordering":
		public["words"] = it.Words
	}
	return toJSON(public)
}

// answer(payloadJSON) — corrige el ejercicio activo. La corrección replica
// exacto la lógica de prototype/index.html:
//   - gapfill: trim + lowercase, comparación exacta contra `answer`.
//   - ordering: palabras unidas con un espacio, trim + lowercase.
//   - truefalse: comparación booleana estricta.
//   - mc: índice de opción seleccionada contra `correct`.
func answer(this js.Value, args []js.Value) interface{} {
	if current == nil {
		return errJSON("no hay ejercicios cargados — llamá a load() primero")
	}
	if current.index >= len(current.items) {
		return toJSON(answerResult{Done: true})
	}
	if current.answered {
		return errJSON("este ejercicio ya fue respondido — llamá a next()")
	}
	if len(args) < 1 {
		return errJSON("uso: exerciseEngine.answer(payloadJSON)")
	}

	var payload answerPayload
	if err := json.Unmarshal([]byte(args[0].String()), &payload); err != nil {
		return errJSON("payload de respuesta inválido: " + err.Error())
	}

	it := current.items[current.index]
	var correct bool
	var correctAnswer string

	switch it.Type {
	case "mc":
		if payload.SelectedIndex == nil {
			return errJSON("falta selectedIndex para un ejercicio mc")
		}
		correct = *payload.SelectedIndex == it.Correct
		if it.Correct >= 0 && it.Correct < len(it.Options) {
			correctAnswer = it.Options[it.Correct]
		}
	case "gapfill":
		if payload.Text == nil {
			return errJSON("falta text para un ejercicio gapfill")
		}
		given := strings.ToLower(strings.TrimSpace(*payload.Text))
		correct = given == strings.ToLower(it.Answer)
		correctAnswer = it.Answer
	case "truefalse":
		if payload.Selected == nil {
			return errJSON("falta selected para un ejercicio truefalse")
		}
		correct = *payload.Selected == it.BoolAnswer
		if it.BoolAnswer {
			correctAnswer = "Vero"
		} else {
			correctAnswer = "Falso"
		}
	case "ordering":
		if len(payload.Words) == 0 {
			return errJSON("falta words para un ejercicio ordering")
		}
		built := strings.ToLower(strings.Join(payload.Words, " "))
		correct = built == strings.ToLower(it.CorrectSentence)
		correctAnswer = it.CorrectSentence
	}

	if correct {
		current.score++
	}
	current.answered = true

	return toJSON(answerResult{
		Correct: correct, CorrectAnswer: correctAnswer,
		Score: current.score, Total: len(current.items),
	})
}

// next() — avanza al siguiente ejercicio. Al terminar la cola, persiste el
// resultado (intentos totales + mejor puntaje) en localStorage.
func next(this js.Value, args []js.Value) interface{} {
	if current == nil {
		return errJSON("no hay ejercicios cargados — llamá a load() primero")
	}
	current.index++
	current.answered = false

	done := current.index >= len(current.items)
	if done {
		rec := loadProgress(current.levelKey)
		isFirstAttempt := rec.Attempts == 0
		rec.Attempts++
		if isFirstAttempt || current.score > rec.BestScore {
			rec.BestScore = current.score
			rec.BestTotal = len(current.items)
		}
		rec.LastPlayed = time.Now().UTC().Format(time.RFC3339)
		saveProgress(current.levelKey, rec)
	}
	return toJSON(map[string]bool{"done": done})
}

// progress(levelKey) — progreso histórico de un nivel, sin necesitar un
// load() previo (para pantallas de resumen).
func progress(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return errJSON("uso: exerciseEngine.progress(levelKey)")
	}
	return toJSON(loadProgress(args[0].String()))
}

func main() {
	api := js.Global().Get("Object").New()
	api.Set("load", js.FuncOf(load))
	api.Set("current", js.FuncOf(currentItem))
	api.Set("answer", js.FuncOf(answer))
	api.Set("next", js.FuncOf(next))
	api.Set("progress", js.FuncOf(progress))
	js.Global().Set("exerciseEngine", api)

	fmt.Println("italianclub exercise-engine (wasm) listo — window.exerciseEngine disponible")

	select {} // mantiene vivo el programa para que los callbacks sigan respondiendo
}
