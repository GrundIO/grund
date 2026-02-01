# VHS Recording Scripts

This directory contains [VHS](https://github.com/charmbracelet/vhs) tape files for generating documentation GIFs.

## Prerequisites

```bash
# Install VHS
brew install charmbracelet/tap/vhs

# Ensure Docker is running
docker info
```

## Recording Scripts

| Tape File | Output | Description |
|-----------|--------|-------------|
| `grund-up.tape` | `grund-up.gif` | Starting services with dependency resolution |
| `grund-init.tape` | `grund-init.gif` | Interactive setup wizard |
| `grund-status.tape` | `grund-status.gif` | Viewing service status |
| `grund-logs.tape` | `grund-logs.gif` | Following service logs |
| `grund-service-init.tape` | `grund-service-init.gif` | Initializing a new service |
| `grund-service-add.tape` | `grund-service-add.gif` | Adding infrastructure to service |
| `grund-config-show.tape` | `grund-config-show.gif` | Viewing configuration |
| `grund-infra-only.tape` | `grund-infra-only.gif` | Starting only infrastructure |
| `grund-full-workflow.tape` | `grund-full-workflow.gif` | Complete workflow demo |

## Usage

### Using Make (Recommended)

```bash
# Record all GIFs
make record-gifs

# Record a single GIF
make record-gif TAPE=grund-up
```

### Manual Recording

```bash
cd scripts/recordings
vhs grund-up.tape -o ../../docs/assets/grund-up.gif
```

## Output Location

GIFs are generated in `docs/assets/`.

## Customization

### Adjust timing

Edit the `Sleep` durations in tape files based on your machine's speed:
- Faster machine: Reduce sleep times
- Slower machine: Increase sleep times

### Change theme

Available themes: `Dracula`, `GitHub`, `Monokai`, `Nord`, `Tokyo Night`, etc.

```tape
Set Theme "GitHub"  # Light theme
Set Theme "Dracula" # Dark theme (default)
```

### Change dimensions

```tape
Set Width 1200   # Terminal width in pixels
Set Height 700   # Terminal height in pixels
Set FontSize 16  # Font size
Set Padding 20   # Padding around terminal
```

## Tips

1. **Clean state**: Run `grund down` before recording `grund-up.tape`
2. **Backup config**: The `grund-init.tape` temporarily removes `~/.grund/`
3. **Pre-pull images**: Pull Docker images beforehand for faster recordings
4. **Test first**: Run tape with shorter sleep times to test, then increase

## Updating GIFs

When Grund's output changes:

1. Edit the relevant `.tape` file if needed
2. Run `make record-gifs` or `make record-gif TAPE=<name>`
3. Commit the updated GIFs
