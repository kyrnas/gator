package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/kyrnas/gator/internal/command"
	"github.com/kyrnas/gator/internal/config"
	"github.com/kyrnas/gator/internal/database"
	"github.com/kyrnas/gator/internal/rss"

	_ "github.com/lib/pq"
)

var (
	comms command.Commands
	s     config.State
)

func middlewareLoggedIn(handler func(s *config.State, cmd command.Command, user database.User) error) func(*config.State, command.Command) error {
	user, err := s.Queries.GetUser(context.Background(), s.Conf.CurrentUserName)
	if err != nil {
		return nil
	}
	return func(s *config.State, cmd command.Command) error {
		return handler(s, cmd, user)
	}
}

func scrapeFeeds(s *config.State, user database.User) {
	feed, err := s.Queries.GetNextFeedToFetch(context.Background(), user.ID)
	if err != nil {
		fmt.Println("Error getting next feed to fetch:", err)
		return
	}
	_, err = s.Queries.MarkFeedFetched(
		context.Background(), 
		database.MarkFeedFetchedParams{
			LastFetchedAt: sql.NullTime{
				Time: time.Now(),
				Valid: true,
			},
			ID: feed.ID,
		},
	)
	if err != nil {
		fmt.Println("Error marking feed as fetched:", err)
		return
	}
	rssfeed, err := rss.FetchFeed(context.Background(), feed.Url)
	if err != nil {
		fmt.Println("Error marking feed as fetched:", err)
		return
	}
	for _, item := range rssfeed.Channel.Item {
		fmt.Println(item.Title)
	}
}

func handlerHelp(s *config.State, cmd command.Command) error {
	fmt.Println("Available commands: ")
	for name := range comms.NameToFunc {
		fmt.Println(" -", name)
	}
	fmt.Println("")
	return nil
}

func handlerLogin(s *config.State, cmd command.Command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("unexpected number of args for login command. Expected: 1. Received: %d", len(cmd.Args))
	}
	user, err := s.Queries.GetUser(context.Background(), cmd.Args[0])
	if err != nil {
		return err
	}
	err = s.Conf.SetUser(user.Name)
	if err != nil {
		return err
	}
	fmt.Println("Set User:", cmd.Args[0])
	return nil
}

func handlerRegister(s *config.State, cmd command.Command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("unexpected number of args for register command. Expected: 1. Received: %d", len(cmd.Args))
	}
	user, err := s.Queries.CreateUser(context.Background(), database.CreateUserParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: cmd.Args[0]})
	if err != nil {
		return err
	}
	fmt.Println("created user:", user)
	err = s.Conf.SetUser(cmd.Args[0])
	if err != nil {
		return err
	}
	return nil
}

func handlerReset(s *config.State, cmd command.Command) error {
	err := s.Queries.DropUsers(context.Background())
	if err != nil {
		return err
	}
	err = s.Conf.SetUser("")
	if err != nil {
		return err
	}
	return nil
}

func handlerUsers(s *config.State, cmd command.Command) error {
	users, err := s.Queries.GetUsers(context.Background())
	if err != nil {
		return err
	}
	for _, user := range users {
		if user.Name == s.Conf.CurrentUserName {
			fmt.Println("*", user.Name, "(current)")
		} else {
			fmt.Println("*", user.Name)
		}
	}
	return nil
}

func handlerAgg(s *config.State, cmd command.Command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("unexpected number of args for agg command. Expected: 1. Received: %d", len(cmd.Args))
	}
	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return err
	}
	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		scrapeFeeds(s, user)
	}
}

func hanlderAddFeed(s *config.State, cmd command.Command, user database.User) error {
	if len(cmd.Args) != 2 {
		return fmt.Errorf("unexpected number of args for addfeed command. Expected: 2. Received: %d", len(cmd.Args))
	}
	feed, err := s.Queries.CreateFeed(
		context.Background(),
		database.CreateFeedParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      cmd.Args[0],
			Url:       cmd.Args[1],
			UserID:    user.ID,
		},
	)
	if err != nil {
		return err
	}
	fmt.Println("created feed:", feed)
	err = hanlderFollow(s, command.Command{
		Name: "follow",
		Args: cmd.Args[1:],
	}, user)
	if err != nil {
		return err
	}
	return nil
}

func hanlderFeeds(s *config.State, cmd command.Command) error {
	feeds, err := s.Queries.GetFeeds(context.Background())
	if err != nil {
		return err
	}
	for _, feed := range feeds {
		fmt.Println(feed.Name, ":", feed.Url)
	}
	return nil
}

func hanlderFollow(s *config.State, cmd command.Command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("unexpected number of args for follow command. Expected: 1. Received: %d", len(cmd.Args))
	}
	feed, err := s.Queries.GetFeed(context.Background(), cmd.Args[0])
	if err != nil {
		return err
	}
	res, err := s.Queries.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    user.ID,
			FeedID:    feed.ID,
		},
	)
	if err != nil {
		return err
	}
	fmt.Println("created feed follow:", res.UserName, "-", res.FeedName)
	return nil
}

func hanlderFollowing(s *config.State, cmd command.Command, user database.User) error {
	feed, err := s.Queries.GetUsersFeedFollows(context.Background(), user.Name)
	if err != nil {
		return err
	}
	for _, f := range feed {
		fmt.Println(f.FeedName, "-", f.FeedUrl)
	}
	return nil
}

func hanlderUnfollow(s *config.State, cmd command.Command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("unexpected number of args for follow command. Expected: 1. Received: %d", len(cmd.Args))
	}
	feed, err := s.Queries.DeleteUsersFeedFollowsByUrl(
		context.Background(), 
		database.DeleteUsersFeedFollowsByUrlParams{
			Name: user.Name,
			Url: cmd.Args[0],
		},
	)
	if err != nil {
		return err
	}
	for _, f := range feed {
		fmt.Println(f.FeedID)
	}
	return nil
}

func initialize() {
	// initialize config from json file
	conf, err := config.Read()
	if err != nil {
		fmt.Println("ERROR: Unable to load config")
		panic(err)
	}

	// create a db connection
	db, err := sql.Open("postgres", conf.DbUrl)
	if err != nil {
		fmt.Println("ERROR: Unable to connect to db")
		panic(err)
	}
	dbQueries := database.New(db)

	// create a global state struct
	s = config.State{
		Conf:    &conf,
		Queries: dbQueries,
	}

	// initialize the map of commands and register known commands
	comms = command.Commands{
		NameToFunc: make(map[string]func(*config.State, command.Command) error),
	}
	comms.Register("help", handlerHelp)
	comms.Register("login", handlerLogin)
	comms.Register("register", handlerRegister)
	comms.Register("reset", handlerReset)
	comms.Register("users", handlerUsers)
	comms.Register("agg", middlewareLoggedIn(handlerAgg))
	comms.Register("addfeed", middlewareLoggedIn(hanlderAddFeed))
	comms.Register("feeds", hanlderFeeds)
	comms.Register("follow", middlewareLoggedIn(hanlderFollow))
	comms.Register("following", middlewareLoggedIn(hanlderFollowing))
	comms.Register("unfollow", middlewareLoggedIn(hanlderUnfollow))
}

func main() {
	initialize()
	args := os.Args
	if len(args) < 2 {
		handlerHelp(&s, command.Command{})
		os.Exit(1)
	}
	cmd := command.Command{
		Name: args[1],
		Args: args[2:],
	}
	err := comms.Run(&s, cmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
