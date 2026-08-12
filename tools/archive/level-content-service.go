// ARCHIVADO 2026-08-12 (Fase 4 del roadmap, "Apagar WordPress") — no se usa
// en el proyecto. Se conserva como referencia de la arquitectura alternativa
// que se consideró y se descartó (ver docs/architecture.md, decisión 3):
// microservicio que consulta la REST API de WordPress en vivo, con caché en
// memoria, en vez de migrar el contenido una sola vez a JSON estático. Con
// las Fases 1-3 completas (gramática, listening y ejercicios 100% migrados
// a content/*.json y embebidos en prototype/index.html), WordPress ya no es
// necesaria como fuente de contenido para la app — puede apagarse sin romper
// nada. Este archivo nunca llegó a correr contra un WordPress real con las
// taxonomías de nivel que necesita (ver nota original abajo); quedó como
// prueba de concepto de la arquitectura descartada, no como código en uso.
//
// ---- comentario original ----
// Microservicio de ejemplo: sirve contenido de OnlineItalianClub.com
// ya filtrado por nivel CEFR, cacheado en memoria para no pegarle
// a WordPress en cada request de la app móvil.
//
// Uso previsto:
//   GET /api/level/A1/modules        -> lista de módulos con conteo de items
//   GET /api/level/A1/grammar        -> posts de gramática etiquetados A1
//   GET /api/level/B2/listening      -> posts de listening etiquetados B2
//
// Requiere que en WordPress cada post tenga, además de su categoría de
// tipo (grammar, listening, vocab...), una taxonomía o tag de nivel
// (a1, a2, b1, b2, c1, c2). Ese etiquetado es la fase 1 del plan y debe
// completarse ANTES de que este servicio tenga sentido.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const wpBase = "https://onlineitalianclub.com/wp-json/wp/v2"

var validLevels = map[string]bool{
	"a1": true, "a2": true, "b1": true, "b2": true, "c1": true, "c2": true,
}

var validModules = map[string]bool{
	"grammar": true, "listening": true, "vocabulary": true,
	"dialogues": true, "verbs": true, "reading": true,
}

// --- cache muy simple, en memoria, con expiración ---

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

type cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

func newCache(ttl time.Duration) *cache {
	return &cache{entries: make(map[string]cacheEntry), ttl: ttl}
}

func (c *cache) get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.data, true
}

func (c *cache) set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{data: data, expiresAt: time.Now().Add(c.ttl)}
}

var contentCache = newCache(15 * time.Minute)

// --- estructura de respuesta hacia la app ---

type levelModuleResponse struct {
	Level  string      `json:"level"`
	Module string      `json:"module"`
	Count  int         `json:"count"`
	Items  []wpPostLite `json:"items"`
}

type wpPostLite struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Link  string `json:"link"`
	Excerpt string `json:"excerpt"`
}

// wpPostRaw refleja el shape real que devuelve wp-json (simplificado)
type wpPostRaw struct {
	ID    int `json:"id"`
	Title struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	Link    string `json:"link"`
	Excerpt struct {
		Rendered string `json:"rendered"`
	} `json:"excerpt"`
}

// fetchFromWordPress golpea la REST API real de WordPress.
// En producción, aquí se filtraría por los IDs de taxonomía de
// nivel + tipo de contenido, una vez completado el etiquetado (fase 1).
func fetchFromWordPress(level, module string) ([]wpPostLite, error) {
	// Nota: esta URL es ilustrativa. WordPress necesita que las
	// taxonomías de nivel y módulo existan y estén expuestas por la
	// REST API (register_taxonomy con 'show_in_rest' => true).
	url := fmt.Sprintf("%s/posts?per_page=20&search=%s", wpBase, module)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw []wpPostRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	out := make([]wpPostLite, 0, len(raw))
	for _, p := range raw {
		out = append(out, wpPostLite{
			ID:      p.ID,
			Title:   strings.TrimSpace(p.Title.Rendered),
			Link:    p.Link,
			Excerpt: strings.TrimSpace(p.Excerpt.Rendered),
		})
	}
	return out, nil
}

func handleLevelModule(w http.ResponseWriter, r *http.Request) {
	// esperado: /api/level/{level}/{module}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 {
		http.Error(w, "ruta esperada: /api/level/{nivel}/{modulo}", http.StatusBadRequest)
		return
	}
	level := strings.ToLower(parts[2])
	module := strings.ToLower(parts[3])

	if !validLevels[level] {
		http.Error(w, "nivel inválido, use a1..c2", http.StatusBadRequest)
		return
	}
	if !validModules[module] {
		http.Error(w, "módulo inválido", http.StatusBadRequest)
		return
	}

	cacheKey := level + ":" + module
	if cached, ok := contentCache.get(cacheKey); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(cached)
		return
	}

	items, err := fetchFromWordPress(level, module)
	if err != nil {
		log.Printf("error consultando WordPress: %v", err)
		http.Error(w, "error obteniendo contenido", http.StatusBadGateway)
		return
	}

	resp := levelModuleResponse{
		Level:  level,
		Module: module,
		Count:  len(items),
		Items:  items,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	contentCache.set(cacheKey, data)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/level/", handleLevelModule)
	mux.HandleFunc("/health", handleHealth)

	// CORS abierto para que la PWA pueda consumir la API desde el móvil.
	// En producción, restringir al dominio real de la app.
	handler := corsMiddleware(mux)

	addr := ":8080"
	log.Printf("microservicio de contenido por nivel escuchando en %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
