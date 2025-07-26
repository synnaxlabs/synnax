This package is a copy of the `encoding/csv` package from the Go standard library.

The standard library package will directly modify data inside CSVs, not just the
structure, by converting carriage returns and line feeds when writing strings to a CSV.

Currently, the writer has been modified so that writing `[][]string`s to the writer will
not modify any of the actual strings.

The reader has not been modified and still converts carriage returns and line feeds when
reading data from a CSV. However, Synnax does not currently use the reader as we don't
yet ingest CSV data into the server.
