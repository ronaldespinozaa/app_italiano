#!/usr/bin/env python3
"""
Scraper + conversor de los ejercicios legacy (`/free_italian_exercises/*.html`)
al formato de datos que consume el motor de `prototype/index.html`
(`EXERCISE_QUEUES`: tipos mc, gapfill, ordering, matching — ver
docs/architecture.md punto 4). Ver docs/roadmap.md Fase 3.

Por qué Python y no Go (a diferencia de grammar-scraper.go / listening-scraper.go):
este scraper no solo extrae texto verbatim, tiene que **entender la lógica de
un motor de quiz JS de hace más de una década** (distintas plantillas, cada
una con su propio layout de array de datos) y convertirla a un formato nuevo
sin cambiar cuál es la respuesta correcta. Es un problema de parsing/heurística
más que de descarga de texto, y la iteración en Python fue más rápida durante
el reconocimiento inicial de formatos (ver migration-log/log.json).

## Cómo se detecta el tipo de ejercicio (por FORMA de los datos, no por el
nombre del script bits/*.js — así también cubre variantes inline sin bits/):

Cada página tiene `var questionN = new Array(...)` (uno por pregunta) y a
veces un `var correctArray = new Array(...)`.

- Si existe `correctArray`: es **mc**. Elementos del array = [antes, opción1,
  opción2, ..., opciónK, después]. `correctArray[i]` es el índice (1-based)
  de la opción correcta dentro de [opción1..opciónK] — verificado leyendo el
  código real de bits/uncountable3.js y bits/dropdown.js (el <select> del
  motor legacy tiene un <option> en blanco en la posición 0, así que
  selectedIndex==correctArray[i] cae exactamente en la opción correspondiente).
- Si NO existe `correctArray` y el array tiene exactamente 4 elementos con
  el último puramente numérico (el ancho del <input>, en píxeles/caracteres):
  es **gapfill**. Elementos = [antes, respuesta, después, ancho(ignorado)].
- Si NO existe `correctArray` y el array tiene exactamente 2 elementos: es
  **matching** (pares). Todos los pares de una misma página se agrupan en
  UN solo ítem `{type:'matching', pairs:[{left,right},...]}` — no una fila
  por par, porque la interacción real (unir izquierda con derecha) es un
  solo ejercicio con N pares, igual que 'ordering' es un solo ejercicio con
  N palabras. Agregado el 2026-08-12 junto con el 5º tipo de componente en
  el motor (ver docs/architecture.md punto 4).
- Si NO existe `correctArray` y el array tiene 3+ elementos sin forma de
  gapfill: es **ordering** — los elementos, en ese orden, forman la frase
  correcta; el motor espera `words` ya barajadas (no se auto-barajan solas).
- Si la página NO tiene ningún `questionN = new Array(...)`: se prueba el
  formato **gappedtext** (`parse_gappedtext`) — un párrafo con 10-15 huecos
  numerados vía variables sueltas (`var one=".."`, `var two=".."`, ...) y
  cada hueco marcado inline con `<INPUT ... name=XXXt> <IMG ... name=XXX>`.
  Cada hueco se convierte en un ítem `gapfill` independiente (no un tipo de
  dato nuevo para "párrafo con huecos") — se fragmenta la vista continua del
  párrafo en items sueltos, con antes/después = el texto real que rodea a
  cada hueco en la página. Agregado el 2026-08-12.

## Qué se sigue excluyendo a propósito (no se fuerza ninguna conversión dudosa):
- Páginas "hub" que solo listan enlaces a sub-páginas de conjugación (ej.
  imperfetto.html -> imperfetto_cantare.html, ...): SÍ se siguen un nivel
  (sus sub-páginas son ejercicios reales en el formato ya soportado).
- Nivel CEFR: se asigna solo por marcador explícito en el nombre de archivo
  (ej. "A1_articoli.html", "B2_forma_passiva.html", "verbi_x_a2.html"). Los
  ítems sin marcador de nivel en el nombre NO se adivinan — van a
  content/exercises-sinnivel.json para revisión manual, siguiendo el
  principio de honestidad de datos del README (mejor "no sé" que un dato
  falso).

USO:
  cd italian-club-app
  python3 tools/exercise_scraper.py
Requiere conexión a internet real. Escribe content/exercises-<nivel>.json
(uno por nivel + "sinnivel") y tools/exercise-scraper-report.json con el
detalle de qué se convirtió y qué se omitió y por qué.
"""
import json
import random
import re
import time
import urllib.request
import urllib.error
from pathlib import Path

INDEX_URL = "https://onlineitalianclub.com/index-of-free-italian-exercises-and-grammar-lessons/"
BASE = "https://onlineitalianclub.com/free_italian_exercises/"
HEADERS = {"User-Agent": "Mozilla/5.0 (compatible; ItalianClubMigrationBot/1.0)"}
LEVELS = ["A1", "A2", "B1", "B2", "C1", "C2"]

random.seed(20260810)  # reproducible: mismo barajado de 'ordering' en cada corrida


def fetch(url, timeout=20):
    req = urllib.request.Request(url, headers=HEADERS)
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.read().decode("utf-8", errors="replace")


def clean_text(s):
    s = s.replace("&#8217;", "'").replace("&#8216;", "'")
    s = s.replace("&#8220;", '"').replace("&#8221;", '"')
    s = s.replace("&#8211;", "–").replace("&#8230;", "…")
    s = s.replace("&amp;", "&").replace("&nbsp;", " ")
    s = re.sub(r"<[^>]+>", " ", s)
    s = re.sub(r"\s+", " ", s)
    return s.strip()


def split_js_array_literal(inner):
    """Parsea el contenido entre paréntesis de `new Array(...)` en una lista
    de strings Python, respetando comillas y comas dentro de comillas."""
    items = []
    buf = []
    in_str = False
    quote = None
    i = 0
    n = len(inner)
    while i < n:
        ch = inner[i]
        if in_str:
            if ch == "\\" and i + 1 < n:
                buf.append(ch)
                buf.append(inner[i + 1])
                i += 2
                continue
            if ch == quote:
                in_str = False
                i += 1
                continue
            buf.append(ch)
            i += 1
            continue
        else:
            if ch in ("'", '"'):
                in_str = True
                quote = ch
                i += 1
                continue
            if ch == ",":
                items.append("".join(buf))
                buf = []
                i += 1
                continue
            if ch.isspace():
                i += 1
                continue
            # texto fuera de comillas y no coma/espacio: ignorar (no debería pasar)
            i += 1
            continue
    if buf or items:
        items.append("".join(buf))
    return [clean_text(x) for x in items]


QUESTION_VAR_RE = re.compile(
    r"question(\d+)\s*=\s*new\s+Array\s*\((.*?)\)\s*;", re.S
)
# Restringido a dígitos/comas/espacios: evita depender de que la línea
# termine en ';' (algunas páginas del sitio la omiten, ej. A1_articoli.html)
# y como el contenido nunca tiene paréntesis propios, no hay riesgo de
# cortar de más ni de menos.
CORRECT_ARRAY_RE = re.compile(r"correctArray\s*=\s*new\s+Array\s*\(([\d,\s]*)\)", re.S)
QUESTION_ARRAY_ORDER_RE = re.compile(r"questionArray\s*=\s*new\s+Array\s*\(([^)]*)\)")
H1_RE = re.compile(r"(?is)<h1[^>]*>(.*?)</h1>")
LOCAL_LINK_RE = re.compile(r'href="([a-zA-Z0-9_.\-()]+\.html)"')

LEVEL_FILENAME_RE = re.compile(r"(?i)(?:^|[_\-])([abc][12])(?:[_\-.(]|$)")


def detect_level(filename):
    m = LEVEL_FILENAME_RE.search(filename)
    if m:
        return m.group(1).upper()
    return None


def parse_questions(html):
    """Devuelve (items_elems: list[list[str]] en el orden de questionArray,
    correct_array: list[int] | None)."""
    qvars = {}
    for m in QUESTION_VAR_RE.finditer(html):
        idx = int(m.group(1))
        elems = split_js_array_literal(m.group(2))
        qvars[idx] = elems

    order_m = QUESTION_ARRAY_ORDER_RE.search(html)
    if order_m:
        order_nums = [int(n) for n in re.findall(r"question(\d+)", order_m.group(1))]
    else:
        order_nums = sorted(qvars.keys())

    ordered = [qvars[n] for n in order_nums if n in qvars]

    correct = None
    cm = CORRECT_ARRAY_RE.search(html)
    if cm:
        raw = [x.strip() for x in cm.group(1).split(",")]
        try:
            correct = [int(x) for x in raw if x != ""]
        except ValueError:
            correct = None
    return ordered, correct


def is_numeric(s):
    return bool(re.fullmatch(r"\d+", s.strip()))


def build_kicker(level, topic):
    return f"Ejercicios · {level or 'SIN_NIVEL'} · {topic}"


# ---- gappedtext*.js: un párrafo con 10-15 huecos numerados -----------------
# Forma de datos totalmente distinta a questionN=new Array(...): las
# respuestas son variables sueltas (var one=".."; var two=".."; ...) y el
# hueco vive inline en el párrafo como <INPUT ... name=XXXt> <IMG ... name=XXX>.
# Cada hueco se convierte en un ítem 'gapfill' independiente (antes/después
# = el texto real que lo rodea en el párrafo), en vez de inventar un tipo de
# dato nuevo para "párrafo con huecos" — más pragmático y sigue siendo 100%
# contenido real, solo que fragmentado en vez de una sola vista continua.
NUM_WORDS = ["one", "two", "three", "four", "five", "six", "seven", "eight",
             "nine", "ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen"]
GAPPEDTEXT_VAR_RE = re.compile(
    r'var\s+(' + "|".join(NUM_WORDS) + r')\s*=\s*"([^"]*)"'
)
BLANK_MARKER_RE = re.compile(
    r"<INPUT[^>]*name=(\w+)t[^>]*>\s*<IMG[^>]*>", re.I
)
STRONG_NUM_RE = re.compile(r"<STRONG>\s*\(\d+\)\s*</STRONG>", re.I)


def parse_gappedtext(html):
    answers = dict(GAPPEDTEXT_VAR_RE.findall(html))
    if not answers:
        return None

    last_script_end = 0
    for marker in ("</SCRIPT>", "</script>"):
        idx = html.rfind(marker)
        if idx != -1:
            last_script_end = max(last_script_end, idx + len(marker))

    end_idx = len(html)
    for marker in ("Back to", "Contact us"):
        idx = html.find(marker, last_script_end)
        if idx != -1:
            end_idx = min(end_idx, idx)

    passage_html = STRONG_NUM_RE.sub(" ", html[last_script_end:end_idx])

    matches = list(BLANK_MARKER_RE.finditer(passage_html))
    if not matches:
        return None

    segments = []
    prev_end = 0
    for m in matches:
        segments.append((passage_html[prev_end:m.start()], m.group(1)))
        prev_end = m.end()
    segments.append((passage_html[prev_end:], None))

    blanks = []
    for i in range(len(segments) - 1):
        before_raw, word_var = segments[i]
        after_raw = segments[i + 1][0]
        answer = answers.get(word_var)
        if not answer:
            continue
        before = clean_text(before_raw)[-220:].strip()
        after = clean_text(after_raw)[:220].strip()
        blanks.append((before, answer, after))
    return blanks or None


def convert_page(url, html, level, topic, report):
    ordered, correct = parse_questions(html)
    if not ordered:
        blanks = parse_gappedtext(html)
        if not blanks:
            return []
        items = []
        for before, answer, after in blanks:
            items.append({
                "type": "gapfill",
                "kicker": build_kicker(level, topic) + " (texto con huecos)",
                "title": topic,
                "before": before,
                "after": after,
                "answer": answer,
                "hint": "",
                "feedbackCorrect": "¡Correcto!",
                "feedbackWrong": "No es correcto. Inténtalo de nuevo.",
                "_source": url,
            })
        return items

    items = []
    matching_pairs = []
    for i, elems in enumerate(ordered):
        n = len(elems)
        try:
            if correct is not None:
                # mc: [antes, opt1..optK, despues], correct[i] es 1-based sobre opt1..optK
                if n < 3:
                    report["skipped"].append({"url": url, "q": i + 1, "reason": f"mc con muy pocos elementos ({n})"})
                    continue
                before, opts, after = elems[0], elems[1:-1], elems[-1]
                if i >= len(correct):
                    report["skipped"].append({"url": url, "q": i + 1, "reason": "correctArray más corto que questionArray"})
                    continue
                correct_idx = correct[i]
                if not (1 <= correct_idx <= len(opts)):
                    report["skipped"].append({"url": url, "q": i + 1, "reason": f"correctArray[{i}]={correct_idx} fuera de rango (1..{len(opts)})"})
                    continue
                correct_text = opts[correct_idx - 1]
                prompt = f"{before} ___ {after}".strip()
                items.append({
                    "type": "mc",
                    "kicker": build_kicker(level, topic),
                    "title": topic,
                    "prompt": prompt,
                    "options": [{"t": o, "correct": (o == correct_text and idx == correct_idx - 1)} for idx, o in enumerate(opts)],
                    "feedbackCorrect": "¡Correcto!",
                    "feedbackWrong": f'No es correcto. La respuesta correcta es: "{correct_text}".',
                    "_source": url,
                })
            elif n == 4 and is_numeric(elems[3]):
                before, answer, after = elems[0], elems[1], elems[2]
                if not answer:
                    report["skipped"].append({"url": url, "q": i + 1, "reason": "gapfill con respuesta vacía"})
                    continue
                items.append({
                    "type": "gapfill",
                    "kicker": build_kicker(level, topic),
                    "title": topic,
                    "before": before,
                    "after": after,
                    "answer": answer,
                    "hint": "",
                    "feedbackCorrect": "¡Correcto!",
                    "feedbackWrong": "No es correcto. Inténtalo de nuevo.",
                    "_source": url,
                })
            elif n == 2:
                left, right = elems[0], elems[1]
                if not left or not right:
                    report["skipped"].append({"url": url, "q": i + 1, "reason": "matching con un lado vacío"})
                    continue
                matching_pairs.append({"left": left, "right": right})
            elif n >= 3:
                words = list(elems)
                correct_sentence = " ".join(words)
                shuffled = words[:]
                tries = 0
                while shuffled == words and tries < 5:
                    random.shuffle(shuffled)
                    tries += 1
                items.append({
                    "type": "ordering",
                    "kicker": build_kicker(level, topic),
                    "title": topic,
                    "words": shuffled,
                    "correct": correct_sentence,
                    "explanation": "",
                    "_source": url,
                })
            else:
                report["skipped"].append({"url": url, "q": i + 1, "reason": f"forma de array no reconocida ({n} elementos, sin correctArray)"})
        except Exception as e:
            report["skipped"].append({"url": url, "q": i + 1, "reason": f"ERROR parseando: {e}"})

    if matching_pairs:
        items.append({
            "type": "matching",
            "kicker": build_kicker(level, topic),
            "title": topic,
            "pairs": matching_pairs,
            "explanation": "",
            "_source": url,
        })
    return items


def discover_pages():
    html = fetch(INDEX_URL)
    links = re.findall(r'<a href="(https://onlineitalianclub\.com/free_italian_exercises/[^"]+\.html)"', html)
    return sorted(set(links))


def main():
    report = {"converted_pages": [], "skipped_pages": [], "skipped": [], "hub_pages_followed": []}
    pages = discover_pages()
    print(f"[discover] {len(pages)} páginas en el índice principal", flush=True)

    all_items = {lvl: [] for lvl in LEVELS}
    all_items["SIN_NIVEL"] = []

    queue = list(pages)
    seen = set()
    visited_hubs = set()

    idx = 0
    while idx < len(queue):
        url = queue[idx]
        idx += 1
        if url in seen:
            continue
        seen.add(url)

        try:
            html = fetch(url)
        except (urllib.error.URLError, TimeoutError) as e:
            report["skipped_pages"].append({"url": url, "reason": f"fetch error: {e}"})
            print(f"[fetch-error] {url}: {e}", flush=True)
            time.sleep(0.3)
            continue

        filename = url.rstrip("/").rsplit("/", 1)[-1]
        h1m = H1_RE.search(html)
        topic = clean_text(h1m.group(1)) if h1m else filename.replace(".html", "").replace("_", " ")

        ordered, _ = parse_questions(html)
        level = detect_level(filename)
        if not ordered:
            # ¿es un texto con huecos múltiples (gappedtext)? probar antes de
            # asumir que es un hub o darla por perdida.
            if parse_gappedtext(html):
                items = convert_page(url, html, level, topic, report)
                if items:
                    bucket = level if level else "SIN_NIVEL"
                    all_items[bucket].extend(items)
                    report["converted_pages"].append({"url": url, "level": bucket, "topic": topic, "items": len(items)})
                    time.sleep(0.3)
                    continue
            # ¿página hub con enlaces a sub-páginas del mismo tema?
            if url not in visited_hubs and "/free_italian_exercises/" in url:
                visited_hubs.add(url)
                local_links = LOCAL_LINK_RE.findall(html)
                added = 0
                for ll in local_links:
                    full = BASE + ll
                    if full not in seen and full not in queue:
                        queue.append(full)
                        added += 1
                if added:
                    report["hub_pages_followed"].append({"url": url, "sub_pages_added": added})
                    print(f"[hub] {url} -> +{added} sub-páginas", flush=True)
                    time.sleep(0.3)
                    continue
            report["skipped_pages"].append({"url": url, "reason": "sin preguntas reconocibles (no es hub ni tiene questionArray)"})
            time.sleep(0.3)
            continue

        items = convert_page(url, html, level, topic, report)
        if items:
            bucket = level if level else "SIN_NIVEL"
            all_items[bucket].extend(items)
            report["converted_pages"].append({"url": url, "level": bucket, "topic": topic, "items": len(items)})
        else:
            report["skipped_pages"].append({"url": url, "reason": "0 ítems convertibles en esta página (ver report.skipped para el detalle)"})

        if len(seen) % 25 == 0:
            print(f"[progress] {len(seen)} páginas procesadas...", flush=True)
        time.sleep(0.3)

    content_dir = Path("content")
    total = 0
    for lvl in LEVELS + ["SIN_NIVEL"]:
        items = all_items[lvl]
        total += len(items)
        out = {
            "level": lvl,
            "migrated_at": time.strftime("%Y-%m-%d"),
            "source_note": "Migrado de /free_italian_exercises/*.html (motor de quiz JS legacy). "
                            "feedback genérico (no pedagógico) — ver README y docs/roadmap.md Fase 3.",
            "exercise_content": items,
        }
        fname = f"exercises-{lvl.lower()}.json" if lvl != "SIN_NIVEL" else "exercises-sinnivel.json"
        with open(content_dir / fname, "w", encoding="utf-8") as f:
            json.dump(out, f, ensure_ascii=False, indent=2)
        print(f"[write] content/{fname}: {len(items)} ítems")

    with open("tools/exercise-scraper-report.json", "w", encoding="utf-8") as f:
        json.dump(report, f, ensure_ascii=False, indent=2)

    print(f"\nTotal ítems convertidos: {total}")
    print(f"Páginas convertidas: {len(report['converted_pages'])}")
    print(f"Páginas omitidas: {len(report['skipped_pages'])}")
    print(f"Preguntas individuales omitidas dentro de páginas convertidas: {len(report['skipped'])}")
    print(f"Hubs seguidos: {len(report['hub_pages_followed'])}")


if __name__ == "__main__":
    main()
