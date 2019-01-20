test:
	go test ./ -v -count=1 -cover

bench:
	go test -benchmem ./ -run=^$$ -bench .