package shared

import (
	"errors"

	ghContext "github.com/cli/cli/v2/context"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/internal/prompter"
)

type AmbiguousBaseRepoError struct {
	Remotes ghContext.Remotes
}

func (e AmbiguousBaseRepoError) Error() string {
	return "multiple remotes detected. please specify which repo to use by providing the -R, --repo argument"
}

type baseRepoFn func() (ghrepo.Interface, error)
type remotesFn func() (ghContext.Remotes, error)

func PromptWhenAmbiguousBaseRepoFunc(baseRepoFn baseRepoFn, prompter prompter.Prompter) baseRepoFn {
	return func() (ghrepo.Interface, error) {
		baseRepo, err := baseRepoFn()
		if err != nil {
			var ambiguousBaseRepoErr AmbiguousBaseRepoError
			if !errors.As(err, &ambiguousBaseRepoErr) {
				return nil, err
			}

			baseRepo, err = promptForRepo(baseRepo, ambiguousBaseRepoErr.Remotes, prompter)
			if err != nil {
				return nil, err
			}
		}

		return baseRepo, nil
	}
}

// RequireNoAmbiguityBaseRepoFunc returns a function to resolve the base repo, ensuring that
// there was only one option, regardless of whether the base repo had been set.
func RequireNoAmbiguityBaseRepoFunc(baseRepo baseRepoFn, remotes remotesFn) baseRepoFn {
	return func() (ghrepo.Interface, error) {
		remotes, err := remotes()
		if err != nil {
			return nil, err
		}

		if remotes.Len() > 1 {
			return nil, AmbiguousBaseRepoError{Remotes: remotes}
		}

		return baseRepo()
	}
}

// REVIEW WARNING: I have not had a close look at this function yet, I do not vouch for it.
func promptForRepo(baseRepo ghrepo.Interface, remotes ghContext.Remotes, prompter prompter.Prompter) (ghrepo.Interface, error) {
	var defaultRepo string
	var remoteArray []string

	// TODO: consider whether we should just go with the default order of remotes because then
	// users that are familiar can just hit enter and achieve the behaviour they had before.
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

	baseRepoInput, errInput := prompter.Select("Select a base repo", defaultRepo, remoteArray)
	if errInput != nil {
		return baseRepo, errInput
	}

	selectedRepo, errSelectedRepo := ghrepo.FromFullName(remoteArray[baseRepoInput])
	if errSelectedRepo != nil {
		return baseRepo, errSelectedRepo
	}

	return selectedRepo, nil
}
