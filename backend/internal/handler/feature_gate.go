package handler

import (
	"net/http"

	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// FeatureUnavailable prevents prototype-only flows from being used as if they
// provided real verification. Re-enable a route only after its external trust
// source and persistence model have been implemented.
func FeatureUnavailable(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Error(c, http.StatusNotImplemented, 50102, feature+" is not configured")
	}
}
