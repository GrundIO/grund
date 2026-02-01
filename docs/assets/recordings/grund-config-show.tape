# grund-config-show.tape
# Records: Viewing configuration and resolved environment variables
# Usage: vhs grund-config-show.tape
#
# Prerequisites:
#   - Services registered in config

Output ../grund-config-show.gif

Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 50ms

# Hidden setup
Hide
Type "cd ~/Personal/grund/docs/demo && export GRUND_CONFIG=$(pwd)/services.yaml && clear"
Enter
Sleep 500ms
Show

# Title
Type "# View global configuration"
Sleep 500ms
Enter
Sleep 300ms

# Show global config
Type "grund config show"
Sleep 300ms
Enter
Sleep 4s

# Title for service config
Type "# View resolved config for a service"
Sleep 500ms
Enter
Sleep 300ms

# Show service config
Type "grund config show user-service"
Sleep 300ms
Enter
Sleep 5s
