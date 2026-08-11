// Scraper de contenido de listening — Fase 2 de la migración (ver docs/roadmap.md).
//
// A diferencia de grammar-scraper.go (que solo migra texto), acá cada ítem tiene
// además un audio real. El sitio NO aloja archivos .mp3 propios para estos
// ejercicios: el audio vive embebido vía un iframe de SoundCloud
// (<iframe src="https://w.soundcloud.com/player/?url=.../tracks/<id>...">).
// Por eso este scraper no descarga binarios de audio — extrae el track ID de
// SoundCloud y lo guarda como referencia, para reproducir el audio real
// haciendo streaming desde SoundCloud (igual que hace el sitio original), no
// para tener una copia offline. Ver docs/architecture.md, decisión 7.
//
// USO:
//   cd italian-club-app
//   go run tools/listening-scraper.go
// (escribe directo en content/listening-<nivel>.json, uno por nivel; no hace
// falta un paso de merge porque es contenido nuevo, no una actualización de
// ítems existentes)
//
// Requiere conexión a internet real.
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

type ListeningItem struct {
	TitleIt        string `json:"title_it"`
	URL            string `json:"url"`
	Content        string `json:"content"`
	SoundcloudTrack string `json:"soundcloud_track_id,omitempty"`
	Status         string `json:"status,omitempty"`
}

type LevelResult struct {
	Level      string          `json:"level"`
	MigratedAt string          `json:"migrated_at"`
	Items      []ListeningItem `json:"listening_content"`
}

const listeningIndexURL = "https://onlineitalianclub.com/index-of-italian-listening-comprehension-exercises/"

var levelPathToKey = map[string]string{
	"online-italian-course-beginner-level-a1":         "A1",
	"online-italian-course-pre-intermediate-level-a2": "A2",
	"online-italian-course-intermediate-b1":            "B1",
	"online-italian-course-upper-intermediate-b2":      "B2",
	"online-italian-course-advanced-c1":                "C1",
	"online-italian-course-proficient-c2":               "C2",
}

var linkRe = regexp.MustCompile(`<a href="(https://onlineitalianclub\.com/[^"]+)"[^>]*>([^<]+)</a>`)
var h1Re = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
var iframeSrcRe = regexp.MustCompile(`(?i)<iframe[^>]*src="([^"]*soundcloud[^"]*)"`)
var trackIDRe = regexp.MustCompile(`tracks(?:%2F|/)(\d+)`)
var tagRe = regexp.MustCompile(`<[^>]+>`)
var spaceRe = regexp.MustCompile(`\s+`)

func fetchOne(url string) (string, error) {
	client := http.Client{Timeout: 20 * time.Second}
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

func cleanText(html string) string {
	text := tagRe.ReplaceAllString(html, " ")
	text = strings.ReplaceAll(text, "&#8217;", "'")
	text = strings.ReplaceAll(text, "&#8216;", "'")
	text = strings.ReplaceAll(text, "&#8220;", "\"")
	text = strings.ReplaceAll(text, "&#8221;", "\"")
	text = strings.ReplaceAll(text, "&#8211;", "–")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = spaceRe.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// discoverLinks descarga el índice de listening y devuelve, por nivel, la
// lista de (título, URL) de cada ejercicio — deduplicada.
func discoverLinks() (map[string][][2]string, error) {
	html, err := fetchOne(listeningIndexURL)
	if err != nil {
		return nil, err
	}
	result := map[string][][2]string{}
	seen := map[string]bool{}
	for _, m := range linkRe.FindAllStringSubmatch(html, -1) {
		href, text := m[1], strings.TrimSpace(m[2])
		if seen[href] {
			continue
		}
		for pathKey, level := range levelPathToKey {
			if strings.Contains(href, pathKey) {
				// excluye los 6 links de navegación de nivel en sí (href termina justo en el path del nivel)
				trimmed := strings.TrimRight(href, "/")
				if strings.HasSuffix(trimmed, pathKey) {
					break
				}
				seen[href] = true
				result[level] = append(result[level], [2]string{text, href})
				break
			}
		}
	}
	return result, nil
}

func scrapeItem(title, url string) ListeningItem {
	item := ListeningItem{TitleIt: title, URL: url}
	html, err := fetchOne(url)
	if err != nil {
		item.Status = "ERROR: " + err.Error()
		return item
	}
	if m := iframeSrcRe.FindStringSubmatch(html); len(m) > 1 {
		if tm := trackIDRe.FindStringSubmatch(m[1]); len(tm) > 1 {
			item.SoundcloudTrack = tm[1]
		}
	}
	if item.SoundcloudTrack == "" {
		item.Status = "ADVERTENCIA: no se encontró embed de audio SoundCloud en la página"
	}
	// título real desde el h1 de la página (más confiable que el texto del link del índice)
	if hm := h1Re.FindStringSubmatch(html); len(hm) > 1 {
		item.TitleIt = cleanText(hm[1])
	}
	start := strings.Index(html, "</h1>")
	if start == -1 {
		start = 0
	} else {
		start += 5
	}
	end := strings.Index(html, "Contact us")
	if end == -1 || end < start {
		end = len(html)
	}
	item.Content = cleanText(html[start:end])
	if len(item.Content) < 30 {
		if item.Status != "" {
			item.Status += "; "
		}
		item.Status += "ADVERTENCIA: contenido extraído muy corto, revisar manualmente"
	}
	return item
}

func main() {
	fmt.Fprintln(os.Stderr, "Descargando índice de listening...")
	byLevel, err := discoverLinks()
	if err != nil {
		log.Fatalf("no se pudo descargar el índice: %v", err)
	}

	for _, level := range []string{"A1", "A2", "B1", "B2", "C1", "C2"} {
		links := byLevel[level]
		fmt.Fprintf(os.Stderr, "[%s] %d ítems encontrados en el índice\n", level, len(links))
		lr := LevelResult{Level: level, MigratedAt: time.Now().Format("2006-01-02")}
		for i, pair := range links {
			title, url := pair[0], pair[1]
			fmt.Fprintf(os.Stderr, "[%s %d/%d] descargando: %s\n", level, i+1, len(links), title)
			item := scrapeItem(title, url)
			if item.Status != "" {
				log.Printf("[%s] %s -> %s", level, item.TitleIt, item.Status)
			}
			lr.Items = append(lr.Items, item)
			time.Sleep(400 * time.Millisecond)
		}

		outPath := fmt.Sprintf("content/listening-%s.json", strings.ToLower(level))
		f, err := os.Create(outPath)
		if err != nil {
			log.Fatalf("no se pudo escribir %s: %v", outPath, err)
		}
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		if err := enc.Encode(lr); err != nil {
			log.Fatalf("no se pudo codificar %s: %v", outPath, err)
		}
		f.Close()
		fmt.Fprintf(os.Stderr, "[%s] escrito %s (%d ítems)\n\n", level, outPath, len(lr.Items))
	}
	fmt.Fprintln(os.Stderr, "Listo. Revisa los items con status ERROR o ADVERTENCIA antes de usarlos en la app.")
}
