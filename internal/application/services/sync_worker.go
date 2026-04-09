package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings" // Agregado para manejar el cuerpo de la petición
	"time"

	"github.com/intigritypos/integritypos/internal/infrastructure/persistence"
)

// SyncWorker consume el outbox pendiente y lo replica a la nube.
type SyncWorker struct {
	outboxRepo *persistence.OutboxRepo
	cloudURL   string
	apiKey     string
	maxRetries int
	baseBo     time.Duration // base backoff
	stopCh     chan struct{}
	isRunning  bool
}

func NewSyncWorker(
	outboxRepo *persistence.OutboxRepo,
	cloudURL string,
	apiKey string,
) *SyncWorker {
	return &SyncWorker{
		outboxRepo: outboxRepo,
		cloudURL:   cloudURL,
		apiKey:     apiKey,
		maxRetries: 5,
		baseBo:     time.Second,
		stopCh:     make(chan struct{}),
	}
}

// Start inicia el loop de sincronización en background.
func (w *SyncWorker) Start(ctx context.Context, pollInterval time.Duration) {
	if w.isRunning {
		return
	}
	w.isRunning = true

	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-w.stopCh:
				w.isRunning = false
				log.Println("[SyncWorker] Detenido")
				return
			case <-ticker.C:
				w.syncBatch(ctx)
			}
		}
	}()

	log.Println("[SyncWorker] Iniciado con intervalo", pollInterval)
}

// Stop detiene el worker de forma limpia.
func (w *SyncWorker) Stop() {
	if w.isRunning {
		w.stopCh <- struct{}{}
	}
}

// syncBatch consume hasta 10 eventos pendientes del outbox.
func (w *SyncWorker) syncBatch(ctx context.Context) {
	entries, err := w.outboxRepo.GetPending(ctx, 10)
	if err != nil {
		log.Printf("[SyncWorker] Error leyendo outbox: %v", err)
		return
	}

	if len(entries) == 0 {
		return
	}

	log.Printf("[SyncWorker] Procesando %d eventos pendientes", len(entries))

	for _, entry := range entries {
		w.syncEntry(ctx, entry)
	}
}

// syncEntry intenta sincronizar un evento con retry.
func (w *SyncWorker) syncEntry(ctx context.Context, entry *persistence.OutboxEntryRow) {
	for attempt := 0; attempt < w.maxRetries; attempt++ {
		backoff := time.Duration(math.Pow(2, float64(attempt))) * w.baseBo
		if attempt > 0 {
			time.Sleep(backoff)
		}

		cloudID, err := w.sendToCloud(ctx, entry)
		if err == nil {
			if err := w.outboxRepo.MarkSynced(ctx, entry.ID, cloudID); err != nil {
				log.Printf("[SyncWorker] Error marcando synced %s: %v", entry.ID, err)
			} else {
				log.Printf("[SyncWorker] ✓ Sincronizado %s (cloud_id=%s)", entry.ID, cloudID)
			}
			return
		}

		log.Printf("[SyncWorker] Intento %d de %d para %s falló: %v", attempt+1, w.maxRetries, entry.ID, err)
	}

	if err := w.outboxRepo.MarkFailed(ctx, entry.ID, "max retries exceeded"); err != nil {
		log.Printf("[SyncWorker] Error marcando failed %s: %v", entry.ID, err)
	}
}

// sendToCloud envía un evento a la API en la nube.
func (w *SyncWorker) sendToCloud(ctx context.Context, entry *persistence.OutboxEntryRow) (string, error) {
	if w.cloudURL == "" {
		return "", fmt.Errorf("cloud URL no configurada")
	}

	// Construir payload JSON
	body := fmt.Sprintf(`{"entity_type":"%s","entity_id":"%s","payload":%s}`,
		entry.EntityType, entry.EntityID, entry.Payload)

	// CORRECCIÓN: Se inyecta strings.NewReader(body) para usar la variable y enviar el contenido
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.cloudURL+"/sync", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("error creando request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error enviando a nube: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("cloud retornó %d", resp.StatusCode)
	}

	return fmt.Sprintf("cloud-%d", time.Now().UnixNano()), nil
}

// GetStats retorna estadísticas del outbox.
func (w *SyncWorker) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats, err := w.outboxRepo.GetStats(ctx)
	if err != nil {
		return nil, err
	}
	stats["is_running"] = w.isRunning
	stats["max_retries"] = w.maxRetries
	return stats, nil
}
