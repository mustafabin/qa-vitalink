package worker

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log"
    "math/rand"
    "net"
    "net/http"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"

    "vitalink/internal/models"
)

/* -------------------------------------------------------------------------
   Configuration
--------------------------------------------------------------------------*/

type WebhookConfig struct {
    URL                string        `json:"url"`
    MaxRetries         int           `json:"max_retries"`          // ← fixed json tag
    InitialRetryDelay  time.Duration `json:"initial_retry_delay"`  // ← fixed field name
    MaxRetryDelay      time.Duration `json:"max_retry_delay"`
    RetryBackoffFactor float64       `json:"retry_backoff_factor"`
    WorkerCount        int           `json:"worker_count"`
    QueueSize          int           `json:"queue_size"`
    Timeout            time.Duration `json:"timeout"`
}

// Default values that are used when no env‑vars are set.
var DefaultConfig = WebhookConfig{
    URL:                "http://localhost:9000/webhook/payment-fulfilled",
    MaxRetries:         5,
    InitialRetryDelay: 1 * time.Second,
    MaxRetryDelay:     30 * time.Second, // ← typo fixed
    RetryBackoffFactor: 2.0,
    WorkerCount:        10,
    QueueSize:          1000,
    Timeout:            10 * time.Second,
}

/* -------------------------------------------------------------------------
   Payload / Job definitions
--------------------------------------------------------------------------*/

type WebhookPayload struct {
    PaymentPageID string                 `json:"payment_page_id"`
    MerchantID    string                 `json:"merchant_id"`
    RvcID         string                 `json:"rvc_id"`
    Status        string                 `json:"status"`
    Last4         string                 `json:"last4"`
    Brand         string                 `json:"brand"`
    Timestamp     time.Time              `json:"timestamp"`
    RawResponse   map[string]interface{} `json:"raw_response,omitempty"`
}

type WebhookJob struct {
    Ctx      context.Context
    Page     *models.PaymentPage
    DcResp   map[string]any
    Attempt  int
    Callback func(error) // optional – called when the job finally succeeds / fails
}

/* -------------------------------------------------------------------------
   Worker implementation
--------------------------------------------------------------------------*/

type WebhookWorker struct {
    config    WebhookConfig
    queue     chan WebhookJob
    waitGroup sync.WaitGroup
    stopChan  chan struct{}
}

// NewWebhookWorker reads environment variables, falls back to defaults and returns a ready‑to‑start worker.
func NewWebhookWorker(cfg WebhookConfig) *WebhookWorker {
    // Resolve values from env – empty values in cfg mean “use default”.
    url := getEnv("WEBHOOK_URL", cfg.URL)
    if url == "" {
        url = DefaultConfig.URL // final fallback
    }
    cfg.URL = url

    if cfg.MaxRetries == 0 {
        cfg.MaxRetries = getEnvAsInt("WEBHOOK_MAX_RETRIES", DefaultConfig.MaxRetries)
    }
    if cfg.InitialRetryDelay == 0 {
        cfg.InitialRetryDelay = getEnvAsDuration("WEBHOOK_INITIAL_RETRY_DELAY", DefaultConfig.InitialRetryDelay)
    }
    if cfg.MaxRetryDelay == 0 {
        cfg.MaxRetryDelay = getEnvAsDuration("WEBHOOK_MAX_RETRY_DELAY", DefaultConfig.MaxRetryDelay)
    }
    if cfg.RetryBackoffFactor == 0 {
        cfg.RetryBackoffFactor = getEnvAsFloat("WEBHOOK_BACKOFF_FACTOR", DefaultConfig.RetryBackoffFactor)
    }
    if cfg.WorkerCount == 0 {
        cfg.WorkerCount = getEnvAsInt("WEBHOOK_WORKER_COUNT", DefaultConfig.WorkerCount)
    }
    if cfg.QueueSize == 0 {
        cfg.QueueSize = getEnvAsInt("WEBHOOK_QUEUE_SIZE", DefaultConfig.QueueSize)
    }
    if cfg.Timeout == 0 {
        cfg.Timeout = getEnvAsDuration("WEBHOOK_TIMEOUT", DefaultConfig.Timeout)
    }

    log.Printf("Webhook worker configured – URL: %s, workers: %d, queue: %d", cfg.URL, cfg.WorkerCount, cfg.QueueSize)

    return &WebhookWorker{
        config:   cfg,
        queue:    make(chan WebhookJob, cfg.QueueSize),
        stopChan: make(chan struct{}),
    }
}

// Start spawns the configured number of goroutine workers.
func (w *WebhookWorker) Start() {
    log.Printf("Starting %d webhook workers", w.config.WorkerCount)
    for i := 0; i < w.config.WorkerCount; i++ {
        w.waitGroup.Add(1) // <- required argument
        go w.worker(i)
    }
}

// Stop gracefully shuts down the workers.
func (w *WebhookWorker) Stop() {
    close(w.stopChan)
    w.waitGroup.Wait()
    log.Println("Webhook worker stopped")
}

// worker loops waiting for jobs or a stop signal.
func (w *WebhookWorker) worker(id int) {
    defer w.waitGroup.Done()
    for {
        select {
        case <-w.stopChan:
            return
        case job := <-w.queue:
            w.processWebhookJob(job)
        }
    }
}

// processWebhookJob sends the webhook and, on failure, retries with back‑off.
func (w *WebhookWorker) processWebhookJob(job WebhookJob) {
    if job.Attempt >= w.config.MaxRetries {
        log.Printf("[worker] max retries exceeded for page %s", job.Page.PageUID)
        if job.Callback != nil {
            job.Callback(errors.New("max retries exceeded"))
        }
        return
    }

    payload := WebhookPayload{
        PaymentPageID: job.Page.PageUID,
        RvcID:         job.Page.RvcID,
        MerchantID:    job.Page.MerchantID,
        Status:        "paid",
        Last4:         getString(job.DcResp, "Last4"),
        Brand:         getString(job.DcResp, "Brand"),
        Timestamp:     time.Now().UTC(),
        RawResponse:   job.DcResp,
    }

    err := w.sendWebhookRequest(job.Ctx, payload)
    if err == nil {
        log.Printf("[worker] webhook succeeded for page %s", job.Page.PageUID)
        if job.Callback != nil {
            job.Callback(nil)
        }
        return
    }

    if !w.isRetryableError(err) {
        log.Printf("[worker] non‑retryable error for page %s: %v", job.Page.PageUID, err)
        if job.Callback != nil {
            job.Callback(err)
        }
        return
    }

    // Schedule a retry with exponential back‑off + jitter
    delay := w.calculateRetryDelay(job.Attempt)
    log.Printf("[worker] attempt %d/%d failed for page %s – retrying in %v: %v",
        job.Attempt+1, w.config.MaxRetries, job.Page.PageUID, delay, err)

    time.AfterFunc(delay, func() {
        w.QueueWebhook(WebhookJob{
            Ctx:      job.Ctx,
            Page:     job.Page,
            DcResp:   job.DcResp,
            Attempt:  job.Attempt + 1,
            Callback: job.Callback,
        })
    })
}

// QueueWebhook puts a job onto the internal channel (non‑blocking, with fallback error).
func (w *WebhookWorker) QueueWebhook(job WebhookJob) {
    select {
    case w.queue <- job:
        // queued successfully
    default:
        log.Printf("[worker] queue full – dropping webhook for page %s", job.Page.PageUID)
        if job.Callback != nil {
            job.Callback(errors.New("webhook queue full"))
        }
    }
}

/* -------------------------------------------------------------------------
   HTTP request handling
--------------------------------------------------------------------------*/

func (w *WebhookWorker) sendWebhookRequest(ctx context.Context, payload WebhookPayload) error {
    body, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal payload: %w", err)
    }
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.config.URL, bytes.NewBuffer(body))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", "Vitalink/1.0")

    client := &http.Client{Timeout: w.config.Timeout}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("request failed: %w", err) // ← fixed formatting
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(b))
    }
    return nil
}

/* -------------------------------------------------------------------------
   Retry / error helpers
--------------------------------------------------------------------------*/

func (w *WebhookWorker) isRetryableError(err error) bool {
    if err == nil {
        return false
    }
    // Network timeout / temporary failures
    var netErr net.Error
    if errors.As(err, &netErr) && netErr.Timeout() {
        return true
    }
    // 5xx server errors are retryable, 429 (rate‑limit) also retryable
    if strings.Contains(err.Error(), "status 5") || strings.Contains(err.Error(), "status 429") {
        return true
    }
    // Generic connection problems
    if strings.Contains(err.Error(), "connection") ||
        strings.Contains(err.Error(), "timeout") ||
        strings.Contains(err.Error(), "reset") {
        return true
    }
    return false
}

// exponential back‑off with jitter (random +/-25%)
func (w *WebhookWorker) calculateRetryDelay(attempt int) time.Duration {
    delay := w.config.InitialRetryDelay
    for i := 0; i < attempt; i++ {
        delay = time.Duration(float64(delay) * w.config.RetryBackoffFactor)
        if delay > w.config.MaxRetryDelay {
            delay = w.config.MaxRetryDelay
        }
    }
    return addJitter(delay)
}

/* -------------------------------------------------------------------------
   Small utilities
--------------------------------------------------------------------------*/

func addJitter(d time.Duration) time.Duration {
    // 0‑25% jitter
    j := time.Duration(rand.Int63n(int64(d / 4)))
    return d + j - (j / 2)
}

func getString(m map[string]any, key string) string {
    if v, ok := m[key]; ok {
        if s, ok := v.(string); ok {
            return s
        }
    }
    return ""
}

/* -------------------------------------------------------------------------
   Env‑var helpers (used by NewWebhookWorker)
--------------------------------------------------------------------------*/

func getEnv(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}

func getEnvAsInt(key string, def int) int {
    if v := os.Getenv(key); v != "" {
        if i, err := strconv.Atoi(v); err == nil {
            return i
        }
    }
    return def
}

func getEnvAsDuration(key string, def time.Duration) time.Duration {
    if v := os.Getenv(key); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            return d
        }
    }
    return def
}

func getEnvAsFloat(key string, def float64) float64 {
    if v := os.Getenv(key); v != "" {
        if f, err := strconv.ParseFloat(v, 64); err == nil {
            return f
        }
    }
    return def
}