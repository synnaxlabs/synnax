# Go Naming Conventions

## Constructors

<table>

<tr>
<th>Name</th>
<th>Meaning</th>
</tr>
<tr>
<td><code>New</code></td>
<td>Pure constructor: no side effects and no need for closure.</td>
</tr>
<tr>
<td><code>Open</code></td>
<td>Constructor for an entity that has side effects and/or forks goroutines. Entities
constructed with <code>Open</code> should implement <code>io.Closer</code>.
</td>
</tr>
<tr>
<tr>
<td><code>Wrap</code></td>
<td>Constructor for an entity whose primary purpose is to extend/adapt the interface
of another entity. `Wrap` also implies that the wrapper does not need to take control of the
wrapped entity's lifecycle.
</td>
</tr>
</table>
