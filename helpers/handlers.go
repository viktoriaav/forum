package helpers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var tmpl *template.Template

func init() {
	tmpl = template.Must(template.ParseGlob("frontend/*.html"))
}

type HeaderData struct {
	LoggedInUser string
}

func IndexHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Retrieve the filter parameters from the query string
	filter := r.URL.Query().Get("filter")
	category := r.URL.Query().Get("category")

	if r.URL.Path != "/" {
		errorHandler(w, "Page not found", 404)
		return
	}

	categories, err := GetCategories(db)
	if err != nil {
		errorHandler(w, "Internal Server Error", 500)
		return
	}

	var posts []Post
	loggedInUsername, _ := GetLoggedInUsername(r, db)

	if filter == "my-likes" {
		posts, err = GetUserLikedPosts(db, loggedInUsername)
	} else if filter == "my-posts" {
		posts, err = GetUserCreatedPosts(db, loggedInUsername)
	} else if category != "" {
		posts, err = GetPostsByCategory(db, category)
	} else {
		posts, err = GetPosts(db)
	}

	if err != nil {
		errorHandler(w, "Internal Server Error", 500)
		return
	}

	headerData := HeaderData{
		LoggedInUser: loggedInUsername,
	}

	data := struct {
		Categories []Category
		Posts      []Post
		Header     HeaderData
	}{
		Categories: categories,
		Posts:      posts,
		Header:     headerData,
	}

	err = tmpl.ExecuteTemplate(w, "index", data)
	if err != nil {
		errorHandler(w, err.Error(), 500)
		return
	}
}

func errorHandler(w http.ResponseWriter, msg string, code int) error {
	tmpl, err := template.ParseFiles("frontend/error.html")
	if err != nil {
		return err
	}
	errorMessage := struct {
		Message string
		Code    int
	}{
		Message: msg,
		Code:    code,
	}
	return tmpl.Execute(w, errorMessage)
}
func CreatePostPageHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	categories, err := GetCategories(db)
	if err != nil {
		return
	}
	loggedInUsername, _ := GetLoggedInUsername(r, db) // Retrieve the logged-in username
	headerData := HeaderData{
		LoggedInUser: loggedInUsername,
	}
	data := struct {
		Categories []Category
		Header     HeaderData
	}{
		Categories: categories,
		Header:     headerData,
	}

	tmpl.ExecuteTemplate(w, "create-post", data)
}

func AddPostHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		errorHandler(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, err := GetLoggedInUsername(r, db)
	if err != nil {
		http.Error(w, "Session error", http.StatusUnauthorized)
		return
	}

	err = r.ParseForm()
	if err != nil {
		http.Error(w, "Form parsing error", http.StatusInternalServerError)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")
	categories := r.Form["categories[]"]

	if len(categories) == 0 {
		http.Error(w, "At least one category must be selected", http.StatusBadRequest)
		return
	}

	var userID int
	err = db.QueryRow("SELECT user_ID FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Println("Database error:", err)
		return
	}

	insertQuery := "INSERT INTO posts (user_ID, title, content, created_at) VALUES (?, ?, ?, ?)"
	createdAt := time.Now()

	result, err := db.Exec(insertQuery, userID, title, content, createdAt)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Println("Database error:", err)
		return
	}

	postID, _ := result.LastInsertId()

	for _, selectedCategory := range categories {
		insertCategoryQuery := "INSERT INTO post_categories (post_ID, category_ID) VALUES (?, (SELECT category_ID FROM categories WHERE category = ?))"
		_, err = db.Exec(insertCategoryQuery, postID, selectedCategory)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			log.Println("Database error:", err)
			return
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		errorHandler(w, "Method not allowed", 405)
		return
	}

	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Convert email and username to lowercase
	lowercaseEmail := strings.ToLower(email)
	lowercaseUsername := strings.ToLower(username)

	// Check if the user already exists in the database
	var existingUser int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE LOWER(email) = ? OR LOWER(username) = ?", lowercaseEmail, lowercaseUsername).Scan(&existingUser)
	if err != nil {
		errorHandler(w, "Database error", 500)
		log.Println("Database error:", err)
		return
	}

	if existingUser > 0 {
		errorHandler(w, "User already exists", 409)
		return
	}
	createdAt := time.Now()
	// Hash the password before storing it
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		errorHandler(w, "Password hashing error", 500)
		log.Println("Password hashing error:", err)
		return
	}

	// Continue with user registration
	query := "INSERT INTO users (email, username, password, created_at) VALUES (?, ?, ?, ?)"
	_, err = db.Exec(query, lowercaseEmail, lowercaseUsername, hashedPassword, createdAt)
	if err != nil {
		errorHandler(w, "Database error", 500)
		log.Println("Database error:", err)
		return
	}
	// Get the user's ID based on the username from the database
	var userID int
	err = db.QueryRow("SELECT user_ID FROM users WHERE username = ?", lowercaseUsername).Scan(&userID)
	if err != nil {
		errorHandler(w, "Database error", 500)
		log.Println("Database error:", err)
		return
	}

	// Generate a session token
	token := GenerateSessionToken()

	// Calculate session duration (same as in the LoginHandler)
	sessionHours := 1
	sessionMinutes := 30
	sessionSeconds := 0
	sessionDuration := time.Duration(sessionHours)*time.Hour + time.Duration(sessionMinutes)*time.Minute + time.Duration(sessionSeconds)*time.Second

	// Calculate expiration time
	expirationTime := time.Now().Add(sessionDuration)

	// Create a new session record in the database
	err = createSession(userID, token, expirationTime, db)
	if err != nil {
		errorHandler(w, "Database error", http.StatusInternalServerError)
		log.Println("Database error:", err)
		return
	}

	// Set the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   token,
		Expires: expirationTime,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func GenerateSessionToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func createSession(userID int, token string, expirationTime time.Time, db *sql.DB) error {
	// Delete expired sessions before creating a new one
	err := DeleteExpiredSessions(db)
	if err != nil {
		return err
	}

	// Check if an active session already exists for the user
	var existingSessionID int
	query := "SELECT session_ID FROM sessions WHERE user_ID = ? AND expires_at > ?"
	err = db.QueryRow(query, userID, time.Now()).Scan(&existingSessionID)

	if err == sql.ErrNoRows { // No active session found, create a new one
		insertQuery := "INSERT INTO sessions (token, user_ID, created_at, expires_at) VALUES (?, ?, ?, ?)"
		_, err = db.Exec(insertQuery, token, userID, time.Now(), expirationTime)
		return err
	} else if err != nil {
		return err
	}

	// Update the existing session with new token and expiration time
	updateQuery := "UPDATE sessions SET token = ?, expires_at = ? WHERE session_ID = ?"
	_, err = db.Exec(updateQuery, token, expirationTime, existingSessionID)
	return nil
}

func LoginHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var userID int
	var hashedPassword []byte // To store the hashed password from the database
	query := "SELECT user_ID, password FROM users WHERE username = ?"
	err := db.QueryRow(query, username).Scan(&userID, &hashedPassword)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Compare the hashed password with the provided password
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate a session token
	token := GenerateSessionToken()

	// Calculate session duration
	sessionHours := 1
	sessionMinutes := 30
	sessionSeconds := 0
	sessionDuration := time.Duration(sessionHours)*time.Hour + time.Duration(sessionMinutes)*time.Minute + time.Duration(sessionSeconds)*time.Second

	// Calculate expiration time
	expirationTime := time.Now().Add(sessionDuration)

	// Create a new session record in the database
	err = createSession(userID, token, expirationTime, db)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Println("Database error:", err)
		return
	}

	// Set the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   token,
		Expires: expirationTime,
	})

	// Redirect to the main page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Get the session token from the cookie
	cookie, err := r.Cookie("session_token")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Delete the session from the sessions table
	deleteQuery := "DELETE FROM sessions WHERE token = ?"
	_, err = db.Exec(deleteQuery, cookie.Value)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Println("Database error:", err)
		return
	}

	// Expire the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})

	// Redirect to the main page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func PostHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Get the postID from the request URL or form data
	postIDStr := strings.TrimPrefix(r.URL.Path, "/post/")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		errorHandler(w, "Invalid post ID", 400)
		return
	}

	// Assuming you have a database connection variable db
	posts, err := GetPosts(db, postID)
	if err != nil {
		fmt.Print("1", err)
		errorHandler(w, "Error retrieving post", 500)
		return
	}

	if len(posts) == 0 {
		errorHandler(w, "We cannot find this post", 404)
		return
	}

	post := posts[0]

	// Get comments for the selected post
	comments, err := GetCommentsForPost(db, post.PostID)
	if err != nil {
		errorHandler(w, "Error retrieving comments", 500)
		return
	}
	loggedInUsername, _ := GetLoggedInUsername(r, db) // Retrieve the logged-in username
	headerData := HeaderData{
		LoggedInUser: loggedInUsername,
	}

	// Create a data structure to pass to the template
	data := struct {
		Post     Post
		Comments []Comment
		Header   HeaderData
	}{
		Post:     post,
		Comments: comments,
		Header:   headerData,
	}

	err = tmpl.ExecuteTemplate(w, "post", data) // Use the "post" template
	if err != nil {
		errorHandler(w, "Internal server error", 500)
		return
	}
}
func SubmitCommentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	username, err := GetLoggedInUsername(r, db)
	if err != nil {
		// Handle unauthenticated user.
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get the user's ID based on the username.
	userID, err := GetUserIDByUsername(username, db)
	if err != nil {
		// Handle error.
		http.Error(w, "Failed to get user ID", http.StatusInternalServerError)
		return
	}

	// Extract the comment and postID from the form data.
	comment := r.FormValue("comment")
	postIDStr := r.FormValue("postID")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		// Handle invalid postID.
		http.Error(w, "Invalid postID", http.StatusBadRequest)
		return
	}

	// Insert the comment into the database using the user's ID.
	_, err = db.Exec("INSERT INTO comments (post_ID, user_ID, content, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)", postID, userID, comment)
	if err != nil {
		// Handle the error.
		fmt.Print(err)
		http.Error(w, "Failed to submit comment", http.StatusInternalServerError)
		return
	}

	// Redirect back to the post page or update the comments section via AJAX.
	http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
}

func UpdateReactionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	username, err := GetLoggedInUsername(r, db)
	if err != nil {
		// Handle unauthenticated user.
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Get the user's ID based on the username.
	loggedInUserID, err := GetUserIDByUsername(username, db)
	if err != nil {
		// Handle error.
		http.Error(w, "Failed to get user ID", http.StatusInternalServerError)
		return
	}
	reactionTypeStr := r.FormValue("action")
	if reactionTypeStr == "" {
		http.Error(w, "Reaction type not provided", http.StatusBadRequest)
		return
	}
	reactionType, err := strconv.Atoi(reactionTypeStr)
	if err != nil || (reactionType != 0 && reactionType != 1) {
		fmt.Print(err)
		http.Error(w, "Invalid reaction type", http.StatusBadRequest)
		return
	}
	targetType := r.FormValue("targetType") // "post" or "comment"
	if targetType != "post" && targetType != "comment" {
		http.Error(w, "Invalid target type", http.StatusBadRequest)
		return
	}
	targetIDStr := r.FormValue("targetID")
	targetID, err := strconv.Atoi(targetIDStr)
	if err != nil || targetID <= 0 {
		http.Error(w, "Invalid target ID", http.StatusBadRequest)
		return
	}
	var tableName string
	var targetColumn string
	if targetType == "post" {
		tableName = "likes"
		targetColumn = "post_ID"
	} else {
		tableName = "likes"
		targetColumn = "comment_ID"
	}
	// Check if the user has already liked or disliked this target (post or comment)
	var existingReactionType int
	err = db.QueryRow("SELECT type FROM "+tableName+" WHERE "+targetColumn+" = ? AND user_ID = ?", targetID, loggedInUserID).Scan(&existingReactionType)
	if err != nil && err != sql.ErrNoRows {
		fmt.Print(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if err == nil {
		// User has already reacted, update the reaction type
		_, err = db.Exec("UPDATE likes SET type = ? WHERE ("+targetType+"_ID = ?) AND user_ID = ?", reactionType, targetID, loggedInUserID)
		if err == nil {
			// Update the likes count in the corresponding target entry (post or comment)
			_, err = db.Exec("UPDATE "+targetType+"s SET "+targetType+"s = (SELECT COUNT(*) FROM likes WHERE "+targetType+"_ID = ? AND type = 0) WHERE "+targetType+"_ID = ?", targetID, targetID)
		}
	} else {
		// User hasn't reacted yet, insert a new reaction
		_, err = db.Exec("INSERT INTO likes ("+targetType+"_ID, user_ID, type) VALUES (?, ?, ?)", targetID, loggedInUserID, reactionType)
		if err == nil {
			// Update the likes count in the corresponding target entry (post or comment)
			_, err = db.Exec("UPDATE "+targetType+"s SET "+targetType+"s = (SELECT COUNT(*) FROM likes WHERE "+targetType+"_ID = ? AND type = 0) WHERE "+targetType+"_ID = ?", targetID, targetID)
		}
	}
	// Redirect back to the same page to refresh the content
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func DeleteExpiredSessions(db *sql.DB) error {
	deleteQuery := "DELETE FROM sessions WHERE expires_at <= ?"
	_, err := db.Exec(deleteQuery, time.Now())
	return err
}
