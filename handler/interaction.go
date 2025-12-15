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
// buttonIDApply has been removed

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
			// The Apply button is removed as updates are now automatic
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
	
	// Case for buttonIDApply is removed.
	}
}

// 3. Handle Modal Submissions -> Saves Data and Automatically Applies Status
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

	// Base response data for updating the original interaction message
	responseUpdate := &discordgo.InteractionResponseData{
		Content:    "Configuration Saved, but update result is pending.",
		Components: configButtons, // RE-INCLUDES THE BUTTONS
	}

	switch data.CustomID {
	case modalIDGeneral:
		// Save General Status Data (Components 0-3)
		currentStatus.Status = strings.ToLower(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
		activity.Type = stringToActivityType(strings.ToLower(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))
		activity.Name = data.Components[2].(*discord
