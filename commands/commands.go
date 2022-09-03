package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/haplesspanda/haplessbot/constants"
	"github.com/haplesspanda/haplessbot/fe8"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var acceptedCommands = map[string]struct{}{"ping": {}, "avatar": {}, "banner": {}, "choose": {}, "order": {}, "fe8": {}}
var client *http.Client

func init() {
	client = &http.Client{}
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

		body := doJsonRequest(request)

		var bodyJson any
		json.Unmarshal(body, &bodyJson)
		log.Printf("Command response: %s", bodyJson)
	}
}

type Option struct {
	Name    string   `json:"name"`
	Type    int      `json:"type"`
	Value   any      `json:"value"`
	Options []Option `json:"options"`
}

type ResolvedEntities struct {
	Users   map[string]UserData        `json:"users"`
	Members map[string]GuildMemberData `json:"members"`
}

type InteractionData struct {
	Type     int              `json:"type"`
	Name     string           `json:"name"`
	Id       string           `json:"id"`
	Options  []Option         `json:"options"`
	Resolved ResolvedEntities `json:"resolved"`
}

type GuildMemberData struct {
	User   UserData `json:"user"`
	Avatar *string  `json:"avatar"`
}

type UserData struct {
	Username      string  `json:"username"`
	Discriminator string  `json:"discriminator"`
	Id            string  `json:"id"`
	Avatar        *string `json:"avatar"`
	Banner        *string `json:"banner"`
}

type EmbedImage struct {
	Url string `json:"url"`
}

type EmbedThumbnail struct {
	Url string `json:"url"`
}

type Embed struct {
	Title       string         `json:"title"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Url         string         `json:"url"`
	Image       EmbedImage     `json:"image"`
	Thumbnail   EmbedThumbnail `json:"thumbnail"`
}

type Attachment struct {
	// Id          string `json:"id"`
	Description string `json:"description"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

func RunInteractionCallback(data InteractionData, memberData GuildMemberData, guildId string, interactionId string, interactionToken string) {
	log.Printf("Processing %s", interactionId)
	if data.Type != 1 {
		log.Printf("Unexpected command type %d, aborting", data.Type)
		return
	}

	userData := memberData.User

	url := fmt.Sprintf("https://discord.com/api/v10/interactions/%s/%s/callback", interactionId, interactionToken)
	log.Println(url)

	type InteractionCallbackData struct {
		Content     string       `json:"content"`
		Embeds      []Embed      `json:"embeds"`
		Attachments []Attachment `json:"attachments"`
	}

	type InteractionCallbackMessage struct {
		Type int                     `json:"type"`
		Data InteractionCallbackData `json:"data"`
	}

	var callbackJson InteractionCallbackMessage
	var attachment *BinaryAttachment
	switch data.Name {
	case "ping":
		callbackJson = InteractionCallbackMessage{
			Type: 4,
			Data: InteractionCallbackData{
				Content: "Pong",
			},
		}
	case "avatar":
		// Pick the user (maybe specified from command)
		var avatarUser UserData
		var avatarGuildMember GuildMemberData
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
			callbackJson = InteractionCallbackMessage{
				Type: 4,
				Data: InteractionCallbackData{
					Embeds: []Embed{{
						Title: fmt.Sprintf("Avatar for %s", fullUser),
						Url:   avatarUrl,
						Image: EmbedImage{Url: avatarUrl},
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
			callbackJson = InteractionCallbackMessage{
				Type: 4,
				Data: InteractionCallbackData{
					Embeds: []Embed{{
						Title: fmt.Sprintf("Avatar for %s", fullUser),
						Url:   avatarUrl,
						Image: EmbedImage{Url: avatarUrl},
					}},
				},
			}
		} else {
			callbackJson = InteractionCallbackMessage{
				Type: 4,
				Data: InteractionCallbackData{
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

		body := doJsonRequest(request)

		var bannerUser UserData

		log.Println(body)
		json.Unmarshal(body, &bannerUser)
		log.Println(bannerUser)

		fullUser := fmt.Sprintf("%s#%s", bannerUser.Username, bannerUser.Discriminator)

		if bannerUser.Banner == nil {
			callbackJson = InteractionCallbackMessage{
				Type: 4,
				Data: InteractionCallbackData{
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
			callbackJson = InteractionCallbackMessage{
				Type: 4,
				Data: InteractionCallbackData{
					Embeds: []Embed{{
						Title: fmt.Sprintf("Banner for %s", fullUser),
						Url:   bannerUrl,
						Image: EmbedImage{Url: bannerUrl},
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
		callbackJson = InteractionCallbackMessage{
			Type: 4,
			Data: InteractionCallbackData{
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

		callbackJson = InteractionCallbackMessage{
			Type: 4,
			Data: InteractionCallbackData{
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
					attachment = &BinaryAttachment{
						ContentType: "image/png",
						Name:        data.ThumbnailImage.Name,
						Filename:    data.ThumbnailImage.Filename,
					}
					callbackJson = InteractionCallbackMessage{
						Type: 4,
						Data: InteractionCallbackData{
							Embeds: []Embed{{
								Title:       data.Name,
								Description: data.Content,
								Thumbnail: EmbedThumbnail{
									Url: thumbnailUrl,
								},
							}},
						},
					}
				} else {
					callbackJson = InteractionCallbackMessage{
						Type: 4,
						Data: InteractionCallbackData{
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
					attachment = &BinaryAttachment{
						ContentType: "image/png",
						Name:        data.ThumbnailImage.Name,
						Filename:    data.ThumbnailImage.Filename,
					}
					callbackJson = InteractionCallbackMessage{
						Type: 4,
						Data: InteractionCallbackData{
							Embeds: []Embed{{
								Title:       data.Name,
								Description: data.Content,
								Thumbnail: EmbedThumbnail{
									Url: thumbnailUrl,
								},
							}},
						},
					}
				} else {
					callbackJson = InteractionCallbackMessage{
						Type: 4,
						Data: InteractionCallbackData{
							Content: *err,
						},
					}
				}
			default:
				log.Printf("Unknown subcommand %s, aborting", arg.Options[0].Name)
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

		body := doJsonRequest(request)

		var bodyJson any
		json.Unmarshal(body, &bodyJson)

		check(err)
		log.Printf("Interaction callback response: %s", bodyJson)
	} else {
		body, boundary := multiPartForm(callbackBytes, *attachment)
		request, err := http.NewRequest("POST", url, body)
		check(err)

		// dumpedRequest, err := httputil.DumpRequestOut(request, true)
		// check(err)
		// log.Printf("Request: %s", dumpedRequest)

		responseBody := doRequest(request, fmt.Sprintf("multipart/form-data; boundary=%s", boundary))

		// TODO: Merge logging with above
		var bodyJson any
		json.Unmarshal(responseBody, &bodyJson)

		check(err)
		log.Printf("Interaction callback response: %s", bodyJson)

	}
}

func removeIndex(input []string, i int) []string {
	result := make([]string, 0)
	result = append(result, input[:i]...)
	result = append(result, input[i+1:]...)
	return result
}

func doJsonRequest(request *http.Request) []byte {
	return doRequest(request, "application/json")
}

func doRequest(request *http.Request, contentType string) []byte {
	appendHeaders(request, contentType)
	response, err := client.Do(request)
	check(err)

	dumpedResponse, err := httputil.DumpResponse(response, true)
	check(err)

	log.Printf("HTTP response: %s", dumpedResponse)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	check(err)

	return body
}

func appendHeaders(request *http.Request, contentType string) {
	authHeader := fmt.Sprintf("Bot %s", constants.TokenId)

	request.Header.Set("Authorization", authHeader)
	request.Header.Set("Content-Type", contentType)
}

type BinaryAttachment struct {
	ContentType string
	Name        string
	Filename    string
}

func multiPartForm(jsonData []byte, attachment BinaryAttachment) (*bytes.Buffer, string) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	var jsonWriter io.Writer
	jsonHeader := make(textproto.MIMEHeader)
	jsonHeader.Set("Content-Disposition", "form-data; name=\"payload_json\"")
	jsonHeader.Set("Content-Type", "application/json")
	jsonWriter, err := writer.CreatePart(jsonHeader)
	if err != nil {
		panic(err)
	}

	_, err = jsonWriter.Write(jsonData)
	if err != nil {
		panic(err)
	}

	var binaryWriter io.Writer
	binaryHeader := make(textproto.MIMEHeader)
	binaryHeader.Set("Content-Disposition", fmt.Sprintf("form-data; name=\"file0\"; filename=\"%s\"", attachment.Name))
	binaryHeader.Set("Content-Type", attachment.ContentType)
	binaryWriter, err = writer.CreatePart(binaryHeader)
	if err != nil {
		panic(err)
	}

	file, err := os.Open(attachment.Filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	io.Copy(binaryWriter, file)
	if err != nil {
		panic(err)
	}

	err = writer.Close()
	if err != nil {
		panic(err)
	}
	return &b, writer.Boundary()
}
