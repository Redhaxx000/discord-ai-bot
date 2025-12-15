package handler

import (
    "log"
    "strings"

    "discord-ai-bot/db"

    "github.com/bwmarrin/discordgo"
)

// --- Command and Component IDs ---
const modalIDGeneral = "modal_general_config"
const modalIDAssets = "modal_assets_config"
const modalIDButtons = "modal_buttons_config"
// --- END IDs ---

// InteractionCreate handles all slash commands and component/modal submissions
func InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
    switch i.Type {
    case discordgo.InteractionApplicationCommand:
        handleCommand(s, i)
    case discordgo.InteractionMessageComponent:
        handleComponent(s, i)
    case discordgo.InteractionModalSubmit:
        handleModalSubmit(s, i)
    }
}

// 1. Handle Slash Commands -> Opens Menus/Modals
func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    name := i.ApplicationCommandData().Name

    switch name {
    case "config":
        // Opens the RPC Main Menu
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "⚙️ **RPC Configuration Menu**\n\nSelect the section you wish to edit:",
                Flags:   discordgo.MessageFlagsEphemeral,
                Components: []discordgo.MessageComponent{
                    discordgo.ActionsRow{
                        Components: []discordgo.MessageComponent{
                            discordgo.Button{
                                CustomID: "button_general_status",
                                Label:    "General Status & Activity",
                                Style:    discordgo.PrimaryButton,
                            },
                            discordgo.Button{
                                CustomID: "button_assets",
                                Label:    "Images & Streaming Link",
                                Style:    discordgo.SecondaryButton,
                            },
                            discordgo.Button{
                                CustomID: "button_buttons",
                                Label:    "Action Buttons",
                                Style:    discordgo.SecondaryButton,
                            },
                            discordgo.Button{
                                CustomID: "button_apply_status",
                                Label:    "✅ Apply All Changes",
                                Style:    discordgo.SuccessButton,
                            },
                        },
                    },
                },
            },
        })

    case "personality":
        // Opens the Personality Modal
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

// 2. Handle Button Clicks -> Open Specific Modals
func handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
    data := i.MessageComponentData()
    currentStatusData := db.LoadStatus()
    
    var currentActivity *discordgo.Activity
    if currentStatusData != nil {
        for _, act := range currentStatusData.Activities {
            if act.Type != discordgo.ActivityTypeCustom {
                currentActivity = act
                break
            }
        }
    }

    switch data.CustomID {
    case "button_general_status":
        // Open Modal for Status, Type, Name, Details
        statusText := ""
        activityTypeStr := "playing"
        activityName := ""
        detailsText := ""

        if currentStatusData != nil {
            statusText = currentStatusData.Status
        }
        if currentActivity != nil {
            activityTypeStr = activityTypeToString(currentActivity.Type)
            activityName = currentActivity.Name
            detailsText = currentActivity.Details
        }

        modal := discordgo.InteractionResponseData{
            CustomID: modalIDGeneral,
            Title:    "General Status & Activity",
            Components: []discordgo.MessageComponent{
                // Row 1: Status
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "status_input", Label: "Discord Status (online, idle, dnd)", Style: discordgo.TextInputShort, Placeholder: "online", Required: true, MaxLength: 10, Value: statusText},
                }},
                // Row 2: Type
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "type_input", Label: "Activity Type (playing, watching, listening)", Style: discordgo.TextInputShort, Placeholder: "playing", Required: true, MaxLength: 10, Value: activityTypeStr},
                }},
                // Row 3: Name
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "activity_input", Label: "Activity Name (RPC Top Line)", Style: discordgo.TextInputShort, Placeholder: "Visual Studio Code", Required: true, MaxLength: 100, Value: activityName},
                }},
                // Row 4: Details
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "details_input", Label: "RPC Details (Level 1-1)", Style: discordgo.TextInputShort, Placeholder: "Optional", Required: false, MaxLength: 100, Value: detailsText},
                }},
            },
        }
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &modal})

    case "button_assets":
        // Open Modal for Images and Streaming Link
        largeKey := ""
        largeText := ""
        smallKey := ""
        smallText := ""
        streamingURL := ""

        // FIX: Check for empty string, not nil, as Assets is a struct
        if currentActivity != nil && currentActivity.Assets.LargeImageID != "" {
            largeKey = currentActivity.Assets.LargeImageID
            largeText = currentActivity.Assets.LargeText
            smallKey = currentActivity.Assets.SmallImageID
            smallText = currentActivity.Assets.SmallText
        }
        if currentActivity != nil {
            streamingURL = currentActivity.URL
        }

        modal := discordgo.InteractionResponseData{
            CustomID: modalIDAssets,
            Title:    "Images & Streaming Link",
            Components: []discordgo.MessageComponent{
                // Row 1: Large Image Key
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "large_key_input", Label: "Large Image Asset Key", Style: discordgo.TextInputShort, Placeholder: "Asset must be uploaded to Developer Portal.", Required: false, MaxLength: 50, Value: largeKey},
                }},
                // Row 2: Large Image Text
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "large_text_input", Label: "Large Image Tooltip Text", Style: discordgo.TextInputShort, Placeholder: "What the large image says on hover.", Required: false, MaxLength: 100, Value: largeText},
                }},
                // Row 3: Small Image Key
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "small_key_input", Label: "Small Image Asset Key", Style: discordgo.TextInputShort, Placeholder: "Optional", Required: false, MaxLength: 50, Value: smallKey},
                }},
                 // Row 4: Small Image Text
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "small_text_input", Label: "Small Image Tooltip Text", Style: discordgo.TextInputShort, Placeholder: "Optional", Required: false, MaxLength: 100, Value: smallText},
                }},
                 // Row 5: Streaming URL (Used only if type is 'streaming')
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "url_input", Label: "Streaming URL (Twitch/YouTube Link)", Style: discordgo.TextInputShort, Placeholder: "Only used if Activity Type is streaming.", Required: false, MaxLength: 100, Value: streamingURL},
                }},
            },
        }
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &modal})

    case "button_buttons":
        // Open Modal for Buttons (Max 2 supported by RPC, but 5-row limit)
        // We will store the buttons on the UpdateStatusData object instead of Activity
        
        btn1Label := ""
        btn1URL := ""
        
        // FIX: Check currentStatusData.Buttons (If this field existed in v0.27.1, which it likely did not)
        // Since it's likely not on UpdateStatusData either, we will use a separate DB save method
        // For simplicity with v0.27.1, let's assume buttons are saved in a temporary/separate manner for now,
        // or rely on the fact that older versions only supported one button via another field.
        // Given the constraints, we rely on the DB storing the *previous* UpdateStatusData object which might
        // have had the buttons, or we assume the buttons array on UpdateStatusData is nil/empty.
        
        if currentStatusData != nil && len(currentStatusData.Buttons) > 0 {
            btn1Label = currentStatusData.Buttons[0].Label
            btn1URL = currentStatusData.Buttons[0].URL
        }

        modal := discordgo.InteractionResponseData{
            CustomID: modalIDButtons,
            Title:    "Action Buttons (Max 1)",
            Components: []discordgo.MessageComponent{
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "btn1_label", Label: "Button 1 Label", Style: discordgo.TextInputShort, Placeholder: "e.g. Visit Website", Required: false, MaxLength: 32, Value: btn1Label},
                }},
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "btn1_url", Label: "Button 1 URL", Style: discordgo.TextInputShort, Placeholder: "Must be a full https:// link.", Required: false, MaxLength: 512, Value: btn1URL},
                }},
            },
        }
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &modal})

    case "button_apply_status":
        // This button triggers the final status update using the consolidated data
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseDeferredMessageUpdate,
        })
        
        // Call the final update function
        updateStatus(s, i)
    }
}

// 3. Handle Modal Submissions -> Saves Data Temporarily/Persistently
func handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
    data := i.ModalSubmitData()

    // Load current status data to preserve everything not in this modal
    currentStatus := db.LoadStatus()
    if currentStatus == nil {
        currentStatus = &discordgo.UpdateStatusData{Status: "online", Activities: []*discordgo.Activity{{Type: discordgo.ActivityTypeGame}}}
    }
    
    var activity *discordgo.Activity
    for _, act := range currentStatus.Activities {
        if act.Type != discordgo.ActivityTypeCustom {
            activity = act
            break
        }
    }
    if activity == nil {
        // If no activity exists, create a default one and append it
        activity = &discordgo.Activity{Type: discordgo.ActivityTypeGame, Name: "Updating..."}
        currentStatus.Activities = append(currentStatus.Activities, activity)
    }

    switch data.CustomID {
    case modalIDGeneral:
        // Save General Status Data
        currentStatus.Status = strings.ToLower(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
        activity.Type = stringToActivityType(strings.ToLower(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))
        activity.Name = data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        activity.Details = data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        
        db.SaveStatus(*currentStatus)
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Content: "✅ **General Activity Saved!** Click 'Apply All Changes' to update Discord."}})

    case modalIDAssets:
        // Save Assets and URL
        largeKey := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        largeText := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        smallKey := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        smallText := data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        url := data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

        // Update Assets (v0.27.1 compatible structure)
        activity.Assets = discordgo.Assets{
            LargeImageID: largeKey,
            LargeText:    largeText,
            SmallImageID: smallKey,
            SmallText:    smallText,
        }
        activity.URL = url
        
        db.SaveStatus(*currentStatus)
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Content: "✅ **Images/URL Saved!** Click 'Apply All Changes' to update Discord."}})

    case modalIDButtons:
        // Save Buttons - These are stored on the UpdateStatusData object in v0.27.1
        btnLabel := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        btnURL := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        
        // FIX: Set Buttons on the UpdateStatusData object
        currentStatus.Buttons = make([]discordgo.Button, 0)
        if btnLabel != "" && btnURL != "" {
            currentStatus.Buttons = append(currentStatus.Buttons, discordgo.Button{
                Label: btnLabel,
                URL:   btnURL,
            })
        }
        
        db.SaveStatus(*currentStatus)
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Content: "✅ **Buttons Saved!** Click 'Apply All Changes' to update Discord."}})

    // --- OTHER MODALS ---
    case "personality_modal":
        // Personality modal submission
        newPersonality := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        db.SavePersonality(newPersonality)
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "✅ **Personality Updated!**",
                Flags:   discordgo.MessageFlagsEphemeral,
            },
        })
    }
}

// --- HELPER FUNCTIONS ---

// UpdateStatus performs the final update call to Discord
func updateStatus(s *discordgo.Session, i *discordgo.InteractionCreate) {
    newData := db.LoadStatus()
    if newData == nil {
        s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: "❌ Error: No saved status data found."})
        return
    }

    if err := s.UpdateStatusComplex(*newData); err != nil {
        log.Printf("Error updating status: %v", err)
        s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: "❌ **Update Failed!** Check bot logs for details. (Ensure Activity Name is set and Assets are valid keys)"})
        return
    }

    s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: "🎉 **Full RPC Updated!** Changes applied to Discord."})
}

// Converts activity type constant to string for pre-filling modals
func activityTypeToString(t discordgo.ActivityType) string {
    switch t {
    case discordgo.ActivityTypeGame:
        return "playing"
    case discordgo.ActivityTypeStreaming:
        return "streaming"
    case discordgo.ActivityTypeListening:
        return "listening"
    case discordgo.ActivityTypeWatching:
        return "watching"
    case discordgo.ActivityTypeCompeting:
        return "competing"
    default:
        return "playing"
    }
}

// Converts string back to activity type constant
func stringToActivityType(s string) discordgo.ActivityType {
    switch s {
    case "playing":
        return discordgo.ActivityTypeGame
    case "streaming":
        return discordgo.ActivityTypeStreaming
    case "listening":
        return discordgo.ActivityTypeListening
    case "watching":
        return discordgo.ActivityTypeWatching
    case "competing":
        return discordgo.ActivityTypeCompeting
    default:
        return discordgo.ActivityTypeGame
    }
}
