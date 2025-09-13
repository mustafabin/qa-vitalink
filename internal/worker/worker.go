package worker

var (
    Worker *WebhookWorker
)

func InitWebhookWorker() {

	// Grabs from env in the NewWebhookWorker function
    config := WebhookConfig{
        URL:               "",
        MaxRetries:        0,  
        InitialRetryDelay: 0,  
        MaxRetryDelay:     0,  
        RetryBackoffFactor: 0, 
        WorkerCount:       0,  
        QueueSize:         0,  
        Timeout:           0,  
    }

    Worker = NewWebhookWorker(config)
    Worker.Start()
}

func ShutdownWebhookWorker() {
    if Worker != nil {
        Worker.Stop()
    }
}