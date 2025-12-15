package handler

import (
    "log"
    "strings"
    "time"

    "discord-ai-bot/ai"
    "discord-ai-bot/db"

    "github.com/bwmarrin/discordgo"
)

// Utility function to create a message reference for replies
func createReply(m *discordgo.MessageCreate) *discordgo.MessageReference {
    return &discordgo.MessageReference{
        MessageID: m.ID,
        ChannelID: m.ChannelID,
        GuildID: m.GuildID,
    }
}

func MessageCreate(s *discordgo.Session) func(*discordgo.Session, *discordgo.MessageCreate) {
    return func(s *discordgo.Session, m *discordgo.MessageCreate) {
        if m.Author.ID == s.State.User.ID { return }

        // --- PING/AI HANDLING ONLY ---
        isPinged := false
        mentionID := s.State.User.ID
        for _, mention := range m.Mentions {
            if mention.ID == mentionID {
                isPinged = true
                break
            }
        }

        if isPinged {
            cleanMessage := strings.TrimSpace(strings.Replace(m.Content, "<@"+mentionID+">", "", 1))
            
            if cleanMessage == "" {
                s.ChannelMessageSendReply(m.ChannelID, "Hello! Ping me with a question.", createReply(m))
                return
            }

            s.ChannelTyping(m.ChannelID)
            startTime := time.Now()

            personality := db.LoadPersonality()
            history := db.LoadGlobalHistory() 

            userMessage := ai.Message{Role: "user", Content: cleanMessage} 
            
            fullHistory := []ai.Message{{Role: "system", Content: personality}}
            fullHistory = append(fullHistory, history...) 
            fullHistory = append(fullHistory, userMessage)

            aiResponseContent, err := ai.GetCerebrasResponse(fullHistory) 
            
            if elapsed := time.Since(startTime); elapsed < time.Second {
                time.Sleep(time.Second - elapsed)
            }

            if err != nil {
                log.Printf("Cerebras API Error: %v", err)
                s.ChannelMessageSendReply(m.ChannelID, "AI Error. Check logs.", createReply(m))
                return
            }

            s.ChannelMessageSendReply(m.ChannelID, aiResponseContent, createReply(m))

            assistantMessage := ai.Message{Role: "assistant", Content: aiResponseContent}
            finalHistory := append(history, userMessage, assistantMessage) 
            db.SaveGlobalHistory(finalHistory)
        }
    }
}
