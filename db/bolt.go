package db

import (
    "encoding/json"
    "log" // <-- ADDED: Needed for log.Fatal/log.Printf
    "os"  // <-- ADDED: Needed for os.ErrNotExist

    "sync"

    "discord-ai-bot/ai"

    bolt "github.com/boltdb/bolt"
)

var db *bolt.DB
var once sync.Once

// --- Global Constants ---
const conversationBucket = "conversations" // <-- DEFINED
const globalKey = "global_conversation"      // <-- DEFINED
const personalityKey = "bot_personality"     // <-- DEFINED
// ------------------------

// InitDB initializes the BoltDB connection (safe to call multiple times)
func InitDB(dbPath string) {
    once.Do(func() {
        var err error
        db, err = bolt.Open(dbPath, 0600, nil)
        if err != nil {
            log.Fatalf("Error opening BoltDB: %v", err)
        }
        
        // Ensure the bucket exists
        err = db.Update(func(tx *bolt.Tx) error {
            _, err := tx.CreateBucketIfNotExists([]byte(conversationBucket))
            return err
        })
        if err != nil {
            log.Fatalf("Error creating BoltDB bucket: %v", err)
        }
    })
}

// LoadGlobalHistory loads the bot's entire conversation history.
func LoadGlobalHistory() []ai.Message {
    var history []ai.Message
    err := db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte(conversationBucket))
        if b == nil {
            return nil
        }
        data := b.Get([]byte(globalKey))
        if data == nil {
            return nil
        }
        return json.Unmarshal(data, &history)
    })

    if err != nil {
        log.Printf("Warning: Error loading history (returning empty): %v", err)
    }
    return history
}

// SaveGlobalHistory saves the updated history.
func SaveGlobalHistory(history []ai.Message) {
    data, err := json.Marshal(history)
    if err != nil {
        log.Printf("Error marshalling history: %v", err)
        return
    }

    err = db.Update(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte(conversationBucket))
        if b == nil {
            return os.ErrNotExist
        }
        return b.Put([]byte(globalKey), data)
    })
    
    if err != nil {
        log.Printf("Error saving history: %v", err)
    }
}

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
