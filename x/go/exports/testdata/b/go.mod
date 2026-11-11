module example.com/b

go 1.27.4

require (
	example.com/a v0.0.0
	example.com/c v0.0.0
)

replace (
	example.com/a => ../a
	example.com/c => ../c
)
