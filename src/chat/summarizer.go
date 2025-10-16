package chat

import (
	"context"
	"fmt"

	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

const (
	// Token threshold to trigger summarization
	summarizationThreshold = 3000

	// Keep the most recent N messages without summarization
	recentMessageWindow = 4
)

// Summarizer handles conversation summarization to reduce token usage
type Summarizer struct {
	llmClient models.LLMInferencer
}

func NewSummarizer(llmClient models.LLMInferencer) *Summarizer {
	return &Summarizer{
		llmClient: llmClient,
	}
}

// ShouldSummarize checks if the session should be summarized
func (s *Summarizer) ShouldSummarize(session *models.ChatSession) bool {
	return session.TotalTokens > summarizationThreshold && len(session.Messages) > recentMessageWindow
}

// SummarizeSession creates a summary of older messages and keeps recent ones
func (s *Summarizer) SummarizeSession(ctx context.Context, session *models.ChatSession) (*models.ChatSession, error) {
	if !s.ShouldSummarize(session) {
		return session, nil
	}

	// Split messages: older (to summarize) vs recent (to keep)
	splitIndex := len(session.Messages) - recentMessageWindow
	if splitIndex <= 0 {
		return session, nil
	}

	olderMessages := session.Messages[:splitIndex]
	recentMessages := session.Messages[splitIndex:]

	// Build conversation text from older messages
	conversationText := ""
	for _, msg := range olderMessages {
		conversationText += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	// Create summarization prompt
	summarizationPrompt := fmt.Sprintf(`Please provide a concise summary of the following conversation. Focus on the key topics, questions asked, and important information exchanged. Keep it under 200 words.

Conversation:
%s

Summary:`, conversationText)

	// Generate summary using LLM
	summaryReq := &models.InferenceRequest{
		Query:       summarizationPrompt,
		MaxTokens:   300,
		Temperature: 0.3, // Lower temperature for more focused summaries
	}

	summary, err := s.llmClient.Infer(ctx, summaryReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// Create a new session with summary + recent messages
	summarizedSession := &models.ChatSession{
		SessionID:       session.SessionID,
		Messages:        []models.ChatMessage{},
		CreatedAt:       session.CreatedAt,
		LastInteraction: session.LastInteraction,
		TotalTokens:     0, // Will be recalculated
		MessageCount:    session.MessageCount,
		ModelPreference: session.ModelPreference,
	}

	// Add summary as a system message
	summarizedSession.Messages = append(summarizedSession.Messages, models.ChatMessage{
		Role:      "system",
		Content:   fmt.Sprintf("[Conversation Summary]: %s", summary),
		Timestamp: session.CreatedAt,
	})

	// Add recent messages
	summarizedSession.Messages = append(summarizedSession.Messages, recentMessages...)

	// Recalculate token count
	totalTokens := 0
	for _, msg := range summarizedSession.Messages {
		totalTokens += len(msg.Content) / 4 // Rough token estimation
	}
	summarizedSession.TotalTokens = totalTokens

	return summarizedSession, nil
}

// BuildOptimizedContext builds context with automatic summarization if needed
func (s *Summarizer) BuildOptimizedContext(ctx context.Context, session *models.ChatSession) (string, *models.ChatSession, error) {
	// Check if summarization is needed
	if s.ShouldSummarize(session) {
		summarizedSession, err := s.SummarizeSession(ctx, session)
		if err != nil {
			// Fall back to regular context if summarization fails
			return s.buildRegularContext(session), session, nil
		}
		return s.buildRegularContext(summarizedSession), summarizedSession, nil
	}

	return s.buildRegularContext(session), session, nil
}

func (s *Summarizer) buildRegularContext(session *models.ChatSession) string {
	if len(session.Messages) == 0 {
		return ""
	}

	context := ""
	for _, msg := range session.Messages {
		if msg.Role == "system" {
			context += msg.Content + "\n\n"
		} else {
			context += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
		}
	}

	return context
}
