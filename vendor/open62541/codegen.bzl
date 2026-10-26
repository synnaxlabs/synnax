# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

"""Runs one open62541 code generator as a Bazel action."""

def _generate_impl(ctx):
    stem = "%s/%s" % (ctx.outputs.outs[0].dirname, ctx.attr.stem)
    ctx.actions.run(
        executable = ctx.executable.tool,
        arguments = [
            ctx.expand_location(a, ctx.attr.srcs).replace("{out}", stem)
            for a in ctx.attr.arguments
        ],
        inputs = ctx.files.srcs,
        outputs = ctx.outputs.outs,
        mnemonic = "Open62541Generate",
        progress_message = "Generating %{label}",
    )

generate = rule(
    implementation = _generate_impl,
    doc = "Runs a generator that writes every file in outs from one path stem.",
    attrs = {
        "arguments": attr.string_list(
            doc = "Arguments. $(execpath) expands, and {out} becomes the stem path.",
        ),
        "outs": attr.output_list(mandatory = True),
        "srcs": attr.label_list(allow_files = True),
        "stem": attr.string(
            doc = "The generator's output path, relative to the directory of outs.",
            mandatory = True,
        ),
        "tool": attr.label(cfg = "exec", executable = True, mandatory = True),
    },
)
