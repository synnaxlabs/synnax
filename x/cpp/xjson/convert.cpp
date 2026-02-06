// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


#include "x/cpp/xjson/convert.h"

namespace xjson {

template<typename T>
ReadConverter make_number_reader() {
    return [](const nlohmann::json &value)
               -> std::pair<telem::Series, xerrors::Error> {
        const auto series = telem::Series(static_cast<T>(value.get<double>()));
        return std::pair<telem::Series, xerrors::Error>(series, xerrors::NIL);
    };
}

template<typename T>
ReadConverter make_strict_number_reader() {
    return [](const nlohmann::json &value)
               -> std::pair<telem::Series, xerrors::Error> {
        const double v = value.get<double>();
        if constexpr (std::is_integral_v<T>) {
            if (v != std::trunc(v))
                return {telem::Series(static_cast<T>(0)), TRUNCATION_ERR};
            if (v < static_cast<double>(std::numeric_limits<T>::min()) ||
                v > static_cast<double>(std::numeric_limits<T>::max()))
                return {telem::Series(static_cast<T>(0)), OVERFLOW_ERR};
        }
        return {telem::Series(static_cast<T>(v)), xerrors::NIL};
    };
}

template<typename T>
ReadConverter make_bool_numeric_reader() {
    return [](const nlohmann::json &value)
               -> std::pair<telem::Series, xerrors::Error> {
        return {
            telem::Series(static_cast<T>(value.get<bool>() ? 1 : 0)),
            xerrors::NIL
        };
    };
}

std::pair<ReadConverter, xerrors::Error>
resolve_read_converter(
    xjson::Type json_type,
    const telem::DataType &target_type,
    bool strict
) {
    // Any → String
    if (target_type == telem::STRING_T) {
        if (json_type == xjson::Type::Number)
            return {[](const nlohmann::json &value)
                        -> std::pair<telem::Series, xerrors::Error> {
                return {telem::Series(value.dump()), xerrors::NIL};
            }, xerrors::NIL};
        if (json_type == xjson::Type::String)
            return {[](const nlohmann::json &value)
                        -> std::pair<telem::Series, xerrors::Error> {
                return {
                    telem::Series(value.get<std::string>()),
                    xerrors::NIL
                };
            }, xerrors::NIL};
        if (json_type == xjson::Type::Boolean)
            return {[](const nlohmann::json &value)
                        -> std::pair<telem::Series, xerrors::Error> {
                return {
                    telem::Series(
                        std::string(value.get<bool>() ? "true" : "false")
                    ),
                    xerrors::NIL
                };
            }, xerrors::NIL};
    }

    // Boolean → Numeric
    if (json_type == xjson::Type::Boolean) {
        if (target_type == telem::FLOAT64_T)
            return {make_bool_numeric_reader<double>(), xerrors::NIL};
        if (target_type == telem::FLOAT32_T)
            return {make_bool_numeric_reader<float>(), xerrors::NIL};
        if (target_type == telem::INT64_T)
            return {make_bool_numeric_reader<int64_t>(), xerrors::NIL};
        if (target_type == telem::INT32_T)
            return {make_bool_numeric_reader<int32_t>(), xerrors::NIL};
        if (target_type == telem::INT16_T)
            return {make_bool_numeric_reader<int16_t>(), xerrors::NIL};
        if (target_type == telem::INT8_T)
            return {make_bool_numeric_reader<int8_t>(), xerrors::NIL};
        if (target_type == telem::UINT64_T)
            return {make_bool_numeric_reader<uint64_t>(), xerrors::NIL};
        if (target_type == telem::UINT32_T)
            return {make_bool_numeric_reader<uint32_t>(), xerrors::NIL};
        if (target_type == telem::UINT16_T)
            return {make_bool_numeric_reader<uint16_t>(), xerrors::NIL};
        if (target_type == telem::UINT8_T)
            return {make_bool_numeric_reader<uint8_t>(), xerrors::NIL};
    }

    // Number → Numeric
    if (json_type == xjson::Type::Number) {
        if (target_type == telem::FLOAT64_T)
            return {make_number_reader<double>(), xerrors::NIL};
        if (target_type == telem::FLOAT32_T)
            return {make_number_reader<float>(), xerrors::NIL};
        if (strict) {
            if (target_type == telem::INT64_T)
                return {make_strict_number_reader<int64_t>(), xerrors::NIL};
            if (target_type == telem::INT32_T)
                return {make_strict_number_reader<int32_t>(), xerrors::NIL};
            if (target_type == telem::INT16_T)
                return {make_strict_number_reader<int16_t>(), xerrors::NIL};
            if (target_type == telem::INT8_T)
                return {make_strict_number_reader<int8_t>(), xerrors::NIL};
            if (target_type == telem::UINT64_T)
                return {make_strict_number_reader<uint64_t>(), xerrors::NIL};
            if (target_type == telem::UINT32_T)
                return {make_strict_number_reader<uint32_t>(), xerrors::NIL};
            if (target_type == telem::UINT16_T)
                return {make_strict_number_reader<uint16_t>(), xerrors::NIL};
            if (target_type == telem::UINT8_T)
                return {make_strict_number_reader<uint8_t>(), xerrors::NIL};
        } else {
            if (target_type == telem::INT64_T)
                return {make_number_reader<int64_t>(), xerrors::NIL};
            if (target_type == telem::INT32_T)
                return {make_number_reader<int32_t>(), xerrors::NIL};
            if (target_type == telem::INT16_T)
                return {make_number_reader<int16_t>(), xerrors::NIL};
            if (target_type == telem::INT8_T)
                return {make_number_reader<int8_t>(), xerrors::NIL};
            if (target_type == telem::UINT64_T)
                return {make_number_reader<uint64_t>(), xerrors::NIL};
            if (target_type == telem::UINT32_T)
                return {make_number_reader<uint32_t>(), xerrors::NIL};
            if (target_type == telem::UINT16_T)
                return {make_number_reader<uint16_t>(), xerrors::NIL};
            if (target_type == telem::UINT8_T)
                return {make_number_reader<uint8_t>(), xerrors::NIL};
        }
    }
    return {nullptr, UNSUPPORTED_ERR};
}

std::pair<nlohmann::json, xerrors::Error>
from_sample_value(const telem::SampleValue &value, xjson::Type target) {
    return std::visit(
        [target](const auto &v) -> std::pair<nlohmann::json, xerrors::Error> {
            using T = std::decay_t<decltype(v)>;
            if constexpr (std::is_same_v<T, std::string>) {
                if (target == xjson::Type::String)
                    return {nlohmann::json(v), xerrors::NIL};
                return {nlohmann::json(), UNSUPPORTED_ERR};
            } else if constexpr (std::is_same_v<T, telem::TimeStamp>) {
                return {nlohmann::json(), UNSUPPORTED_ERR};
            } else {
                switch (target) {
                    case xjson::Type::Number:
                        return {nlohmann::json(v), xerrors::NIL};
                    case xjson::Type::String: {
                        auto s = nlohmann::json(v).dump();
                        if constexpr (std::is_floating_point_v<T>) {
                            if (s.find('.') != std::string::npos) {
                                s.erase(s.find_last_not_of('0') + 1);
                                if (s.back() == '.') s.pop_back();
                            }
                        }
                        return {nlohmann::json(s), xerrors::NIL};
                    }
                    case xjson::Type::Boolean:
                        return {nlohmann::json(v != 0), xerrors::NIL};
                }
                return {nlohmann::json(), UNSUPPORTED_ERR};
            }
        },
        value
    );
}

xerrors::Error
check_from_sample_value(const telem::DataType &type, xjson::Type target) {
    if (type == telem::STRING_T)
        return target == xjson::Type::String ? xerrors::NIL : UNSUPPORTED_ERR;
    if (type == telem::FLOAT64_T || type == telem::FLOAT32_T ||
        type == telem::INT64_T || type == telem::INT32_T ||
        type == telem::INT16_T || type == telem::INT8_T ||
        type == telem::UINT64_T || type == telem::UINT32_T ||
        type == telem::UINT16_T || type == telem::UINT8_T)
        return xerrors::NIL;
    return UNSUPPORTED_ERR;
}

nlohmann::json zero_value(xjson::Type format) {
    switch (format) {
        case xjson::Type::Number: return 0;
        case xjson::Type::String: return "";
        case xjson::Type::Boolean: return false;
    }
    return nullptr;
}

}
