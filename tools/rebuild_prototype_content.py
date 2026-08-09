#!/usr/bin/env python3
"""
Regenera el bloque GRAMMAR_CONTENT dentro de prototype/index.html usando
tu contenido migrado local (content/grammar-*.json), que debería tener
mejor cobertura que la copia usada originalmente para armar el prototipo.

Uso (desde la raíz del repo):
  python3 tools/rebuild_prototype_content.py
"""
import json
import re
from pathlib import Path

LEVELS = ['a1', 'a2', 'b1', 'b2', 'c1', 'c2']

def build_js():
    out = "const GRAMMAR_CONTENT = {\n"
    for lv in LEVELS:
        path = Path('content') / f'grammar-{lv}.json'
        if not path.exists():
            print(f"AVISO: no existe {path}, se omite {lv.upper()}")
            continue
        d = json.loads(path.read_text(encoding='utf-8'))
        out += f"  {lv.upper()}: [\n"
        for i in d['grammar_content']:
            title = i['title_it'].replace("\\", "\\\\").replace("'", "\\'")
            content = (i.get('content') or '(contenido pendiente de migrar)')
            content = content.replace("\\", "\\\\").replace("'", "\\'").replace("\n", " ")
            verified = 'status' not in i
            out += f"    {{title:'{title}', content:'{content}', verified:{'true' if verified else 'false'}}},\n"
        out += "  ],\n"
    out += "};\n"
    return out

def main():
    proto_path = Path('prototype/index.html')
    html = proto_path.read_text(encoding='utf-8')

    start_marker = "const GRAMMAR_CONTENT = {"
    if start_marker not in html:
        print("ERROR: no se encontró el bloque GRAMMAR_CONTENT en el prototipo.")
        return

    start = html.index(start_marker)
    # el bloque termina en el primer "};\n" después del inicio
    end = html.index("};\n", start) + len("};\n")

    new_block = build_js()
    new_html = html[:start] + new_block + html[end:]

    proto_path.write_text(new_html, encoding='utf-8')
    print(f"Listo. prototype/index.html actualizado con tu contenido local ({len(new_block)} caracteres).")

if __name__ == "__main__":
    main()
