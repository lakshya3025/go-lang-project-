package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"quizapp/models"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// Add these type definitions at the top of the file
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"-"`
}

type Quiz struct {
	ID        int        `json:"id"`
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
}

type Question struct {
	ID             int         `json:"id"`
	QuizID         int         `json:"quiz_id"`
	Text           string      `json:"text"`
	Options        []string    `json:"options"`
	Answer         string      `json:"answer"`
	ImageURL       string      `json:"image_url,omitempty"`
	Context        string      `json:"context,omitempty"`
	WordDefinition interface{} `json:"word_definition,omitempty"`
}

type QuizWithScore struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	UserScore     float64 `json:"user_score"`
	Rank          int     `json:"rank"`
	TotalAttempts int     `json:"total_attempts"`
	HighScore     float64 `json:"high_score"`
}

type TopScore struct {
	Rank     int     `json:"rank"`
	Username string  `json:"username"`
	Score    float64 `json:"score"`
}

// Add these types if not already present
type LeaderboardEntry struct {
	Rank     int     `json:"rank"`
	Username string  `json:"username"`
	Score    float64 `json:"score"`
	QuizName string  `json:"quiz_name"`
}

// Add this type definition at the top with other types
type UserStats struct {
	QuizzesTaken int     `json:"quizzes_taken"`
	AverageScore float64 `json:"average_score"`
	GlobalRank   int     `json:"global_rank"`
}

// Initialize creates a new database connection and sets up the schema
func Initialize(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	if err := createTables(); err != nil {
		return err
	}

	// Add a test user if none exists
	var count int
	err = DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err = DB.Exec(`
			INSERT INTO users (username, email, password)
			VALUES (?, ?, ?)
		`, "test", "test@example.com", "test123")
		if err != nil {
			return err
		}
	}

	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		if err := DB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
			return err
		}
	}
	return nil
}

func createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS quizzes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			created_by INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (created_by) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS questions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			quiz_id INTEGER,
			text TEXT NOT NULL,
			options TEXT NOT NULL,
			answer TEXT NOT NULL,
			image_url TEXT,
			context TEXT,
			word_definition TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (quiz_id) REFERENCES quizzes(id)
		)`,
		`CREATE TABLE IF NOT EXISTS scores (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			quiz_id INTEGER,
			score REAL NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, quiz_id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (quiz_id) REFERENCES quizzes(id)
		)`,
		`CREATE TABLE IF NOT EXISTS quiz_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			quiz_id INTEGER NOT NULL,
			score REAL NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (quiz_id) REFERENCES quizzes(id),
			UNIQUE(user_id, quiz_id)
		)`,
	}

	for _, query := range queries {
		if _, err := DB.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

// GetUserByUsername retrieves a user by username
func GetUserByUsername(username string) (*User, error) {
	var user User
	err := DB.QueryRow(`
		SELECT id, username, email, password 
		FROM users 
		WHERE username = ?
	`, username).Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser creates a new user
func CreateUser(username, email, password string) error {
	// First, check if username or email already exists
	var count int
	err := DB.QueryRow(`
		SELECT COUNT(*) 
		FROM users 
		WHERE username = ? OR email = ?
	`, username, email).Scan(&count)

	if err != nil {
		return fmt.Errorf("failed to check existing user: %v", err)
	}

	if count > 0 {
		return fmt.Errorf("UNIQUE constraint failed: username or email already exists")
	}

	// If no existing user, insert the new user
	_, err = DB.Exec(`
		INSERT INTO users (username, email, password, created_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, username, email, password)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("UNIQUE constraint failed: username or email already exists")
		}
		return fmt.Errorf("failed to create user: %v", err)
	}

	return nil
}

// GetUserCreatedQuizzes retrieves all quizzes created by a user
func GetUserCreatedQuizzes(userID int) ([]Quiz, error) {
	log.Printf("Fetching quizzes for user ID: %d", userID)
	rows, err := DB.Query(`
		SELECT id, title
		FROM quizzes
		WHERE created_by = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		log.Printf("Error fetching quizzes: %v", err)
		return nil, err
	}
	defer rows.Close()

	var quizzes []Quiz
	for rows.Next() {
		var quiz Quiz
		if err := rows.Scan(&quiz.ID, &quiz.Title); err != nil {
			log.Printf("Error scanning quiz: %v", err)
			continue
		}
		quizzes = append(quizzes, quiz)
	}
	log.Printf("Found %d quizzes", len(quizzes))
	return quizzes, nil
}

// GetQuizWithQuestions retrieves a quiz and its questions
func GetQuizWithQuestions(quizID string) (*Quiz, error) {
	var quiz Quiz
	err := DB.QueryRow(`
		SELECT id, title 
		FROM quizzes 
		WHERE id = ?
	`, quizID).Scan(&quiz.ID, &quiz.Title)
	if err != nil {
		return nil, fmt.Errorf("failed to get quiz: %v", err)
	}

	rows, err := DB.Query(`
		SELECT id, text, options, answer 
		FROM questions 
		WHERE quiz_id = ? 
		ORDER BY id
	`, quizID)
	if err != nil {
		return nil, fmt.Errorf("failed to get questions: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var q Question
		var optionsStr string
		err := rows.Scan(&q.ID, &q.Text, &optionsStr, &q.Answer)
		if err != nil {
			log.Printf("Error scanning question: %v", err)
			continue
		}
		q.Options = strings.Split(optionsStr, "|")
		quiz.Questions = append(quiz.Questions, q)
	}

	return &quiz, nil
}

// SaveQuizScore saves or updates a user's quiz score
func SaveQuizScore(userID, quizID int, score float64) error {
	log.Printf("Saving score for user %d, quiz %d: %.2f", userID, quizID, score)
	result, err := DB.Exec(`
		INSERT INTO quiz_results (user_id, quiz_id, score)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, quiz_id) 
		DO UPDATE SET score = CASE WHEN score < ? THEN ? ELSE score END
	`, userID, quizID, score, score, score)
	if err != nil {
		log.Printf("Error saving score: %v", err)
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
	} else {
		log.Printf("Score saved successfully. Rows affected: %d", rows)
	}
	return nil
}

// GetUserQuizzes retrieves all quizzes with user scores
func GetUserQuizzes(userID int) ([]QuizWithScore, error) {
	rows, err := DB.Query(`
		WITH UserScores AS (
			SELECT quiz_id, score,
				   RANK() OVER (PARTITION BY quiz_id ORDER BY score DESC) as rank
			FROM quiz_results
		)
		SELECT 
			q.id, 
			q.title,
			COALESCE(qr.score, 0) as user_score,
			COALESCE(us.rank, 0) as rank,
			COUNT(DISTINCT qr2.user_id) as total_attempts,
			MAX(qr2.score) as high_score
		FROM quizzes q
		LEFT JOIN quiz_results qr ON q.id = qr.quiz_id AND qr.user_id = ?
		LEFT JOIN UserScores us ON q.id = us.quiz_id AND qr.score = us.score
		LEFT JOIN quiz_results qr2 ON q.id = qr2.quiz_id
		GROUP BY q.id, q.title, qr.score, us.rank
		ORDER BY q.created_at DESC
	`, userID)
	if err != nil {
		log.Printf("Error fetching quizzes: %v", err)
		return nil, err
	}
	defer rows.Close()

	var quizzes []QuizWithScore
	for rows.Next() {
		var quiz QuizWithScore
		var totalAttempts int
		var highScore float64
		err := rows.Scan(
			&quiz.ID,
			&quiz.Title,
			&quiz.UserScore,
			&quiz.Rank,
			&totalAttempts,
			&highScore,
		)
		if err != nil {
			log.Printf("Error scanning quiz: %v", err)
			continue
		}
		quiz.TotalAttempts = totalAttempts
		quiz.HighScore = highScore
		quizzes = append(quizzes, quiz)
	}

	return quizzes, nil
}

// GetTopScores retrieves the top scoring users
func GetTopScores(limit int) ([]TopScore, error) {
	rows, err := DB.Query(`
		WITH UserScores AS (
			SELECT 
				u.username,
				AVG(qr.score) as avg_score,
				RANK() OVER (ORDER BY AVG(qr.score) DESC) as rank
			FROM users u
			JOIN quiz_results qr ON u.id = qr.user_id
			GROUP BY u.id, u.username
		)
		SELECT rank, username, avg_score
		FROM UserScores
		WHERE rank <= ?
		ORDER BY rank
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top scores: %v", err)
	}
	defer rows.Close()

	var scores []TopScore
	for rows.Next() {
		var score TopScore
		err := rows.Scan(&score.Rank, &score.Username, &score.Score)
		if err != nil {
			log.Printf("Error scanning top score: %v", err)
			continue
		}
		scores = append(scores, score)
	}

	return scores, nil
}

// GetLeaderboard retrieves the leaderboard data
func GetLeaderboard() ([]LeaderboardEntry, error) {
	rows, err := DB.Query(`
		SELECT 
			u.username,
			q.title as quiz_name,
			qr.score,
			RANK() OVER (PARTITION BY qr.quiz_id ORDER BY qr.score DESC) as rank
		FROM quiz_results qr
		JOIN users u ON qr.user_id = u.id
		JOIN quizzes q ON qr.quiz_id = q.id
		ORDER BY qr.quiz_id, rank
		LIMIT 50
	`)
	if err != nil {
		log.Printf("Error fetching leaderboard: %v", err)
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		err := rows.Scan(&entry.Username, &entry.QuizName, &entry.Score, &entry.Rank)
		if err != nil {
			log.Printf("Error scanning leaderboard entry: %v", err)
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Add this function to properly handle GitHub users
func GetOrCreateGithubUser(githubUser *models.GithubUser) (*User, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	var user User
	err = tx.QueryRow(`
		SELECT id, username, email 
		FROM users 
		WHERE email = ? OR username = ?
	`, githubUser.Email, githubUser.Login).Scan(&user.ID, &user.Username, &user.Email)

	if err == sql.ErrNoRows {
		// Create new user
		result, err := tx.Exec(`
			INSERT INTO users (username, email, password)
			VALUES (?, ?, ?)
		`, githubUser.Login, githubUser.Email, "github-auth")
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %v", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get user id: %v", err)
		}

		user = User{
			ID:       int(id),
			Username: githubUser.Login,
			Email:    githubUser.Email,
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to query user: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return &user, nil
}

func GetAvailableQuizzes(userID int) ([]QuizWithScore, error) {
	rows, err := DB.Query(`
		SELECT 
			q.id,
			q.title,
			COALESCE(qr.score, 0) as user_score,
			COALESCE(r.rank, 0) as rank,
			COALESCE(a.attempts, 0) as total_attempts,
			COALESCE(hs.high_score, 0) as high_score
		FROM quizzes q
		LEFT JOIN quiz_results qr ON q.id = qr.quiz_id AND qr.user_id = ?
		LEFT JOIN (
			SELECT quiz_id, user_id, RANK() OVER (PARTITION BY quiz_id ORDER BY score DESC) as rank
			FROM quiz_results
		) r ON q.id = r.quiz_id AND r.user_id = ?
		LEFT JOIN (
			SELECT quiz_id, COUNT(*) as attempts
			FROM quiz_results
			GROUP BY quiz_id
		) a ON q.id = a.quiz_id
		LEFT JOIN (
			SELECT quiz_id, MAX(score) as high_score
			FROM quiz_results
			GROUP BY quiz_id
		) hs ON q.id = hs.quiz_id
		ORDER BY q.id DESC
	`, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query quizzes: %v", err)
	}
	defer rows.Close()

	var quizzes []QuizWithScore
	for rows.Next() {
		var quiz QuizWithScore
		err := rows.Scan(
			&quiz.ID,
			&quiz.Title,
			&quiz.UserScore,
			&quiz.Rank,
			&quiz.TotalAttempts,
			&quiz.HighScore,
		)
		if err != nil {
			log.Printf("Error scanning quiz: %v", err)
			continue
		}
		quizzes = append(quizzes, quiz)
	}

	return quizzes, nil
}

// Add these functions at the bottom of the file
func GetUserByID(userID int) (*User, error) {
	var user User
	err := DB.QueryRow(`
		SELECT id, username, email 
		FROM users 
		WHERE id = ?
	`, userID).Scan(&user.ID, &user.Username, &user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}
	return &user, nil
}

func GetUserStats(userID int) (*UserStats, error) {
	var stats UserStats

	// Get number of quizzes taken and average score
	err := DB.QueryRow(`
		SELECT COUNT(DISTINCT quiz_id) as quizzes_taken,
			   COALESCE(AVG(score), 0) as average_score
		FROM quiz_results
		WHERE user_id = ?
	`, userID).Scan(&stats.QuizzesTaken, &stats.AverageScore)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %v", err)
	}

	// Get global rank based on average score
	err = DB.QueryRow(`
		WITH UserRanks AS (
			SELECT user_id,
				   AVG(score) as avg_score,
				   RANK() OVER (ORDER BY AVG(score) DESC) as rank
			FROM quiz_results
			GROUP BY user_id
		)
		SELECT COALESCE(rank, 0)
		FROM UserRanks
		WHERE user_id = ?
	`, userID).Scan(&stats.GlobalRank)
	if err != nil {
		if err == sql.ErrNoRows {
			stats.GlobalRank = 0
		} else {
			return nil, fmt.Errorf("failed to get global rank: %v", err)
		}
	}

	return &stats, nil
}
