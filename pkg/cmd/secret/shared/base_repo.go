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

			baseRepoOptions := make([]string, len(ambiguousBaseRepoErr.Remotes))
			for i, remote := range ambiguousBaseRepoErr.Remotes {
				baseRepoOptions[i] = ghrepo.FullName(remote)
			}

			selectedBaseRepo, err := prompter.Select("Select a base repo", baseRepoOptions[0], baseRepoOptions)
			if err != nil {
				return nil, err
			}

			selectedRepo, err := ghrepo.FromFullName(baseRepoOptions[selectedBaseRepo])
			if err != nil {
				return nil, err
			}

			return selectedRepo, nil
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
