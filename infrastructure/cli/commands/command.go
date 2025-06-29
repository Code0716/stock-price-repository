package commands

import "github.com/urfave/cli/v2"

type Command struct {
	// The name of the command
	Name string
	// A list of aliases for the command
	Aliases []string
	// A short description of the usage of this command
	Usage string
	// Custom text to show on USAGE section of help
	UsageText string
	// The function to call when this command is invoked
	Action cli.ActionFunc
	// List of flags to parse
	Flags []cli.Flag
	// List of child commands
	Subcommands []*Command
}

func (c *Command) CliCommand() *cli.Command {
	cc := &cli.Command{
		Name:      c.Name,
		Aliases:   c.Aliases,
		Usage:     c.Usage,
		UsageText: c.UsageText,
		Action:    c.Action,
		Flags:     c.Flags,
	}
	subCommands := make([]*cli.Command, len(c.Subcommands))
	for i, subCommand := range c.Subcommands {
		subCommands[i] = subCommand.CliCommand()
	}
	cc.Subcommands = subCommands
	return cc
}
