# grund-logs.tape
# Records: Viewing service logs
# Usage: vhs grund-logs.tape
#
# Note: Starts user-service in hidden setup to have logs to show

Output docs/assets/grund-logs.gif

Set FontSize 14
Set Width 1200
Set Height 420
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 30ms
Set Shell "bash"

# Setup PATH, environment, and start service (hidden)
Hide
Type "export PATH=$PATH:$HOME/go/bin"
Enter
Sleep 200ms
Type "export GRUND_CONFIG=$HOME/Personal/grund/docs/demo/services.yaml"
Enter
Sleep 200ms
# Start service first so we have logs
Type "grund up user-service"
Enter
Sleep 10s
Type "clear"
Enter
Sleep 300ms
Show

# Title
Type "# View logs from user-service"
Sleep 300ms
Enter
Sleep 200ms

# Run logs
Type "grund logs user-service"
Sleep 200ms
Enter

# Show logs
Sleep 3s

# Cleanup (hidden)
Hide
Type "grund down"
Enter
Sleep 2s
Show
