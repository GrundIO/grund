# grund-service-add.tape
# Records: Adding infrastructure to an existing service
# Usage: vhs grund-service-add.tape

Output docs/assets/grund-service-add.gif

Set FontSize 16
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 30ms
Set Shell "bash"

# Setup: create service with grund service init (hidden)
Hide
Type "export PATH=$PATH:$HOME/go/bin"
Enter
Sleep 200ms
Type "rm -rf /tmp/demo-service && mkdir -p /tmp/demo-service && cd /tmp/demo-service"
Enter
Sleep 300ms

# Run grund service init
Type "grund service init"
Enter
Sleep 500ms

# 1-5: Service basics - Enter for defaults
Enter
Sleep 300ms
Enter
Sleep 300ms
Enter
Sleep 300ms
Enter
Sleep 300ms
Enter
Sleep 300ms

# 6-11: Infrastructure (all No) - Enter for defaults
Enter
Sleep 300ms
Enter
Sleep 300ms
Enter
Sleep 300ms
Enter
Sleep 300ms
Enter
Sleep 300ms
Enter
Sleep 300ms

# 12. Dependent services - Enter for empty
Enter
Sleep 500ms

# 13. Write this configuration? - Enter for Yes
Enter
Sleep 500ms

Type "clear"
Enter
Sleep 300ms
Show

# Title
Type "# Add infrastructure to existing service"
Sleep 300ms
Enter
Sleep 200ms

# Show current file
Type "cat grund.yaml"
Enter
Sleep 2s

# Clear screen so add commands are at top
Type "clear"
Enter
Sleep 200ms

# Add PostgreSQL
Type "grund service add postgres myapp_db"
Sleep 200ms
Enter
Sleep 500ms

# Add Redis
Type "grund service add redis"
Sleep 200ms
Enter
Sleep 500ms

# Add SQS queue
Type "grund service add queue order-events"
Sleep 200ms
Enter
Sleep 500ms

# Clear and show the updated file
Type "clear"
Enter
Type "# Updated grund.yaml"
Enter
Sleep 200ms
Type "cat grund.yaml"
Enter
Sleep 3s

# Cleanup (hidden)
Hide
Type "rm -rf /tmp/demo-service"
Enter
Show
