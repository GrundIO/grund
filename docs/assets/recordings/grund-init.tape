# grund-init.tape
# Records: Interactive setup wizard for first-time users
# Usage: vhs grund-init.tape
#
# Prerequisites:
#   - Backup ~/.grund/ if you have existing config
#   - This will create a fresh ~/.grund/config.yaml

Output ../grund-init.gif

Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 50ms

# Hidden setup - backup and remove existing config
Hide
Type "mv ~/.grund ~/.grund.backup 2>/dev/null; clear"
Enter
Sleep 500ms
Show

# Title
Type "# First-time setup with grund init"
Sleep 500ms
Enter
Sleep 300ms

# Run init
Type "grund init"
Sleep 300ms
Enter
Sleep 2s

# Step 1: Global config - press enter to accept
Enter
Sleep 2s

# Step 2: Projects folder - type path
Type "~/Personal"
Sleep 300ms
Enter
Sleep 3s

# Navigate selection (if services are found)
# Press enter to confirm selection
Enter
Sleep 2s

# Step 3: AI Skills - skip for demo
Type "n"
Enter
Sleep 2s

# Show completion
Sleep 2s

# Restore backup (hidden)
Hide
Type "rm -rf ~/.grund && mv ~/.grund.backup ~/.grund 2>/dev/null"
Enter
Show
