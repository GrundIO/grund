# grund-up.tape
# Records: Starting services with dependency resolution
# Usage: vhs grund-up.tape
#
# Prerequisites:
#   - Docker running
#   - Demo services set up in docs/demo/
#   - Run `grund down` before recording for clean state

Output ../grund-up.gif

Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 50ms

# Hidden setup
Hide
Type "cd ~/Personal/grund/docs/demo"
Enter
Sleep 500ms
Type "export GRUND_CONFIG=$(pwd)/services.yaml"
Enter
Sleep 500ms
Type "docker compose -p grund down 2>/dev/null"
Enter
Sleep 2s
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

# Wait for completion (adjust based on your machine)
Sleep 25s

# Show success
Sleep 2s
