package db

import (
    "encoding/json"
    "log"
    "os"
    "sync"

    "discord-ai-bot/ai" // Import the Message struct

    bolt "github.com/boltdb/bolt"
)

var db *bolt.DB
var once sync.Once

const conversationBucket = "conversations"
const globalKey = "global_conversation"

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
