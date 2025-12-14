package handler

import (
    "log"
    "strings"

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

            // 2. Load GLOBAL Conversation History
            history := db.LoadGlobalHistory() 

            // 3. Add the current user message to the history
            userMessage := ai.Message{Role: "user", Content: cleanMessage}
            newHistory := append(history, userMessage)

            // OPTIONAL: Limit history size to prevent running out of context and hitting rate limits
            // if len(newHistory) > 10 { // e.g., keep the last 10 messages
            //     newHistory = newHistory[len(newHistory)-10:]
            // }

            // 4. Call Cerebras API
            s.ChannelMessageSend(m.ChannelID, "*Thinking...*")
            aiResponseContent, err := ai.GetCerebrasResponse(newHistory)
            
            if err != nil {
                log.Printf("Cerebras API Error: %v", err)
                s.ChannelMessageSend(m.ChannelID, "Sorry, I ran into an issue connecting to the AI. Check the logs.")
                return
            }

            // 5. Send the AI response to the channel
            s.ChannelMessageEdit(m.ChannelID, m.ID, aiResponseContent) // Edit the "Thinking..." message
            s.ChannelMessageSend(m.ChannelID, aiResponseContent)

            // 6. Add the AI response to the history and save (GLOBAL)
            assistantMessage := ai.Message{Role: "assistant", Content: aiResponseContent}
            finalHistory := append(newHistory, assistantMessage)

            db.SaveGlobalHistory(finalHistory)
        }
    }
}
