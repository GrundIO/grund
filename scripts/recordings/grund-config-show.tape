# grund-config-show.tape
# Records: Viewing configuration and resolved environment variables
# Usage: vhs grund-config-show.tape

Output docs/assets/grund-config-show.gif

Set FontSize 16
Set Width 1200
Set Height 650
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 30ms
Set Shell "bash"

# Setup PATH and environment (hidden)
Hide
Type "export PATH=$PATH:$HOME/go/bin"
Enter
Sleep 200ms
Type "export GRUND_CONFIG=$HOME/Personal/grund/docs/demo/services.yaml"
Enter
Sleep 200ms
Type "clear"
Enter
Sleep 300ms
Show

# Title
Type "# View global configuration"
Sleep 300ms
Enter
Sleep 200ms

# Show global config
Type "grund config show"
Sleep 200ms
Enter
Sleep 2s

# Clear before service config
Type "clear"
Enter

# Title for service config
Type "# View resolved config for a service"
Sleep 300ms
Enter
Sleep 200ms

# Show service config
Type "grund config show user-service"
Sleep 200ms
Enter
Sleep 3s
