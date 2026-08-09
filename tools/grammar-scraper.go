// Scraper de contenido de gramática — fase final de la migración.
// A diferencia del microservicio anterior (que consultaba la REST API),
// este programa descarga el HTML real de cada página y extrae el texto
// entre el título (H1) y el pie de página, que es donde vive la
// explicación real de la lección.
//
// USO:
//   go run grammar-scraper.go > grammar-b2-c1-c2.json
//
// Requiere conexión a internet real (no funciona en el sandbox de esta
// sesión, que tiene la red restringida a un allowlist de dominios de
// desarrollo). Pensado para correr en tu propia máquina o servidor.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type GrammarItem struct {
	TitleIt string `json:"title_it"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Status  string `json:"status,omitempty"`
}

type LevelResult struct {
	Level      string        `json:"level"`
	MigratedAt string        `json:"migrated_at"`
	Items      []GrammarItem `json:"grammar_content"`
}

// URLs reales identificadas manualmente en los índices de B2, C1 y C2
// del sitio (ver la conversación / los *-content.json de cada nivel).
var pending = map[string][][2]string{
	"A1": {
		{"La forma 'Lei'", "https://onlineitalianclub.com/italian-grammar-first-meetings/"},
	},
	"A2": {
		{"Condizionale - come si usa (video)", "https://onlineitalianclub.com/free-italian-exercises-and-resources/online-italian-course-pre-intermediate-level-a2/come-si-usa-il-condizionale-video/"},
		{"Verbi modali al passato prossimo", "https://onlineitalianclub.com/free-italian-exercises-and-resources/online-italian-course-pre-intermediate-level-a2/verbi-modali-passato-prossimo-modal-verbs/"},
	},
	"B2": {
		{"Aggettivi determinativi e relazionali", "https://onlineitalianclub.com/aggettivi-determinativi-e-relazionali/"},
		{"Avverbi e congiunzioni con valore di connettori", "https://onlineitalianclub.com/avverbi-e-congiunzioni-con-valore-di-connettori-adverbs-and-conjunctions-used-as-linkers/"},
		{"Costruzione scissa", "https://onlineitalianclub.com/la-costruzione-scissa/"},
		{"Forma diminutiva e accrescitiva", "https://onlineitalianclub.com/la-forma-diminutiva-e-accrescitiva-diminutives-and-augmentatives/"},
		{"Forma passiva", "https://onlineitalianclub.com/la-forma-passiva-2/"},
		{"Imperativo (parte 2)", "https://onlineitalianclub.com/imperativo-parte-2/"},
		{"Parole composte", "https://onlineitalianclub.com/le-parole-composte-compound-words/"},
		{"Passato remoto", "https://onlineitalianclub.com/passato-remoto-the-remote-past-tense/"},
		{"Posizione degli aggettivi", "https://onlineitalianclub.com/la-posizione-degli-aggettivi/"},
		{"Pronome relativo 'che'", "https://onlineitalianclub.com/il-pronome-relativo-che/"},
		{"Pronome relativo doppio 'chi'", "https://onlineitalianclub.com/il-pronome-relativo-doppio-chi/"},
		{"Quando si usa il congiuntivo", "https://onlineitalianclub.com/quando-usare-il-congiuntivo/"},
		{"'Si' spersonalizzante", "https://onlineitalianclub.com/il-si-spersonalizzante/"},
	},
	"C1": {
		{"Costruzione fare + infinito", "https://onlineitalianclub.com/la-costruzione-fare-infinito-the-construction-make-infinitive/"},
		{"Discorso indiretto", "https://onlineitalianclub.com/il-discorso-indiretto/"},
		{"Gerundio composto", "https://onlineitalianclub.com/gerundio-composto/"},
		{"Gerundio semplice", "https://onlineitalianclub.com/gerundio-semplice/"},
		{"Infinito", "https://onlineitalianclub.com/linfinito-the-infinitive/"},
		{"Locuzioni avverbiali", "https://onlineitalianclub.com/locuzioni-avverbiali-adverb-phrases/"},
		{"Mica", "https://onlineitalianclub.com/mica/"},
		{"Parole di difficile comprensione e uso", "https://onlineitalianclub.com/parole-di-difficile-comprensione-e-uso-difficult-words/"},
		{"Participio", "https://onlineitalianclub.com/il-participio/"},
		{"Piuttosto che", "https://onlineitalianclub.com/piuttosto-che/"},
		{"Pronomi e aggettivi indefiniti", "https://onlineitalianclub.com/pronomi-e-aggettivi-indefiniti/"},
		{"Quando usare 'non'", "https://onlineitalianclub.com/quando-usare-non/"},
		{"Uso e omissione degli articoli", "https://onlineitalianclub.com/uso-e-omissione-degli-articoli/"},
		{"Verbi pronominali", "https://onlineitalianclub.com/verbi-pronominali/"},
	},
	"C2": {
		{"Altri usi del congiuntivo", "https://onlineitalianclub.com/altri-usi-del-congiuntivo/"},
		{"Articoli con nomi propri e toponimi", "https://onlineitalianclub.com/articoli-determinativi-con-nomi-propri-e-di-citta/"},
		{"Concordanze verbali", "https://onlineitalianclub.com/concordanze-verbali/"},
		{"Congiunzioni", "https://onlineitalianclub.com/congiunzioni-connectives/"},
		{"Connettivi vari", "https://onlineitalianclub.com/italian-grammar-conjunctions-connettivi/"},
		{"Finché / Finché non", "https://onlineitalianclub.com/finche-finche-non/"},
		{"Particelle pronominali 'ci' e 'ne'", "https://onlineitalianclub.com/particelle-pronominali-ci-ne/"},
		{"Plurale dei nomi composti", "https://onlineitalianclub.com/il-plurale-dei-nomi-composti-plural-form-of-italian-compound-nouns/"},
		{"Preposizioni (semplici)", "https://onlineitalianclub.com/le-preposizioni/"},
		{"Preposizioni articolate", "https://onlineitalianclub.com/preposizioni-articolate/"},
		{"Pronomi dimostrativi", "https://onlineitalianclub.com/pronomi-dimostrativi/"},
		{"Pronomi relativi 'che' e 'cui'", "https://onlineitalianclub.com/pronomi-relativi-che-e-cui/"},
		{"Pronomi relativi possessivi", "https://onlineitalianclub.com/pronomi-relativi-possessivi/"},
	},
}

// Ítems cuya URL exacta no se confirmó manualmente durante la sesión.
// El programa las busca automáticamente en el índice alfabético del
// sitio antes de intentar descargarlas.
var toDiscover = map[string][]string{
	"B1": {
		"congiuntivo imperfetto",
		"congiuntivo passato",
		"forma passiva",
		"periodo ipotetico della possibilità",
		"periodo ipotetico dell'impossibilità",
		"trapassato prossimo",
		"uso di \"prima di\"",
	},
}

const grammarIndexURL = "https://onlineitalianclub.com/index-of-free-italian-exercises-and-grammar-lessons/index-of-italian-grammar-lessons/"

var tagRe = regexp.MustCompile(`<[^>]+>`)
var spaceRe = regexp.MustCompile(`\s+`)

func extractBody(html string) string {
	// El contenido real vive entre el primer <h1> y el marcador de
	// "Contact us" / footer que aparece en todas las páginas del sitio.
	start := strings.Index(html, "<h1")
	if start == -1 {
		start = 0
	}
	end := strings.Index(html, "Contact us")
	if end == -1 || end < start {
		end = len(html)
	}
	if end > len(html) {
		end = len(html)
	}
	chunk := html[start:end]
	// quita el propio h1 (ya tenemos el título por separado)
	if h1end := strings.Index(chunk, "</h1>"); h1end != -1 {
		chunk = chunk[h1end+5:]
	}
	text := tagRe.ReplaceAllString(chunk, " ")
	text = strings.ReplaceAll(text, "&#8217;", "'")
	text = strings.ReplaceAll(text, "&#8220;", "\"")
	text = strings.ReplaceAll(text, "&#8221;", "\"")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = spaceRe.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func fetchOne(url string) (string, error) {
	client := http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ItalianClubMigrationBot/1.0)")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// discoverURL busca dentro del HTML del índice alfabético un enlace
// <a href="..."> cuyo texto visible contenga la palabra clave dada
// (insensible a mayúsculas/tildes simples), y devuelve su href.
func discoverURL(indexHTML, keyword string) string {
	pattern := `(?is)<a href="([^"]+)"[^>]*>([^<]*` + regexp.QuoteMeta(keyword) + `[^<]*)</a>`
	re := regexp.MustCompile(pattern)
	if m := re.FindStringSubmatch(indexHTML); len(m) > 1 {
		return m[1]
	}
	return ""
}

func main() {
	results := []LevelResult{}

	// Fase 1: niveles con URL ya confirmada (A1, A2 pendientes + B2/C1/C2)
	for _, level := range []string{"A1", "A2", "B2", "C1", "C2"} {
		items, ok := pending[level]
		if !ok {
			continue
		}
		lr := LevelResult{Level: level, MigratedAt: time.Now().Format("2006-01-02")}
		for _, pair := range items {
			title, url := pair[0], pair[1]
			fmt.Fprintf(os.Stderr, "[%s] descargando: %s\n", level, title)

			html, err := fetchOne(url)
			item := GrammarItem{TitleIt: title, URL: url}
			if err != nil {
				item.Status = "ERROR: " + err.Error()
				log.Printf("FALLO %s (%s): %v", title, url, err)
			} else {
				content := extractBody(html)
				if len(content) < 30 {
					item.Status = "ADVERTENCIA: contenido extraído muy corto, revisar manualmente"
				}
				item.Content = content
			}
			lr.Items = append(lr.Items, item)
			time.Sleep(500 * time.Millisecond)
		}
		results = append(results, lr)
	}

	// Fase 2: ítems de B1 sin URL confirmada — se descubren primero
	// buscando en el índice alfabético del sitio.
	if keywords, ok := toDiscover["B1"]; ok {
		fmt.Fprintln(os.Stderr, "\n[B1] descargando índice alfabético para descubrir URLs pendientes...")
		indexHTML, err := fetchOne(grammarIndexURL)
		lr := LevelResult{Level: "B1-descubiertos", MigratedAt: time.Now().Format("2006-01-02")}
		if err != nil {
			log.Printf("No se pudo descargar el índice: %v", err)
			for _, kw := range keywords {
				lr.Items = append(lr.Items, GrammarItem{TitleIt: kw, Status: "ERROR: no se pudo acceder al índice del sitio"})
			}
		} else {
			for _, kw := range keywords {
				url := discoverURL(indexHTML, kw)
				item := GrammarItem{TitleIt: kw}
				if url == "" {
					item.Status = "NO ENCONTRADO en el índice — requiere búsqueda manual"
					lr.Items = append(lr.Items, item)
					continue
				}
				item.URL = url
				fmt.Fprintf(os.Stderr, "[B1] descubierto '%s' -> %s\n", kw, url)
				html, ferr := fetchOne(url)
				if ferr != nil {
					item.Status = "ERROR: " + ferr.Error()
				} else {
					item.Content = extractBody(html)
					if len(item.Content) < 30 {
						item.Status = "ADVERTENCIA: contenido extraído muy corto, revisar manualmente"
					}
				}
				lr.Items = append(lr.Items, item)
				time.Sleep(500 * time.Millisecond)
			}
		}
		results = append(results, lr)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(results); err != nil {
		log.Fatal(err)
	}
	fmt.Fprintln(os.Stderr, "\nListo. Revisa los items con status ERROR, ADVERTENCIA o NO ENCONTRADO antes de usarlos en la app.")
}
