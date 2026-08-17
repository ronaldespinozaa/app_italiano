package main

import "testing"

func TestApplySM2_FailResetsStreak(t *testing.T) {
	card := SRSCard{EF: 2.5, Interval: 20, Reps: 5}
	got := applySM2(card, 1, "2026-08-17")
	if got.Reps != 0 {
		t.Errorf("reps = %d, quería 0 (falló, la racha se resetea)", got.Reps)
	}
	if got.Interval != 1 {
		t.Errorf("interval = %d, quería 1 (vuelve a empezar la progresión)", got.Interval)
	}
	if got.Due != "2026-08-18" {
		t.Errorf("due = %q, quería 2026-08-18 (hoy + 1 día)", got.Due)
	}
}

func TestApplySM2_SuccessProgression(t *testing.T) {
	// Progresión estándar de SM-2 para éxitos consecutivos: 1 -> 6 -> interval*EF.
	card := newSRSCard() // EF 2.5, reps 0
	today := "2026-08-17"

	card = applySM2(card, 4, today)
	if card.Reps != 1 || card.Interval != 1 {
		t.Fatalf("1er repaso: reps=%d interval=%d, quería reps=1 interval=1", card.Reps, card.Interval)
	}

	card = applySM2(card, 4, today)
	if card.Reps != 2 || card.Interval != 6 {
		t.Fatalf("2do repaso: reps=%d interval=%d, quería reps=2 interval=6", card.Reps, card.Interval)
	}

	prevInterval, prevEF := card.Interval, card.EF
	card = applySM2(card, 4, today)
	wantInterval := int(float64(prevInterval) * prevEF)
	if card.Reps != 3 {
		t.Errorf("3er repaso: reps=%d, quería 3", card.Reps)
	}
	if card.Interval < wantInterval-1 || card.Interval > wantInterval+1 {
		t.Errorf("3er repaso: interval=%d, quería ~%d (interval anterior * EF)", card.Interval, wantInterval)
	}
}

func TestApplySM2_EasinessFactorHasFloor(t *testing.T) {
	card := SRSCard{EF: 1.3, Interval: 1, Reps: 1}
	// Muchas respuestas de calidad mínima (0) en fila no deben hundir EF
	// por debajo del piso de 1.3 que define el algoritmo.
	for i := 0; i < 20; i++ {
		card = applySM2(card, 0, "2026-08-17")
	}
	if card.EF < 1.3 {
		t.Errorf("EF = %v, no debería bajar de 1.3 (piso del algoritmo)", card.EF)
	}
}

func TestApplySM2_QualityIsClamped(t *testing.T) {
	low := applySM2(newSRSCard(), -5, "2026-08-17")
	high := applySM2(newSRSCard(), 99, "2026-08-17")
	wantLow := applySM2(newSRSCard(), 0, "2026-08-17")
	wantHigh := applySM2(newSRSCard(), 5, "2026-08-17")
	if low != wantLow {
		t.Errorf("quality -5 no se clampeó a 0: got %+v, quería %+v", low, wantLow)
	}
	if high != wantHigh {
		t.Errorf("quality 99 no se clampeó a 5: got %+v, quería %+v", high, wantHigh)
	}
}

func TestPickVocabIndex_PrefersOverdueThenNewThenNearestFuture(t *testing.T) {
	// idx 0: vencido hace tiempo. idx 1: nunca calificado. idx 2: vencido hoy mismo.
	srs := map[string]SRSCard{
		"0": {EF: 2.5, Due: "2026-08-10"},
		"2": {EF: 2.5, Due: "2026-08-17"},
	}
	got := pickVocabIndex(srs, 3, "2026-08-17")
	if got != 0 {
		t.Errorf("con vencidos disponibles, got idx %d, quería 0 (el más atrasado)", got)
	}

	// Sin vencidos: el nunca calificado gana sobre uno programado a futuro.
	srs2 := map[string]SRSCard{
		"0": {EF: 2.5, Due: "2026-08-20"},
	}
	got2 := pickVocabIndex(srs2, 2, "2026-08-17")
	if got2 != 1 {
		t.Errorf("sin vencidos, got idx %d, quería 1 (el nunca calificado)", got2)
	}

	// Todo calificado y a futuro: gana el due más próximo.
	srs3 := map[string]SRSCard{
		"0": {EF: 2.5, Due: "2026-08-25"},
		"1": {EF: 2.5, Due: "2026-08-19"},
	}
	got3 := pickVocabIndex(srs3, 2, "2026-08-17")
	if got3 != 1 {
		t.Errorf("todo a futuro, got idx %d, quería 1 (due más próximo)", got3)
	}
}

func TestPickVocabIndex_EmptyStateReturnsFirstItem(t *testing.T) {
	got := pickVocabIndex(map[string]SRSCard{}, 5, "2026-08-17")
	if got != 0 {
		t.Errorf("sin estado guardado, got idx %d, quería 0 (primer ítem nuevo)", got)
	}
}

func TestVocabProgressCounts(t *testing.T) {
	srs := map[string]SRSCard{
		"0": {Reps: 4, Due: "2026-08-10"}, // visto, vencido, dominado (reps>=3)
		"1": {Reps: 1, Due: "2026-08-20"}, // visto, no vencido, no dominado
	}
	seen, due, mastered := vocabProgressCounts(srs, 4, "2026-08-17")
	if seen != 2 {
		t.Errorf("seen = %d, quería 2", seen)
	}
	if due != 1 {
		t.Errorf("due = %d, quería 1", due)
	}
	if mastered != 1 {
		t.Errorf("mastered = %d, quería 1", mastered)
	}
}
