# grund-service-add.tape
# Records: Adding infrastructure to an existing service
# Usage: vhs grund-service-add.tape
#
# Prerequisites:
#   - A directory with an existing grund.yaml

Output ../grund-service-add.gif

Set FontSize 16
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 50ms

# Hidden setup - create a minimal grund.yaml
Hide
Type "mkdir -p /tmp/demo-service && cd /tmp/demo-service"
Enter
Sleep 300ms
Type @"cat > grund.yaml << 'EOF'
version: "1"
service:
  name: demo-service
  type: go
  port: 8080
requires:
  services: []
  infrastructure: {}
EOF"@
Enter
Sleep 500ms
Type "clear"
Enter
Sleep 500ms
Show

# Title
Type "# Add infrastructure to existing service"
Sleep 500ms
Enter
Sleep 300ms

# Add PostgreSQL
Type "grund service add postgres myapp_db"
Sleep 300ms
Enter
Sleep 2s

# Add Redis
Type "grund service add redis"
Sleep 300ms
Enter
Sleep 2s

# Add SQS queue
Type "grund service add queue order-events"
Sleep 300ms
Enter
Sleep 2s

# Add service dependency
Type "grund service add dependency user-service"
Sleep 300ms
Enter
Sleep 2s

# Show the updated file
Type "cat grund.yaml"
Enter
Sleep 5s

# Cleanup (hidden)
Hide
Type "rm -rf /tmp/demo-service"
Enter
Show
