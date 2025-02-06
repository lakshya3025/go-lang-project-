package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"quizapp/database"
	"quizapp/middleware"
	"quizapp/services"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
)

var (
	store         *sessions.CookieStore
	templates     *template.Template
	templateFuncs = template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
		"formatScore": func(score float64) string {
			return fmt.Sprintf("%.1f", score)
		},
		"split": strings.Split,
	}
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Set default session key
	sessionKey := "development-secret-key-123"
	if key := os.Getenv("SESSION_KEY"); key != "" {
		sessionKey = key
	} else {
		log.Println("Warning: Using default session key. This is not secure for production.")
	}

	store = sessions.NewCookieStore([]byte(sessionKey))
	middleware.SetStore(store)

	// Initialize database
	log.Println("Initializing database...")
	if err := database.Initialize("quiz.db"); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	log.Println("Database initialized successfully")

	// Parse templates with functions
	log.Println("Parsing templates...")
	templates = template.Must(template.New("").Funcs(templateFuncs).ParseGlob("templates/*.html"))
	log.Println("Templates parsed successfully")
}

func main() {
	defer database.Close()
	r := mux.NewRouter()

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Auth routes
	r.HandleFunc("/register", handleRegister).Methods("GET", "POST")
	r.HandleFunc("/login", handleLogin).Methods("GET", "POST")
	r.HandleFunc("/logout", handleLogout).Methods("POST")

	// Quiz routes
	r.HandleFunc("/", middleware.RequireAuth(handleHome)).Methods("GET")
	r.HandleFunc("/quiz/{id}", middleware.RequireAuth(handleQuiz)).Methods("GET")
	r.HandleFunc("/api/submit-quiz", middleware.RequireAuth(handleQuizSubmission)).Methods("POST")

	// Admin routes (protected)
	r.HandleFunc("/admin/create-quiz", middleware.RequireAuth(handleCreateQuiz)).Methods("GET", "POST")

	// Leaderboard route
	r.HandleFunc("/leaderboard", middleware.RequireAuth(handleLeaderboard)).Methods("GET")

	// Past quizzes route
	r.HandleFunc("/past-quizzes", middleware.RequireAuth(handlePastQuizzes)).Methods("GET")

	// GitHub routes
	r.HandleFunc("/auth/github", handleGithubAuth)
	r.HandleFunc("/auth/github/callback", handleGithubCallback)

	// Add logging
	port := ":8080"
	log.Printf("Server starting on http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, r))
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if err := templates.ExecuteTemplate(w, "register.html", map[string]interface{}{
			"Error": r.URL.Query().Get("error"),
		}); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
		}
		return
	}

	// Get form values
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Basic validation
	if username == "" || email == "" || password == "" {
		http.Redirect(w, r, "/register?error=All fields are required", http.StatusSeeOther)
		return
	}

	// Create user
	if err := database.CreateUser(username, email, password); err != nil {
		log.Printf("Failed to create user: %v", err)
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			http.Redirect(w, r, "/register?error=Username or email already exists", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/register?error=Failed to create account", http.StatusSeeOther)
		return
	}

	// Log success and redirect
	log.Printf("New user registered: %s (%s)", username, email)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		err := templates.ExecuteTemplate(w, "login.html", nil)
		if err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	log.Printf("Login attempt: username=%s", username)

	user, err := database.GetUserByUsername(username)
	if err != nil {
		log.Printf("Login error: %v", err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Simple password check for now
	if user.Password != password {
		log.Printf("Password mismatch for user: %s", username)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	session, err := store.Get(r, "quiz-session")
	if err != nil {
		log.Printf("Session error: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	session.Values["userID"] = user.ID
	if err := session.Save(r, w); err != nil {
		log.Printf("Session save error: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Login successful: username=%s, userID=%d", username, user.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleCreateQuiz(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "quiz-session")
	userID, ok := session.Values["userID"].(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "GET" {
		categories, err := services.FetchCategories()
		if err != nil {
			http.Error(w, "Failed to fetch categories", http.StatusInternalServerError)
			return
		}

		templates.ExecuteTemplate(w, "create_quiz.html", map[string]interface{}{
			"Categories": categories,
		})
		return
	}

	var request struct {
		Title         string `json:"title"`
		Category      int    `json:"category"`
		Difficulty    string `json:"difficulty"`
		QuestionCount int    `json:"questionCount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	questions, err := services.FetchQuizQuestions(request.Category, request.Difficulty, request.QuestionCount)
	if err != nil {
		http.Error(w, "Failed to fetch questions", http.StatusInternalServerError)
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		INSERT INTO quizzes (title, created_by)
		VALUES (?, ?)
	`, request.Title, userID)
	if err != nil {
		http.Error(w, "Failed to create quiz", http.StatusInternalServerError)
		return
	}

	quizID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Failed to get quiz ID", http.StatusInternalServerError)
		return
	}

	for _, q := range questions {
		options := append([]string{q.CorrectAnswer}, q.IncorrectAnswers...)
		optionsStr := strings.Join(options, "|")
		_, err = tx.Exec(`
			INSERT INTO questions (quiz_id, text, options, answer)
			VALUES (?, ?, ?, ?)
		`, quizID, q.Question, optionsStr, q.CorrectAnswer)
		if err != nil {
			http.Error(w, "Failed to create questions", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to create quiz", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"quizId": quizID,
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "quiz-session")
	session.Values = map[interface{}]interface{}{}
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "quiz-session")
	userID, ok := session.Values["userID"].(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get user info
	user, err := database.GetUserByID(userID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Get user stats
	stats, err := database.GetUserStats(userID)
	if err != nil {
		log.Printf("Error getting user stats: %v", err)
		stats = &database.UserStats{
			QuizzesTaken: 0,
			AverageScore: 0,
			GlobalRank:   0,
		}
	}

	// Get top 5 performers
	topScores, err := database.GetTopScores(5)
	if err != nil {
		log.Printf("Error getting top scores: %v", err)
		topScores = []database.TopScore{} // Use empty list on error
	}

	data := map[string]interface{}{
		"Username":     user.Username,
		"QuizzesTaken": stats.QuizzesTaken,
		"AverageScore": stats.AverageScore,
		"GlobalRank":   stats.GlobalRank,
		"TopScores":    topScores,
	}

	if err := templates.ExecuteTemplate(w, "home.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
}

func handleQuiz(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	quizID := vars["id"]

	quiz, err := database.GetQuizWithQuestions(quizID)
	if err != nil {
		log.Printf("Error getting quiz: %v", err)
		http.Error(w, "Quiz not found", http.StatusNotFound)
		return
	}

	if err := templates.ExecuteTemplate(w, "quiz.html", quiz); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
}

func handleQuizSubmission(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "quiz-session")
	userID, ok := session.Values["userID"].(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var submission struct {
		QuizID  int      `json:"quizId"`
		Answers []string `json:"answers"`
	}

	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get all questions and answers
	rows, err := database.DB.Query(`
		SELECT text, answer 
		FROM questions 
		WHERE quiz_id = ? 
		ORDER BY id
	`, submission.QuizID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var correctAnswers int
	var questions []map[string]interface{}

	for i := 0; rows.Next(); i++ {
		var questionText, correctAnswer string
		if err := rows.Scan(&questionText, &correctAnswer); err != nil {
			continue
		}

		userAnswer := ""
		if i < len(submission.Answers) {
			userAnswer = submission.Answers[i]
		}

		isCorrect := userAnswer == correctAnswer
		if isCorrect {
			correctAnswers++
		}

		questions = append(questions, map[string]interface{}{
			"text":          questionText,
			"isCorrect":     isCorrect,
			"userAnswer":    userAnswer,
			"correctAnswer": correctAnswer,
		})
	}

	totalQuestions := len(questions)
	if totalQuestions == 0 {
		http.Error(w, "Quiz not found", http.StatusNotFound)
		return
	}

	score := float64(correctAnswers) / float64(totalQuestions) * 100

	// Save the score
	if err := database.SaveQuizScore(userID, submission.QuizID, score); err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Get user's rank for this quiz
	var rank int
	err = database.DB.QueryRow(`
		WITH RankedScores AS (
			SELECT user_id, score,
				   RANK() OVER (ORDER BY score DESC) as rank
			FROM quiz_results
			WHERE quiz_id = ?
		)
		SELECT COALESCE(rank, 0)
		FROM RankedScores
		WHERE user_id = ?
	`, submission.QuizID, userID).Scan(&rank)
	if err != nil {
		rank = 0 // Default to 0 if error
	}

	// Return the results
	json.NewEncoder(w).Encode(map[string]interface{}{
		"score":          score,
		"correctAnswers": correctAnswers,
		"totalQuestions": totalQuestions,
		"questions":      questions,
		"rank":           rank,
	})
}

func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	results, err := database.GetLeaderboard()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if err := templates.ExecuteTemplate(w, "leaderboard.html", map[string]interface{}{
		"Results": results,
	}); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
}

func handleGithubAuth(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, services.GetGithubAuthURL(), http.StatusTemporaryRedirect)
}

func handleGithubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}

	githubUser, err := services.GetGithubUser(code)
	if err != nil {
		log.Printf("Failed to get GitHub user: %v", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// Create or get user
	user, err := database.GetOrCreateGithubUser(githubUser)
	if err != nil {
		log.Printf("Failed to process user: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Set session
	session, _ := store.Get(r, "quiz-session")
	session.Values["userID"] = user.ID
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func handlePastQuizzes(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "quiz-session")
	userID, ok := session.Values["userID"].(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	pastQuizzes, err := database.GetUserQuizzes(userID)
	if err != nil {
		log.Printf("Error getting past quizzes: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if err := templates.ExecuteTemplate(w, "past-quizzes.html", map[string]interface{}{
		"PastQuizzes": pastQuizzes,
	}); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
}
