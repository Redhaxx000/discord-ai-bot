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
    if err := godotenv.Load(); err != nil {
        log.Println("Note: No .env file found.")
    }

    token := os.Getenv("DISCORD_BOT_TOKEN")
    port := os.Getenv("PORT") 
    dbPath := os.Getenv("DB_PATH")

    if token == "" { log.Fatal("FATAL: DISCORD_BOT_TOKEN not set.") }
    if dbPath == "" { dbPath = "bot_memory.db" }

    db.InitDB(dbPath) 

    dg, err := discordgo.New("Bot " + token)
    if err != nil { log.Fatalf("FATAL: Error creating Discord session: %v", err) }

    // 1. Load and Set Saved Status
    initialStatus := db.LoadStatus()
    if initialStatus != nil {
        dg.UpdateStatusComplex(*initialStatus)
    }

    // 2. Register Handlers
    dg.AddHandler(handler.MessageCreate(dg)) // AI Chat Handler
    dg.AddHandler(handler.InteractionCreate) // NEW: UI/Slash Command Handler

    dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

    if err = dg.Open(); err != nil {
        log.Fatalf("FATAL: Error opening connection: %v", err)
    }
    
    // 3. REGISTER SLASH COMMANDS
    // We register them globally (might take up to an hour to appear, 
    // strictly for development you can pass a Guild ID as the second arg instead of "")
    log.Println("Registering slash commands...")
    commands := []*discordgo.ApplicationCommand{
        {
            Name:        "config",
            Description: "Open the UI to edit Bot Status, Activity, and RPC Assets",
        },
        {
            Name:        "personality",
            Description: "Open the UI to edit the AI Personality",
        },
    }

    for _, v := range commands {
        _, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", v)
        if err != nil {
            log.Panicf("Cannot create '%v' command: %v", v.Name, err)
        }
    }

    log.Println("✅ Bot is running with Slash Commands active.")
    
    if port == "" { port = "8080" }
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Discord Bot is running.")
    })
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
