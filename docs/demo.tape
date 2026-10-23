# VHS tape for the README demo GIF. https://github.com/charmbracelet/vhs
# Record with:  vhs docs/demo.tape   (produces docs/demo.gif)
# Tip: in another terminal, send an email to the shown address mid-recording so
# the live-arrival animation is captured.

Output docs/demo.gif

Set FontSize 18
Set Width 1100
Set Height 640
Set Padding 20
Set Theme "Catppuccin Mocha"

Type "mailbust demo-inbox"
Enter
Sleep 2s

# (send an email to demo-inbox@mailbust.com here to show live arrival)
Sleep 5s

Down
Sleep 1s
Enter
Sleep 3s
Escape
Sleep 1s

Type "q"
Sleep 1s
