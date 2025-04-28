-- name: CreateFeedFollow :one
INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING 
    *,
    (SELECT name FROM users WHERE users.id = $4) AS user_name,
    (SELECT name FROM feeds WHERE feeds.id = $5) AS feed_name;

-- name: GetUsersFeedFollows :many
SELECT users.name AS user_name, feeds.name AS feed_name, feeds.url AS feed_url FROM feed_follows
INNER JOIN users ON feed_follows.user_id = users.id
INNER JOIN feeds ON feed_follows.feed_id = feeds.id
WHERE users.name = $1;


-- name: DeleteUsersFeedFollowsByUrl :many
DELETE FROM feed_follows
WHERE 
    user_id = (SELECT id FROM users WHERE users.name = $1) 
        AND 
    feed_id = (SELECT id FROM feeds WHERE url = $2)
RETURNING *;