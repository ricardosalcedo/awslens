# awslens demo recording
# Run: vhs demo.tape

Output docs/demo.gif
Set Width 1200
Set Height 600
Set FontSize 16
Set Theme "Catppuccin Mocha"
Set TypingSpeed 80ms

# ── Launch ────────────────────────────────────────────────────────
Type "awslens"
Sleep 500ms
Enter
Sleep 3s

# ── Profile picker — select awslens-readonly (3rd item) ──────────
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Enter
Sleep 2s

# ── Costs (index 17) — navigate down from EC2 ────────────────────
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 300ms
Enter
Sleep 2s

# ── Navigate to previous months ──────────────────────────────────
Type "["
Sleep 3s
Type "["
Sleep 3s

# ── Back to current month ────────────────────────────────────────
Type "]"
Sleep 3s
Type "]"
Sleep 3s
Escape
Sleep 1s

# ── CloudFormation (index 16) — 1 up from Costs ──────────────────
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Enter
Sleep 4s
Escape
Sleep 1s

# ── Quit ──────────────────────────────────────────────────────────
Escape
Sleep 500ms
Type "q"
Sleep 1s
