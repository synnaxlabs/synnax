# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

"""Takes the headers of a C++ library without linking it."""

load("@rules_cc//cc/common:cc_info.bzl", "CcInfo")

def _cc_headers_impl(ctx):
    return [CcInfo(compilation_context = ctx.attr.lib[CcInfo].compilation_context)]

cc_headers = rule(
    implementation = _cc_headers_impl,
    attrs = {"lib": attr.label(providers = [CcInfo])},
    doc = "Provides the headers of lib to dependents without linking lib.",
)
