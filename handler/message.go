package handler

import (
    "log"
    "strings"
    "time" // Needed for the typing indicator duration

    "discord-ai-bot/ai"
    "discord-ai-bot/db"

    "github.com/bwmarrin/discordgo"
)

// MessageCreate returns a function that handles Discord messages.
func MessageCreate(s *discordgo.Session) func(*discordgo.Session, *discordgo.MessageCreate) {
    // The inner function is the actual handler
    return func(s *discordgo.Session, m *discordgo.MessageCreate) {
        // Ignore messages from the bot itself
        if m.Author.ID == s.State.User.ID {
            return
        }

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
            cleanMessage := strings.TrimSpace(strings.Replace(m.Content, "<@"+mentionID+">", "", 1))
            
            if cleanMessage == "" {
                s.ChannelMessageSend(m.ChannelID, "Hello! Ping me with a question and I'll remember the context.")
                return
            }

            // 2. Start the Discord Typing Indicator
            // This tells Discord the bot is "typing" while it waits for the AI.
            s.ChannelTyping(m.ChannelID)
            
            // OPTIONAL: Keep the indicator running for a minimum time if the AI is super fast.
            startTime := time.Now()

            // 3. Load GLOBAL Conversation History
            history := db.LoadGlobalHistory() 

            // 4. Add the current user message to the history
            userMessage := ai.Message{Role: "user", Content: cleanMessage}
            newHistory := append(history, userMessage)

            // 5. Call Cerebras API
            aiResponseContent, err := ai.GetCerebrasResponse(newHistory)
            
            // Ensure typing indicator runs for at least 1 second if the response was instant
            if elapsed := time.Since(startTime); elapsed < time.Second {
                time.Sleep(time.Second - elapsed)
            }
            // Note: The typing indicator usually stops automatically when a message is sent.

            if err != nil {
                log.Printf("Cerebras API Error: %v", err)
                s.ChannelMessageSend(m.ChannelID, "Sorry, I ran into an issue connecting to the AI. Check the logs.")
                return
            }

            // 6. Send the final AI response to the channel (No "Thinking..." message needed)
            s.ChannelMessageSend(m.ChannelID, aiResponseContent)

            // 7. Add the AI response to the history and save (GLOBAL)
            assistantMessage := ai.Message{Role: "assistant", Content: aiResponseContent}
            finalHistory := append(newHistory, assistantMessage)

            db.SaveGlobalHistory(finalHistory)
        }
    }
}
