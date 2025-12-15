package handler

import (
    "log"
    "strings"
    "time" // Added for Timestamp logic (will need to be parsed from the modal)

    "discord-ai-bot/db"

    "github.com/bwmarrin/discordgo"
)

// InteractionCreate handles all slash commands and modal submissions
// ... (InteractionCreate and handleCommand remain the same) ...

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
                    // Row 2: Custom Status Message (New)
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
                     // Row 5: RPC Details and State (Combined into one row)
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
                    // Note: RPC State, URL, and Assets will be assumed to be set via subsequent modals 
                    // or for simplicity, we will merge the most common asset field into the main modal
                    // as done previously. Due to Discord's 5-row limit on Modals, we must be selective.
                    // We will keep the LargeImage Asset key as it's the most impactful.
                },
            },
        })
        if err != nil {
            log.Printf("Error sending modal: %v", err)
        }
    
    // ... (rest of handleCommand for personality) ...
    case "personality":
        // ... (personality modal logic remains the same) ...
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
        // Extract Data (NOTE: Indexing is based on the Modal components above)
        statusStr := strings.ToLower(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
        customStatusText := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        typeStr := strings.ToLower(data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
        activityText := data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        detailsText := data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        // Note: Asset key is removed from the modal to make room for new fields.
        // We will keep loading the LAST saved assets.

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
        
        // Load existing status to preserve other non-editable fields (like Assets)
        currentStatus := db.LoadStatus()
        var currentActivity *discordgo.Activity
        if currentStatus != nil && len(currentStatus.Activities) > 0 {
            currentActivity = currentStatus.Activities[0]
        }
        
        // Define the Activity
        activity := discordgo.Activity{
            Name:    activityText,
            Type:    activityType,
            Details: detailsText, // New
            // State: is left out for simplicity but can be added back if needed
        }
        
        // Preserve existing Assets and URL if the activity name/type hasn't changed drastically
        if currentActivity != nil {
            // Only update Assets/URL if they were previously set.
            activity.Assets = currentActivity.Assets 
            activity.URL = currentActivity.URL
        }
        
        // Define the final status update
        newData := discordgo.UpdateStatusData{
            Status: statusStr,
            // If the user set a custom status, we send *two* activities.
            // Discord displays the first non-nil activity. For custom status,
            // it must be ActivityTypeCustom.
            Activities: []*discordgo.Activity{}, 
        }

        // Add Custom Status if provided
        if customStatusText != "" {
            newData.Activities = append(newData.Activities, &discordgo.Activity{
                Name: customStatusText,
                Type: discordgo.ActivityTypeCustom,
            })
        }

        // Add the main Rich Presence Activity
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
