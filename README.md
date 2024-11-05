# Forum

This project consists in creating a web forum that allows :

- communication between users (posts, comments, likes/dislikes);
- associating categories to posts (for logged-in users when creating a new post);
- liking and disliking posts and comments (logged-in users);
- filtering posts (logged-in users).

## Storing the Data

In order to store the data in this forum (like users, posts, comments, etc.) the database library SQLite is used.

SELECT, CREATE and INSERT queries are used.

## Authentication

The client is able to register as a new user on the forum, by inputting their credentials. A login session is created to access the forum and be able to add posts and comments.

Cookies are used to allow each user to have only one opened session. Each of these sessions contain an expiration date (24h). It is up to you to decide how long the cookie stays "alive". UUID is used as a session ID.

## Instructions for user registration:

- An email is required:
    - When the email is already taken, an error response is returned;
- Username is required:
    - When the username is already taken, an error response is returned;
- Password is required:
    - The password is encrypted when stored.

## Communication

In order for users to communicate between each other, they are able to create posts and comments.

- Only registered users are able to create posts and comments;
- When registered users are creating a post they can associate one or more categories (tags) to it;
- The implementation and choice of the categories (tags) was up to the developers;
- The posts and comments are visible to all users (registered or not);
- Non-registered users are only able to see posts and comments.

## Likes and Dislikes

Only registered users are able to like or dislike posts and comments.

The number of likes and dislikes are visible by all users (registered or not).

## Filter

A filter mechanism has been implemented, that will allow users to filter the displayed posts by:

- categories (tags);
- created posts;
- liked posts.

The last two are only available for registered users and must refer to the logged-in user.

## Docker

For the forum project Docker is used.

How to:

- Build the Docker image by running the following command: 
````
docker build -t your-image-name .
````

- Once the image is built, you can run a container based on the image using the following command: 
````
docker run -p 8000:8000 your-image-name
````

- The container will start, and your Go application will be accessible at http://localhost:8080 in your web browser.
- To stop and remove the image, run the following command: 
```
docker rm -f $(docker ps -a -q)
docker rmi -f $(docker images -q)
```

Make sure you have Docker installed and running on your machine before building and running the Docker image.

## Allowed Packages

- All standard Go packages are allowed;
- sqlite3;
- bcrypt;
- UUID;

No frontend libraries or frameworks like React, Angular, Vue etc. have been used.

## Usage

1. You can start the program by running the following command:
```
go run .
```
2. Open http://localhost:8000
3. To end the server:
```
CTRL + C
```


## Developers

- [Olia Priadkina/Olha_Priadkina](https://01.kood.tech/git/Olha_Priadkina)
- [Viktoriia/vavstanc](https://01.kood.tech/git/vavstanc)