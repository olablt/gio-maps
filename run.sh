# tmux new -s gio-maps
tmux send-keys -t gio-maps.0 C-c;
tmux send-keys -t gio-maps.0 C-l;
tmux send-keys -t gio-maps.0 "tmux clear-history" ENTER

# tmux send-keys -t gio-maps.0 "go run ." ENTER
tmux send-keys -t gio-maps.0 "go run ./apps/hello/" ENTER
