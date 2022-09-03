package fe8

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

type Characters map[string]Character

type BaseStats struct {
	Level      int
	Class      string
	Hp         int
	StrOrMag   int
	Skl        int
	Spd        int
	Lck        int
	Def        int
	Res        int
	Mov        int
	Con        int
	WeaponRank string
	Affinity   string
}

type GrowthRates struct {
	Hp       int
	StrOrMag int
	Skl      int
	Spd      int
	Lck      int
	Def      int
	Res      int
}

type Recruitment struct {
	Chapter     string
	Description string
}

type Metadata struct {
	ClassDiscriminator  string
	UsesStr             bool
	StartsFullyPromoted bool
	StartsTrainee       bool
	ThumbnailUrl        string
}

type Character struct {
	Name               string
	Stats              BaseStats
	Growths            GrowthRates
	EirikaRecruitment  Recruitment
	EphraimRecruitment Recruitment
	Meta               Metadata
}

var characters Characters

func init() {
	populateData()
}

func normalizeName(unnormalizedName string) string {
	return strings.ToLower(unnormalizedName)
}

func populateData() {
	eirikaRecruitmentData := readFile("fe8/data/recruitment_eirika.tsv")
	ephraimRecruitmentData := readFile("fe8/data/recruitment_ephraim.tsv")
	baseStatsData := readFile("fe8/data/basestats.tsv")
	growthsData := readFile("fe8/data/growths.tsv")
	metadataData := readFile("fe8/data/meta.tsv")

	initializeCharacterData(eirikaRecruitmentData)
	addEirikaRecruitment(eirikaRecruitmentData)
	addEphraimRecruitment(ephraimRecruitmentData)
	addBaseStats(baseStatsData)
	addGrowths(growthsData)
	addMetadata(metadataData)
}

func readFile(filename string) [][]string {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.Comma = '\t'
	data, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	// log.Printf("Data: %v", data)
	return data
}

func initializeCharacterData(data [][]string) {
	characters = make(Characters)
	for _, entry := range data {
		normalizedName := normalizeName(entry[0])
		characters[normalizedName] = Character{
			Name: entry[0],
		}
	}
}

func addEirikaRecruitment(data [][]string) {
	for _, entry := range data {
		if len(entry) != 4 {
			log.Println("Found row with wrong number of entries, skipping")
			continue
		}
		normalizedName := normalizeName(entry[0])
		character, ok := characters[normalizedName]
		if !ok {
			panic(ok)
		}
		character.EirikaRecruitment = Recruitment{
			Chapter:     entry[2],
			Description: entry[3],
		}

		characters[normalizedName] = character
	}
}

func addEphraimRecruitment(data [][]string) {
	for _, entry := range data {
		if len(entry) != 4 {
			log.Println("Found row with wrong number of entries, skipping")
			continue
		}
		normalizedName := normalizeName(entry[0])
		character, ok := characters[normalizedName]
		if !ok {
			panic(ok)
		}
		character.EphraimRecruitment = Recruitment{
			Chapter:     entry[2],
			Description: entry[3],
		}
		characters[normalizedName] = character
	}
}

func addBaseStats(data [][]string) {
	for _, entry := range data {
		if len(entry) != 14 {
			log.Println("Found row with wrong number of entries, skipping")
			continue
		}
		normalizedName := normalizeName(entry[0])
		character, ok := characters[normalizedName]
		if !ok {
			panic(ok)
		}
		character.Stats = BaseStats{
			Level:      parseInt(entry[1]),
			Class:      entry[2],
			Hp:         parseInt(entry[3]),
			StrOrMag:   parseInt(entry[4]),
			Skl:        parseInt(entry[5]),
			Spd:        parseInt(entry[6]),
			Lck:        parseInt(entry[7]),
			Def:        parseInt(entry[8]),
			Res:        parseInt(entry[9]),
			Mov:        parseInt(entry[10]),
			Con:        parseInt(entry[11]),
			WeaponRank: entry[12],
			Affinity:   entry[13],
		}
		characters[normalizedName] = character
	}
}

func addGrowths(data [][]string) {
	for _, entry := range data {
		if len(entry) != 8 {
			log.Println("Found row with wrong number of entries, skipping")
			continue
		}
		normalizedName := normalizeName(entry[0])
		character, ok := characters[normalizedName]
		if !ok {
			panic(ok)
		}
		character.Growths = GrowthRates{
			Hp:       parseInt(entry[1]),
			StrOrMag: parseInt(entry[2]),
			Skl:      parseInt(entry[3]),
			Spd:      parseInt(entry[4]),
			Lck:      parseInt(entry[5]),
			Def:      parseInt(entry[6]),
			Res:      parseInt(entry[7]),
		}
		characters[normalizedName] = character
	}
}

func addMetadata(data [][]string) {
	for _, entry := range data {
		if len(entry) != 6 {
			log.Println("Found row with wrong number of entries, skipping")
			continue
		}
		normalizedName := normalizeName(entry[0])
		character, ok := characters[normalizedName]
		if !ok {
			panic(ok)
		}
		character.Meta = Metadata{
			ClassDiscriminator:  entry[1],
			UsesStr:             entry[2] == "Str",
			StartsFullyPromoted: entry[3] == "True",
			StartsTrainee:       entry[4] == "True",
			ThumbnailUrl:        entry[5],
		}
		characters[normalizedName] = character
	}
}

func parseInt(str string) int {
	result, err := strconv.Atoi(str)
	if err != nil {
		panic(err)
	}
	return result
}

type CharacterResponse struct {
	Name           string
	Content        string
	ThumbnailImage *CharacterImage
}

type CharacterImage struct {
	Name     string
	Filename string
}

func GetCharacterData(characterName string) (*CharacterResponse, *string) {
	normalizedName := normalizeName(characterName)
	character, ok := characters[normalizedName]
	if !ok {
		err := fmt.Sprintf("Unknown character: %s", characterName)
		return nil, &err
	}

	promotions, err := GetPromotions(character.Stats.Class, character.Meta.ClassDiscriminator, character.Meta.StartsFullyPromoted)
	if err != nil {
		return nil, err
	}
	promotionsString := ""
	if len(*promotions) > 0 {
		promotionsString = fmt.Sprintf("Promotions: %s", strings.Join(*promotions, ", "))
	}
	classString :=
		formatClassList([]DisplayClass{{Name: character.Stats.Class, Level: character.Stats.Level}})

	statsString := formatStats(
		fmt.Sprint(character.Stats.Hp),
		fmt.Sprint(character.Stats.StrOrMag),
		fmt.Sprint(character.Stats.Skl),
		fmt.Sprint(character.Stats.Spd),
		fmt.Sprint(character.Stats.Lck),
		fmt.Sprint(character.Stats.Def),
		fmt.Sprint(character.Stats.Res),
		fmt.Sprint(character.Stats.Mov),
		fmt.Sprint(character.Stats.Con),
		character.Meta.UsesStr)
	description := fmt.Sprintf(`
%s
Starting Stats: %s
Weapon Rank: %s
Affinity: %s
%s`,
		classString,
		statsString,
		character.Stats.WeaponRank,
		character.Stats.Affinity,
		promotionsString,
	)

	filename := character.Meta.ThumbnailUrl
	result := CharacterResponse{
		Name:    character.Name,
		Content: description,
		ThumbnailImage: &CharacterImage{
			Name:     fmt.Sprintf("%s.png", normalizedName),
			Filename: filename,
		},
	}
	return &result, nil
}

func GetAverageStats(characterName string, level int, promotion *string, promotionLevel *int, secondPromotion *string, secondPromotionLevel *int) (*CharacterResponse, *string) {
	normalizedName := normalizeName(characterName)
	character, ok := characters[normalizedName]
	if !ok {
		err := fmt.Sprintf("Unknown character: %s", characterName)
		return nil, &err
	}

	if level < character.Stats.Level || level > 20 {
		err := fmt.Sprintf("Invalid level: %d", level)
		return nil, &err
	}
	if character.Meta.StartsTrainee && level > 10 {
		err := fmt.Sprintf("Invalid trainee class level: %d", level)
		return nil, &err
	}
	if promotion == nil && promotionLevel != nil {
		err := "Missing promotion class"
		return nil, &err
	}
	if promotion != nil && promotionLevel == nil {
		err := "Missing promotion level"
		return nil, &err
	}
	if promotionLevel != nil && (*promotionLevel < 1 || *promotionLevel > 20) {
		err := fmt.Sprintf("Invalid promotion level: %d", promotionLevel)
		return nil, &err
	}

	if !character.Meta.StartsTrainee && (secondPromotion != nil || secondPromotionLevel != nil) {
		err := fmt.Sprintf("Cannot promote non-trainee %s a second time", character.Name)
		return nil, &err
	}
	if (promotion == nil || promotionLevel == nil) && (secondPromotion != nil || secondPromotionLevel != nil) {
		err := "Missing first promotion, but second promotion was requested"
		return nil, &err
	}
	if secondPromotion == nil && secondPromotionLevel != nil {
		err := "Missing second promotion class"
		return nil, &err
	}
	if secondPromotion != nil && secondPromotionLevel == nil {
		err := "Missing second promotion level"
		return nil, &err
	}
	if secondPromotionLevel != nil && (*secondPromotionLevel < 1 || *secondPromotionLevel > 20) {
		err := fmt.Sprintf("Invalid second promotion level: %d", secondPromotionLevel)
		return nil, &err
	}

	class := GetClass(character.Stats.Class, character.Meta.ClassDiscriminator, character.Meta.StartsFullyPromoted)
	// log.Printf("Class: %v", class)

	startingLevel := character.Stats.Level

	averageStats := AverageStats{
		Hp:       float64(character.Stats.Hp),
		StrOrMag: float64(character.Stats.StrOrMag),
		Skl:      float64(character.Stats.Skl),
		Spd:      float64(character.Stats.Spd),
		Lck:      float64(character.Stats.Lck),
		Def:      float64(character.Stats.Def),
		Res:      float64(character.Stats.Res),
		Mov:      character.Stats.Mov,
		Con:      character.Stats.Con,
	}

	averageStats = applyLevels(startingLevel, level, averageStats, character.Growths, *class.MaxStats)

	classes := []DisplayClass{{Name: character.Stats.Class, Level: level}}

	if promotion != nil && promotionLevel != nil {
		actualPromotion := *promotion
		actualPromotionLevel := *promotionLevel

		// Apply promotion bonuses
		updatedStats, promotedClassName, err := applyPromotion(character.Stats.Class, actualPromotion, character.Meta.ClassDiscriminator, averageStats)
		if err != nil {
			return nil, err
		}

		averageStats = *updatedStats

		promotionClass := GetClass(actualPromotion, character.Meta.ClassDiscriminator, !character.Meta.StartsTrainee)
		// log.Printf("Promoted class: %v", promotionClass)

		// Apply levelup bonuses
		averageStats = applyLevels(1, actualPromotionLevel, averageStats, character.Growths, *promotionClass.MaxStats)

		classes = append(classes, DisplayClass{Name: *promotedClassName, Level: actualPromotionLevel})

		if character.Meta.StartsTrainee && secondPromotion != nil && secondPromotionLevel != nil {
			actualSecondPromotion := *secondPromotion
			actualSecondPromotionLevel := *secondPromotionLevel

			// Apply promotion bonuses
			updatedStats, secondPromotedClassName, err := applyPromotion(*promotedClassName, actualSecondPromotion, character.Meta.ClassDiscriminator, averageStats)
			if err != nil {
				return nil, err
			}

			averageStats = *updatedStats

			secondPromotionClass := GetClass(actualSecondPromotion, character.Meta.ClassDiscriminator, true)

			// Apply levelup bonuses
			averageStats = applyLevels(1, actualSecondPromotionLevel, averageStats, character.Growths, *secondPromotionClass.MaxStats)

			classes = append(classes, DisplayClass{Name: *secondPromotedClassName, Level: actualSecondPromotionLevel})
		}
	}

	classString := formatClassList(classes)

	statsString := formatStats(
		fmt.Sprintf("%.2f", averageStats.Hp),
		fmt.Sprintf("%.2f", averageStats.StrOrMag),
		fmt.Sprintf("%.2f", averageStats.Skl),
		fmt.Sprintf("%.2f", averageStats.Spd),
		fmt.Sprintf("%.2f", averageStats.Lck),
		fmt.Sprintf("%.2f", averageStats.Def),
		fmt.Sprintf("%.2f", averageStats.Res),
		fmt.Sprint(averageStats.Mov),
		fmt.Sprint(averageStats.Con),
		character.Meta.UsesStr)

	description := fmt.Sprintf(`
%s
Average Stats: %s`, classString, statsString)

	filename := character.Meta.ThumbnailUrl
	result := CharacterResponse{
		Name:    character.Name,
		Content: description,
		ThumbnailImage: &CharacterImage{
			Name:     fmt.Sprintf("%s.png", normalizedName),
			Filename: filename,
		},
	}

	return &result, nil
}

// Return values: stats, canonical class name, error
func applyPromotion(startingClass string, promotionClass string, classDescriminator string, averageStatsBefore AverageStats) (*AverageStats, *string, *string) {
	promotionData, err := GetPromotion(startingClass, promotionClass, classDescriminator)
	if err != nil {
		return nil, nil, err
	}

	result := AverageStats{
		Hp:       averageStatsBefore.Hp + float64(promotionData.Hp),
		StrOrMag: averageStatsBefore.StrOrMag + float64(promotionData.StrOrMag),
		Skl:      averageStatsBefore.Skl + float64(promotionData.Skl),
		Spd:      averageStatsBefore.Spd + float64(promotionData.Spd),
		// Luck never increased on promotions!
		Lck: averageStatsBefore.Lck,
		Def: averageStatsBefore.Def + float64(promotionData.Def),
		Res: averageStatsBefore.Res + float64(promotionData.Res),
		Mov: averageStatsBefore.Mov + promotionData.Mov,
		Con: averageStatsBefore.Con + promotionData.Con,
	}

	return &result, &promotionData.PromotedClass, nil
}

func applyLevels(startingLevel int, endingLevel int, averageStatsBefore AverageStats, growths GrowthRates, maxStats MaximumStats) AverageStats {
	levelDiff := endingLevel - startingLevel
	// log.Printf("Promoted class: %v", promotionClass)

	result := AverageStats{
		Hp:       calculateAverageStatFloat(averageStatsBefore.Hp, growths.Hp, levelDiff, maxStats.Hp),
		StrOrMag: calculateAverageStatFloat(averageStatsBefore.StrOrMag, growths.StrOrMag, levelDiff, maxStats.StrOrMag),
		Skl:      calculateAverageStatFloat(averageStatsBefore.Skl, growths.Skl, levelDiff, maxStats.Skl),
		Spd:      calculateAverageStatFloat(averageStatsBefore.Spd, growths.Spd, levelDiff, maxStats.Spd),
		Lck:      calculateAverageStatFloat(averageStatsBefore.Lck, growths.Lck, levelDiff, maxStats.Lck),
		Def:      calculateAverageStatFloat(averageStatsBefore.Def, growths.Def, levelDiff, maxStats.Def),
		Res:      calculateAverageStatFloat(averageStatsBefore.Res, growths.Res, levelDiff, maxStats.Res),
		Mov:      averageStatsBefore.Mov,
		Con:      averageStatsBefore.Con,
	}
	return result
}

type AverageStats struct {
	Hp       float64
	StrOrMag float64
	Skl      float64
	Spd      float64
	Lck      float64
	Def      float64
	Res      float64
	Mov      int
	Con      int
}

func calculateAverageStatFloat(starting float64, growthRate int, levelDiff int, maximum int) float64 {
	uncappedResult := float64(starting) + float64(levelDiff)*float64(growthRate)*.01
	return math.Min(uncappedResult, float64(maximum))
}

func formatStats(hp string, strOrMag string, skl string, spd string, lck string, def string, res string, mov string, con string, usesStr bool) string {
	var strOrMagLabel string
	if usesStr {
		strOrMagLabel = "Str"
	} else {
		strOrMagLabel = "Mag"
	}
	return fmt.Sprintf("```"+
		`
HP     %s 
%s    %s    Lck    %s
Skl    %s    Def    %s
Spd    %s    Res    %s
Mov    %s    Con    %s`+"```",
		maybeAddWhitespace(hp),
		strOrMagLabel,
		maybeAddWhitespace(strOrMag),
		maybeAddWhitespace(lck),
		maybeAddWhitespace(skl),
		maybeAddWhitespace(def),
		maybeAddWhitespace(spd),
		maybeAddWhitespace(res),
		maybeAddWhitespace(mov),
		maybeAddWhitespace(con))
}

func maybeAddWhitespace(num string) string {
	// TODO: Kind of annoying to go float -> string -> float, consider reorganizing
	number, _ := strconv.ParseFloat(num, 64)
	if number < 10 {
		return " " + num
	} else {
		return num
	}
}

type DisplayClass struct {
	Name  string
	Level int
}

func formatClassList(classes []DisplayClass) string {
	if len(classes) == 0 {
		log.Printf("Formatting empty class list")
		return ""
	}
	label := "Classes"
	if len(classes) == 1 {
		label = "Class"
	}

	var formattedClasses []string
	for _, class := range classes {
		formattedClasses = append(formattedClasses, formatClass(class))
	}

	result := fmt.Sprintf("%s: %s", label, strings.Join(formattedClasses, ", "))
	return result
}

func formatClass(class DisplayClass) string {
	return fmt.Sprintf("%s %d", class.Name, class.Level)
}
