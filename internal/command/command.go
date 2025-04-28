package command

import (
	"fmt"

	"github.com/kyrnas/gator/internal/config"
)



type Command struct {
	Name string
	Args []string
}

type Commands struct {
	NameToFunc map[string]func(*config.State, Command) error
}

func (comm *Commands) Run(s *config.State, cmd Command) error {
	f, exists := comm.NameToFunc[cmd.Name]
	if !exists {
		return fmt.Errorf("Unknown command: %s", cmd.Name)
	}
	err := f(s, cmd)
	return err
}

func (comm *Commands) Register(name string, f func(*config.State, Command) error) {
	comm.NameToFunc[name] = f
}
