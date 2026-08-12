# -*- coding: utf-8 -*-
"""
Clasificación manual (rol: profesor de italiano) de las 938 preguntas de
content/exercises-sinnivel.json que exercise_scraper.py no pudo asignar a un
nivel CEFR por nombre de archivo. Este archivo es el registro de CRITERIO,
no una herramienta reproducible que se corra de nuevo (ya se aplicó una vez
y content/exercises-sinnivel.json quedó vacío) — se conserva para que quede
explícito por qué cada página terminó en el nivel en que terminó.

Se clasifica por PÁGINA de origen (nombre de archivo), no por título: varios
títulos se repiten entre páginas de tiempos verbales distintos (ej. "Essere"
existe como drill de passato remoto Y de congiuntivo presente, con nivel
distinto cada vez), así que agrupar por título habría mezclado niveles.

Criterio, en orden de prioridad:
1. Match directo contra el tema ya clasificado en content/grammar-*.json
   cuando existe (ej. "Aggettivi possessivi" ya está en grammar-a2.json
   -> A2). Es la fuente más confiable porque ya es una decisión tomada
   y documentada en este mismo proyecto, no un criterio nuevo e
   independiente que podría contradecirla.
2. Para los ~53 drills de conjugación verbal (alcanzados desde páginas
   "hub" como pres_ind.html/imperfetto.html/condizionale.html/etc., sin
   nombre de tema propio — el nombre de archivo es solo el verbo:
   "essere.html", "cantare.html"...) se clasificó por TIEMPO VERBAL según
   la progresión CEFR estándar para italiano como lengua extranjera,
   también anclada en grammar-*.json donde ese tiempo aparece:
     - Presente indicativo (pres_ind_*)               -> A1
       (grammar-a1.json: "Verbi utili", "Verbi regolari")
     - Imperfetto (imperfetto_*)                        -> A2
       (grammar-a2.json: "Imperfetto")
     - Condizionale semplice (condizionale_*)           -> A2
       (grammar-a2.json: "Condizionale semplice")
     - Futuro semplice (futuro_semplice_*)               -> A2
       (grammar-a2.json: "Futuro semplice")
     - Congiuntivo presente (congiuntivo_presente_*)     -> B1
       (grammar-b1.json: "Congiuntivo presente")
     - Passato remoto (passato_remoto_*)                 -> B2
       (grammar-b2.json: "Passato remoto")
3. Para el resto (temas sueltos sin match exacto en grammar-*.json —
   comparativos, pronombres relativos, conectores, etc.) se usó juicio
   experto estándar de progresión CEFR para italiano LE.

Aplicado el 2026-08-12: 115 páginas de origen, 938 preguntas, 0 sin
clasificar. Ver migration-log/log.json para la nota de hito.
"""

LEVELS = {
    # --- páginas de tema suelto (criterio 1 o 3) ---
    "a_o_in.html": "A1",
    "aggettivi_contrari.html": "A1",
    "aggettivi_possessivi.html": "A2",
    "aggettiviindefiniti.html": "C1",
    "aggrelirr.html": "B2",
    "articoli.html": "A1",
    "ce_o_ci_sono_part_one.html": "A1",
    "ce_o_ci_sono_part_two.html": "A1",
    "con_senza_articolo.html": "C1",
    "concordanzatempi.html": "C2",
    "condizionale_o_congiuntivo.html": "B1",
    "condizionali.html": "A2",
    "congiuntivo_concordanza.html": "B1",
    "congiuntivo_indicativo.html": "B2",
    "connettivi.html": "B2",
    "connettivi1.html": "B2",
    "connettivi2.html": "B2",
    "discorso_indiretto_liv.html": "C1",
    "es1.html": "B1",
    "es6_1.html": "B2",
    "es6_2.html": "B2",
    "farcela.html": "A2",
    "fra_periodo_ipotetico.html": "B1",
    "fra_periodo_ipotetico2.html": "B1",
    "il_senso_degli_avverbi1.html": "B2",
    "il_senso_degli_avverbi2.html": "B2",
    "imperativo_formale_informale.html": "A2",
    "le_preposizioni_semplici.html": "B1",
    "metterci_mettere_mettersi.html": "C1",
    "nomi.html": "A1",
    "nomi2.html": "A1",
    "nomi3.html": "A1",
    "nomi_irregolari_machile_o_femminile.html": "A2",
    "nomi_irregolari_trasforma_al_femminile.html": "A2",
    "nonpleonastico.html": "C1",
    "paroleneo.html": "B2",
    "participio_passato_irregolare_1.html": "A2",
    "participio_passato_irregolare_2.html": "A2",
    "participio_passato_regolare.html": "A1",
    "passato_prossimo_essere_o_avere.html": "A1",
    "passato_prossimo_imperfetto.html": "A2",
    "periodo_ipotetico.html": "B1",
    "pinocchio_remoto_1.html": "B2",
    "pinocchio_remoto_2.html": "B2",
    "plurale_irregolare.html": "A1",
    "plurale_regolare.html": "A1",
    "posizioneaggettivi.html": "B2",
    "preposizioni.html": "B1",
    "preposizioni_articolate.html": "A2",
    "presente_passato_prossimo_imperfetto.html": "A2",
    "pronomi.html": "A1",
    "pronominali.html": "A2",
    "rel_o_ass.html": "B1",
    "tempo_del_congiuntivo.html": "B2",
    "un_po_di_preposizioni.html": "B1",
    "uso_del_condizionale.html": "A2",
    "verbi_irregolari.html": "A2",
    "verbi_pronominali.html": "C1",
    "word_order_lavori_o_studi.html": "B1",
    "word_order_lingue.html": "B1",
    "word_order_psicologia.html": "B1",
    "word_order_tecnologia.html": "B1",
}

# --- drills de conjugación verbal (por tiempo, criterio 2) ---
VERB_STEMS = {
    "pres_ind_": "A1",
    "imperfetto_": "A2",
    "condizionale_": "A2",
    "futuro_semplice_": "A2",
    "congiuntivo_presente_": "B1",
    "passato_remoto_": "B2",
}


def classify(filename):
    if filename in LEVELS:
        return LEVELS[filename]
    for stem, lvl in VERB_STEMS.items():
        if filename.startswith(stem):
            return lvl
    return None
