# GUPS

> used to express reproof, derision, or remonstrance

-- Mirriam Webster Dictionary

![Gupi-chan](/gups.png)


## What is it?

Tool to help manage Github pull requests through Slack by providing a daily todo
list of pull requests via Slack private messages.

The Slack messages take the form of:

```text
Ready:
- [age] repo/pr: title

Pending:
- [age] repo/pr: title

Requested:
- [age] repo/pr: title

Open:
- [age] repo/pr: title

Inspirational Quotes:
> quote
```

To classify and manage PRs Gups uses a per repo owner lists to whom PRs within a
repo are assigned for reviewed. These pending reviews are represented within the
`Pending` category. When a PR has at least 2 reviews by owner of the repos then
it is classified as `Ready` for both the repo owners and the author. Gups is
also aware of Github's requested review feature so if a user is tagged for
review then the PR will be classified as `Requested` for that user.

Otherwise, the `age` field measures how old the PR and the `Inspirational
Quotes` section gives user a REALLY bad reason to check up on Gups on a daily
basis.

*Note that a screenshots would be better but my notifications are loaded with
private information.*


## How To Build

```sh
go build -mod=vendor ./...
```

Uses golang's mod feature and dependencies are vendored.


## Usage

```sh
CONFIG=<path> GITHUB_TOKEN=<token> SLACK_TOKEN=<token> gups [-dry-run] [-dump-users] 
```

Environment variables are as follows:

| Key | Example | Value |
| - | - | - |
| `CONFIG` | `/etc/gups.json` | Path to [configuration file](#config) |
| `GITHUB_TOKEN` | `1234567890abcdef1234567890abcdef12345678` | [Github token](https://github.blog/2013-05-16-personal-api-tokens/) |
| `SLACK_TOKEN` | `i-dont-remember-what-it-looks-like` | [Slack internal app token](https://slack.com/intl/en-ca/help/articles/215770388) |

Getting a Github token is pretty straight-forward. For a slack token you'll need
to manually create a Gups app and install it within your workspace. Once
installed you'll be given a token that you can give to Gups.

Providing no command line arguments will execute Gups default behaviour which is
to scan Github and send Slack notifications. The command line arguments are
utilities provided by Gups:

| Argument | Effect |
| - | - |
| `-dry-run` | Sends the Slack notification to the console instead of Slack |
| `-dump-users` | Dumps all the visible users in the Slack workspace |



## Config

The configuration file provided through the `CONFIG` environment variable is a
json file with the following form:

```json
{
	"github_to_slack_user": {
		"RAttab": "remi.attab"
	},
	
	"repos": [
		{ "path": "RAttab/gups", "owners: [ "RAttab" ] }
	]
}
```

`github_to_slack_user` contains a mapping of Github username to Slack username
which basically tells Gups how to reach a given Github user on Slack. Note that
if a Github user is not present in this list then it will be ignored by
Gups. The `-dump-users` command line argument utility can be useful to figure
out if a Slack user is accessible with the provided token which can be a problem when
dealing with multiple Slack workspace.

`repos` lists all the Github repos to be scanned by Gups. The `path` entry is
the simplified Github path for the repo which takes the form
`<github-username>/<repo-name>`. The `owner` entry, is a list of Github users
configured in the `github_to_slack_user` section. Note that an empty owner list
is also acceptable in which case only the requested reviews will be pulled from
the repo.

You can test your config via the `-dry-run` command line argument which will
execute the Gups workflow but will dump the notifications messages to the
console of sending them to Slack.


## Credits

* **Gupi-chan logos**: https://make.girls.moe/
* **Inpirational Quotes**: https://icanhazdadjoke.com/

