package control

type State[S comparable, R comparable] struct {
	Subject   S
	Resource  R
	Authority Authority
}
