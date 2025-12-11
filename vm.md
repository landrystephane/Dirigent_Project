sudo docker compose up
tmux new -d -s master_node 'cd cmd/master_node; sudo /usr/local/go/bin/go run main.go --config ./config.yaml'
tmux new -d -s data 'cd cmd/data_plane; sudo /usr/local/go/bin/go run main.go --config ./config.yaml'
tmux new -d -s worker 'cd cmd/worker_node && sudo env "PATH=$PATH" /usr/local/go/bin/go run main.go --config ./config.yaml'


tmux attach -t master_node
# detach without killing: Ctrl-b d