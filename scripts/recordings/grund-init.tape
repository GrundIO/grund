# grund-init.tape
# Records: Interactive setup wizard for first-time users
# Usage: vhs grund-init.tape
#
# Note: This will backup and restore ~/.grund/ during recording

Output docs/assets/grund-init.gif

Set FontSize 16
Set Width 1200
Set Height 490
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 30ms
Set Shell "bash"

# Setup PATH and backup existing config/skills (hidden)
Hide
Type "export PATH=$PATH:$HOME/go/bin"
Enter
Sleep 200ms
Type "mv ~/.grund ~/.grund.backup 2>/dev/null"
Enter
Sleep 100ms
# Remove existing skills for clean demo
Type "mv ~/.claude/skills/using-grund ~/.claude/skills/using-grund.backup 2>/dev/null"
Enter
Sleep 100ms
Type "mv ~/.cursor/skills-cursor/using-grund ~/.cursor/skills-cursor/using-grund.backup 2>/dev/null"
Enter
Sleep 200ms
Type "clear"
Enter
Sleep 300ms
Show

# Title
Type "# First-time setup with grund init"
Sleep 300ms
Enter
Sleep 200ms

# Run init
Type "grund init"
Sleep 200ms
Enter
Sleep 1s

# Step 1: Global config created automatically (no prompt for fresh install)
# Wait for Step 2 prompt

# Step 2: Projects folder - clear default and type demo path
Ctrl+U
Sleep 100ms
Type "~/Personal/grund/docs/demo"
Sleep 200ms
Enter
Sleep 1s

# "Found N projects. Add them to services registry?" - Enter for Yes
Enter
Sleep 500ms

# MultiSelect - select projects with Space, then Enter to confirm
Space
Sleep 300ms
Down
Space
Sleep 300ms
Down
Space
Sleep 300ms
Enter
Sleep 1s

# Step 3: "Set up AI assistant skills?" - Enter for Yes
Enter
Sleep 500ms

# "Which AI assistant?" - Select (Claude Code/Cursor/Both, default: Both)
# Enter to accept "Both"
Enter
Sleep 1s

# Show completion
Sleep 1s

# Restore backups (hidden)
Hide
Type "rm -rf ~/.grund && mv ~/.grund.backup ~/.grund 2>/dev/null"
Enter
Sleep 100ms
Type "rm -rf ~/.claude/skills/using-grund && mv ~/.claude/skills/using-grund.backup ~/.claude/skills/using-grund 2>/dev/null"
Enter
Sleep 100ms
Type "rm -rf ~/.cursor/skills-cursor/using-grund && mv ~/.cursor/skills-cursor/using-grund.backup ~/.cursor/skills-cursor/using-grund 2>/dev/null"
Enter
Show
