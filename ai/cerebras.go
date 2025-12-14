package ai

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
)

// Message is the standard structure for LLM chat history.
type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

// CerebrasRequest models the API request payload.
type CerebrasRequest struct {
    Model    string    `json:"model"`
    Messages []Message `json:"messages"`
}

// CerebrasResponse models the API response structure.
type CerebrasResponse struct {
    Choices []struct {
        Message Message `json:"message"`
    } `json:"choices"`
}

// GetCerebrasResponse sends the conversation history to the Cerebras API.
func GetCerebrasResponse(history []Message) (string, error) {
    apiKey := os.Getenv("CEREBRAS_API_KEY")
    if apiKey == "" {
        return "", fmt.Errorf("CEREBRAS_API_KEY not set")
    }

    // The model and API endpoint may need updating based on Cerebras's current documentation
    url := "https://api.cerebras.com/v1/completions" 
    model := "llama-3.3-70b" // Example model

    reqPayload := CerebrasRequest{
        Model:    model,
        Messages: history,
    }

    body, _ := json.Marshal(reqPayload)
    
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil {
        return "", fmt.Errorf("creating request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer " + apiKey)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", fmt.Errorf("making API call: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        var errBody bytes.Buffer
        errBody.ReadFrom(resp.Body)
        return "", fmt.Errorf("API call failed with status %d: %s", resp.StatusCode, errBody.String())
    }

    var apiResp CerebrasResponse
    if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
        return "", fmt.Errorf("decoding API response: %w", err)
    }

    if len(apiResp.Choices) == 0 {
        return "Sorry, the AI did not provide a response.", nil
    }

    return apiResp.Choices[0].Message.Content, nil
}
