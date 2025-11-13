#pragma once

#include <cstdint>
#include <memory>

#include "x/cpp/telem/series.h"

namespace arc::wasm {
inline uint64_t value_at(const std::shared_ptr<telem::Series> &s, const int i) {
    switch (s->data_type().density()) {
        case 1:
            return s->at<uint8_t>(i);
        case 2:
            return s->at<uint16_t>(i);
        case 4: {
            if (const auto dt = s->data_type(); dt == telem::FLOAT32_T) {
                const auto f = s->at<float>(i);
                uint32_t bits;
                memcpy(&bits, &f, sizeof(float));
                return bits;
            }
            return s->at<uint32_t>(i);
        }
        case 8: {
            if (const auto dt = s->data_type(); dt == telem::FLOAT64_T) {
                const auto d = s->at<double>(i);
                uint64_t bits;
                memcpy(&bits, &d, sizeof(double));
                return bits;
            }
            return static_cast<uint64_t>(s->at<int64_t>(i));
        }
        default:
            return 0;
    }
}

inline void
set_value_at(const std::shared_ptr<telem::Series> &s, const int i, const uint64_t v) {
    switch (const auto dt = s->data_type(); dt.density()) {
        case 1:
            s->set(i, static_cast<uint8_t>(v));
            break;
        case 2:
            s->set(i, static_cast<uint16_t>(v));
            break;
        case 4: {
            if (dt == telem::FLOAT32_T) {
                const auto bits = static_cast<uint32_t>(v);
                float f;
                memcpy(&f, &bits, sizeof(float));
                s->set(i, f);
            } else
                s->set(i, static_cast<uint32_t>(v));
            break;
        }
        case 8: {
            if (dt == telem::FLOAT64_T) {
                double d;
                memcpy(&d, &v, sizeof(double));
                s->set(i, d);
            } else
                s->set(i, static_cast<int64_t>(v));
            break;
        }
    }
}

}
