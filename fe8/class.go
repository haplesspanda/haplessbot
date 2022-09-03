package fe8

import (
	"fmt"
	"log"
)

type Classes map[string]Class

type MaximumStats struct {
	Hp       int
	StrOrMag int
	Skl      int
	Spd      int
	Lck      int
	Def      int
	Res      int
	Mov      int
	Con      int
}

type Promotion struct {
	StartingClass string
	PromotedClass string
	Hp            int
	StrOrMag      int
	Skl           int
	Spd           int
	Def           int
	Res           int
	Con           int
	Mov           int
	WeaponRanks   string
}

type Class struct {
	Name       string
	MaxStats   *MaximumStats
	Promotions *[]Promotion
}

var classes Classes

func init() {
	maxStatsData := readFile("fe8/data/maxstats.tsv")
	promotionsData := readFile("fe8/data/promotions.tsv")

	initializeClassData(promotionsData, maxStatsData)
	addPromotions(promotionsData)
	addMaxStats(maxStatsData)
}

func initializeClassData(data [][]string, additionalData [][]string) {
	classes = make(Classes)

	// Add starting and promotion class for all promotions
	for _, entry := range data {
		classes[normalizeName(entry[0])] = Class{
			Name: entry[0],
		}
		classes[normalizeName(entry[1])] = Class{
			Name: entry[1],
		}
	}

	// Add classes with stats data (for standalone classes like Dancer)
	for _, entry := range additionalData {
		classes[normalizeName(entry[0])] = Class{
			Name: entry[0],
		}
	}
}

func addPromotions(data [][]string) {
	for _, entry := range data {
		if len(entry) != 11 {
			log.Println("Found row with wrong number of entries, skipping")
			continue
		}
		class, ok := classes[normalizeName(entry[0])]
		if !ok {
			panic(ok)
		}
		promotion := Promotion{
			StartingClass: entry[0],
			PromotedClass: entry[1],
			Hp:            parseInt(entry[2]),
			StrOrMag:      parseInt(entry[3]),
			Skl:           parseInt(entry[4]),
			Spd:           parseInt(entry[5]),
			Def:           parseInt(entry[6]),
			Res:           parseInt(entry[7]),
			Con:           parseInt(entry[8]),
			Mov:           parseInt(entry[9]),
			WeaponRanks:   entry[10],
		}
		if class.Promotions == nil {
			promotions := make([]Promotion, 0)
			class.Promotions = &promotions
		}
		newPromotions := append(*class.Promotions, promotion)
		class.Promotions = &newPromotions

		log.Println(class.Promotions)

		classes[normalizeName(entry[0])] = class
	}
}

func addMaxStats(data [][]string) {
	for _, entry := range data {
		if len(entry) != 10 {
			log.Println("Found row with wrong number of entries, skipping")
			continue
		}
		class, ok := classes[normalizeName(entry[0])]
		if !ok {
			panic(ok)
		}
		class.MaxStats = &MaximumStats{
			Hp:       parseInt(entry[1]),
			StrOrMag: parseInt(entry[2]),
			Skl:      parseInt(entry[3]),
			Spd:      parseInt(entry[4]),
			Lck:      parseInt(entry[5]),
			Def:      parseInt(entry[6]),
			Res:      parseInt(entry[7]),
			Mov:      parseInt(entry[8]),
			Con:      parseInt(entry[9]),
		}

		classes[normalizeName(entry[0])] = class
	}
}

func GetClass(className string, classDiscriminator string, fullyPromoted bool) Class {
	// Class without discriminator
	result, exists := classes[normalizeName(className)]
	if exists && result.MaxStats != nil {
		return result
	}

	// Class with discriminator
	result, exists = classes[normalizeName(fmt.Sprintf("%s (%s)", className, classDiscriminator))]
	if exists && result.MaxStats != nil {
		return result
	}

	// Fallback for unpromoted unit
	if fullyPromoted {
		panic(fmt.Sprintf("Unknown fully-promoted class: %s", className))
	} else {
		// TODO: If this ends up caring about con, choose between foot/mounted appropriately
		result, exists = classes[normalizeName("Non-promoted (foot)")]
		if !exists || result.MaxStats == nil {
			panic(fmt.Sprintf("Could not get default unpromoted class stats for: %s", className))
		}
	}

	return result
}

func GetPromotions(className string, classDiscriminator string, fullyPromoted bool) (*[]string, *string) {
	class, exists := classes[normalizeName(className)]
	if !exists || class.Promotions == nil {
		class, exists = classes[normalizeName(fmt.Sprintf("%s (%s)", className, classDiscriminator))]
	}

	if !exists || class.Promotions == nil {
		if fullyPromoted {
			return &[]string{}, nil
		} else {
			err := fmt.Sprintf("Could not get class promotions for: %s (%s)", className, classDiscriminator)
			return nil, &err
		}
	}

	result := make([]string, 0)
	for _, promotion := range *class.Promotions {
		result = append(result, promotion.PromotedClass)
	}
	return &result, nil
}

func GetPromotion(startingClass string, promotionClass string, classDiscriminator string) (*Promotion, *string) {
	fullStartingClass := fmt.Sprintf("%s (%s)", startingClass, classDiscriminator)
	fullPromotionClass := fmt.Sprintf("%s (%s)", promotionClass, classDiscriminator)
	class, exists := classes[normalizeName(startingClass)]
	if !exists || class.Promotions == nil {
		class, exists = classes[normalizeName(fullStartingClass)]
	}

	if !exists || class.Promotions == nil {
		err := fmt.Sprintf("Could not get class promotions for class: %s (%s)", startingClass, classDiscriminator)
		return nil, &err
	}

	var promotion *Promotion
	// Note: This loop is a somewhat dangerous pattern because it returns non-matches without the break!
	for _, element := range *class.Promotions {
		if normalizeName(element.PromotedClass) == normalizeName(promotionClass) {
			promotion = &element
			break
		} else if normalizeName(element.PromotedClass) == normalizeName(fullPromotionClass) {
			promotion = &element
			break
		}
	}

	if promotion == nil {
		err := fmt.Sprintf("Could not find promotion from %s to %s", startingClass, promotionClass)
		return nil, &err
	}

	log.Printf("Returning promotion %v", promotion)
	return promotion, nil
}
