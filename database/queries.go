package database

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const createtables string = `
CREATE TABLE IF NOT EXISTS categories (
	category_ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL ,
	category TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS users (
	user_ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL ,
	email TEXT DEFAULT NULL ,
	username TEXT NOT NULL UNIQUE ,
	password TEXT DEFAULT NULL ,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS posts (
	post_ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL ,
	user_ID INTEGER NOT NULL ,
	title TEXT NOT NULL ,
	content TEXT NOT NULL ,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ,
	FOREIGN KEY(user_ID) REFERENCES users(user_ID)
);
CREATE TABLE IF NOT EXISTS comments (
	comment_ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL ,
	post_ID INTEGER NOT NULL ,
	user_ID INTEGER NOT NULL ,
	content TEXT NOT NULL ,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(post_ID) REFERENCES posts(post_ID),
	FOREIGN KEY(user_ID) REFERENCES users(user_ID)
);
CREATE TABLE IF NOT EXISTS sessions (
	session_ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL ,
	token TEXT NOT NULL ,
	user_ID INTEGER NOT NULL ,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ,
	expires_at INTEGER NOT NULL ,
	FOREIGN KEY(user_ID) REFERENCES users(user_ID)
);
CREATE TABLE IF NOT EXISTS likes ( 
    like_ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL ,
    post_ID INTEGER , 
    comment_ID INTEGER , 
    user_ID INTEGER NOT NULL , 
    type INTEGER NOT NULL , 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP , 
    FOREIGN KEY(post_ID) REFERENCES posts(post_ID) , 
    FOREIGN KEY(comment_id) REFERENCES comments(comment_id) , 
    FOREIGN KEY(user_ID) REFERENCES users(user_ID)
);
CREATE TABLE IF NOT EXISTS post_categories (
    post_ID INTEGER NOT NULL, 
	category_ID INTEGER NOT NULL, 
    FOREIGN KEY(post_ID) REFERENCES posts(post_ID) , 
    FOREIGN KEY(category_ID) REFERENCES categories(category_ID)
);`

func OpenDB() (*sql.DB, error) {
	dbPath := "./database/database.db"

	// Check if the database file exists
	if _, err := os.Stat(dbPath); errors.Is(err, os.ErrNotExist) {
		// Open a new database connection
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return nil, err
		}

		// Tables
		if _, err := db.Exec(createtables); err != nil {
			return nil, err
		}

		paths := []string{"./database/sql/fill_tables.sql"}
		for _, path := range paths {
			sqlFile, err := os.ReadFile(path)
			if err != nil {
				log.Println("Error reading file:", err)
				continue
			}

			queries := strings.Split(string(sqlFile), ";")
			for _, query := range queries {
				query = strings.TrimSpace(query)
				if query == "" {
					continue
				}

				tx, err := db.Begin()
				if err != nil {
					log.Println("Error starting transaction:", err)
					continue
				}

				_, err = tx.Exec(query)
				if err != nil {
					log.Println("Error executing query:", err)
					tx.Rollback()
					continue
				}

				err = tx.Commit()
				if err != nil {
					log.Println("Error committing transaction:", err)
					continue
				}
			}
		}
		return db, nil
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	return db, nil
}
