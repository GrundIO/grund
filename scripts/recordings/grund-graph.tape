# grund-graph.tape
# Records: Visualizing service dependency graph
# Usage: vhs grund-graph.tape

Output docs/assets/grund-graph.gif

Set FontSize 16
Set Width 1000
Set Height 400
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 30ms
Set Shell "bash"

# Setup PATH and environment (hidden)
Hide
Type "export PATH=$PATH:$HOME/go/bin"
Enter
Sleep 500ms
Type "export GRUND_CONFIG=$HOME/Personal/grund/docs/demo/services.yaml"
Enter
Sleep 500ms
Type "clear"
Enter
Sleep 500ms
Show

# Title
Type "# Visualize dependency graph"
Sleep 500ms
Enter
Sleep 300ms

# Render graph as SVG
Type "grund graph"
Sleep 300ms
Enter

# Wait for render
Sleep 3s

# Show DOT output
Type "# Print as DOT format"
Sleep 500ms
Enter
Sleep 300ms

Type "grund graph --format dot"
Sleep 300ms
Enter

# Show the output
Sleep 3s
