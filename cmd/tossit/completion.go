package main

import (
	"fmt"
	"os"
)

const bashCompletion = `_tossit() {
    local cur prev commands global_opts send_opts recv_opts relay_opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    commands="send receive relay update help"
    global_opts="--relay --relay-token --stream --password --version --help"
    send_opts="--relay --relay-token --stream --password"
    recv_opts="--relay --relay-token --password --dir"
    relay_opts="--config --port --storage --expire --max-size --rate-limit --auth-token --allow-ips --ui --ui-password --admin-password --help"

    case "$prev" in
        --relay|--relay-token|--password|--dir|--config|--port|--storage|--expire|--max-size|--rate-limit|--auth-token|--allow-ips|--ui-password|--admin-password|-p|-d)
            return 0
            ;;
        --ui)
            COMPREPLY=( $(compgen -W "true false" -- "$cur") )
            return 0
            ;;
    esac

    local cmd=""
    for ((i=1; i < COMP_CWORD; i++)); do
        case "${COMP_WORDS[i]}" in
            send|s) cmd="send"; break;;
            receive|recv|r) cmd="receive"; break;;
            relay) cmd="relay"; break;;
        esac
    done

    case "$cmd" in
        send)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=( $(compgen -W "$send_opts" -- "$cur") )
            else
                COMPREPLY=( $(compgen -f -- "$cur") )
            fi
            ;;
        receive)
            COMPREPLY=( $(compgen -W "$recv_opts" -- "$cur") )
            ;;
        relay)
            COMPREPLY=( $(compgen -W "$relay_opts" -- "$cur") )
            ;;
        "")
            if [[ "$cur" == -* ]]; then
                COMPREPLY=( $(compgen -W "$global_opts" -- "$cur") )
            else
                COMPREPLY=( $(compgen -W "$commands" -- "$cur") $(compgen -f -- "$cur") )
            fi
            ;;
    esac
}
complete -o default -F _tossit tossit
`

const zshCompletion = `#compdef tossit

_tossit() {
    local -a commands
    commands=(
        'send:Upload and share files'
        'receive:Download files'
        'relay:Run a self-hosted relay server'
        'update:Check for updates'
        'help:Show help'
    )

    _arguments -C \
        '1:command:->cmd' \
        '*::arg:->args'

    case "$state" in
        cmd)
            _alternative \
                'commands:command:_describe "command" commands' \
                'files:file:_files'
            ;;
        args)
            case "${words[1]}" in
                send|s)
                    _arguments \
                        '--relay[Relay server URL]:url:' \
                        '--relay-token[Auth token for private relay]:token:' \
                        '--stream[Real-time streaming]' \
                        '--password[Password-protect the transfer]:password:' \
                        '-p[Password-protect the transfer]:password:' \
                        '*:file:_files'
                    ;;
                receive|recv|r)
                    _arguments \
                        '--relay[Relay server URL]:url:' \
                        '--relay-token[Auth token for private relay]:token:' \
                        '--password[Password for protected transfer]:password:' \
                        '-p[Password for protected transfer]:password:' \
                        '--dir[Save files to directory]:directory:_directories' \
                        '-d[Save files to directory]:directory:_directories' \
                        '1:code:'
                    ;;
                relay)
                    _arguments \
                        '--config[Load config from JSON file]:file:_files' \
                        '--port[Port to listen on]:port:' \
                        '--storage[Storage directory]:directory:_directories' \
                        '--expire[Transfer expiry]:duration:' \
                        '--max-size[Max file size]:size:' \
                        '--rate-limit[Max connections per minute per IP]:number:' \
                        '--auth-token[Require token for access]:token:' \
                        '--allow-ips[Comma-separated IP allowlist]:ips:' \
                        '--ui[Enable web UI]:bool:(true false)' \
                        '--ui-password[Password to access web UI]:password:' \
                        '--admin-password[Admin password]:password:' \
                        '--help[Show help]'
                    ;;
            esac
            ;;
    esac
}

_tossit "$@"
`

const fishCompletion = `complete -c tossit -f

# Subcommands
complete -c tossit -n '__fish_use_subcommand' -a send -d 'Upload and share files'
complete -c tossit -n '__fish_use_subcommand' -a receive -d 'Download files'
complete -c tossit -n '__fish_use_subcommand' -a relay -d 'Run a self-hosted relay server'
complete -c tossit -n '__fish_use_subcommand' -a update -d 'Check for updates'
complete -c tossit -n '__fish_use_subcommand' -a help -d 'Show help'
complete -c tossit -n '__fish_use_subcommand' -F

# Global options
complete -c tossit -l version -d 'Show version'
complete -c tossit -l help -d 'Show help'

# Send options
complete -c tossit -n '__fish_seen_subcommand_from send s' -l relay -x -d 'Relay server URL'
complete -c tossit -n '__fish_seen_subcommand_from send s' -l relay-token -x -d 'Auth token'
complete -c tossit -n '__fish_seen_subcommand_from send s' -l stream -d 'Real-time streaming'
complete -c tossit -n '__fish_seen_subcommand_from send s' -l password -x -d 'Password-protect'
complete -c tossit -n '__fish_seen_subcommand_from send s' -s p -x -d 'Password-protect'
complete -c tossit -n '__fish_seen_subcommand_from send s' -F

# Receive options
complete -c tossit -n '__fish_seen_subcommand_from receive recv r' -l relay -x -d 'Relay server URL'
complete -c tossit -n '__fish_seen_subcommand_from receive recv r' -l relay-token -x -d 'Auth token'
complete -c tossit -n '__fish_seen_subcommand_from receive recv r' -l password -x -d 'Password'
complete -c tossit -n '__fish_seen_subcommand_from receive recv r' -s p -x -d 'Password'
complete -c tossit -n '__fish_seen_subcommand_from receive recv r' -l dir -x -d 'Output directory'
complete -c tossit -n '__fish_seen_subcommand_from receive recv r' -s d -x -d 'Output directory'

# Relay options
complete -c tossit -n '__fish_seen_subcommand_from relay' -l config -rF -d 'Config file'
complete -c tossit -n '__fish_seen_subcommand_from relay' -l port -x -d 'Port'
complete -c tossit -n '__fish_seen_subcommand_from relay' -l storage -x -d 'Storage directory'
complete -c tossit -n '__fish_seen_subcommand_from relay' -l expire -x -d 'Transfer expiry'
complete -c tossit -n '__fish_seen_subcommand_from relay' -l max-size -x -d 'Max file size'
complete -c tossit -n '__fish_seen_subcommand_from relay' -l rate-limit -x -d 'Rate limit per IP'
complete -c tossit -n '__fish_seen_subcommand_from relay' -l auth-token -x -d 'Auth token'
complete -c tossit -n '__fish_seen_subcommand_from relay' -l allow-ips -x -d 'IP allowlist'
complete -c tossit -n '__fish_seen_subcommand_from relay' -l ui -xa 'true false' -d 'Enable web UI'
complete -c tossit -n '__fish_seen_subcommand_from relay' -l ui-password -x -d 'UI password'
complete -c tossit -n '__fish_seen_subcommand_from relay' -l admin-password -x -d 'Admin password'
complete -c tossit -n '__fish_seen_subcommand_from relay' -l help -d 'Show help'
`

func runCompletion(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: tossit completion <bash|zsh|fish>")
		os.Exit(1)
	}

	switch args[0] {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	case "fish":
		fmt.Print(fishCompletion)
	default:
		fmt.Fprintf(os.Stderr, "Unknown shell: %s (supported: bash, zsh, fish)\n", args[0])
		os.Exit(1)
	}
}
