# grund-service-init.tape
# Records: Initializing a new service with interactive wizard
# Usage: vhs grund-service-init.tape

Output docs/assets/grund-service-init.gif

Set FontSize 16
Set Width 1200
Set Height 490
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 30ms
Set Shell "bash"

# Setup PATH and temp directory (hidden)
Hide
Type "export PATH=$PATH:$HOME/go/bin"
Enter
Sleep 200ms
Type "rm -rf /tmp/payment-service && mkdir -p /tmp/payment-service && cd /tmp/payment-service"
Enter
Sleep 200ms
Type "clear"
Enter
Sleep 300ms
Show

# Title
Type "# Initialize a new service"
Sleep 300ms
Enter
Sleep 200ms

# Run service init
Type "grund service init"
Sleep 200ms
Enter
Sleep 1s

# 1. Service name (Text, default: payment-service from folder) - Enter for default
Enter
Sleep 500ms

# 2. Service type (Select: go/python/node, default: go) - Enter for default
Enter
Sleep 500ms

# 3. Port (Int, default: 8080) - Enter for default
Enter
Sleep 500ms

# 4. Dockerfile path (Text, default: Dockerfile) - Enter for default
Enter
Sleep 500ms

# 5. Health check endpoint (Text, default: /health) - Enter for default
Enter
Sleep 500ms

# 6. Need PostgreSQL? (Confirm, default: No) - Right arrow to Yes, then Enter
Right
Enter
Sleep 500ms

# 6a. Database name (Text, default: payment-service_db) - Enter for default
Enter
Sleep 500ms

# 6b. Migrations path (Text, default: empty) - Enter for empty
Enter
Sleep 500ms

# 7. Need MongoDB? (Confirm, default: No) - Enter for No
Enter
Sleep 500ms

# 8. Need Redis? (Confirm, default: No) - Right arrow to Yes, then Enter
Right
Enter
Sleep 500ms

# 9. Need SQS queues? (Confirm, default: No) - Enter for No
Enter
Sleep 500ms

# 10. Need SNS topics? (Confirm, default: No) - Enter for No
Enter
Sleep 500ms

# 11. Need S3 buckets? (Confirm, default: No) - Enter for No
Enter
Sleep 500ms

# 12. Dependent services (Text) - Enter for empty
Enter
Sleep 1s

# 13. Write this configuration? (Confirm, default: Yes) - Enter for Yes
Enter
Sleep 1s

# Show the generated file
Type "cat grund.yaml"
Enter
Sleep 3s

# Cleanup (hidden)
Hide
Type "rm -rf /tmp/payment-service"
Enter
Show
