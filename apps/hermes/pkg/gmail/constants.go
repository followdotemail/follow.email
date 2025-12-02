package gmail

var SystemLabels = map[string]bool{
	"INBOX":     true,
	"SENT":      true,
	"DRAFT":     true,
	"SPAM":      true,
	"TRASH":     true,
	"STARRED":   true,
	"IMPORTANT": true,
	"UNREAD":    true,
	"ARCHIVED":  true,
	"CHAT":      true,
}

var CategoryLabels = map[string]string{
	"CATEGORY_PERSONAL":   "personal",
	"CATEGORY_SOCIAL":     "social",
	"CATEGORY_PROMOTIONS": "promotions",
	"CATEGORY_UPDATES":    "updates",
	"CATEGORY_FORUMS":     "forums",
}

func IsSystemLabel(label string) bool {
	return SystemLabels[label]
}

func IsCategory(labelID string) bool {
	_, exists := CategoryLabels[labelID]
	return exists
}

func GetCategoryName(labelID string) string {
	if name, exists := CategoryLabels[labelID]; exists {
		return name
	}

	return "" // NEED TO HANDLE THIS CASE OR WILL MAKE IT NULL
}
