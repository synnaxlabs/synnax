"""Module extension for fetching the ESI registry blob from GitHub releases."""

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_file")

def _esi_impl(ctx):
    http_file(
        name = "esi_registry_blob",
        urls = [
            "https://github.com/synnaxlabs/esi/releases/download/latest/registry_blob.inc",
        ],
        sha256 = "fd1a5dae9d00efa9bfff17cc94db1752b0c736b9a5048f9496f6fe2265584d25",
        downloaded_file_path = "registry_blob.inc",
    )

esi = module_extension(implementation = _esi_impl)
