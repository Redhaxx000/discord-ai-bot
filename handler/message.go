package handler

import (
    "log"
    "strings"
    "time"
    "fmt" 

    "discord-ai-bot/ai"
    "discord-ai-bot/db"

    "github.com/bwmarrin/discordgo"
)

const commandPrefix = "!setpersonality"

// Utility function to create a message reference for replies
func createReply(m *discordgo.MessageCreate) *discordgo.MessageReference {
    return &discordgo.MessageReference{
        MessageID: m.ID,
        ChannelID: m.ChannelID,
        GuildID: m.GuildID,
    }
}

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
                // REPLY: Command usage error
                s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Please provide a new personality description after the command, e.g., `%s You are a sarcastic pirate.`", commandPrefix), createReply(m))
                return
            }

            db.SavePersonality(newPersonality)
            // REPLY: Command success message
            s.ChannelMessageSendReply(m.ChannelID, "✅ **Personality Updated!** The bot is now defined as: ```"+newPersonality+"```", createReply(m))
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
            cleanMessage := strings.TrimSpace(strings.Replace(m.Content, "<@"+mentionID+">", "", 1))
            
            if cleanMessage == "" {
                // REPLY: No message content
                s.ChannelMessageSendReply(m.ChannelID, "Hello! Ping me with a question and I'll remember the context.", createReply(m))
                return
            }

            // 2. Start the Discord Typing Indicator
            s.ChannelTyping(m.ChannelID)
            startTime := time.Now()

            // 3. Load GLOBAL Personality and Conversation History
            personality := db.LoadPersonality()
            history := db.LoadGlobalHistory() 

            // 4. Construct the full history for the AI, starting with the system prompt
            userMessage := ai.Message{Role: "user", Content: cleanMessage} 
            
            fullHistory := []ai.Message{
                {Role: "system", Content: personality},
            }
            fullHistory = append(fullHistory, history...) 
            fullHistory = append(fullHistory, userMessage)

            // 5. Call Cerebras API (using fullHistory)
            aiResponseContent, err := ai.GetCerebrasResponse(fullHistory) 
            
            // Timing delay
            if elapsed := time.Since(startTime); elapsed < time.Second {
                time.Sleep(time.Second - elapsed)
            }

            if err != nil {
                log.Printf("Cerebras API Error: %v", err)
                // REPLY: Error message (ONLY ONE SEND IN ERROR BLOCK)
                s.ChannelMessageSendReply(m.ChannelID, "Sorry, I ran into an issue connecting to the AI. Check the logs.", createReply(m))
                return
            }

            // 6. Send the final AI response (ONLY ONCE)
            // REPLY: The final AI answer (ONLY ONE SEND IN SUCCESS BLOCK)
            s.ChannelMessageSendReply(m.ChannelID, aiResponseContent, createReply(m))

            // 7. Update and Save conversation history (only the user/assistant messages)
            assistantMessage := ai.Message{Role: "assistant", Content: aiResponseContent}
            
            finalHistory := append(history, userMessage, assistantMessage) 

            db.SaveGlobalHistory(finalHistory)
        }
    }
}
