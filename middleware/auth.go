package middleware

import (
	"net/http"

	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore

func SetStore(s *sessions.CookieStore) {
	store = s
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "quiz-session")
		if err != nil {
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}

		userID, ok := session.Values["userID"].(int)
		if !ok || userID == 0 {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next(w, r)
	}
}
