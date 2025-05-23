<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

## sparrow completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(sparrow completion zsh)

To load completions for every new session, execute once:

#### Linux:

	sparrow completion zsh > "${fpath[1]}/_sparrow"

#### macOS:

	sparrow completion zsh > $(brew --prefix)/share/zsh/site-functions/_sparrow

You will need to start a new shell for this setup to take effect.


```
sparrow completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.sparrow.yaml)
```

### SEE ALSO

* [sparrow completion](sparrow_completion.md)	 - Generate the autocompletion script for the specified shell

