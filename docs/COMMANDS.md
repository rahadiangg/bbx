# `bbx` command reference

For interactive help: `bbx help` or `bbx <command> --help`.

## Top-level commands

```
bbx auth         Manage Bamboo authentication
bbx config       View and edit bbx configuration
bbx plan         Manage Bamboo plans (pipelines)
bbx build        Trigger, stop, and inspect builds
bbx queue        Inspect the Bamboo build queue
bbx deployment   Trigger and inspect deployments
bbx version      Print bbx version

# Future stubs (return ExitNotImpl=6):
bbx permissions, users, system, triggers, trusted-keys, session, avatars
```

## auth

```
bbx auth login    [--name <ctx>] [--base-url <url>] [--token <pat>] [--insecure]
bbx auth logout   [--name <ctx>]
bbx auth whoami
```

## config

```
bbx config view              [--show-secrets]
bbx config contexts
bbx config use-context <name>
bbx config set <key>=<value> [<key>=<value> ...] [--context <ctx>]
    keys: base-url, token, token-env, insecure-skip-verify
```

## plan

```
bbx plan list                       [--all] [--limit N] [--max-results N] [--expand ...]
bbx plan get      <plan-key>
bbx plan enable   <plan-key>
bbx plan disable  <plan-key>
bbx plan delete   <plan-key> --yes
```

### plan branch

```
bbx plan branch list   <plan-key>                  [--all] [--limit N] [--max-results N]
bbx plan branch get    <plan-key> <branch-name>
bbx plan branch create <plan-key> <branch-name>    [--vcs-branch <name>]
```

### plan variable

```
bbx plan variable list   <plan-key>
bbx plan variable get    <plan-key> <name>
bbx plan variable set    <plan-key> <name> <value>
bbx plan variable delete <plan-key> <name>
```

## build

```
bbx build trigger   <plan-key>             [--var key=value ...]
bbx build stop      <build-result-key>
bbx build continue  <build-result-key>
bbx build status    <build-result-key>
bbx build get       <build-result-key>
bbx build history   <plan-key>             [--all] [--limit N] [--max-results N]
bbx build latest                            [--max-results N]
```

### build comment

```
bbx build comment list   <build-result-key>
bbx build comment add    <build-result-key> <content>
bbx build comment delete <build-result-key> <comment-id>
```

### build label

```
bbx build label list   <build-result-key>
bbx build label add    <build-result-key> <label>
bbx build label delete <build-result-key> <label>
```

## queue

```
bbx queue list
```

## deployment

```
bbx deployment queue
bbx deployment trigger --environment-id N --version-id N
bbx deployment cancel  <deployment-result-id>
bbx deployment result  <deployment-result-id>
bbx deployment preview [--project-id N] [--plan-result-key KEY]
```

## Global flags

| Flag                       | Purpose                                                       |
|----------------------------|---------------------------------------------------------------|
| `--config <path>`          | Override config file path                                     |
| `--context <name>`         | Use a non-current context for this invocation                 |
| `-o, --output <format>`    | `table`, `json`, or `yaml` (auto-detects when unset)          |
| `-v, --verbose`            | Increase verbosity (`-v`, `-vv`)                              |
