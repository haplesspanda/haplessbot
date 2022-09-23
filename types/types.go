package types

type Option struct {
	Name    string   `json:"name"`
	Type    int      `json:"type"`
	Value   any      `json:"value"`
	Options []Option `json:"options"`
}

type ResolvedEntities struct {
	Users       map[string]UserData        `json:"users"`
	Members     map[string]GuildMemberData `json:"members"`
	Attachments map[string]Attachment      `json:"attachments"`
}

type InteractionCreateDetails struct {
	Token   string          `json:"token"`
	Data    InteractionData `json:"data"`
	Member  GuildMemberData `json:"member"`
	Id      string          `json:"id"`
	GuildId string          `json:"guild_id"`
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
	Id          string `json:"id"`
	Description string `json:"description"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Url         string `json:"url"`
	Size        int    `json:"size"`
}

type InteractionCallbackData struct {
	Content     string       `json:"content"`
	Embeds      []Embed      `json:"embeds"`
	Attachments []Attachment `json:"attachments"`
}

type InteractionCallbackMessage struct {
	Type int                     `json:"type"`
	Data InteractionCallbackData `json:"data"`
}
