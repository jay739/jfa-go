# Contributing to Omnifin

Thanks for your interest! Omnifin is a personal hard fork of [hrfee/jfa-go](https://github.com/hrfee/jfa-go) maintained by a single person, so contributions are very welcome but please read this first.

## Where to file what

- **Bug or feature specific to Omnifin's additions** (Jellyfin/Jellystat announcement variables, the redesigned email templates, the security hardening, the dashboard widgets, etc.) → file it here.
- **Bug or feature that also affects upstream jfa-go** → please file at [hrfee/jfa-go](https://github.com/hrfee/jfa-go) where it benefits a much wider audience. We'll happily pull fixes back.

## Getting set up

```sh
git clone https://github.com/jay739/omnifin.git
cd omnifin
go install github.com/swaggo/swag/cmd/swag@v1.16.4
npm install
make all
./build/omnifin -data ./local-data
```

The first run launches a setup wizard on `http://localhost:8056`.

### Stack

- Go 1.24+ — backend, single binary
- TypeScript — admin UI, compiled with esbuild
- Tailwind CSS + a17t — styling
- MJML — transactional email templates compiled to HTML
- BadgerDB — embedded key-value store
- Gin — HTTP framework

## Pull request guidelines

- **One concern per PR.** A bug fix and a new feature should be two PRs.
- **Run `go vet` and `go test` locally** before pushing. CI will run them on PR but it's faster to catch locally.
- **Match existing style.** No automatic reformatting that touches lines you're not changing.
- **Keep commits focused.** `git rebase -i` to squash WIP commits before review.
- **For UI changes, include a screenshot** in the PR description.
- **Don't add new dependencies** without flagging it in the PR — single-maintainer project, every new dep is technical debt.

## What's actively wanted

- More announcement template variables that pull live data from Jellyfin/Jellystat
- Better mobile responsiveness on the admin panel
- Translations
- Bug reports with reproduction steps

## What's likely to be rejected

- New backend dependencies for things solvable with the standard library
- Sweeping reformatting / "style cleanup" PRs
- Features that significantly diverge from upstream jfa-go's API or schema (we want migration in and out of the project to stay easy)

## Reporting security issues

Please don't open a public issue for security problems. Email the maintainer directly — see the GitHub profile.

## Credits

If you contribute, you'll be added to the contributors list. The original jfa-go work by Harvey Tindall remains attributed throughout.
