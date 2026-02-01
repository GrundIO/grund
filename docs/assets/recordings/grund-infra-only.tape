# grund-infra-only.tape
# Records: Starting only infrastructure for local development
# Usage: vhs grund-infra-only.tape
#
# Prerequisites:
#   - Docker running
#   - Demo services configured

Output ../grund-infra-only.gif

Set FontSize 16
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 50ms

# Hidden setup
Hide
Type "cd ~/Personal/grund/docs/demo && export GRUND_CONFIG=$(pwd)/services.yaml"
Enter
Sleep 300ms
Type "docker compose -p grund down 2>/dev/null"
Enter
Sleep 2s
Type "clear"
Enter
Sleep 500ms
Show

# Title
Type "# Start only infrastructure (run service locally)"
Sleep 500ms
Enter
Sleep 300ms

# Run with --infra-only
Type "grund up user-service --infra-only"
Sleep 300ms
Enter

# Wait for infrastructure
Sleep 15s

# Show what's running
Type "grund status"
Enter
Sleep 4s
