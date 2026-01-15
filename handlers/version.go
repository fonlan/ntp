package handlers

import (
	"net/http"
)

type VersionHandler struct {
	version string
}

func NewVersionHandler(version string) *VersionHandler {
	return &VersionHandler{version: version}
}

func (h *VersionHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"version": h.version,
	})
}
