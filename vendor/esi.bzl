# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

"""Module extension for fetching the ESI registry blob from GitHub releases."""

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_file")

def _esi_impl(_):
    http_file(
        name = "esi_registry_blob",
        urls = [
            "https://github.com/synnaxlabs/esi/releases/download/latest/registry_blob.inc",
        ],
        sha256 = "fd1a5dae9d00efa9bfff17cc94db1752b0c736b9a5048f9496f6fe2265584d25",
        downloaded_file_path = "registry_blob.inc",
    )

esi = module_extension(implementation = _esi_impl)
