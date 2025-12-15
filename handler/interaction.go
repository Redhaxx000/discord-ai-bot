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
// Custom IDs for the buttons
const buttonIDGeneral = "button_general_config"
const buttonIDAssets = "button_assets_config"

// --- END IDs ---

// Generic pointer helper function for strings
func ptr(s string) *string {
    return &s
}

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

// Define the two remaining button components (Apply removed)
var configButtons = []discordgo.MessageComponent{
	// Row 1: The two action buttons
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Edit General Status",
				Style:    discordgo.PrimaryButton,
				CustomID: buttonIDGeneral,
			},
			discordgo.Button{
				Label:    "Edit Assets/URL",
				Style:    discordgo.SecondaryButton,
				CustomID: buttonIDAssets,
			},
		},
	},
}


// 1. Handle Slash Commands -> Opens Menus/Modals
func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := i.ApplicationCommandData().Name

	switch name {
	case "config":
		// 1. IMMEDIATELY DEFER to acknowledge the command within 3 seconds.
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})

		if err != nil {
			log.Printf("CRITICAL: Error deferring /config command: %v", err)
			return
		}

		// 2. Send the actual config menu as a FOLLOWUP message, using the Buttons.
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content:    "**RPC Configuration Menu**\n\nSubmit a modal to automatically update your RPC status.",
			Components: configButtons, // Use the defined buttons
			Flags:      discordgo.MessageFlagsEphemeral,
		})

		if err != nil {
			log.Printf("ERROR: Failed to send /config followup message (menu): %v", err)
		}

	case "personality":
		// Opens the Personality Modal
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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

		if err != nil {
			log.Printf("Error responding to /personality command: %v", err)
			return
		}
	}
}

// 2. Handle Component Interactions (Buttons only)
func handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	
	selectedValue := data.CustomID 

	// Load status data for pre-filling modals
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

	switch selectedValue {
	case buttonIDGeneral:
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
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &modal})
		if err != nil {
			log.Printf("Error responding to component modal (General): %v", err)
		}

	case buttonIDAssets:
		// Open Modal for Images and Streaming Link
		largeKey := ""
		largeText := ""
		smallKey := ""
		smallText := ""
		streamingURL := ""

		// Check for existing assets
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
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &modal})
		if err != nil {
			log.Printf("Error responding to component modal (Assets): %v", err)
		}
	}
}

// 3. Handle Modal Submissions -> Saves Data and Automatically Applies Status
func handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	// 1. IMMEDIATELY DEFER the response to prevent the "Unknown Interaction" error.
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Printf("CRITICAL: Error deferring modal response: %v", err)
		return
	}

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

	// Base response data for updating the original interaction message
	responseUpdate := &discordgo.WebhookEdit{
		// FIX: Use ptr helper for Content
		Content:    ptr("Configuration Saved, but update result is pending."),
		// FIX: Use address operator (&) for Components
		Components: &configButtons,
	}
	
	// A placeholder for the final message content
	var finalMessageContent string

	switch data.CustomID {
	case modalIDGeneral:
		// Save General Status Data (Components 0-3)
		currentStatus.Status = strings.ToLower(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
		activity.Type = stringToActivityType(strings.ToLower(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))
		activity.Name = data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		activity.Details = data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

		db.SaveStatus(*currentStatus)
		
		// --- AUTOMATIC STATUS UPDATE ---
		if err := s.UpdateStatusComplex(*currentStatus); err != nil {
			log.Printf("Error updating status (General): %v", err)
			finalMessageContent = "General settings saved, but **Status Update FAILED!** Check bot logs for details."
		} else {
			finalMessageContent = "General settings saved and **Status Updated Successfully!**"
		}
		// -------------------------------

	case modalIDAssets:
		// Save Assets and URL (Components 0-4)
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
		
		// --- AUTOMATIC STATUS UPDATE ---
		if err := s.UpdateStatusComplex(*currentStatus); err != nil {
			log.Printf("Error updating status (Assets): %v", err)
			finalMessageContent = "Images/URL saved, but **Status Update FAILED!** Check bot logs for details. (Ensure Assets are valid keys)"
		} else {
			finalMessageContent = "Images/URL saved and **Status Updated Successfully!**"
		}
		// -------------------------------
		
	case "personality_modal":
		// Personality modal submission
		newPersonality := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		db.SavePersonality(newPersonality)
		
		// Use FollowupMessageCreate since this is a separate command interaction (/personality)
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Personality Updated!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return // Exit early as the personality modal is handled differently

	default:
		// Should not happen, but update the message anyway
		finalMessageContent = "Unknown submission type."
	}
	
	// 2. Edit the original message (from the deferral) with the final status and buttons.
	// FIX: Use ptr helper for the final message content
	responseUpdate.Content = ptr(finalMessageContent)
	_, err = s.InteractionResponseEdit(i.Interaction, responseUpdate)
	if err != nil {
		log.Printf("Error editing deferred interaction response: %v", err)
	}
}

// --- HELPER FUNCTIONS ---

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
