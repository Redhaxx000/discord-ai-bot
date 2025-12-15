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
                Title:    "Bot Configuration & RPC",
                Components: []discordgo.MessageComponent{
                    // Row 1: Status
                    discordgo.ActionsRow{
                        Components: []discordgo.MessageComponent{
                            discordgo.TextInput{
                                CustomID:    "status_input",
                                Label:       "Status (online, idle, dnd, invisible)",
                                Style:       discordgo.TextInputShort,
                                Placeholder: "online",
                                Required:    true,
                                MaxLength:   10,
                            },
                        },
                    },
                    // Row 2: Activity Type
                    discordgo.ActionsRow{
                        Components: []discordgo.MessageComponent{
                            discordgo.TextInput{
                                CustomID:    "type_input",
                                Label:       "Type (playing, watching, listening)",
                                Style:       discordgo.TextInputShort,
                                Placeholder: "playing",
                                Required:    true,
                                MaxLength:   10,
                            },
                        },
                    },
                    // Row 3: Activity Text
                    discordgo.ActionsRow{
                        Components: []discordgo.MessageComponent{
                            discordgo.TextInput{
                                CustomID:    "activity_input",
                                Label:       "Activity Text",
                                Style:       discordgo.TextInputShort,
                                Placeholder: "Visual Studio Code",
                                Required:    true,
                                MaxLength:   100,
                            },
                        },
                    },
                     // Row 4: Large Image Asset Key
                     discordgo.ActionsRow{
                        Components: []discordgo.MessageComponent{
                            discordgo.TextInput{
                                CustomID:    "asset_input",
                                Label:       "Large Image Asset Key (Optional)",
                                Style:       discordgo.TextInputShort,
                                Placeholder: "my_logo_key",
                                Required:    false,
                                MaxLength:   50,
                            },
                        },
                    },
                },
            },
        })
        if err != nil {
            log.Printf("Error sending modal: %v", err)
        }

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
                                Style:       discordgo.TextInputParagraph, // Big text box
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
        // Extract Data
        statusStr := strings.ToLower(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
        typeStr := strings.ToLower(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
        activityText := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
        assetKey := data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

        // Logic to Map Strings to Discord Types
        var activityType discordgo.ActivityType
        switch typeStr {
        case "playing": activityType = discordgo.ActivityTypeGame
        case "streaming": activityType = discordgo.ActivityTypeStreaming
        case "listening": activityType = discordgo.ActivityTypeListening
        case "watching": activityType = discordgo.ActivityTypeWatching
        case "competing": activityType = discordgo.ActivityTypeCompeting
        default: activityType = discordgo.ActivityTypeGame // Default
        }

        // Construct Assets if key provided
        var assets discordgo.Assets // FIXED: Correct Type Name
        if assetKey != "" {
            assets = discordgo.Assets{ // FIXED: Correct Type Name
                LargeImageID: assetKey, // FIXED: Field is LargeImageID, not LargeImage
                LargeText:    activityText,
            }
        }

        newData := discordgo.UpdateStatusData{
            Status: statusStr,
            Activities: []*discordgo.Activity{
                {
                    Name:   activityText,
                    Type:   activityType,
                    Assets: assets,
                },
            },
        }

        // Save and Update
        db.SaveStatus(newData)
        if err := s.UpdateStatusComplex(newData); err != nil {
            log.Printf("Error updating status: %v", err)
        }

        // Respond to user (Hidden message)
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "✅ **Configuration Updated!**",
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
