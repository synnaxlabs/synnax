#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from freighter import FileTransport
from synnax import ontology
from synnax.project import Key as ProjectKey
from synnax.project import ontology_id as project_ontology_id
from x.fs import FilePath


class Client:
    """Imports and exports resources from and to the Core.

    Each call moves exactly one envelope, streamed to or from disk: ``import_`` streams
    a file from disk, and ``export`` streams the response straight into a destination
    file.
    """

    _file_transport: FileTransport

    def __init__(self, file_transport: FileTransport) -> None:
        self._file_transport = file_transport

    def import_(
        self,
        source: FilePath,
        project: ProjectKey | None = None,
    ) -> ontology.ID:
        """Imports the resource at source and returns its new ontology ID.

        The source file's name travels with the request: when the file's contents carry
        no top-level ``name`` field, the Core names the imported resource after the
        file, with the extension stripped. A ``name`` in the file always wins.

        :param source: a file path streamed from disk.
        :param project: the key of the project to create the imported resource under.
            The Core creates the resource and its project relationship in a single
            transaction, so a parenting failure rolls the import back.
        :returns: the new resource's ontology ID.
        """
        params = (
            {"project": str(project_ontology_id(project))}
            if project is not None
            else None
        )
        return self._file_transport.upload("/imex/import", source, ontology.ID, params)

    def export(self, id: ontology.ID, dest: FilePath) -> None:
        """Exports the resource identified by id, streaming it into dest.

        The response body is streamed straight into dest — the on-disk format is driven
        by the destination's extension.

        :param id: the ontology ID of the resource to export.
        :param dest: a file path to stream the response body into.
        """
        self._file_transport.download("/imex/export", id, dest)
