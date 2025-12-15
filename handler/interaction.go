package handler

import (
    "log"
    "strings"

    "discord-ai-bot/db"

    "github.com/bwmarrin/discordgo"
)

// InteractionCreate handles all slash commands and modal submissions
func InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
    switch i.Type {
    case discordgo.InteractionApplicationCommand:
        handleCommand(s, i)
    case discordgo.InteractionModalSubmit:
        handleModalSubmit(s, i)
    }
}

// 1. Handle the Slash Command -> Open the Modal (UI)
func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    name := i.ApplicationCommandData().Name

    switch name {
    case "config":
        // Create the Modal for Status/RPC
        err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseModal,
            Data: &discordgo.InteractionResponseData{
                CustomID: "config_modal",
                Title:    "Bot Status & Full RPC",
                Components: []discordgo.MessageComponent{
                    // Row 1: Status
                    discordgo.ActionsRow{
                        Components: []discordgo.MessageComponent{
                            discordgo.TextInput{
                                CustomID:    "status_input",
                                Label:       "Discord Status (online, idle, dnd, invisible)",
                                Style:       discordgo.TextInputShort,
                                Placeholder: "online",
                                Required:    true,
                                MaxLength:   10,
                            },
                        },
                    },
                    // Row 2: Custom Status Message
                    discordgo.ActionsRow{
                        Components: []discordgo.MessageComponent{
                            discordgo.TextInput{
                                CustomID:    "custom_status_input",
                                Label:       "Custom Status Text (e.g. 'Back in 5')",
                                Style:       discordgo.TextInputShort,
                                Placeholder: "Leave empty for no custom status.",
                                Required:    false,
                                MaxLength:   100,
                            },
                        },
                    },
                    // Row 3: Activity Type
                    discordgo.ActionsRow{
                        Components: []discordgo.MessageComponent{
                            discordgo.TextInput{
                                CustomID:    "type_input",
                                Label:       "Activity Type (playing, watching, listening)",
                                Style:       discordgo.TextInputShort,
                                Placeholder: "playing",
                                Required:    true,
                                MaxLength:   10,
                            },
                        },
                    },
                    // Row 4: Activity Text (Name of the game/activity)
                    discordgo.ActionsRow{
                        Components: []discordgo.MessageComponent{
                            discordgo.TextInput{
                                CustomID:    "activity_input",
                                Label:       "Activity Name (e.g. 'Visual Studio Code')",
                                Style:       discordgo.TextInputShort,
                                Placeholder: "Required for RPC",
                                Required:    true,
                                MaxLength:   100,
                            },
                        },
                    },
                     // Row 5: RPC Details
                     discordgo.ActionsRow{
                        Components: []discordgo.MessageComponent{
                            discordgo.TextInput{
                                CustomID:    "details_input",
                                Label:       "RPC Details (e.g. 'Level 1-1')",
                                Style:       discordgo.TextInputShort,
                                Placeholder: "Optional",
                                Required:    false,
                                MaxLength:   100,
                            },
                        },
                    },
                },
            },
        })
        if err != nil {
            log.Printf("Error sending modal: %v", err)
        }
    
    // ... (rest of handleCommand for personality) ...
    case "personality":
        // Create the Modal for Personality
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseModal,
            Data: &discordgo.InteractionResponseData{
                CustomID: "personality_modal",
                Title:    "Edit AI Personality",
                Components: []discordgo.MessageComponent{
                    discordgo.ActionsRow{
                        Components: []discordgo.MessageComponent{
                            discordgo.TextInput{
                                CustomID:    "persona_input",
                                Label:       "System Prompt",
                                Style:       discordgo.TextInputParagraph,
                                Placeholder: "You are a helpful assistant...",
                                Required:    true,
                                MaxLength:   2000,
                            },
                        },
                    },
                },
            },
        })
    }
}

// 2. Handle the Form Submission (Data Processing)
func handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
    data := i.ModalSubmitData()

    if data.CustomID == "config_modal" {
        // Extract Data (Indexing is based on the Modal components above)
        statusStr := strings.ToLower(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
        customStatusText := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        typeStr := strings.ToLower(data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
        activityText := data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        detailsText := data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        
        // Logic to Map Strings to Discord Types
        var activityType discordgo.ActivityType
        switch typeStr {
        case "playing": activityType = discordgo.ActivityTypeGame
        case "streaming": activityType = discordgo.ActivityTypeStreaming
        case "listening": activityType = discordgo.ActivityTypeListening
        case "watching": activityType = discordgo.ActivityTypeWatching
        case "competing": activityType = discordgo.ActivityTypeCompeting
        default: activityType = discordgo.ActivityTypeGame
        }
        
        // Load existing status to preserve Assets and URL
        currentStatus := db.LoadStatus()
        var currentActivity *discordgo.Activity
        if currentStatus != nil && len(currentStatus.Activities) > 0 {
            currentActivity = currentStatus.Activities[0]
        }
        
        // Define the Rich Presence Activity
        activity := discordgo.Activity{
            Name:    activityText,
            Type:    activityType,
            Details: detailsText,
        }
        
        // Preserve existing Assets and URL if previously set
        if currentActivity != nil {
            activity.Assets = currentActivity.Assets 
            activity.URL = currentActivity.URL
        }
        
        // Define the final status update
        newData := discordgo.UpdateStatusData{
            Status: statusStr,
            Activities: []*discordgo.Activity{}, 
        }

        // 1. Add Custom Status if provided (ActivityTypeCustom must be first if present)
        if customStatusText != "" {
            newData.Activities = append(newData.Activities, &discordgo.Activity{
                Name: customStatusText,
                Type: discordgo.ActivityTypeCustom,
            })
        }

        // 2. Add the main Rich Presence Activity
        newData.Activities = append(newData.Activities, &activity)


        // Save and Update
        db.SaveStatus(newData)
        if err := s.UpdateStatusComplex(newData); err != nil {
            log.Printf("Error updating status: %v", err)
        }

        // Respond to user (Hidden message)
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "✅ **Configuration Updated!** Check your profile to see the new status and RPC.",
                Flags:   discordgo.MessageFlagsEphemeral,
            },
        })
    } else if data.CustomID == "personality_modal" {
        newPersonality := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        
        db.SavePersonality(newPersonality)

        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "✅ **Personality Updated!**\nNow acts as: " + newPersonality,
                Flags:   discordgo.MessageFlagsEphemeral,
            },
        })
    }
}
