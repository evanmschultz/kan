Output .artifacts/vhs/regression_subtasks.gif

Set Shell "zsh"
Set FontSize 20
Set Width 1200
Set Height 720
Set Framerate 30

Type "rm -f /tmp/kan-vhs-regression-subtasks.db"
Enter
Sleep 300ms
Type "KAN_DB_PATH=/tmp/kan-vhs-regression-subtasks.db ./kan"
Enter
Sleep 1200ms

Type "n"
Type "Parent task"
Enter
Sleep 500ms

Type "s"
Type "Subtask one"
Enter
Sleep 500ms

Type "s"
Type "Subtask two"
Enter
Sleep 700ms

Type "i"
Sleep 1200ms
Type "i"
Sleep 500ms

Type "]"
Sleep 700ms
Type "i"
Sleep 1200ms
Type "i"
Sleep 500ms

Type "q"
Sleep 300ms
