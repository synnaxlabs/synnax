// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <cctype>
#include <cstddef>
#include <cstdint>
#include <limits>
#include <optional>
#include <string>
#include <string_view>

#include "x/cpp/date/date.h"
#include "x/cpp/tz/tz.h"

namespace x::tz {
namespace {
struct Entry {
    std::string_view name;
    size_t offset;
    size_t size;
};

// Defines DATA and ENTRIES.
#include "x/cpp/tz/zoneinfo.inc"

constexpr int64_t SECONDS_PER_MINUTE = 60;
constexpr int64_t SECONDS_PER_HOUR = 60 * SECONDS_PER_MINUTE;
constexpr int64_t SECONDS_PER_DAY = 24 * SECONDS_PER_HOUR;

const x::errors::Error BAD_DATA(x::errors::VALIDATION, "malformed time zone data");

std::string lower(std::string_view s) {
    std::string out(s);
    for (auto &c: out)
        c = static_cast<char>(std::tolower(static_cast<unsigned char>(c)));
    return out;
}

/// @brief reads big-endian fields from TZif data. A read past the end sets failed and
/// returns nothing.
class Reader {
    std::string_view data;
    size_t pos = 0;

public:
    bool failed = false;

    explicit Reader(const std::string_view data): data(data) {}

    std::string_view read(const size_t n) {
        if (n > this->data.size() - this->pos) {
            this->failed = true;
            this->pos = this->data.size();
            return {};
        }
        const auto out = this->data.substr(this->pos, n);
        this->pos += n;
        return out;
    }

    std::optional<uint32_t> big4() {
        const auto b = this->read(4);
        if (b.size() != 4) return std::nullopt;
        uint32_t v = 0;
        for (const char c: b)
            v = v << 8 | static_cast<unsigned char>(c);
        return v;
    }

    std::optional<uint64_t> big8() {
        const auto b = this->read(8);
        if (b.size() != 8) return std::nullopt;
        uint64_t v = 0;
        for (const char c: b)
            v = v << 8 | static_cast<unsigned char>(c);
        return v;
    }

    [[nodiscard]] std::string_view rest() const { return this->data.substr(this->pos); }
};

/// @brief returns the text in s up to its first NUL.
std::string until_nul(const std::string_view s) {
    return std::string(s.substr(0, s.find('\0')));
}

/// @brief returns the days since the Unix epoch of the first day of month in year. A
/// month of 13 is January of the next year.
int64_t first_of(const int64_t year, const int64_t month) {
    if (month == 13)
        return date::days_from_civil({static_cast<uint16_t>(year + 1), 1, 1});
    return date::days_from_civil(
        {static_cast<uint16_t>(year), static_cast<uint8_t>(month), 1}
    );
}

// The rule string parser below mirrors tzset in Go's time package.

/// @brief reads a number from min to max at the start of s.
std::optional<int64_t>
rule_num(std::string_view &s, const int64_t min, const int64_t max) {
    if (s.empty()) return std::nullopt;
    int64_t num = 0;
    size_t i = 0;
    for (; i < s.size(); i++) {
        const char c = s[i];
        if (c < '0' || c > '9') {
            if (i == 0 || num < min) return std::nullopt;
            s = s.substr(i);
            return num;
        }
        num = num * 10 + (c - '0');
        if (num > max) return std::nullopt;
    }
    if (num < min) return std::nullopt;
    s = {};
    return num;
}

/// @brief reads a zone name at the start of s, stripping the angle brackets of a
/// quoted name.
std::optional<std::string> rule_name(std::string_view &s) {
    if (s.empty()) return std::nullopt;
    if (s[0] != '<') {
        for (size_t i = 0; i < s.size(); i++) {
            const char c = s[i];
            if ((c >= '0' && c <= '9') || c == ',' || c == '-' || c == '+') {
                if (i < 3) return std::nullopt;
                std::string name(s.substr(0, i));
                s = s.substr(i);
                return name;
            }
        }
        if (s.size() < 3) return std::nullopt;
        std::string name(s);
        s = {};
        return name;
    }
    for (size_t i = 0; i < s.size(); i++)
        if (s[i] == '>') {
            std::string name(s.substr(1, i - 1));
            s = s.substr(i + 1);
            return name;
        }
    return std::nullopt;
}

/// @brief reads a [+-]hh[:mm[:ss]] offset at the start of s, in seconds.
std::optional<int64_t> rule_offset(std::string_view &s) {
    if (s.empty()) return std::nullopt;
    bool neg = false;
    if (s[0] == '+')
        s = s.substr(1);
    else if (s[0] == '-') {
        s = s.substr(1);
        neg = true;
    }
    const auto hours = rule_num(s, 0, int64_t{24} * 7);
    if (!hours) return std::nullopt;
    int64_t off = *hours * SECONDS_PER_HOUR;
    if (s.empty() || s[0] != ':') return neg ? -off : off;
    s = s.substr(1);
    const auto mins = rule_num(s, 0, 59);
    if (!mins) return std::nullopt;
    off += *mins * SECONDS_PER_MINUTE;
    if (s.empty() || s[0] != ':') return neg ? -off : off;
    s = s.substr(1);
    const auto secs = rule_num(s, 0, 59);
    if (!secs) return std::nullopt;
    off += *secs;
    return neg ? -off : off;
}

enum class RuleKind { Julian, DayOfYear, MonthWeekDay };

struct Rule {
    RuleKind kind = RuleKind::Julian;
    int64_t day = 0;
    int64_t week = 0;
    int64_t month = 0;
    int64_t time = 0;
};

/// @brief reads a Jn, n, or Mm.w.d date, with an optional /time, at the start of s.
std::optional<Rule> rule_date(std::string_view &s) {
    if (s.empty()) return std::nullopt;
    Rule r;
    if (s[0] == 'J') {
        s = s.substr(1);
        const auto day = rule_num(s, 1, 365);
        if (!day) return std::nullopt;
        r.kind = RuleKind::Julian;
        r.day = *day;
    } else if (s[0] == 'M') {
        s = s.substr(1);
        const auto month = rule_num(s, 1, 12);
        if (!month || s.empty() || s[0] != '.') return std::nullopt;
        s = s.substr(1);
        const auto week = rule_num(s, 1, 5);
        if (!week || s.empty() || s[0] != '.') return std::nullopt;
        s = s.substr(1);
        const auto day = rule_num(s, 0, 6);
        if (!day) return std::nullopt;
        r.kind = RuleKind::MonthWeekDay;
        r.day = *day;
        r.week = *week;
        r.month = *month;
    } else {
        const auto day = rule_num(s, 0, 365);
        if (!day) return std::nullopt;
        r.kind = RuleKind::DayOfYear;
        r.day = *day;
    }
    if (s.empty() || s[0] != '/') {
        r.time = 2 * SECONDS_PER_HOUR;
        return r;
    }
    s = s.substr(1);
    const auto time = rule_offset(s);
    if (!time) return std::nullopt;
    r.time = *time;
    return r;
}

/// @brief returns the seconds from the start of year at which r takes effect, for a
/// zone off seconds east of UTC.
int64_t rule_time(const int64_t year, const Rule &r, const int64_t off) {
    const int64_t jan1 = first_of(year, 1);
    int64_t s = 0;
    switch (r.kind) {
        case RuleKind::Julian:
            s = (r.day - 1) * SECONDS_PER_DAY;
            // A Julian day never counts February 29.
            if (first_of(year, 3) - first_of(year, 2) == 29 && r.day >= 60)
                s += SECONDS_PER_DAY;
            break;
        case RuleKind::DayOfYear:
            s = r.day * SECONDS_PER_DAY;
            break;
        case RuleKind::MonthWeekDay: {
            const int64_t first = first_of(year, r.month);
            const int64_t length = first_of(year, r.month + 1) - first;
            int64_t d = r.day - date::weekday_from_days(first);
            if (d < 0) d += 7;
            for (int64_t i = 1; i < r.week; i++) {
                if (d + 7 >= length) break;
                d += 7;
            }
            s = (first - jan1 + d) * SECONDS_PER_DAY;
            break;
        }
    }
    return s + r.time - off;
}

/// @brief returns the local time that the POSIX rule string s gives at sec, or nothing
/// when s is malformed.
std::optional<Info> rule_info(std::string_view s, const int64_t sec) {
    auto std_name = rule_name(s);
    if (!std_name) return std::nullopt;
    auto std_offset = rule_offset(s);
    if (!std_offset) return std::nullopt;
    // A rule string's offsets are added to local time to get UTC.
    *std_offset = -*std_offset;
    if (s.empty() || s[0] == ',')
        return Info{static_cast<int32_t>(*std_offset), std::move(*std_name)};

    auto dst_name = rule_name(s);
    if (!dst_name) return std::nullopt;
    int64_t dst_offset = 0;
    if (s.empty() || s[0] == ',')
        dst_offset = *std_offset + SECONDS_PER_HOUR;
    else {
        const auto off = rule_offset(s);
        if (!off) return std::nullopt;
        dst_offset = -*off;
    }

    // The default daylight rules of tzcode.
    if (s.empty()) s = ",M3.2.0,M11.1.0";
    if (s[0] != ',' && s[0] != ';') return std::nullopt;
    s = s.substr(1);
    const auto start = rule_date(s);
    if (!start || s.empty() || s[0] != ',') return std::nullopt;
    s = s.substr(1);
    const auto end = rule_date(s);
    if (!end || !s.empty()) return std::nullopt;

    int64_t days = sec / SECONDS_PER_DAY;
    if (sec % SECONDS_PER_DAY < 0) days--;
    const date::Date d = date::civil_from_days(static_cast<int32_t>(days));
    const int64_t year = d.year;
    const int64_t yday = days - date::days_from_civil({d.year, 1, 1}) + 1;
    // Go's tzset truncates this remainder toward zero, and so does this.
    const int64_t ysec = (yday - 1) * SECONDS_PER_DAY + sec % SECONDS_PER_DAY;

    int64_t start_sec = rule_time(year, *start, *std_offset);
    int64_t end_sec = rule_time(year, *end, dst_offset);
    auto std_info = Info{static_cast<int32_t>(*std_offset), std::move(*std_name)};
    auto dst_info = Info{static_cast<int32_t>(dst_offset), std::move(*dst_name)};
    if (end_sec < start_sec) {
        std::swap(start_sec, end_sec);
        std::swap(std_info, dst_info);
    }
    if (ysec < start_sec || ysec >= end_sec) return std_info;
    return dst_info;
}
}

std::pair<std::shared_ptr<const Zone>, x::errors::Error>
Zone::parse(std::string name, const std::string_view data) {
    Reader d(data);
    if (d.read(4) != "TZif") return {nullptr, BAD_DATA};
    const auto header = d.read(16);
    if (header.size() != 16) return {nullptr, BAD_DATA};
    int version = 0;
    switch (header[0]) {
        case '\0':
            version = 1;
            break;
        case '2':
            version = 2;
            break;
        case '3':
            version = 3;
            break;
        default:
            return {nullptr, BAD_DATA};
    }

    enum { UTC_LOCAL, STD_WALL, LEAP, TIME, TYPE, CHAR };
    std::array<size_t, 6> n{};
    const auto read_counts = [&] {
        for (auto &count: n) {
            const auto v = d.big4();
            if (!v) return false;
            count = *v;
        }
        return true;
    };
    if (!read_counts()) return {nullptr, BAD_DATA};

    // Versions 2 and 3 repeat the data in 64-bit form after the 32-bit form.
    const bool is64 = version > 1;
    if (is64) {
        d.read(
            n[TIME] * 4 + n[TIME] + n[TYPE] * 6 + n[CHAR] + n[LEAP] * 8 + n[STD_WALL] +
            n[UTC_LOCAL] + 4 + 16
        );
        if (!read_counts()) return {nullptr, BAD_DATA};
    }
    const size_t size = is64 ? 8 : 4;

    Reader times(d.read(n[TIME] * size));
    const auto indices = d.read(n[TIME]);
    Reader type_data(d.read(n[TYPE] * 6));
    const auto abbreviations = d.read(n[CHAR]);
    d.read(n[LEAP] * (size + 4));
    d.read(n[STD_WALL]);
    d.read(n[UTC_LOCAL]);
    if (d.failed) return {nullptr, BAD_DATA};

    auto zone = std::make_shared<Zone>();
    zone->zone_name = std::move(name);
    const auto rest = d.rest();
    if (rest.size() > 2 && rest.front() == '\n' && rest.back() == '\n')
        zone->rule = std::string(rest.substr(1, rest.size() - 2));

    if (n[TYPE] == 0) return {nullptr, BAD_DATA};
    zone->types.reserve(n[TYPE]);
    for (size_t i = 0; i < n[TYPE]; i++) {
        const auto offset = type_data.big4();
        const auto flags = type_data.read(2);
        if (!offset || flags.size() != 2) return {nullptr, BAD_DATA};
        const auto index = static_cast<unsigned char>(flags[1]);
        if (index >= abbreviations.size()) return {nullptr, BAD_DATA};
        zone->types.push_back({
            .offset = static_cast<int32_t>(*offset),
            .dst = flags[0] != 0,
            .abbreviation = until_nul(abbreviations.substr(index)),
        });
    }

    zone->transitions.reserve(n[TIME]);
    for (size_t i = 0; i < n[TIME]; i++) {
        int64_t at = 0;
        if (is64) {
            const auto v = times.big8();
            if (!v) return {nullptr, BAD_DATA};
            at = static_cast<int64_t>(*v);
        } else {
            const auto v = times.big4();
            if (!v) return {nullptr, BAD_DATA};
            at = static_cast<int32_t>(*v);
        }
        const auto type = static_cast<unsigned char>(indices[i]);
        if (type >= zone->types.size()) return {nullptr, BAD_DATA};
        zone->transitions.push_back({.at = at, .type = type});
    }
    // A fixed zone gets one transition at the start of time.
    if (zone->transitions.empty())
        zone->transitions.push_back(
            {.at = std::numeric_limits<int64_t>::min(), .type = 0}
        );
    return {zone, x::errors::NIL};
}

size_t Zone::first_type() const {
    // Mirrors lookupFirstZone in Go's time package.
    bool first_used = false;
    for (const auto &t: this->transitions)
        if (t.type == 0) first_used = true;
    if (!first_used) return 0;
    if (!this->transitions.empty() && this->types[this->transitions[0].type].dst)
        for (int i = static_cast<int>(this->transitions[0].type) - 1; i >= 0; i--)
            if (!this->types[i].dst) return static_cast<size_t>(i);
    for (size_t i = 0; i < this->types.size(); i++)
        if (!this->types[i].dst) return i;
    return 0;
}

Info Zone::at(const int64_t unix_seconds) const {
    if (this->transitions.empty() || unix_seconds < this->transitions[0].at) {
        const auto &t = this->types[this->first_type()];
        return {t.offset, t.abbreviation};
    }
    size_t lo = 0;
    size_t hi = this->transitions.size();
    while (hi - lo > 1) {
        const size_t m = lo + (hi - lo) / 2;
        if (unix_seconds < this->transitions[m].at)
            hi = m;
        else
            lo = m;
    }
    if (lo == this->transitions.size() - 1 && !this->rule.empty())
        if (auto info = rule_info(this->rule, unix_seconds)) return std::move(*info);
    const auto &t = this->types[this->transitions[lo].type];
    return {t.offset, t.abbreviation};
}

Database::Database() {
    const std::string_view data(reinterpret_cast<const char *>(DATA), sizeof(DATA));
    for (const auto &e: ENTRIES)
        this->files.emplace(lower(e.name), File{e.name, data.substr(e.offset, e.size)});
}

std::pair<std::shared_ptr<const Zone>, x::errors::Error>
Database::load(const std::string_view name) {
    const auto key = lower(name);
    if (const auto it = this->zones.find(key); it != this->zones.end())
        return {it->second, x::errors::NIL};
    const auto it = this->files.find(key);
    if (it == this->files.end())
        return {
            nullptr,
            x::errors::Error(
                x::errors::NOT_FOUND,
                "time zone \"" + std::string(name) + "\""
            )
        };
    auto [zone, err] = Zone::parse(std::string(it->second.name), it->second.data);
    if (err) return {nullptr, err};
    this->zones.emplace(key, zone);
    return {zone, x::errors::NIL};
}

std::vector<std::string> Database::names() const {
    std::vector<std::string> out;
    out.reserve(std::size(ENTRIES));
    for (const auto &e: ENTRIES)
        out.emplace_back(e.name);
    return out;
}

}
