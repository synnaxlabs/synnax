# Go Naming Conventions

## Constructors

`New` -> Pure constructor. No side effects. No goroutines. Should not accept a context.

`Open` -> Construction of an entity that requires an `io.Closer`.

`Wrap` -> Construction of entity whose primary purpose is to extend/adapt the interface of another entity.
