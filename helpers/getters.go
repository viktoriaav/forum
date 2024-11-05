package helpers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// CATEGORIES
type Category struct {
	CategoryID int
	Category   string
}

func GetCategories(db *sql.DB) ([]Category, error) {
	if db == nil {
		return nil, errors.New("nil database connection")
	}
	var categories []Category // Declare the slice here
	query := "SELECT category FROM categories"
	rows, err := db.Query(query)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var category Category
		err := rows.Scan(&category.Category)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, nil
}

// POSTS
type Post struct {
	PostID       int
	Username     string
	Title        string
	Content      string
	Categories   []Category // Modify this field
	PostCategory string
	CreatedAt    string
	Likes        int
	Dislikes     int
	CommentCount int
}

func GetPosts(db *sql.DB, postID ...int) ([]Post, error) {
	if db == nil {
		return nil, errors.New("nil database connection")
	}
	var posts []Post
	query := `
		SELECT p.post_ID, u.username, p.title, p.content, p.created_at,
			   c.category,
			   COALESCE(SUM(CASE WHEN l.type = 0 THEN 1 ELSE 0 END), 0) AS likes,
			   COALESCE(SUM(CASE WHEN l.type = 1 THEN 1 ELSE 0 END), 0) AS dislikes,
			   COALESCE(COUNT(com.comment_ID), 0) AS comment_count
		FROM posts AS p
		INNER JOIN users AS u ON p.user_ID = u.user_ID
		INNER JOIN post_categories AS pc ON p.post_ID = pc.post_ID
		INNER JOIN categories AS c ON pc.category_ID = c.category_ID
		LEFT JOIN likes AS l ON p.post_ID = l.post_ID
		LEFT JOIN comments AS com ON p.post_ID = com.post_ID
	`

	if len(postID) > 0 {
		// If postID is provided, fetch only that post
		query += "WHERE p.post_ID = ?"
		rows, err := db.Query(query, postID[0])
		if err != nil {
			fmt.Println("Error executing query:", err)
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var post Post
			err := rows.Scan(
				&post.PostID, &post.Username, &post.Title, &post.Content, &post.CreatedAt,
				&post.PostCategory, &post.Likes, &post.Dislikes, &post.CommentCount,
			)
			if err != nil {
				fmt.Println("Error scanning row:", err) // Debugging line
				return nil, err
			}

			// Calculate likes, dislikes, and comment count for each post
			likes, dislikes, commentCount, err := GetPostStats(db, post.PostID)
			if err != nil {
				return nil, err
			}

			post.Likes = likes
			post.Dislikes = dislikes
			post.CommentCount = commentCount

			// Fetch categories for the current post
			categories, err := GetCategoriesForPost(db, post.PostID)
			if err != nil {
				return nil, err
			}
			post.PostCategory = strings.Join(categories, " ") // Join the categories into a single string

			posts = append(posts, post)
		}

	} else {
		// Fetch all posts
		query += `
		GROUP BY p.post_ID, u.username, p.title, p.content, p.created_at
		`
		rows, err := db.Query(query)
		if err != nil {
			fmt.Println("Error executing query:", err)
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var post Post
			var category Category
			err := rows.Scan(
				&post.PostID, &post.Username, &post.Title, &post.Content, &post.CreatedAt,
				&category.Category, &post.Likes, &post.Dislikes, &post.CommentCount,
			)
			if err != nil {
				fmt.Println("Error scanning row:", err) // Debugging line
				return nil, err
			}
			// Calculate likes, dislikes, and comment count for each post
			likes, dislikes, commentCount, err := GetPostStats(db, post.PostID)
			if err != nil {
				return nil, err
			}

			post.Likes = likes
			post.Dislikes = dislikes
			post.CommentCount = commentCount

			// Fetch categories for the current post
			categories, err := GetCategoriesForPost(db, post.PostID)
			if err != nil {
				return nil, err
			}
			post.PostCategory = strings.Join(categories, " ") // Join the categories into a single string

			posts = append(posts, post)
		}
	}
	return posts, nil
}
func GetCategoriesForPost(db *sql.DB, postID int) ([]string, error) {
	categories := []string{}
	query := `
		SELECT c.category
		FROM categories AS c
		INNER JOIN post_categories AS pc ON c.category_ID = pc.category_ID
		WHERE pc.post_ID = ?
	`
	rows, err := db.Query(query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		err := rows.Scan(&category)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, nil
}

func GetPostStats(db *sql.DB, postID int) (int, int, int, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN l.type = 0 THEN 1 ELSE 0 END), 0) AS likes,
			COALESCE(SUM(CASE WHEN l.type = 1 THEN 1 ELSE 0 END), 0) AS dislikes,
			COALESCE(COUNT(DISTINCT com.comment_ID), 0) AS comment_count
		FROM posts AS p
		LEFT JOIN likes AS l ON p.post_ID = l.post_ID
		LEFT JOIN comments AS com ON p.post_ID = com.post_ID
		WHERE p.post_ID = ?
	`

	var likes, dislikes, commentCount int
	err := db.QueryRow(query, postID).Scan(&likes, &dislikes, &commentCount)
	if err != nil {
		return 0, 0, 0, err
	}

	return likes, dislikes, commentCount, nil
}

type Comment struct {
	CommentID int
	Username  string
	Content   string
	Likes     int
	Dislikes  int
}

func GetCommentsForPost(db *sql.DB, postID int) ([]Comment, error) {
	var comments []Comment
	query := `
		SELECT com.comment_ID, u.username, com.content
		FROM comments AS com
		INNER JOIN users AS u ON com.user_ID = u.user_ID
		WHERE com.post_ID = ?
	`
	rows, err := db.Query(query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.CommentID, &comment.Username, &comment.Content)
		if err != nil {
			return nil, err
		}
		// Calculate likes, dislikes count for each comment
		likes, dislikes, err := GetCommentStats(db, comment.CommentID)
		if err != nil {
			return nil, err
		}

		comment.Likes = likes
		comment.Dislikes = dislikes
		comments = append(comments, comment)
	}

	return comments, nil
}

func GetCommentStats(db *sql.DB, commentID int) (int, int, error) {
	query := `
	SELECT
		COALESCE(SUM(CASE WHEN l.type = 0 THEN 1 ELSE 0 END), 0) AS likes,
		COALESCE(SUM(CASE WHEN l.type = 1 THEN 1 ELSE 0 END), 0) AS dislikes
		FROM comments AS com
		LEFT JOIN likes AS l ON com.comment_ID = l.comment_ID
		WHERE com.comment_ID = ?
	`

	var likes, dislikes int
	err := db.QueryRow(query, commentID).Scan(&likes, &dislikes)
	if err != nil {
		return 0, 0, err
	}

	return likes, dislikes, nil
}

// USERname
func GetLoggedInUsername(r *http.Request, db *sql.DB) (string, error) {
	// Retrieve the session token from the request cookies
	sessionCookie, err := r.Cookie("session_token")
	if err != nil {
		return "", err
	}
	sessionToken := sessionCookie.Value

	// Query the sessions table to find an active session with the given token
	query := `
        SELECT u.username
        FROM sessions s
        INNER JOIN users u ON s.user_ID = u.user_ID
        WHERE s.token = ? AND s.expires_at > strftime('%s', 'now')
        LIMIT 1
    `
	var username string
	err = db.QueryRow(query, sessionToken).Scan(&username)
	if err != nil {
		return "", err
	}
	return username, nil
}

// Function to get the user ID based on the username.
func GetUserIDByUsername(username string, db *sql.DB) (int, error) {
	var userID int
	query := "SELECT user_ID FROM users WHERE username = ? LIMIT 1"
	err := db.QueryRow(query, username).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func GetUserLikedPosts(db *sql.DB, username string) ([]Post, error) {
	// Fetch posts liked by the user
	query := `
        SELECT p.post_ID, u.username, p.title, p.content, p.created_at,
               c.category,
               COALESCE(SUM(CASE WHEN l.type = 0 THEN 1 ELSE 0 END), 0) AS likes,
               COALESCE(SUM(CASE WHEN l.type = 1 THEN 1 ELSE 0 END), 0) AS dislikes,
               COALESCE(COUNT(com.comment_ID), 0) AS comment_count
        FROM posts AS p
        INNER JOIN users AS u ON p.user_ID = u.user_ID
        INNER JOIN post_categories AS pc ON p.post_ID = pc.post_ID
        INNER JOIN categories AS c ON pc.category_ID = c.category_ID
        LEFT JOIN likes AS l ON p.post_ID = l.post_ID
        LEFT JOIN comments AS com ON p.post_ID = com.post_ID
        WHERE p.post_ID IN (
            SELECT post_ID FROM likes WHERE user_ID = (SELECT user_ID FROM users WHERE username = ?)
        )
        GROUP BY p.post_ID, u.username, p.title, p.content, p.created_at
    `

	rows, err := db.Query(query, username)
	if err != nil {
		fmt.Println("Error executing query:", err)
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		// Populate the Post instance based on the query result
		err := rows.Scan(
			&post.PostID, &post.Username, &post.Title, &post.Content, &post.CreatedAt,
			&post.PostCategory, &post.Likes, &post.Dislikes, &post.CommentCount,
		)
		if err != nil {
			fmt.Println("Error scanning row:", err)
			return nil, err
		}

		// Calculate likes, dislikes, and comment count for each post
		likes, dislikes, commentCount, err := GetPostStats(db, post.PostID)
		if err != nil {
			return nil, err
		}

		post.Likes = likes
		post.Dislikes = dislikes
		post.CommentCount = commentCount

		// Fetch categories for the current post
		categories, err := GetCategoriesForPost(db, post.PostID)
		if err != nil {
			return nil, err
		}
		post.PostCategory = strings.Join(categories, " ") // Join the categories into a single string

		posts = append(posts, post)
	}

	return posts, nil
}

func GetUserCreatedPosts(db *sql.DB, username string) ([]Post, error) {
	// Fetch posts created by the user
	query := `
        SELECT p.post_ID, u.username, p.title, p.content, p.created_at,
               c.category,
               COALESCE(SUM(CASE WHEN l.type = 0 THEN 1 ELSE 0 END), 0) AS likes,
               COALESCE(SUM(CASE WHEN l.type = 1 THEN 1 ELSE 0 END), 0) AS dislikes,
               COALESCE(COUNT(com.comment_ID), 0) AS comment_count
        FROM posts AS p
        INNER JOIN users AS u ON p.user_ID = u.user_ID
        INNER JOIN post_categories AS pc ON p.post_ID = pc.post_ID
        INNER JOIN categories AS c ON pc.category_ID = c.category_ID
        LEFT JOIN likes AS l ON p.post_ID = l.post_ID
        LEFT JOIN comments AS com ON p.post_ID = com.post_ID
        WHERE u.username = ?
        GROUP BY p.post_ID, u.username, p.title, p.content, p.created_at
    `

	rows, err := db.Query(query, username)
	if err != nil {
		fmt.Println("Error executing query:", err)
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		// Populate the Post instance based on the query result
		err := rows.Scan(
			&post.PostID, &post.Username, &post.Title, &post.Content, &post.CreatedAt,
			&post.PostCategory, &post.Likes, &post.Dislikes, &post.CommentCount,
		)
		if err != nil {
			fmt.Println("Error scanning row:", err)
			return nil, err
		}
		// Calculate likes, dislikes, and comment count for each post
		likes, dislikes, commentCount, err := GetPostStats(db, post.PostID)
		if err != nil {
			return nil, err
		}

		post.Likes = likes
		post.Dislikes = dislikes
		post.CommentCount = commentCount

		// Fetch categories for the current post
		categories, err := GetCategoriesForPost(db, post.PostID)
		if err != nil {
			return nil, err
		}
		post.PostCategory = strings.Join(categories, " ") // Join the categories into a single string

		posts = append(posts, post)
	}

	return posts, nil
}

func GetPostsByCategory(db *sql.DB, category string) ([]Post, error) {
	var posts []Post
	if category == "all" {
		posts, err := GetPosts(db)
		if err != nil {
			fmt.Println("Error getting posts:", err)
			return nil, err
		}
		return posts, nil
	} else {
		query := `
        SELECT p.post_ID, u.username, p.title, p.content, p.created_at,
               c.category,
               COALESCE(SUM(CASE WHEN l.type = 0 THEN 1 ELSE 0 END), 0) AS likes,
               COALESCE(SUM(CASE WHEN l.type = 1 THEN 1 ELSE 0 END), 0) AS dislikes,
               COALESCE(COUNT(com.comment_ID), 0) AS comment_count
        FROM posts AS p
        INNER JOIN users AS u ON p.user_ID = u.user_ID
        INNER JOIN post_categories AS pc ON p.post_ID = pc.post_ID
        INNER JOIN categories AS c ON pc.category_ID = c.category_ID
        LEFT JOIN likes AS l ON p.post_ID = l.post_ID
        LEFT JOIN comments AS com ON p.post_ID = com.post_ID
        WHERE c.category = ? -- Filter by the selected category
        GROUP BY p.post_ID, u.username, p.title, p.content, p.created_at, c.category
    `
		rows, err := db.Query(query, category)
		if err != nil {
			fmt.Println("Error executing query:", err)
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var post Post
			// Populate the Post instance based on the query result
			err := rows.Scan(
				&post.PostID, &post.Username, &post.Title, &post.Content, &post.CreatedAt,
				&post.PostCategory, &post.Likes, &post.Dislikes, &post.CommentCount,
			)
			if err != nil {
				fmt.Println("Error scanning row:", err)
				return nil, err
			}

			// Fetch categories for the current post
			categories, err := GetCategoriesForPost(db, post.PostID)
			if err != nil {
				return nil, err
			}
			post.PostCategory = strings.Join(categories, " ") // Join the categories into a single string

			posts = append(posts, post)
		}

		return posts, nil
	}
}
