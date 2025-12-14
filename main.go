package main

import (
    "fmt"
    "log"
    "net/http"
    "os"

    "discord-ai-bot/db"
    "discord-ai-bot/handler"

    "github.com/bwmarrin/discordgo"
    "github.com/joho/godotenv"
)

func main() {
    // 1. Load Environment Variables from .env file (for local testing)
    if err := godotenv.Load(); err != nil {
        log.Println("Note: No .env file found, relying on system environment variables (good for Render).")
    }

    token := os.Getenv("DISCORD_BOT_TOKEN")
    port := os.Getenv("PORT") 
    dbPath := os.Getenv("DB_PATH")

    if token == "" {
        log.Fatal("FATAL: DISCORD_BOT_TOKEN environment variable not set.")
    }
    if dbPath == "" {
        dbPath = "bot_memory.db" // Default if not set in .env
    }

    // 2. Initialize BoltDB for global memory
    db.InitDB(dbPath) 

    // 3. Create Discord Session
    dg, err := discordgo.New("Bot " + token)
    if err != nil {
        log.Fatalf("FATAL: Error creating Discord session: %v", err)
    }

    // Required intents to read message content
    dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

    // 4. Register Message Handler
    dg.AddHandler(handler.MessageCreate(dg))

    // 5. Open Discord connection
    if err = dg.Open(); err != nil {
        log.Fatalf("FATAL: Error opening Discord connection: %v", err)
    }
    log.Println("✅ Bot is running and connected to Discord.")
    
    // 6. Start HTTP server to satisfy Render's requirement to bind to a port
    if port == "" {
        port = "8080"
    }

    // Simple health check endpoint
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Discord Bot is running and healthy.")
    })

    log.Printf("🌍 Starting HTTP server on port %s for Render health check...", port)
    // This blocks indefinitely, keeping the Go program and the bot alive.
    log.Fatal(http.ListenAndServe(":"+port, nil))

    // Note: The defer dg.Close() will never be reached in normal operation
    // because log.Fatal(http.ListenAndServe) will exit the process.
}
