package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"matrix-api/models"
)

type StatisticsClient struct {
	URL        string
	HTTPClient *http.Client
}

func (client StatisticsClient) Calculate(ctx context.Context, operation string, matrices map[string]models.Matrix) (json.RawMessage, error) {
	payload, err := json.Marshal(struct {
		Operation string                   `json:"operation"`
		Matrices  map[string]models.Matrix `json:"matrices"`
	}{operation, matrices})
	if err != nil {
		return nil, fmt.Errorf("codificar la solicitud de estadísticas: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("crear la solicitud de estadísticas: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("llamar al servicio de estadísticas: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("leer la respuesta de estadísticas: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("el servicio de estadísticas respondió %d: %s", response.StatusCode, body)
	}

	var result struct {
		Statistics json.RawMessage `json:"statistics"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decodificar la respuesta de estadísticas: %w", err)
	}

	return result.Statistics, nil
}
