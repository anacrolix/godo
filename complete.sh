_godo() {
    COMPREPLY=()
    # COMP_CWORD < 0 breaks this function, and I don't know what it means.
    [[ $COMP_CWORD < 0 ]] && return
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev=("${COMP_WORDS[@]:1:COMP_CWORD-1}")

    if [[ ${cur} == -* ]]; then return; fi
    if [[ ${cur} == /* ]]; then return; fi
    for arg in "${prev[@]}"; do
    	if [[ "${arg}" != -* ]]; then
            # the package spec argument has already happened
    		return
    	fi
    done
    COMPREPLY=( $(compgen -W "$(go-list-cmd ${cur})" -- ${cur}) )
}
complete -F _godo -o default godo
