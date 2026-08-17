//go:build js && wasm

// Motor de ejercicios y vocabulario compilado a WebAssembly.
//
// Milestone 3 del plan "Go -> WASM": alcanza paridad con el motor JS real de
// prototype/index.html (que avanzó mucho en paralelo — ver
// wasmapp/README.md para el detalle), antes de reemplazarlo:
//   - Los 5 tipos de EXERCISE_QUEUES: mc, gapfill, truefalse, ordering,
//     matching (el 5º, agregado junto con la migración de ~123 ejercicios
//     legacy que lo necesitaban).
//   - La cola de ejercicios es INFINITA (current.index % len(items)), igual
//     que `exQueueIndex % queue.length` en el JS — no hay concepto de
//     "ronda terminada".
//   - El progreso se persiste en la MISMA clave de localStorage que ya usa
//     el JS (`italianClubProgress_v1`, esquema {[nivel]: {gram, listen,
//     vocab, ex, exCorrect}}), tocando solo `ex`/`exCorrect` — así, cuando
//     se reemplace el motor JS por este, el progreso ya guardado del
//     usuario sigue funcionando sin migración de datos.
//
// Expone en window.exerciseEngine: load, current, answer, next, progress.
//
// Repetición espaciada (SM-2) para vocabulario: pieza separada del motor de
// ejercicios de arriba (opera sobre REAL_VOCAB, no sobre EXERCISE_QUEUES).
// El algoritmo en sí (SRSCard, applySM2, pickVocabIndex) vive en sm2.go, sin
// build tag de js/wasm, para poder testearlo con `go test` sin un runner de
// WebAssembly. Acá solo el glue: localStorage + API expuesta en
// window.vocabEngine (load, current, answer, progress).
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"syscall/js"
)

// Pair es un ítem de un ejercicio `matching`.
type Pair struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

// ExerciseItem es el superset de los 5 tipos, con el mismo shape que ya usa
// EXERCISE_QUEUES en el prototipo (adaptado a JSON con nombres en
// minúscula). Cada tipo solo llena los campos que le corresponden.
type ExerciseItem struct {
	Type string `json:"type"` // mc | gapfill | truefalse | ordering | matching
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

	// matching
	Pairs []Pair `json:"pairs,omitempty"`
}

// answerPayload es lo que manda la UI al contestar. Solo el campo que
// corresponde al tipo de la pregunta activa debe venir seteado.
type answerPayload struct {
	SelectedIndex   *int     `json:"selectedIndex,omitempty"`   // mc
	Text            *string  `json:"text,omitempty"`            // gapfill
	Selected        *bool    `json:"selected,omitempty"`        // truefalse
	Words           []string `json:"words,omitempty"`           // ordering
	MatchesComplete bool     `json:"matchesComplete,omitempty"` // matching
}

type answerResult struct {
	Error         string `json:"error,omitempty"`
	Correct       bool   `json:"correct"`
	CorrectAnswer string `json:"correctAnswer,omitempty"` // texto legible, para el feedback
}

// levelBucket refleja EXACTO el shape que ya escribe levelBucket() en
// prototype/index.html, más el campo nuevo `vocabSrs` que agrega este motor.
// gram/listen quedan intactos acá: este motor nunca los toca, solo los
// preserva al leer-modificar-escribir el blob completo — mismo criterio para
// vocabSrs desde el lado de exerciseEngine (arriba) y viceversa desde
// vocabEngine (abajo): como los dos comparten este mismo struct, un
// read-modify-write de uno nunca pisa los datos del otro. omitempty en
// VocabSRS para no ensuciar con `"vocabSrs":null` el progreso ya guardado de
// usuarios que todavía no usaron flashcards.
type levelBucket struct {
	Gram      map[string]bool    `json:"gram"`
	Listen    map[string]bool    `json:"listen"`
	Vocab     map[string]bool    `json:"vocab"`
	Ex        map[string]bool    `json:"ex"`
	ExCorrect map[string]bool    `json:"exCorrect"`
	VocabSRS  map[string]SRSCard `json:"vocabSrs,omitempty"`
}

func emptyBucket() levelBucket {
	return levelBucket{
		Gram: map[string]bool{}, Listen: map[string]bool{}, Vocab: map[string]bool{},
		Ex: map[string]bool{}, ExCorrect: map[string]bool{},
	}
}

const progressStorageKey = "italianClubProgress_v1"

func loadFullProgress() map[string]levelBucket {
	data := map[string]levelBucket{}
	raw := js.Global().Get("localStorage").Call("getItem", progressStorageKey)
	if raw.IsNull() || raw.IsUndefined() {
		return data
	}
	_ = json.Unmarshal([]byte(raw.String()), &data) // si está corrupto, arrancamos de cero
	return data
}

func saveFullProgress(data map[string]levelBucket) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	js.Global().Get("localStorage").Call("setItem", progressStorageKey, string(b))
}

// recordExerciseAttempt replica record ExerciseAttempt() del JS: marca el
// ítem como visto siempre, y su corrección refleja el ÚLTIMO intento (se
// borra de exCorrect si el intento actual fue incorrecto), no un historial.
func recordExerciseAttempt(levelKey string, idx int, correct bool) {
	data := loadFullProgress()
	lb, ok := data[levelKey]
	if !ok {
		lb = emptyBucket()
	}
	key := fmt.Sprintf("%d", idx)
	lb.Ex[key] = true
	if correct {
		lb.ExCorrect[key] = true
	} else {
		delete(lb.ExCorrect, key)
	}
	data[levelKey] = lb
	saveFullProgress(data)
}

// exerciseProgress replica la parte "ex" de progressSummary() del JS.
func exerciseProgress(levelKey string, total int) map[string]interface{} {
	data := loadFullProgress()
	lb, ok := data[levelKey]
	if !ok {
		lb = emptyBucket()
	}
	seen := len(lb.Ex)
	correct := len(lb.ExCorrect)
	pct := 0
	if total > 0 {
		pct = (seen * 100) / total
	}
	return map[string]interface{}{
		"seen": seen, "correct": correct, "total": total, "pct": pct,
	}
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

type engineState struct {
	levelKey string
	items    []ExerciseItem
	index    int
	answered bool
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
		switch it.Type {
		case "mc", "gapfill", "truefalse", "ordering", "matching":
			// ok
		default:
			return errJSON(fmt.Sprintf("ítem %d: tipo desconocido %q", i, it.Type))
		}
	}

	current = &engineState{levelKey: levelKey, items: items}
	return toJSON(map[string]interface{}{
		"loaded":   len(items),
		"progress": exerciseProgress(levelKey, len(items)),
	})
}

// current() — el ejercicio activo. La cola es infinita (igual que
// `exQueueIndex % queue.length` en el JS): nunca hay {"done":true}.
func currentItem(this js.Value, args []js.Value) interface{} {
	if current == nil {
		return errJSON("no hay ejercicios cargados — llamá a load() primero")
	}
	pos := current.index % len(current.items)
	it := current.items[pos]

	public := map[string]interface{}{
		"id": it.ID, "type": it.Type, "queueIndex": pos,
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
	case "matching":
		type shuffledRight struct {
			Text     string `json:"text"`
			RightIdx int    `json:"rightIdx"` // índice original — el left correcto es lefts[rightIdx]
		}
		lefts := make([]string, len(it.Pairs))
		shuffled := make([]shuffledRight, len(it.Pairs))
		for i, p := range it.Pairs {
			lefts[i] = p.Left
			shuffled[i] = shuffledRight{Text: p.Right, RightIdx: i}
		}
		rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		public["lefts"] = lefts
		public["rightsShuffled"] = shuffled
	}
	return toJSON(public)
}

// answer(payloadJSON) — corrige el ejercicio activo y persiste el intento
// de inmediato (igual que el JS: recordExerciseAttempt se llama en el
// momento de contestar, no al avanzar). La corrección replica exacto la
// lógica de prototype/index.html:
//   - gapfill: trim + lowercase, comparación exacta contra `answer`.
//   - ordering: palabras unidas con un espacio, trim + lowercase.
//   - truefalse: comparación booleana estricta.
//   - mc: índice de opción seleccionada contra `correct`.
//   - matching: no tiene estado "incorrecto final" — como en el JS, solo se
//     contesta una vez que el cliente ya unió todos los pares (el
//     emparejamiento se valida solo del lado del cliente comparando
//     rightIdx, no hay nada que ocultar), así que siempre es correct=true.
func answer(this js.Value, args []js.Value) interface{} {
	if current == nil {
		return errJSON("no hay ejercicios cargados — llamá a load() primero")
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

	pos := current.index % len(current.items)
	it := current.items[pos]
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
	case "matching":
		if !payload.MatchesComplete {
			return errJSON("matching se contesta con matchesComplete:true recién cuando el cliente unió todos los pares")
		}
		correct = true
	}

	current.answered = true
	recordExerciseAttempt(current.levelKey, pos, correct)

	return toJSON(answerResult{Correct: correct, CorrectAnswer: correctAnswer})
}

// next() — avanza al siguiente ejercicio de la cola (con wraparound).
func next(this js.Value, args []js.Value) interface{} {
	if current == nil {
		return errJSON("no hay ejercicios cargados — llamá a load() primero")
	}
	current.index++
	current.answered = false
	return toJSON(map[string]bool{"ok": true})
}

// progress(levelKey) — igual que la parte "ex" de progressSummary() en el
// JS. Si hay una cola cargada para ese nivel, usa su longitud real como
// total; si no, total queda en 0 (llamar a load() primero para un total
// preciso).
func progress(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return errJSON("uso: exerciseEngine.progress(levelKey)")
	}
	levelKey := args[0].String()
	total := 0
	if current != nil && current.levelKey == levelKey {
		total = len(current.items)
	}
	return toJSON(exerciseProgress(levelKey, total))
}

// ---------------------------------------------------------------------
// Vocabulario: repetición espaciada (SM-2)
// ---------------------------------------------------------------------
// El algoritmo puro (SRSCard, applySM2, pickVocabIndex, vocabProgressCounts)
// vive en sm2.go. Acá solo el estado activo + el glue de localStorage,
// mismo patrón que engineState/current arriba.

type vocabEngineState struct {
	levelKey    string
	items       []VocabItem
	activeIndex int
	hasActive   bool
	answered    bool
}

var vocab *vocabEngineState

func vocabSRSMap(levelKey string) map[string]SRSCard {
	data := loadFullProgress()
	lb, ok := data[levelKey]
	if !ok || lb.VocabSRS == nil {
		return map[string]SRSCard{}
	}
	return lb.VocabSRS
}

func vocabCardFor(levelKey string, idx int) SRSCard {
	card, seen := vocabSRSMap(levelKey)[fmt.Sprintf("%d", idx)]
	if !seen {
		return newSRSCard()
	}
	return card
}

// saveVocabCard persiste la card SM-2 actualizada Y marca el ítem como
// "visto" en Vocab[idx] — el mismo campo que antes escribía
// markSeen('vocab', ...) en el JS (ver prototype/index.html), para que la
// barra de progreso de "Vocabulario" siga funcionando con el mismo criterio
// de progressSummary(). La diferencia semántica: antes se marcaba "visto" al
// dar vuelta la tarjeta (con solo mirar la traducción alcanzaba); ahora se
// marca al calificarla (el usuario ya se autoevaluó) — más fiel a lo que
// "visto" debería significar en un sistema de repetición espaciada real.
func saveVocabCard(levelKey string, idx int, card SRSCard) {
	data := loadFullProgress()
	lb, ok := data[levelKey]
	if !ok {
		lb = emptyBucket()
	}
	if lb.VocabSRS == nil {
		lb.VocabSRS = map[string]SRSCard{}
	}
	if lb.Vocab == nil {
		lb.Vocab = map[string]bool{}
	}
	key := fmt.Sprintf("%d", idx)
	lb.VocabSRS[key] = card
	lb.Vocab[key] = true
	data[levelKey] = lb
	saveFullProgress(data)
}

// vocabProgress arma el mapa de progreso que devuelven tanto load() como
// progress(): además de seen/total/pct (mismo shape que exerciseProgress,
// para que la UI los trate igual), agrega due/mastered — específicos de
// SM-2, sin equivalente en el motor de ejercicios.
func vocabProgress(levelKey string, total int) map[string]interface{} {
	srs := vocabSRSMap(levelKey)
	seen, due, mastered := vocabProgressCounts(srs, total, today())
	pct := 0
	if total > 0 {
		pct = (seen * 100) / total
	}
	return map[string]interface{}{
		"seen": seen, "total": total, "pct": pct, "due": due, "mastered": mastered,
	}
}

// vocabLoad(levelKey, itemsJSON) — reemplaza el set de vocabulario activo.
// itemsJSON tiene el mismo shape que vocabSetForLevel() en el JS
// ([{word,back},...]) — se puede pasar directo, sin conversión.
func vocabLoad(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return errJSON("uso: vocabEngine.load(levelKey, itemsJSON)")
	}
	levelKey := args[0].String()

	var items []VocabItem
	if err := json.Unmarshal([]byte(args[1].String()), &items); err != nil {
		return errJSON("JSON de vocabulario inválido: " + err.Error())
	}
	if len(items) == 0 {
		return errJSON("la lista de vocabulario está vacía")
	}

	vocab = &vocabEngineState{levelKey: levelKey, items: items}
	return toJSON(map[string]interface{}{
		"loaded":   len(items),
		"progress": vocabProgress(levelKey, len(items)),
	})
}

// vocabCurrent() — la tarjeta activa. Si no hay una ya elegida (recién
// cargado, o la anterior ya se calificó), pickVocabIndex() elige la
// siguiente según el estado SM-2 guardado. Igual que exerciseEngine, se
// devuelve `back` (la traducción) sin ocultar nada: a diferencia de un
// ejercicio, acá no hay "hacer trampa" posible — el dato ya está visible en
// REAL_VOCAB del lado del JS; el flip es solo una animación de UI.
func vocabCurrent(this js.Value, args []js.Value) interface{} {
	if vocab == nil {
		return errJSON("no hay vocabulario cargado — llamá a load() primero")
	}
	if !vocab.hasActive {
		vocab.activeIndex = pickVocabIndex(vocabSRSMap(vocab.levelKey), len(vocab.items), today())
		vocab.hasActive = true
		vocab.answered = false
	}
	idx := vocab.activeIndex
	item := vocab.items[idx]
	card := vocabCardFor(vocab.levelKey, idx)
	return toJSON(map[string]interface{}{
		"index": idx, "word": item.Word, "back": item.Back,
		"isNew": card.Due == "" && card.Reps == 0,
		"reps":  card.Reps, "interval": card.Interval, "due": card.Due,
	})
}

type vocabAnswerPayload struct {
	Quality int `json:"quality"`
}

// vocabAnswer(payloadJSON) — califica la tarjeta activa (payload:
// {"quality": N}, escala 0-5 de SM-2) y avanza. El flip en la UI es
// independiente de esto: se puede dar vuelta la tarjeta sin calificar
// (llamando a current() de nuevo, que devuelve la misma tarjeta mientras no
// se conteste), igual de espíritu que exerciseEngine, que no fuerza un
// avance hasta next().
func vocabAnswer(this js.Value, args []js.Value) interface{} {
	if vocab == nil {
		return errJSON("no hay vocabulario cargado — llamá a load() primero")
	}
	if !vocab.hasActive {
		return errJSON("no hay tarjeta activa — llamá a current() primero")
	}
	if vocab.answered {
		return errJSON("esta tarjeta ya fue calificada — llamá a current() para pedir la siguiente")
	}
	if len(args) < 1 {
		return errJSON("uso: vocabEngine.answer(payloadJSON)")
	}

	var payload vocabAnswerPayload
	if err := json.Unmarshal([]byte(args[0].String()), &payload); err != nil {
		return errJSON("payload de respuesta inválido: " + err.Error())
	}

	idx := vocab.activeIndex
	card := vocabCardFor(vocab.levelKey, idx)
	card = applySM2(card, payload.Quality, today())
	saveVocabCard(vocab.levelKey, idx, card)

	vocab.answered = true
	vocab.hasActive = false // fuerza a pickVocabIndex a elegir de nuevo en el próximo current()

	return toJSON(map[string]interface{}{
		"ef": card.EF, "interval": card.Interval, "reps": card.Reps, "due": card.Due,
	})
}

// vocabProgressAPI(levelKey) — progreso de vocabulario para ese nivel. Si
// hay un set cargado para ese nivel, usa su longitud real como total; si no,
// total queda en 0 (llamar a load() primero para un total preciso) — mismo
// criterio que progress() del motor de ejercicios.
func vocabProgressAPI(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return errJSON("uso: vocabEngine.progress(levelKey)")
	}
	levelKey := args[0].String()
	total := 0
	if vocab != nil && vocab.levelKey == levelKey {
		total = len(vocab.items)
	}
	return toJSON(vocabProgress(levelKey, total))
}

func main() {
	api := js.Global().Get("Object").New()
	api.Set("load", js.FuncOf(load))
	api.Set("current", js.FuncOf(currentItem))
	api.Set("answer", js.FuncOf(answer))
	api.Set("next", js.FuncOf(next))
	api.Set("progress", js.FuncOf(progress))
	js.Global().Set("exerciseEngine", api)

	vapi := js.Global().Get("Object").New()
	vapi.Set("load", js.FuncOf(vocabLoad))
	vapi.Set("current", js.FuncOf(vocabCurrent))
	vapi.Set("answer", js.FuncOf(vocabAnswer))
	vapi.Set("progress", js.FuncOf(vocabProgressAPI))
	js.Global().Set("vocabEngine", vapi)

	fmt.Println("italianclub exercise-engine + vocab-engine (wasm) listo — window.exerciseEngine y window.vocabEngine disponibles")

	select {} // mantiene vivo el programa para que los callbacks sigan respondiendo
}
