# grund-service-init.tape
# Records: Initializing a new service with interactive wizard
# Usage: vhs grund-service-init.tape
#
# Prerequisites:
#   - Create a temp directory for the demo
#   - Clean up after recording

Output ../grund-service-init.gif

Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 50ms

# Hidden setup
Hide
Type "mkdir -p /tmp/demo-service && cd /tmp/demo-service && clear"
Enter
Sleep 500ms
Show

# Title
Type "# Initialize a new service"
Sleep 500ms
Enter
Sleep 300ms

# Run service init
Type "grund service init"
Sleep 300ms
Enter
Sleep 1s

# Service name
Type "payment-service"
Enter
Sleep 1s

# Service type (select go)
Enter
Sleep 1s

# Port
Type "8080"
Enter
Sleep 1s

# Dockerfile path
Enter
Sleep 1s

# Health endpoint
Enter
Sleep 1s

# PostgreSQL? yes
Type "y"
Enter
Sleep 500ms

# Database name
Type "payments_db"
Enter
Sleep 1s

# Redis? yes
Type "y"
Enter
Sleep 1s

# SQS? no
Type "n"
Enter
Sleep 500ms

# SNS? no
Type "n"
Enter
Sleep 500ms

# S3? no
Type "n"
Enter
Sleep 1s

# Dependencies? no
Type "n"
Enter
Sleep 2s

# Show the generated file
Type "cat grund.yaml"
Enter
Sleep 4s

# Cleanup (hidden)
Hide
Type "rm -rf /tmp/demo-service"
Enter
Show
