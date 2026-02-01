# grund-full-workflow.tape
# Records: Complete workflow - start services, check status, view logs
# Usage: vhs grund-full-workflow.tape
#
# Prerequisites:
#   - Docker running

Output docs/assets/grund-full-workflow.gif

Set FontSize 14
Set Width 1200
Set Height 490
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 30ms
Set Shell "bash"

# Setup PATH and environment (hidden)
Hide
Type "export PATH=$PATH:$HOME/go/bin"
Enter
Sleep 200ms
Type "export GRUND_CONFIG=$HOME/Personal/grund/docs/demo/services.yaml"
Enter
Sleep 200ms
Type "grund down 2>/dev/null"
Enter
Sleep 2s
Type "clear"
Enter
Sleep 300ms
Show

# Step 1: Start services
Type "# 1. Start notification-service with all dependencies"
Enter
Sleep 200ms
Type "grund up notification-service"
Enter
Sleep 12s

Type "clear"
Enter

# Step 2: Check status
Type "# 2. Check service status"
Enter
Sleep 200ms
Type "grund status"
Enter
Sleep 2s

Type "clear"
Enter

# Step 3: View logs
Type "# 3. View logs from order-service"
Enter
Sleep 200ms
Type "grund logs order-service --tail 10"
Enter
Sleep 2s

Type "clear"
Enter

# Step 4: Stop everything
Type "# 4. Stop all services"
Enter
Sleep 200ms
Type "grund down"
Enter
Sleep 4s

# Clear for clean loop
Type "clear"
Enter
Sleep 100ms
