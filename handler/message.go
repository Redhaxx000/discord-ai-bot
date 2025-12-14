package handler

import (
    "log"
    "strings"
    "time"
    "fmt" // New import for command responses

    "discord-ai-bot/ai"
    "discord-ai-bot/db"

    "github.com/bwmarrin/discordgo"
)

const commandPrefix = "!setpersonality"

// MessageCreate returns a function that handles Discord messages.
func MessageCreate(s *discordgo.Session) func(*discordgo.Session, *discordgo.MessageCreate) {
    // The inner function is the actual handler
    return func(s *discordgo.Session, m *discordgo.MessageCreate) {
        // Ignore messages from the bot itself
        if m.Author.ID == s.State.User.ID {
            return
        }

        // --- COMMAND HANDLING ---
        if strings.HasPrefix(m.Content, commandPrefix) {
            newPersonality := strings.TrimSpace(strings.TrimPrefix(m.Content, commandPrefix))
            
            if newPersonality == "" {
                s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Please provide a new personality description after the command, e.g., `%s You are a sarcastic pirate.`", commandPrefix))
                return
            }

            db.SavePersonality(newPersonality)
            s.ChannelMessageSend(m.ChannelID, "✅ **Personality Updated!** The bot is now defined as: ```"+newPersonality+"```")
            return
        }
        // --- END COMMAND HANDLING ---


        // --- PING/AI HANDLING ---

        // Check if the bot was explicitly pinged (mentioned)
        isPinged := false
        mentionID := s.State.User.ID
        for _, mention := range m.Mentions {
            if mention.ID == mentionID {
                isPinged = true
                break
            }
        }

        if isPinged {
            // 1. Clean the message content (remove the bot's mention)
            // ... (Message cleaning logic remains the same) ...
            cleanMessage := strings.TrimSpace(strings.Replace(m.Content, "<@"+mentionID+">", "", 1))
            
            if cleanMessage == "" {
                s.ChannelMessageSend(m.ChannelID, "Hello! Ping me with a question and I'll remember the context.")
                return
            }

            // 2. Start the Discord Typing Indicator
            s.ChannelTyping(m.ChannelID)
            startTime := time.Now()

            // 3. Load GLOBAL Personality and Conversation History
            personality := db.LoadPersonality() // <-- NEW: Load the personality
            history := db.LoadGlobalHistory() 

            // 4. Construct the full history for the AI, starting with the system prompt
            fullHistory := []ai.Message{
                {Role: "system", Content: personality}, // <-- NEW: Prepend the system prompt
            }
            fullHistory = append(fullHistory, history...) // Append previous conversation history
            fullHistory = append(fullHistory, ai.Message{Role: "user", Content: cleanMessage}) // Append new user message

            // 5. Call Cerebras API (using fullHistory)
            aiResponseContent, err := ai.GetCerebrasResponse(fullHistory) // <-- Updated argument
            
            // ... (Timing and error handling remains the same) ...
            if elapsed := time.Since(startTime); elapsed < time.Second {
                time.Sleep(time.Second - elapsed)
            }

            if err != nil {
                log.Printf("Cerebras API Error: %v", err)
                s.ChannelMessageSend(m.ChannelID, "Sorry, I ran into an issue connecting to the AI. Check the logs.")
                return
            }

            // 6. Send the final AI response
            s.ChannelMessageSend(m.ChannelID, aiResponseContent)

            // 7. Update and Save conversation history (only the user/assistant messages)
            assistantMessage := ai.Message{Role: "assistant", Content: aiResponseContent}
            
            // NOTE: We save only the user and assistant messages for history, NOT the system message.
            finalHistory := append(history, userMessage, assistantMessage)

            db.SaveGlobalHistory(finalHistory)
        }
    }
}
