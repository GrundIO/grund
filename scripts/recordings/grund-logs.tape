# grund-logs.tape
# Records: Following service logs
# Usage: vhs grund-logs.tape
#
# Prerequisites:
#   - Services should be running
#   - Generate some traffic for interesting logs

Output ../grund-logs.gif

Set FontSize 14
Set Width 1200
Set Height 600
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
Type "# Follow logs from user-service"
Sleep 500ms
Enter
Sleep 300ms

# Run logs with follow
Type "grund logs user-service -f"
Sleep 300ms
Enter

# Wait for some logs
Sleep 8s

# Stop following
Ctrl+C
Sleep 1s
