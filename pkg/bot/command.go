package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/google/go-github/github"
)

type command struct {
	owner     string // repo owner
	ownerType string // type of repo owner

	repo   string // repo name
	number int    // number of issue (or pullrequest)
	author string // author of issue (or pullrequest)
	user   string // command user

	cmd  string   // command name
	args []string // command arguments. optional

	event interface{} // github event
}

func (c *command) succeed() string {
	return fmt.Sprintf("%s - succeed!", c.info())
}

func (c *command) invalid() string {
	return fmt.Sprintf("%s - invalid!", c.info())
}

func (c *command) failed() string {
	return fmt.Sprintf("%s - failed!", c.info())
}

func (c *command) info() string {
	return fmt.Sprintf("[%s/%s #%d(%s)] %s: %s %s",
		c.owner, c.repo, c.number, c.author,
		c.user, c.cmd, strings.Join(c.args, " "),
	)
}

// cmdClose handles command /close
func (b *Bot) cmdClose(c *command) bool {
	// check command syntax
	if len(c.args) != 0 {
		glog.Info(c.invalid())
		return true
	}

	ctx := context.Background()

	// close command can only be used by authors or collaborators
	if c.user != c.author {
		isCollab, _, err := b.git.Repositories.IsCollaborator(ctx, c.owner, c.repo, c.user)
		if err != nil {
			glog.Errorf("%s err: %v", c.failed(), err)
			return false
		}
		if !isCollab {
			glog.Infof("%s user is neither author nor a collaborator, ignore.", c.failed())
			return true
		}
	}

	// close issue as user requested
	state := new(string)
	*state = "closed"
	if _, _, err := b.git.Issues.Edit(ctx, c.owner, c.repo, c.number, &github.IssueRequest{State: state}); err != nil {
		glog.Errorf("%s err: %v", c.failed(), err)
		return false
	}
	glog.Info(c.succeed())
	return true
}

// cmdAssign handles command /[un]assign [[@]...]
func (b *Bot) cmdAssign(c *command) bool {
	// check command syntax
	if len(c.args) > 1 {
		glog.Info(c.invalid())
		return true
	}

	// ignore assign command when repo owner is not an organization
	if c.ownerType != "Organization" {
		glog.Infof("repo owner is not an organization, ignore.")
		return true
	}

	ctx := context.Background()
	assignee := c.user
	if len(c.args) == 1 {
		assignee = strings.TrimPrefix(c.args[0], "@")
	}

	// TODO(dunjut) check membership

	// assign/unassign issue to/from assignee as requested.
	var err error
	if c.cmd == "/assign" {
		_, _, err = b.git.Issues.AddAssignees(ctx, c.owner, c.repo, c.number, []string{assignee})
	} else { // /unassign
		_, _, err = b.git.Issues.RemoveAssignees(ctx, c.owner, c.repo, c.number, []string{assignee})
	}
	if err != nil {
		glog.Errorf("%s err: %v", c.failed(), err)
		return false
	}
	glog.Info(c.succeed())
	return true
}
