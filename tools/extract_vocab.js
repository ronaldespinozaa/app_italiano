#!/usr/bin/env node
// Extrae REAL_VOCAB de prototype/index.html y lo convierte al esquema JSON
// que consume wasmapp/main.go (window.vocabEngine.load): [{word,back},...].
//
// Mismo motivo que tools/extract_exercise_queues.js para usar Node y no un
// parser JSON: REAL_VOCAB es un objeto LITERAL de JS (comillas simples,
// strings con "/" y apóstrofos italianos como l'aspetto), no JSON válido.
//
// Uso (desde italian-club-app/):
//   node tools/extract_vocab.js
//
// Escribe wasmapp/data/vocab-<NIVEL>.json (uno por nivel), con el shape:
//   { "level": "A1", "extracted_from": "prototype/index.html",
//     "items": [ {word, back}, ... ] }
//
// Solo A1/A2/B1/B2/C1 — C2 no tiene listas de vocabulario en el sitio
// original (vocabSetForLevel() en prototype/index.html devuelve 'NO_VOCAB'
// para C2 sin siquiera mirar PLACEHOLDER_VOCAB), así que no hay nada real
// que extraer para ese nivel.
//
// OJO: al igual que wasmapp/data/exercises-*.json, esto es un derivado
// gitignored del contenido real (que vive en prototype/index.html), no una
// fuente nueva — se regenera cuando cambie REAL_VOCAB.
"use strict";
const fs = require("fs");
const path = require("path");

const repoRoot = path.join(__dirname, "..");
const protoPath = path.join(repoRoot, "prototype", "index.html");
const html = fs.readFileSync(protoPath, "utf8");

const marker = "const REAL_VOCAB = ";
const start = html.indexOf(marker);
if (start === -1) {
  console.error("No se encontró 'const REAL_VOCAB =' en " + protoPath);
  process.exit(1);
}
const exprStart = start + marker.length;

// Bracket-matching consciente de strings/escapes y de comentarios de línea
// (REAL_VOCAB tiene un comentario `//` adentro, antes de C1) — igual técnica
// que extract_exercise_queues.js, con el agregado de saltear comentarios
// para no confundir un '{' o ';' mencionado ahí con código real.
let depth = 0;
let inString = null; // "'" | '"' | null
let inLineComment = false;
let i = exprStart;
for (; i < html.length; i++) {
  const c = html[i];
  const next = html[i + 1];
  if (inLineComment) {
    if (c === "\n") inLineComment = false;
    continue;
  }
  if (inString) {
    if (c === "\\") { i++; continue; }
    if (c === inString) inString = null;
    continue;
  }
  if (c === "/" && next === "/") { inLineComment = true; i++; continue; }
  if (c === "'" || c === '"') { inString = c; continue; }
  if (c === "{" || c === "[") depth++;
  else if (c === "}" || c === "]") depth--;
  else if (c === ";" && depth === 0) break;
}
if (i >= html.length) {
  console.error("No se encontró el ';' de cierre de REAL_VOCAB");
  process.exit(1);
}
const objLiteralSrc = html.slice(exprStart, i);

// eslint-disable-next-line no-new-func
const REAL_VOCAB = new Function("return (" + objLiteralSrc + ")")();

const outDir = path.join(repoRoot, "wasmapp", "data");
fs.mkdirSync(outDir, { recursive: true });
let totalItems = 0;

for (const [level, list] of Object.entries(REAL_VOCAB)) {
  const items = list.map((entry) => {
    const [word, back] = entry.split("/");
    return { word, back };
  });

  const outPath = path.join(outDir, `vocab-${level.toLowerCase()}.json`);
  const payload = { level, extracted_from: "prototype/index.html", item_count: items.length, items };
  fs.writeFileSync(outPath, JSON.stringify(payload, null, 2) + "\n", "utf8");
  console.log(`${level}: ${items.length} ítems -> ${path.relative(repoRoot, outPath)}`);
  totalItems += items.length;
}

console.log(`\nTotal: ${totalItems} ítems extraídos. (C2 no tiene vocabulario real, no se generó vocab-c2.json.)`);
