# grund-full-workflow.tape
# Records: Complete workflow from start to API testing
# Usage: vhs grund-full-workflow.tape
#
# Prerequisites:
#   - Docker running
#   - Demo services configured
#   - curl installed

Output ../grund-full-workflow.gif

Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 40ms

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

# Step 1: Start services
Type "# 1. Start the notification service (with all dependencies)"
Enter
Sleep 300ms
Type "grund up notification-service"
Enter
Sleep 25s

# Step 2: Check status
Type "# 2. Check service status"
Enter
Sleep 300ms
Type "grund status"
Enter
Sleep 3s

# Step 3: Test APIs
Type "# 3. Test the APIs"
Enter
Sleep 300ms

Type "curl -s http://localhost:8080/health | jq ."
Enter
Sleep 2s

Type "curl -s -X POST http://localhost:8080/users -H 'Content-Type: application/json' -d '{\"email\":\"demo@test.com\",\"name\":\"Demo User\"}' | jq ."
Enter
Sleep 2s

Type "curl -s -X POST http://localhost:8081/orders -H 'Content-Type: application/json' -d '{\"user_id\":1,\"product\":\"Widget\",\"quantity\":3}' | jq ."
Enter
Sleep 2s

# Step 4: View logs
Type "# 4. Check the logs"
Enter
Sleep 300ms
Type "grund logs order-service --tail 5"
Enter
Sleep 3s

# Done
Type "# Done! All services running with infrastructure"
Enter
Sleep 2s
