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
const selectMenuID = "select_config_option" // ID for the Select Menu

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

		// 2. Send the actual config menu as a FOLLOWUP message, using a Select Menu.
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "**RPC Configuration Menu**\n\nSelect the configuration section you wish to edit or apply changes:",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    selectMenuID,
							Placeholder: "Choose a configuration action...",
							Options: []discordgo.SelectMenuOption{
								{
									Label: "Edit General Status & Activity",
									Value: "select_general_status",
									Description: "Set Discord Status, Activity Type, Name, and Details.",
								},
								{
									Label: "Edit Images and Streaming Link",
									Value: "select_assets",
									Description: "Set Large/Small Assets, Tooltip Text, and Streaming URL.",
								},
								{
									Label: "Apply All Changes (Update Status)",
									Value: "select_apply_status",
									Description: "Save and immediately push all saved settings to Discord.",
								},
							},
						},
					},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
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

// 2. Handle Component Interactions (Select Menu and future components)
func handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	
	// Determine the selected value, whether it's from the Select Menu or a direct button click (if any added later)
	var selectedValue string
	if data.CustomID == selectMenuID {
		if len(data.Values) > 0 {
			selectedValue = data.Values[0]
		}
	} else {
		// Fallback for non-select menu components
		selectedValue = data.CustomID
	}

	// Load status data for pre-filling modals or applying status
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
	case "select_general_status":
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

	case "select_assets":
		// Open Modal for Images and Streaming Link (RPC Buttons REMOVED for v0.27.1 compatibility)
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

	case "select_apply_status":
		// This option triggers the final status update
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
		// Save General Status Data (Components 0-3)
		currentStatus.Status = strings.ToLower(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
		activity.Type = stringToActivityType(strings.ToLower(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))
		activity.Name = data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		activity.Details = data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

		db.SaveStatus(*currentStatus)
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Content: "General Activity Saved! Use the Select Menu to 'Apply All Changes'."}})
		if err != nil {
			log.Printf("Error responding to modal submission (General): %v", err)
		}

	case modalIDAssets:
		// Save Assets and URL (Components 0-4) (RPC Buttons REMOVED)
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
		// NOTE: activity.Buttons is left alone/ignored, as v0.27.1 does not support it.

		db.SaveStatus(*currentStatus)
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Content: "Images/URL Saved! Use the Select Menu to 'Apply All Changes'."}})
		if err != nil {
			log.Printf("Error responding to modal submission (Assets): %v", err)
		}

	case "personality_modal":
		// Personality modal submission
		newPersonality := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		db.SavePersonality(newPersonality)
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Personality Updated!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error responding to modal submission (Personality): %v", err)
		}
	}
}

// --- HELPER FUNCTIONS ---

// UpdateStatus performs the final update call to Discord
func updateStatus(s *discordgo.Session, i *discordgo.InteractionCreate) {
	newData := db.LoadStatus()
	if newData == nil {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: "Error: No saved status data found."})
		return
	}

	if err := s.UpdateStatusComplex(*newData); err != nil {
		log.Printf("Error updating status: %v", err)
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: "Update Failed! Check bot logs for details. (Ensure Activity Name is set and Assets are valid keys)"})
		return
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: "Full RPC Updated! Changes applied to Discord."})
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
