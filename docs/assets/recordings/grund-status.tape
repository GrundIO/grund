# grund-status.tape
# Records: Viewing service status
# Usage: vhs grund-status.tape
#
# Prerequisites:
#   - Some services should be running for a good demo
#   - Run `grund up user-service` before recording

Output ../grund-status.gif

Set FontSize 16
Set Width 1000
Set Height 500
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 50ms

# Hidden setup
Hide
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
Sleep 4s
