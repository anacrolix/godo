_godo() {
	local cur prev
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev=("${COMP_WORDS[@]:1:COMP_CWORD-1}")

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
