# GUPS

> used to express reproof, derision, or remonstrance

-- Mirriam Webster Dictionary

![Gupi-chan](/gups.png)


## What is it?

Tool to help manage Github pull requests through Slack by assigning users to PRs
and notifying them via Slack private messages. Also include a daily todo list
notified via slack.

PR management is done via a fairly simple rule engine which works with user
pools. The author of a PR is tested against a user pool and if there's a match
then we can pick a certain number of users from one or more user pool. The
selected users are then assigned as requested reviewers in Github and a Slack
notification is sent to the user.

The complete Slack summary takes the form of:

```text
Assigned:
- [age] repo/pr: title

Pending:
- [age] repo/pr: title

Ready:
- [age] repo/pr: title

Open:
- [age] repo/pr: title

Requested:
- [age] repo/pr: title

Inspirational Quotes:
> quote
```

Where:
- Assigned: the user was assigned to this PR
- Pending: the user is responsible for reviewing this PR
- Ready: this user's PR is ready to be merged
- Open: this user's PR is still waiting reviews
- Requested: this user was manually requested to review the given PR


## How To Build

```sh
make
```

Uses golang's mod feature and dependencies are vendored.


## Usage

```sh
CONFIG=<path> GITHUB_TOKEN=<token> SLACK_TOKEN=<token> gups [-dry-run] [-dump-users] [-full]
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
| `-full` | Sends a full summary of pending, open and ready PRs.  |
| `-dry-run` | Sends the Slack notification to the console instead of Slack |
| `-dump-users` | Dumps all the visible users in the Slack workspace |


## Config

The configuration file provided through the `CONFIG` environment variable is a
json file with the following form:

```json
{
	"github_to_slack_user": {
		"github-user-a": "slack-user-a",
		"github-user-b": "slack-user-b",
		"github-user-c": "slack-user-c"
	},
	
	"skip_pr_labels": [ "wip" ],
	
	"pools": {
		"team-a": [ "github-user-a", "github-user-b" ],
		"team-b": [ "github-user-c" ]
	},
	
	"ruleset": {
		"my-rules": [
			{ "if": "team-a", "pick": [ "team-a:1" ] },
			{ "if": "team-b", "pick": [ "team-b:1" ] },
			{ "pick": [ "team-a:1", "team-b:1" ] }
		]
	},
	
	"repos": [
		{ "path": "my-org/my-repo", "rule": "my-rules" },
		{ "path": "my-org/my-other-repo", "rule": "my-rules" }
	]
}
```

`github_to_slack_user` contains a mapping of Github username to Slack username
which basically tells Gups how to reach a given Github user on Slack. Note that
if a Github user is not present in this list then it will be ignored by
Gups. The `-dump-users` command line argument utility can be useful to figure
out if a Slack user is accessible with the provided token which can be a problem when
dealing with multiple Slack workspace.

`skip_pr_labels` contains a list of labels that, when found on a PR, indicate
that the PR should be skipped.

`pools` contains a mapping of pool names to a list of Github users that belong
to this given pool. The pool name is used within the `ruleset` section.

`ruleset` contains a mapping of the ruleset name to a list of rules. The rules
are evaluated in order where if the `if` field is present then the author or the
PR is tested against the users in the referenced pool. If no `if` field is
provided then it always matches. If a match is found then the `pick` field
indicates how to assign reviewers using the format `<pool>:<count>` where
`team-a:2` indicates that 2 users should be picked from the pool `team-a`. Once
assignement is done, then the subsequent rules are not evaluated.

`repos` lists all the Github repos to be scanned by Gups. The `path` entry is
the simplified Github path for the repo which takes the form
`<github-username>/<repo-name>`. The `rule` entry references one of the ruleset
specificed in the `ruleset` section of the configuration file.

## Additional Notes

Gups uses Github's requested reviewers as it's persistance layer. Which means
that whenever the rule engine executes, it takes into account any currently
assigned reviewers. In other words, if a user has a review request and belongs
to a pool then, Gups will assume that this user is assigned to review the PR and
will therefore adjust it's picking mechanism accordingly. This will happen
regardless of whether the user was assigned by Gups or not.

Note that this leads to a fair number of edge conditions that Gups tries to
handle as gracefully as possible.


## Credits

* **Gupi-chan logos**: https://make.girls.moe/
* **Inpirational Quotes**: https://icanhazdadjoke.com/

