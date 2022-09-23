package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	fe8savereader "github.com/haplesspanda/fe8savereader/format"
	"github.com/haplesspanda/haplessbot/constants"
	"github.com/haplesspanda/haplessbot/fe8"
	"github.com/haplesspanda/haplessbot/rest"
	"github.com/haplesspanda/haplessbot/types"
)

var maxContentLength = 2000

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var acceptedCommands = map[string]struct{}{"ping": {}, "avatar": {}, "banner": {}, "choose": {}, "order": {}, "fe8": {}}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Send specified commands to discord HTTP endpoint
func DefineCommands(commands []string) {
	for _, element := range commands {
		_, exists := acceptedCommands[element]
		if !exists {
			continue
		}

		commandFile := fmt.Sprintf("commands/def/%s.json", element)
		dat, err := os.ReadFile(commandFile)
		check(err)
		log.Printf("Read command: %s", dat)

		url := fmt.Sprintf("https://discord.com/api/v10/applications/%d/commands", constants.ApplicationId)
		log.Printf("Sending command to endpoint: %s", url)

		request, err := http.NewRequest("POST", url, bytes.NewBuffer(dat))
		check(err)

		body := rest.DoJsonRequest(request)

		var bodyJson any
		json.Unmarshal(body, &bodyJson)
		log.Printf("Command response: %s", bodyJson)
	}
}

func RunInteractionCallback(details types.InteractionCreateDetails) {
	var data, memberData, guildId, interactionId, interactionToken = details.Data, details.Member, details.GuildId, details.Id, details.Token

	log.Printf("Processing %s", interactionId)
	if data.Type != 1 {
		log.Printf("Unexpected command type %d, aborting", data.Type)
		return
	}

	userData := memberData.User

	url := fmt.Sprintf("https://discord.com/api/v10/interactions/%s/%s/callback", interactionId, interactionToken)
	log.Println(url)

	followupUrl := fmt.Sprintf("https://discord.com/api/v10/webhooks/%d/%s", constants.ApplicationId, interactionToken)

	var callbackJson types.InteractionCallbackMessage
	var attachment *rest.BinaryAttachment
	var followupJson *types.InteractionCallbackData // TODO: Improve type, this is not correct
	switch data.Name {
	case "ping":
		callbackJson = types.InteractionCallbackMessage{
			Type: 4,
			Data: types.InteractionCallbackData{
				Content: "Pong",
			},
		}
	case "avatar":
		// Pick the user (maybe specified from command)
		var avatarUser types.UserData
		var avatarGuildMember types.GuildMemberData
		var userOption *string
		var preferServer = true
		if data.Options != nil && len(data.Options) > 0 {
			for _, option := range data.Options {
				switch option.Name {
				case "user":
					stringValue := option.Value.(string)
					userOption = &stringValue
				case "show_server_profile":
					preferServer = option.Value.(bool)
				default:
					log.Printf("Aborting, unexpected avatar option: %s", data.Options[0].Name)
					return
				}
			}
		}

		if userOption != nil {
			avatarUser = data.Resolved.Users[*userOption]
			avatarGuildMember = data.Resolved.Members[*userOption]
		} else {
			avatarUser = userData
			avatarGuildMember = memberData
		}

		fullUser := fmt.Sprintf("%s#%s", avatarUser.Username, avatarUser.Discriminator)

		if preferServer && avatarGuildMember.Avatar != nil {
			var avatarExtension string
			if strings.HasPrefix(*avatarGuildMember.Avatar, "a_") {
				avatarExtension = "gif"
			} else {
				avatarExtension = "png"
			}
			avatarUrl := fmt.Sprintf("https://cdn.discordapp.com/guilds/%s/users/%s/avatars/%s.%s?size=4096", guildId, avatarUser.Id, *avatarGuildMember.Avatar, avatarExtension)
			callbackJson = types.InteractionCallbackMessage{
				Type: 4,
				Data: types.InteractionCallbackData{
					Embeds: []types.Embed{{
						Title: fmt.Sprintf("Avatar for %s", fullUser),
						Url:   avatarUrl,
						Image: types.EmbedImage{Url: avatarUrl},
					}},
				},
			}
		} else if avatarUser.Avatar != nil {
			var avatarExtension string
			if strings.HasPrefix(*avatarUser.Avatar, "a_") {
				avatarExtension = "gif"
			} else {
				avatarExtension = "png"
			}
			avatarUrl := fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.%s?size=4096", avatarUser.Id, *avatarUser.Avatar, avatarExtension)
			callbackJson = types.InteractionCallbackMessage{
				Type: 4,
				Data: types.InteractionCallbackData{
					Embeds: []types.Embed{{
						Title: fmt.Sprintf("Avatar for %s", fullUser),
						Url:   avatarUrl,
						Image: types.EmbedImage{Url: avatarUrl},
					}},
				},
			}
		} else {
			callbackJson = types.InteractionCallbackMessage{
				Type: 4,
				Data: types.InteractionCallbackData{
					Content: fmt.Sprintf("User %s has no avatar!", fullUser),
				},
			}
		}
	case "banner":
		var bannerUserId string
		if data.Options != nil && len(data.Options) > 0 {
			if data.Options[0].Name != "user" {
				log.Printf("Aborting, unexpected banner option: %s", data.Options[0].Name)
				return
			}
			bannerUserId = data.Options[0].Value.(string)
		} else {
			bannerUserId = userData.Id
		}

		// Execute get on user for banner URL
		getUserUrl := fmt.Sprintf("https://discord.com/api/v10/users/%s", bannerUserId)
		request, err := http.NewRequest("GET", getUserUrl, nil)
		check(err)

		body := rest.DoJsonRequest(request)

		var bannerUser types.UserData

		log.Println(body)
		json.Unmarshal(body, &bannerUser)
		log.Println(bannerUser)

		fullUser := fmt.Sprintf("%s#%s", bannerUser.Username, bannerUser.Discriminator)

		if bannerUser.Banner == nil {
			callbackJson = types.InteractionCallbackMessage{
				Type: 4,
				Data: types.InteractionCallbackData{
					Content: fmt.Sprintf("User %s has no banner!", fullUser),
				},
			}
		} else {
			var bannerExtension string
			if strings.HasPrefix(*bannerUser.Banner, "a_") {
				bannerExtension = "gif"
			} else {
				bannerExtension = "png"
			}
			bannerUrl := fmt.Sprintf("https://cdn.discordapp.com/banners/%s/%s.%s?size=4096", bannerUser.Id, *bannerUser.Banner, bannerExtension)
			callbackJson = types.InteractionCallbackMessage{
				Type: 4,
				Data: types.InteractionCallbackData{
					Embeds: []types.Embed{{
						Title: fmt.Sprintf("Banner for %s", fullUser),
						Url:   bannerUrl,
						Image: types.EmbedImage{Url: bannerUrl},
					}},
				},
			}
		}
	case "choose":
		if data.Options == nil || len(data.Options) < 2 {
			log.Printf("Aborting, not enough parameters: %v", data.Options)
			return
		}

		selectedOption := data.Options[rand.Intn(len(data.Options))]
		callbackJson = types.InteractionCallbackMessage{
			Type: 4,
			Data: types.InteractionCallbackData{
				Content: fmt.Sprintf("The answer is %s", selectedOption.Value),
			},
		}
	case "order":
		if data.Options == nil || len(data.Options) < 2 {
			log.Printf("Aborting, not enough parameters: %v", data.Options)
			return
		}

		numOptions := len(data.Options)

		options := make([]string, numOptions)
		for index, option := range data.Options {
			options[index] = option.Value.(string)
		}

		result := make([]string, 0)
		for len(options) > 0 {
			selectedIndex := rand.Intn(len(options))
			selectedOption := options[selectedIndex]
			result = append(result, selectedOption)
			options = removeIndex(options, selectedIndex)
		}

		resultString := ""
		for _, res := range result {
			resultString += fmt.Sprintf("\n%s", res)
		}

		callbackJson = types.InteractionCallbackMessage{
			Type: 4,
			Data: types.InteractionCallbackData{
				Content: fmt.Sprintf("The order is %s", resultString),
			},
		}
	case "fe8":
		if data.Options == nil || len(data.Options) != 1 {
			log.Printf("Aborting, wrong parameters: %v", data.Options)
			return
		}

		arg := data.Options[0]
		switch arg.Name {
		case "character":
			if arg.Options == nil || len(arg.Options) != 1 {
				log.Printf("Aborting, wrong parameters: %v", arg.Options)
				return
			}

			switch arg.Options[0].Name {
			case "info":
				subArg := arg.Options[0]
				if subArg.Options == nil || len(subArg.Options) != 1 {
					log.Printf("Aborting, wrong parameters: %v", subArg.Options)
					return
				}

				characterName := subArg.Options[0].Value.(string)
				data, err := fe8.GetCharacterData(characterName)

				if err == nil {
					thumbnailUrl := fmt.Sprintf("attachment://%s", data.ThumbnailImage.Name)
					attachment = &rest.BinaryAttachment{
						ContentType: "image/png",
						Name:        data.ThumbnailImage.Name,
						Filename:    data.ThumbnailImage.Filename,
					}
					callbackJson = types.InteractionCallbackMessage{
						Type: 4,
						Data: types.InteractionCallbackData{
							Embeds: []types.Embed{{
								Title:       data.Name,
								Description: data.Content,
								Thumbnail: types.EmbedThumbnail{
									Url: thumbnailUrl,
								},
							}},
						},
					}
				} else {
					callbackJson = types.InteractionCallbackMessage{
						Type: 4,
						Data: types.InteractionCallbackData{
							Content: *err,
						},
					}
				}
			case "averagestats":
				subArg := arg.Options[0]
				if subArg.Options == nil || len(subArg.Options) < 2 || len(subArg.Options) > 6 {
					log.Printf("Aborting, wrong parameters: %v", subArg.Options)
					return
				}

				var characterName string
				var level int
				var promotion *string
				var promotionLevel *int
				var secondPromotion *string
				var secondPromotionLevel *int
				for _, option := range subArg.Options {
					switch option.Name {
					case "character":
						characterName = option.Value.(string)
					case "level":
						level = int(option.Value.(float64))
					case "promotion":
						value := option.Value.(string)
						promotion = &value
					case "promotionlevel":
						value := int(option.Value.(float64))
						promotionLevel = &value
					case "secondpromotion":
						value := option.Value.(string)
						secondPromotion = &value
					case "secondpromotionlevel":
						value := int(option.Value.(float64))
						secondPromotionLevel = &value
					default:
						log.Printf("Unknown argument %s, aborting", option.Name)
						return
					}
				}
				data, err := fe8.GetAverageStats(characterName, level, promotion, promotionLevel, secondPromotion, secondPromotionLevel)

				if err == nil {
					thumbnailUrl := fmt.Sprintf("attachment://%s", data.ThumbnailImage.Name)
					attachment = &rest.BinaryAttachment{
						ContentType: "image/png",
						Name:        data.ThumbnailImage.Name,
						Filename:    data.ThumbnailImage.Filename,
					}
					callbackJson = types.InteractionCallbackMessage{
						Type: 4,
						Data: types.InteractionCallbackData{
							Embeds: []types.Embed{{
								Title:       data.Name,
								Description: data.Content,
								Thumbnail: types.EmbedThumbnail{
									Url: thumbnailUrl,
								},
							}},
						},
					}
				} else {
					callbackJson = types.InteractionCallbackMessage{
						Type: 4,
						Data: types.InteractionCallbackData{
							Content: *err,
						},
					}
				}
			default:
				log.Printf("Unknown subcommand %s, aborting", arg.Options[0].Name)
				return
			}
		case "savefile":
			if data.Options == nil || len(data.Options) != 1 {
				log.Printf("Aborting, wrong parameters: %v", data.Options)
				return
			}

			switch arg.Options[0].Name {
			case "read":
				subArg := arg.Options[0]
				if subArg.Options == nil || len(subArg.Options) != 1 {
					log.Printf("Aborting, wrong parameters: %v", subArg.Options)
					return
				}

				fileRef := subArg.Options[0].Value.(string)
				attachment := data.Resolved.Attachments[fileRef]

				response, err := http.Get(attachment.Url)
				check(err)
				defer response.Body.Close()

				data, err := io.ReadAll(response.Body)
				check(err)

				outputBuffer := new(bytes.Buffer)
				fe8savereader.Read(bytes.NewReader(data), outputBuffer)

				outputString := outputBuffer.String()
				if len(outputString) > maxContentLength {
					followupJson = &types.InteractionCallbackData{
						Content: outputString[maxContentLength:],
					}
					outputString = outputString[:maxContentLength]
				}
				// TODO: Better long message support

				callbackJson = types.InteractionCallbackMessage{
					Type: 4,
					Data: types.InteractionCallbackData{
						Content: outputString,
					},
				}

				log.Printf("%s", outputBuffer)

			case "compare":
				subArg := arg.Options[0]
				if subArg.Options == nil || len(subArg.Options) != 2 {
					log.Printf("Aborting, wrong parameters: %v", subArg.Options)
					return
				}

				oldFileRef := subArg.Options[0].Value.(string)
				oldAttachment := data.Resolved.Attachments[oldFileRef]

				oldResponse, err := http.Get(oldAttachment.Url)
				check(err)
				defer oldResponse.Body.Close()
				oldData, err := io.ReadAll(oldResponse.Body)
				check(err)

				newFileRef := subArg.Options[1].Value.(string)
				newAttachment := data.Resolved.Attachments[newFileRef]

				newResponse, err := http.Get(newAttachment.Url)
				check(err)
				defer newResponse.Body.Close()
				newData, err := io.ReadAll(newResponse.Body)
				check(err)

				outputBuffer := new(bytes.Buffer)
				fe8savereader.Diff(bytes.NewReader(oldData), bytes.NewReader(newData), outputBuffer)

				outputString := outputBuffer.String()
				if len(outputString) > maxContentLength {
					followupJson = &types.InteractionCallbackData{
						Content: outputString[maxContentLength:],
					}
					outputString = outputString[:maxContentLength]
				}
				// TODO: Better long message support

				callbackJson = types.InteractionCallbackMessage{
					Type: 4,
					Data: types.InteractionCallbackData{
						Content: outputString,
					},
				}

				log.Printf("%s", outputBuffer)
			default:
				log.Printf("Unknown subcommand %s, aborting", arg.Name)
				return
			}

		default:
			log.Printf("Unknown subcommand %s, aborting", arg.Name)
			return
		}
	default:
		log.Printf("Unexpected command %s, aborting", data.Name)
		return
	}

	callbackBytes, err := json.Marshal(callbackJson)
	check(err)

	if attachment == nil {
		request, err := http.NewRequest("POST", url, bytes.NewBuffer(callbackBytes))
		check(err)

		body := rest.DoJsonRequest(request)

		var bodyJson any
		json.Unmarshal(body, &bodyJson)

		check(err)
		log.Printf("Interaction callback response: %s", bodyJson)
	} else {
		body, boundary := rest.MultiPartForm(callbackBytes, *attachment)
		request, err := http.NewRequest("POST", url, body)
		check(err)

		// dumpedRequest, err := httputil.DumpRequestOut(request, true)
		// check(err)
		// log.Printf("Request: %s", dumpedRequest)

		responseBody := rest.DoRequest(request, fmt.Sprintf("multipart/form-data; boundary=%s", boundary))

		// TODO: Merge logging with above
		var bodyJson any
		json.Unmarshal(responseBody, &bodyJson)

		check(err)
		log.Printf("Interaction callback response: %s", bodyJson)

	}

	if followupJson != nil {
		followupBytes, err := json.Marshal(followupJson)
		check(err)

		request, err := http.NewRequest("POST", followupUrl, bytes.NewBuffer(followupBytes))
		check(err)
		body := rest.DoJsonRequest(request)

		var bodyJson any
		json.Unmarshal(body, &bodyJson)

		check(err)
		log.Printf("Followup response: %s", bodyJson)
	}
}

func removeIndex(input []string, i int) []string {
	result := make([]string, 0)
	result = append(result, input[:i]...)
	result = append(result, input[i+1:]...)
	return result
}
