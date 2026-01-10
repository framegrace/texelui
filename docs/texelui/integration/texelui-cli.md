# Bash Dialog Creator (TexelUI CLI)

The `texelui` CLI lets you build interactive dialogs from Bash by describing a JSON spec. It starts a local UI server on demand, renders the dialog in your terminal, and lets your script wait for events, read values, and stream command output.

## Quick Start

```bash
#!/usr/bin/env bash
set -euo pipefail

session=$(texelui open --spec - <<'JSON'
{
  "title": "Deploy",
  "layout": { "type": "form", "gap": 1, "padding": 1, "label_width": 12 },
  "widgets": [
    { "id": "env", "type": "combobox", "label": "Env", "options": ["dev", "staging", "prod"] },
    { "id": "tag", "type": "input", "label": "Tag", "placeholder": "v1.2.3" },
    { "id": "run", "type": "button", "label": "Deploy" },
    { "id": "status", "type": "label", "text": "" }
  ]
}
JSON
)

export TEXELUI_SESSION="$session"

texelui wait --events click:run
eval "$(texelui get --ids env,tag --format sh)"
texelui set --id status --text "Deploying $env:$tag..."
texelui close
```

Notes:
- `texelui` auto-starts its background server and reuses `TEXELUI_SESSION` by default.
- When running from this repo, use `go run ./cmd/texelui` instead of `texelui`.

## Workflow

1. `texelui open` with a JSON spec (returns a session id).
2. `texelui wait` for events and optionally return values.
3. Use `texelui get` / `set` / `append` / `run` to interact with widgets.
4. `texelui close` when finished.

Only one session can be active per server. For parallel dialogs, run with a different socket via `--socket` or `TEXELUI_SOCKET`.

## Commands

### open
```bash
texelui open --spec path/to/spec.json
texelui open --spec -
```
- Reads a JSON spec and opens a dialog.
- Returns a session id on stdout.

### wait
```bash
texelui wait --events click:run,click:exit
texelui wait --values root,pattern --format sh
texelui wait --value status
```
- `--events` is a comma-separated filter list. Filters are `type:id`, with `*` allowed in either position.
- Events are returned as `type:id` (for example `click:run`).
- `--value` returns a single widget value as a raw string.
- `--values` returns multiple widget values. Use `--format sh` for shell assignments or `--format json` for JSON.

### get
```bash
texelui get --ids root,pattern --format sh
```
- Reads widget values by id.
- `--format sh` prints `key='value'` lines (safe to `eval`), `--format json` prints JSON.

### set
```bash
texelui set --id status --text "Running..."
texelui set --id root --value "/tmp"
texelui set --id follow --checked true
```
- `--text` updates labels and buttons.
- `--value` updates input, combobox, and textarea values.
- `--checked` updates checkboxes.

### append
```bash
texelui append --id log --text "Line of output\n"
```
- Appends text to `textarea` / `log` widgets.

### run
```bash
texelui run --stdout log --stderr log --clear log -- find . -name "*.go"
```
- Runs a command and streams stdout/stderr line-by-line into widgets.
- `--clear` empties a widget before running.
- `--cwd` runs the command in a specific directory.
- The CLI exits with the child process exit code when non-zero.

### close
```bash
texelui close
```
- Closes the active session and shuts down the server.

### server and socket
```bash
texelui --server --socket /tmp/texelui.sock
```
- `--server` runs the server daemon directly (the CLI auto-starts it if needed).
- `--socket` overrides the socket path for all commands.

## JSON Spec

Top-level object:
```json
{
  "title": "Window title",
  "layout": { "type": "form" },
  "widgets": []
}
```

### Layout
- `type`: `form` (default) or `vbox`.
- `gap`: spacing between rows (form) or children (vbox).
- `padding`: uniform padding around content.
- `label_width`: label column width. For `vbox`, it aligns label+field rows when labels are used.

### Widgets

Every widget needs a unique `id` and a `type`.

Common fields:
- `label`: row label (form) or inline text for checkbox/button.
- `text`: explicit text for label/button.
- `value`: initial value (string/number/bool depending on widget).
- `width`/`height`: size hints.
- `flex`: when using `vbox`, makes the widget grow.

Supported widget types:

#### input / number
- Fields: `value`, `placeholder`, `width`.
- Emits `change` events on edits.
- `number` is an input variant without numeric validation (values are strings).

#### combobox
- Fields: `options`, `value`, `editable`, `width`.
- Default value is the first option if `value` is empty.
- Emits `change` events.

#### checkbox
- Fields: `label`, `value`.
- `value` is `true`/`false`.
- Emits `change` events.

#### button
- Fields: `label` or `text`, `width`.
- Emits `click` events.

#### label
- Fields: `text` or `label`, `width`.
- Use `texelui set --text` to update.

#### textarea
- Fields: `value`, `width`, `height`, `readonly`.
- `height` defaults to 4 rows.
- Emits `change` events when editable.

#### log
- Same as `textarea` but intended for output streaming.
- Does not emit `change` events.
- Works with `texelui append` and `texelui run`.

### Form layout rules
- Inputs, numbers, and comboboxes use `label` as the left column label.
- Checkboxes, buttons, and labels are full-width rows (no label column).
- Textareas/logs can include a label row above the field when `label` is set.

### VBox layout rules
- Widgets are stacked vertically with `gap` spacing.
- If `label` is set and the widget is not inline (`checkbox`/`button`) or `label` itself, the UI builds a label+field row.
- Use `flex: true` to make a widget expand.

## Events

- `click:<id>` from buttons.
- `change:<id>` from input, combobox, checkbox, and textarea (not log).
- `close:session` when the dialog closes (including Ctrl+C or Esc).

Event filters accept wildcards: `*`, `click:*`, `*:run`.

## Environment Variables

- `TEXELUI_SESSION`: default session id for commands.
- `TEXELUI_SOCKET`: override the socket path.
- Socket default: `$XDG_RUNTIME_DIR/texelui/daemon-$UID.sock`, falling back to `$TMPDIR`.

## Full Example (command runner)

This example opens a form, waits for Run/Exit, runs a command with the form
values, streams output to a text area, and updates a status label. The loop
keeps running until Exit is pressed.

```bash
#!/usr/bin/env bash
set -euo pipefail

session=$(texelui open --spec - <<'JSON'
{
  "title": "Find Files",
  "layout": { "type": "form", "gap": 1, "padding": 1, "label_width": 14 },
  "widgets": [
    { "id": "root", "type": "input", "label": "Root", "value": "." },
    { "id": "pattern", "type": "input", "label": "Pattern", "value": "*.go" },
    { "id": "max_depth", "type": "number", "label": "Max Depth", "value": 3 },
    { "id": "follow", "type": "checkbox", "label": "Follow symlinks", "value": false },
    { "id": "run", "type": "button", "label": "Run" },
    { "id": "exit", "type": "button", "label": "Exit" },
    { "id": "status", "type": "label", "text": "" },
    { "id": "log", "type": "textarea", "label": "Output", "height": 12, "readonly": true }
  ]
}
JSON
)

export TEXELUI_SESSION="$session"

while true; do
  event=$(texelui wait --events click:run,click:exit)
  case "$event" in
    click:exit) break ;;
    click:run)
      eval "$(texelui get --ids root,pattern,max_depth,follow --format sh)"

      texelui set --id status --text "Running..."
      cmd=(find "$root")
      [ -n "$max_depth" ] && cmd+=(-maxdepth "$max_depth")
      [ "$follow" = "true" ] && cmd+=(-L)
      cmd+=(-name "$pattern")

      if texelui run --stdout log --stderr log --clear log -- "${cmd[@]}"; then
        texelui set --id status --text "Done."
      else
        rc=$?
        texelui set --id status --text "Failed (rc=$rc)."
      fi
      ;;
  esac
done

texelui close
```
