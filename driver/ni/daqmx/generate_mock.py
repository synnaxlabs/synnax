#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Generates mock.h from api.h.

Parses every virtual method declared on the DAQmx API interface in api.h and emits
MockAPI, a subclass whose methods all succeed (return 0) and record the call name and
its arguments as JSON for assertion in tests. Scalars are recorded by value and char
arrays as strings; other arrays and pointers are recorded as "<array>" and "<ptr>"
placeholders, and TaskHandle arguments are skipped.

Regenerate whenever api.h changes; never edit mock.h by hand.

Usage: python3 driver/ni/daqmx/generate_mock.py && clang-format -i driver/ni/daqmx/mock.h
"""

import re
from pathlib import Path

DIR = Path(__file__).parent
API = (DIR / "api.h").read_text()

HEADER = """\
// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Code generated from api.h by generate_mock.py. Regenerate when api.h changes.

#pragma once

#include <string>
#include <vector>

#include "nlohmann/json.hpp"

#include "driver/ni/daqmx/api.h"

namespace driver::ni::daqmx {
/// @brief a mock DAQmx API that records every call and its scalar/string
/// arguments as JSON for assertion in tests. All calls succeed.
class MockAPI final : public API {
public:
    /// @brief one entry per API call, in call order: {"fn": name, "args": {...}}.
    std::vector<nlohmann::json> calls;

    /// @brief returns the recorded calls to the given function, in order.
    [[nodiscard]] std::vector<nlohmann::json> calls_to(const std::string &fn) const {
        std::vector<nlohmann::json> out;
        for (const auto &c: calls)
            if (c["fn"] == fn) out.push_back(c);
        return out;
    }
"""

FOOTER = """};
}
"""


def parse_param(param: str):
    param = " ".join(param.split())
    is_array = param.endswith("[]")
    if is_array:
        param = param[:-2].strip()
    is_ptr = "*" in param
    parts = param.replace("*", " ").split()
    name = parts[-1]
    typ = " ".join(parts[:-1])
    return typ, name, is_array, is_ptr


def record_line(typ: str, name: str, is_array: bool, is_ptr: bool):
    base = typ.removeprefix("const ").strip()
    if base == "TaskHandle":
        return None
    if is_array and base == "char":
        return (
            f'        if ({name} != nullptr) args["{name}"] = std::string({name});\n'
        )
    if is_array:
        return f'        if ({name} != nullptr) args["{name}"] = "<array>";\n'
    if (
        is_ptr
        or base == "void"
        or base == "CVIAbsoluteTime"
        or base.endswith("CallbackPtr")
    ):
        return f'        args["{name}"] = "<ptr>";\n'
    return f'        args["{name}"] = {name};\n'


def emit(name: str, params: str):
    parsed = [parse_param(p) for p in params.split(",")] if params.strip() else []
    sig_params = ", ".join(
        p.strip().replace("\n", " ") for p in params.split(",") if p.strip()
    )
    sig_params = " ".join(sig_params.split())
    out = f"    int32 {name}({sig_params}) override {{\n"
    out += "        nlohmann::json args = nlohmann::json::object();\n"
    recorded = [
        line
        for typ, pname, is_array, is_ptr in parsed
        if (line := record_line(typ, pname, is_array, is_ptr)) is not None
    ]
    out += "".join(recorded)
    if not recorded:
        out += "\n"
    out += f'        this->calls.push_back({{{{"fn", "{name}"}}, {{"args", args}}}});\n'
    out += "        return 0;\n"
    out += "    }\n\n"
    return out


methods = re.findall(r"virtual\s+int32\s+(\w+)\(([^)]*)\)\s*=\s*0;", API, re.DOTALL)
body = "".join(emit(name, params) for name, params in methods)
(DIR / "mock.h").write_text(HEADER + "\n" + body.rstrip("\n") + "\n" + FOOTER)
print(f"generated {len(methods)} methods")
