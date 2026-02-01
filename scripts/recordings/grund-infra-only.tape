# grund-infra-only.tape
# Records: Starting only infrastructure for local development
# Usage: vhs grund-infra-only.tape
#
# Prerequisites:
#   - Docker running

Output docs/assets/grund-infra-only.gif

Set FontSize 16
Set Width 1200
Set Height 420
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 30ms
Set Shell "bash"

# Setup PATH and environment (hidden)
Hide
Type "export PATH=$PATH:$HOME/go/bin"
Enter
Sleep 300ms
Type "export GRUND_CONFIG=$HOME/Personal/grund/docs/demo/services.yaml"
Enter
Sleep 300ms
Type "grund down 2>/dev/null"
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
Sleep 6s

# Show what's running
Type "grund status"
Enter
Sleep 2s
