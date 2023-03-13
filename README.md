# Kubernetes Port-Forward Manager (k8s.kpm) \[Preview\]

This tool will help you to do bulk port mappings using kubectl command which is installed and configured in your
machine. This tool expects configured kubectl tool in your machine prior to using this tool.

## Compiling

- Checkout the code from github
- Then run `make` to compile the tool
- Copy the `kpm` binary in the source folder to a desired location and add to `PATH` variable

Note: More information will be added how to compile on different platforms later.

## Usage

| Flag              | Description                       |
|--                 |--                                 |
| -h                |   Print usage help                |
| -l                |   Print configured profile names  |
| -p \<profile\>    | Use the provided profile name     |
| -complete         | Install bash|zsh completions      |
| -uncomplete       | Uninstall bash|zsh completions    |