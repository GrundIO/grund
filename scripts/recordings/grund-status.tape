# grund-status.tape
# Records: Viewing service status
# Usage: vhs grund-status.tape

Output docs/assets/grund-status.gif

Set FontSize 16
Set Width 1000
Set Height 350
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
Type "clear"
Enter
Sleep 500ms
Show

# Title
Type "# Check running services"
Sleep 500ms
Enter
Sleep 300ms

# Run status
Type "grund status"
Sleep 300ms
Enter

# Show the table
Sleep 2s
