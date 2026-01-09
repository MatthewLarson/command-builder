# cb command wrapper
# Source this file in your .bashrc or .zshrc

function cb() {
    if [ "$1" = "exec" ]; then
        # Capture the constructed command from the Go binary
        local cmd_str
        # Ensure we call the binary, not this function (though usually fine as first arg is different)
        # explicit path or assumes 'cb' binary is in PATH and differs from function (?)
        # Actually functions shadow binaries. We need `command cb exec`
        cmd_str=$(command cb exec)
        
        if [ -n "$cmd_str" ]; then
            # Add the built command to the history list
            history -s "$cmd_str"
            
            # Execute the command
            eval "$cmd_str"
            
            # Optional: Attempt to remove "cb" commands from history session
            # This is complex to do reliably across all shells/configs.
            # For now, we prioritize ensuring the *result* is in history.
        fi
    else
        # Pass through to the binary
        command cb "$@"
    fi
}
