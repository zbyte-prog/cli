package shared

import (
	"fmt"

	ghContext "github.com/cli/cli/v2/context"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/internal/prompter"
)

func ValidateHasOnlyOneRemote(hasRepoOverride bool, remotes func() (ghContext.Remotes, error)) error {
	if !hasRepoOverride && remotes != nil {
		remotes, err := remotes()
		if err != nil {
			return err
		}

		if remotes.Len() > 1 {
			return fmt.Errorf("multiple remotes detected %v. please specify which repo to use by providing the -R or --repo argument", remotes)
		}
	}

	return nil
}

func PromptForRepo(baseRepo ghrepo.Interface, remotes func() (ghContext.Remotes, error), survey prompter.Prompter) (ghrepo.Interface, error) {
	var defaultRepo string
	var remoteArray []string

	if remotes, _ := remotes(); remotes != nil {
		if defaultRemote, _ := remotes.ResolvedRemote(); defaultRemote != nil {
			// this is a remote explicitly chosen via `repo set-default`
			defaultRepo = ghrepo.FullName(defaultRemote)
		} else if len(remotes) > 0 {
			// as a fallback, just pick the first remote
			defaultRepo = ghrepo.FullName(remotes[0])
		}

		for _, remote := range remotes {
			remoteArray = append(remoteArray, ghrepo.FullName(remote))
		}
	}

	baseRepoInput, errInput := survey.Select("Select a base repo", defaultRepo, remoteArray)
	if errInput != nil {
		return baseRepo, errInput
	}

	selectedRepo, errSelectedRepo := ghrepo.FromFullName(remoteArray[baseRepoInput])
	if errSelectedRepo != nil {
		return baseRepo, errSelectedRepo
	}

	return selectedRepo, nil
}
