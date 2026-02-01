# grund-up.tape
# Records: Starting services with dependency resolution
# Usage: vhs grund-up.tape
#
# Prerequisites:
#   - Docker running
#   - Run `grund down` before recording for clean state

Output docs/assets/grund-up.gif

Set FontSize 16
Set Width 1200
Set Height 490
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 30ms
Set Shell "bash"

# Setup PATH and environment (hidden)
Hide
Type "export PATH=$PATH:$HOME/go/bin"
Enter
Sleep 500ms
Type "export GRUND_CONFIG=$HOME/Personal/grund/docs/demo/services.yaml"
Enter
Sleep 500ms
Type "grund down 2>/dev/null"
Enter
Sleep 3s
Type "clear"
Enter
Sleep 500ms
Show

# Title comment
Type "# Start notification-service with all dependencies"
Sleep 500ms
Enter
Sleep 300ms

# Run the command
Type "grund up notification-service"
Sleep 300ms
Enter

# Wait for completion
Sleep 13s

# Show success
Sleep 2s
