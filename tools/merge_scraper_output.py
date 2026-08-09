#!/usr/bin/env python3
"""
Fusiona grammar-final.json (salida de grammar-scraper.go) con los
content/grammar-*.json existentes, nivel por nivel.

Uso:
  cd italian-club-app
  python3 tools/merge_scraper_output.py tools/grammar-final.json

Reglas de fusión:
- Si el ítem del scraper tiene contenido real (sin "status" de error) y el
  existente tenía status "pendiente/externo", se reemplaza y se actualiza
  el título del status a "migrado con grammar-scraper.go".
- Si el ítem del scraper falló (status ERROR/ADVERTENCIA/NO ENCONTRADO), se
  deja el existente intacto y se agrega una nota para revisión manual.
- Nunca se sobreescribe contenido que ya estaba marcado como 100% verbatim
  del sitio (sin campo "status").
"""
import json
import sys
from pathlib import Path

def load(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))

def save(path, data):
    Path(path).write_text(
        json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8"
    )

def normalize_level_key(level_key):
    # el scraper usa "B1-descubiertos" para el lote de descubrimiento automático
    return level_key.replace("-descubiertos", "").upper()

def merge_level(scraper_items, existing_path):
    if not Path(existing_path).exists():
        print(f"  AVISO: no existe {existing_path}, se omite")
        return 0

    existing = load(existing_path)
    by_title = {i["title_it"]: i for i in existing["grammar_content"]}
    updated = 0

    for s_item in scraper_items:
        title = s_item.get("title_it", "")
        matched = None
        # match exacto o por coincidencia parcial (títulos pueden variar levemente)
        if title in by_title:
            matched = by_title[title]
        else:
            for k in by_title:
                if title.lower() in k.lower() or k.lower() in title.lower():
                    matched = by_title[k]
                    break

        if not matched:
            print(f"  SIN MATCH: '{title}' no encontrado en {existing_path}, se omite")
            continue

        scraper_ok = s_item.get("content") and not s_item.get("status")
        existing_had_status = "status" in matched

        if scraper_ok and existing_had_status:
            matched["content"] = s_item["content"]
            matched["url"] = s_item.get("url", matched.get("url", ""))
            matched.pop("status", None)
            matched["migrated_via"] = "grammar-scraper.go"
            updated += 1
        elif not scraper_ok:
            note = s_item.get("status", "sin contenido")
            print(f"  SCRAPER FALLÓ para '{title}': {note} — se conserva el existente")
        # si scraper_ok pero el existente YA era verbatim (sin status), no se toca

    save(existing_path, existing)
    return updated

def main():
    if len(sys.argv) != 2:
        print("Uso: python3 merge_scraper_output.py <ruta a grammar-final.json>")
        sys.exit(1)

    scraper_data = load(sys.argv[1])
    content_dir = Path("content")

    total_updated = 0
    for level_block in scraper_data:
        level = normalize_level_key(level_block["level"])
        existing_path = content_dir / f"grammar-{level.lower()}.json"
        print(f"Fusionando {level} -> {existing_path}")
        total_updated += merge_level(level_block["grammar_content"], existing_path)

    print(f"\nTotal de ítems actualizados a contenido verbatim: {total_updated}")

if __name__ == "__main__":
    main()
