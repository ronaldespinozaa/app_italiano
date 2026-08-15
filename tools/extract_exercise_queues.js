#!/usr/bin/env node
// Extrae EXERCISE_QUEUES de prototype/index.html y lo convierte al esquema
// JSON que consume wasmapp/main.go (window.exerciseEngine.load).
//
// Por qué Node y no Python: EXERCISE_QUEUES es un objeto LITERAL de JS
// (comillas simples, claves sin comillas, escapes tipo \\'), no JSON válido
// — parsearlo a mano con regex es frágil. Node lo puede evaluar como el
// JS que realmente es.
//
// Uso (desde italian-club-app/):
//   node tools/extract_exercise_queues.js
//
// Escribe wasmapp/data/exercises-<NIVEL>.json (uno por nivel), con el shape:
//   { "level": "A1", "extracted_from": "prototype/index.html",
//     "items": [ {type, id, ...campos según tipo}, ... ] }
//
// OJO: esto NO va en content/ — ahí ya vive content/exercises-<nivel>.json,
// el contenido migrado canónico (formato legacy exercise_content, con
// kicker/title, fuente de EXERCISE_QUEUES). Este script genera un derivado
// con el shape que consume wasmapp/main.go, específico de este módulo —
// mismo criterio que wasmapp/dist/ (build artifact, gitignored).
//
// La conversión de shape (prototype -> wasmapp) es mecánica:
//   - mc: options:[{t,correct}] -> options:[string], correct:index
//   - gapfill: answer/hint/before/after se copian tal cual
//   - truefalse: answer (bool) -> boolAnswer
//   - ordering: correct (string) -> correctSentence
//   - matching: pairs:[{left,right}] se copia tal cual
"use strict";
const fs = require("fs");
const path = require("path");

const repoRoot = path.join(__dirname, "..");
const protoPath = path.join(repoRoot, "prototype", "index.html");
const html = fs.readFileSync(protoPath, "utf8");

const marker = "const EXERCISE_QUEUES = ";
const start = html.indexOf(marker);
if (start === -1) {
  console.error("No se encontró 'const EXERCISE_QUEUES =' en " + protoPath);
  process.exit(1);
}
const exprStart = start + marker.length;

// Bracket-matching consciente de strings/escapes, para encontrar el ';'
// que cierra la declaración sin confundirse con llaves/comillas dentro de
// los valores.
let depth = 0;
let inString = null; // "'" | '"' | null
let i = exprStart;
for (; i < html.length; i++) {
  const c = html[i];
  if (inString) {
    if (c === "\\") { i++; continue; } // saltea el carácter escapado
    if (c === inString) inString = null;
    continue;
  }
  if (c === "'" || c === '"') { inString = c; continue; }
  if (c === "{" || c === "[") depth++;
  else if (c === "}" || c === "]") depth--;
  else if (c === ";" && depth === 0) break;
}
if (i >= html.length) {
  console.error("No se encontró el ';' de cierre de EXERCISE_QUEUES");
  process.exit(1);
}
const objLiteralSrc = html.slice(exprStart, i);

// eslint-disable-next-line no-new-func
const EXERCISE_QUEUES = new Function("return (" + objLiteralSrc + ")")();

const outDir = path.join(repoRoot, "wasmapp", "data");
fs.mkdirSync(outDir, { recursive: true });
let totalItems = 0;

for (const [level, queue] of Object.entries(EXERCISE_QUEUES)) {
  const items = queue.map((ex, idx) => {
    const base = { type: ex.type, id: `${level}-${idx}` };
    switch (ex.type) {
      case "mc": {
        const options = ex.options.map((o) => o.t);
        const correct = ex.options.findIndex((o) => o.correct);
        return { ...base, prompt: ex.prompt, options, correct };
      }
      case "gapfill":
        return { ...base, before: ex.before, after: ex.after, answer: ex.answer, hint: ex.hint };
      case "truefalse":
        return { ...base, statement: ex.statement, boolAnswer: ex.answer };
      case "ordering":
        return { ...base, words: ex.words, correctSentence: ex.correct };
      case "matching":
        return { ...base, pairs: ex.pairs };
      default:
        throw new Error(`tipo desconocido '${ex.type}' en ${level}[${idx}] ('${ex.title}')`);
    }
  });

  const outPath = path.join(outDir, `exercises-${level.toLowerCase()}.json`);
  const payload = { level, extracted_from: "prototype/index.html", item_count: items.length, items };
  fs.writeFileSync(outPath, JSON.stringify(payload, null, 2) + "\n", "utf8");
  console.log(`${level}: ${items.length} ítems -> ${path.relative(repoRoot, outPath)}`);
  totalItems += items.length;
}

console.log(`\nTotal: ${totalItems} ítems extraídos.`);
