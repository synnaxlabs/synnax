// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <string_view>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/json/text.h"

namespace x::json {
std::pair<Pointer, errors::Error> Pointer::parse(const std::string_view text) {
    if (text.empty()) return {Pointer{}, errors::NIL};
    if (text[0] != '/')
        return {
            Pointer{},
            errors::Error(
                errors::VALIDATION,
                "json pointer " + quote(text) + R"( must be empty or start with "/")"
            ),
        };
    Pointer p;
    std::string token;
    for (std::size_t i = 1; i <= text.size(); i++) {
        if (i == text.size() || text[i] == '/') {
            p.tokens.push_back(std::move(token));
            token.clear();
            continue;
        }
        if (text[i] != '~') {
            token += text[i];
            continue;
        }
        const char next = i + 1 < text.size() ? text[i + 1] : '\0';
        if (next != '0' && next != '1')
            return {
                Pointer{},
                errors::Error(
                    errors::VALIDATION,
                    "json pointer " + quote(text) +
                        R"( is malformed: "~" must be followed by "0" or "1")"
                ),
            };
        token += next == '0' ? '~' : '/';
        i++;
    }
    return {std::move(p), errors::NIL};
}

std::string Pointer::to_string() const {
    std::string out;
    for (const auto &token: this->tokens) {
        out += '/';
        for (const char c: token) {
            if (c == '~')
                out += "~0";
            else if (c == '/')
                out += "~1";
            else
                out += c;
        }
    }
    return out;
}
}
