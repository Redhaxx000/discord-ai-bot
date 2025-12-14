package db

import (
    // ... existing imports ...
    "sync"
    "discord-ai-bot/ai" // Import the Message struct
    bolt "github.com/boltdb/bolt"
)

// ... existing consts ...
const personalityKey = "bot_personality" // <-- New constant for the personality key

var db *bolt.DB
var once sync.Once

// ... InitDB, LoadGlobalHistory, SaveGlobalHistory functions remain the same ...

// LoadPersonality loads the bot's system prompt (personality).
func LoadPersonality() string {
    var personality string
    err := db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte(conversationBucket))
        if b == nil {
            return nil
        }
        data := b.Get([]byte(personalityKey))
        if data != nil {
            personality = string(data)
        }
        return nil
    })

    if err != nil {
        log.Printf("Warning: Error loading personality: %v", err)
    }
    // Set a default personality if none is found
    if personality == "" {
        personality = "You are a helpful AI assistant."
    }
    return personality
}

// SavePersonality saves the new system prompt (personality).
func SavePersonality(personality string) {
    err := db.Update(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte(conversationBucket))
        if b == nil {
            return os.ErrNotExist
        }
        return b.Put([]byte(personalityKey), []byte(personality))
    })
    
    if err != nil {
        log.Printf("Error saving personality: %v", err)
    }
}
