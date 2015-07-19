_godo() {
	local cur prev
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[@]:1:COMP_CWORD-1}"

    if [[ ${cur} == -* ]]; then return; fi
    if [[ ${cur} == /* ]]; then return; fi
    # if [[ ${cur} == .* ]]; then return; fi
    COMPREPLY=( $(compgen -W "$(go-list-cmd ${cur})" -- ${cur}) )
}
complete -F _godo godo
