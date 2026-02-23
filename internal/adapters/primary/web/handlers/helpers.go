package handlers

import (
	"encoding/json"
	"net/http"
)

// ✅ Constante pour satisfaire le DRY (Don't Repeat Yourself)
const contentTypeJSON = "application/json"

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)

	// ✅ Le "_ =" indique qu'on ignore consciemment l'erreur d'encodage
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"status":  "error",
		"message": message,
	})
}
