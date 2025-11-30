package handlers

import (
    "net/http"
    
    "follow-email-backend/internal/services"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

type LabelHandler struct {
    db           *gorm.DB
    labelService *services.LabelService
}

func NewLabelHandler(db *gorm.DB, labelService *services.LabelService) *LabelHandler {
    return &LabelHandler{
        db:           db,
        labelService: labelService,
    }
}

// GetLabels returns all labels for the authenticated user
func (h *LabelHandler) GetLabels(c *gin.Context) {
    clerkID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
        return
    }
    
    userID, err := h.getUserUUIDFromClerkID(clerkID.(string))
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
        return
    }
    
    labels, err := h.labelService.GetUserLabels(c.Request.Context(), userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get labels"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"labels": labels})
}


// SyncLabels syncs labels from Gmail
func (h *LabelHandler) SyncLabels(c *gin.Context) {
    clerkID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
        return
    }
    
    userID, err := h.getUserUUIDFromClerkID(clerkID.(string))
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
        return
    }
    
    err = h.labelService.SyncUserLabels(c.Request.Context(), userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync labels: " + err.Error()})
        return
    }
    
    // Return updated labels
    labels, _ := h.labelService.GetUserLabels(c.Request.Context(), userID)
    c.JSON(http.StatusOK, gin.H{
        "message": "Labels synced successfully",
        "labels":  labels,
    })
}

func (h *LabelHandler) getUserUUIDFromClerkID(clerkID string) (uuid.UUID, error) {
    var user struct{ ID uuid.UUID }
    err := h.db.Table("users").Select("id").Where("clerk_id = ?", clerkID).First(&user).Error
    return user.ID, err
}