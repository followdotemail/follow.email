package gmail

import (
	"encoding/json"
)

type ParsedLabels struct {
	Category          string
	SystemLabels      []string
	GmailUserLabelIDs []string
}


// ParseGmailLabels separates Gmail labels into categories, system labels, and user labels
func ParseGmailLabels(labelIDs []string) ParsedLabels {
	result := ParsedLabels{
		SystemLabels:      []string{},
		GmailUserLabelIDs: []string{},
	}

	for _, labelID := range labelIDs {
		switch {
		case IsCategory(labelID):
			result.Category = GetCategoryName(labelID)
		case IsSystemLabel(labelID):
			result.SystemLabels = append(result.SystemLabels, labelID)
		default:
			// User created label - store gmail's id (string)
			result.GmailUserLabelIDs = append(result.GmailUserLabelIDs, labelID)
		}
	}

	return result
}


// ToJson converts a slice to JSON string for database storage
func ToJson(data interface{}) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}
