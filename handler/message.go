package handler

import (
    "log"
    "strings"
    "strconv" // NEW: Needed to parse activity type from string
    "time"
    "fmt" 

    "discord-ai-bot/ai"
    "discord-ai-bot/db"

    "github.com/bwmarrin/discordgo"
)

const commandPrefix = "!setpersonality"
const statusCommandPrefix = "!setstatus" // NEW: Status command

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

        // 1. Personality Command
        if strings.HasPrefix(m.Content, commandPrefix) {
            newPersonality := strings.TrimSpace(strings.TrimPrefix(m.Content, commandPrefix))
            
            if newPersonality == "" {
                s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Please provide a new personality description after the command, e.g., `%s You are a sarcastic pirate.`", commandPrefix), createReply(m))
                return
            }

            db.SavePersonality(newPersonality)
            s.ChannelMessageSendReply(m.ChannelID, "✅ **Personality Updated!** The bot is now defined as: ```"+newPersonality+"```", createReply(m))
            return
        }

        // 2. Status Command (NEW LOGIC)
        if strings.HasPrefix(m.Content, statusCommandPrefix) {
            
            // Expected format: !setstatus <status> <type> <text...>
            args := strings.Fields(strings.TrimPrefix(m.Content, statusCommandPrefix))
            
            if len(args) < 3 {
                // Not enough arguments
                s.ChannelMessageSendReply(m.ChannelID, 
                    fmt.Sprintf("❌ Invalid format. Usage: `%s <status> <type> <text>`. Status: `online`/`idle`/`dnd`. Type: `playing`/`watching`/`listening`/`streaming`.", statusCommandPrefix), 
                    createReply(m))
                return
            }

            status := args[0]
            activityTypeStr := strings.ToLower(args[1])
            activityText := strings.Join(args[2:], " ")
            
            // Map string type to discordgo constant
            var activityType discordgo.ActivityType
            switch activityTypeStr {
            case "playing":
                activityType = discordgo.ActivityTypeGame
            case "streaming":
                activityType = discordgo.ActivityTypeStreaming
            case "listening":
                activityType = discordgo.ActivityTypeListening
            case "watching":
                activityType = discordgo.ActivityTypeWatching
            default:
                s.ChannelMessageSendReply(m.ChannelID, "❌ Invalid activity type. Use `playing`, `streaming`, `listening`, or `watching`.", createReply(m))
                return
            }

            // Set the new status
            err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
                Status: status,
                Activities: []*discordgo.Activity{
                    {
                        Name: activityText,
                        Type: activityType,
                    },
                },
            })

            if err != nil {
                log.Printf("Error setting bot status: %v", err)
                s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("❌ Failed to set status: %v", err), createReply(m))
                return
            }

            s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("✅ **Status Updated!** Status: `%s`, Activity: `%s %s`", status, strings.Title(activityTypeStr), activityText), createReply(m))
            return
        }
        // --- END COMMAND HANDLING ---


        // --- PING/AI HANDLING (Remains the same as the last corrected version) ---
        
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
            // 1. Clean the message content
            cleanMessage := strings.TrimSpace(strings.Replace(m.Content, "<@"+mentionID+">", "", 1))
            
            if cleanMessage == "" {
                s.ChannelMessageSendReply(m.ChannelID, "Hello! Ping me with a question and I'll remember the context.", createReply(m))
                return
            }

            // 2. Start the Discord Typing Indicator
            s.ChannelTyping(m.ChannelID)
            startTime := time.Now()

            // 3. Load GLOBAL Personality and Conversation History
            personality := db.LoadPersonality()
            history := db.LoadGlobalHistory() 

            // 4. Construct the full history for the AI
            userMessage := ai.Message{Role: "user", Content: cleanMessage} 
            
            fullHistory := []ai.Message{
                {Role: "system", Content: personality},
            }
            fullHistory = append(fullHistory, history...) 
            fullHistory = append(fullHistory, userMessage)

            // 5. Call Cerebras API 
            aiResponseContent, err := ai.GetCerebrasResponse(fullHistory) 
            
            // Timing delay
            if elapsed := time.Since(startTime); elapsed < time.Second {
                time.Sleep(time.Second - elapsed)
            }

            if err != nil {
                log.Printf("Cerebras API Error: %v", err)
                s.ChannelMessageSendReply(m.ChannelID, "Sorry, I ran into an issue connecting to the AI. Check the logs.", createReply(m))
                return
            }

            // 6. Send the final AI response (ONLY ONCE)
            s.ChannelMessageSendReply(m.ChannelID, aiResponseContent, createReply(m))

            // 7. Update and Save conversation history
            assistantMessage := ai.Message{Role: "assistant", Content: aiResponseContent}
            
            finalHistory := append(history, userMessage, assistantMessage) 

            db.SaveGlobalHistory(finalHistory)
        }
    }
}
