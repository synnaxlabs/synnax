// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <cerrno>
#include <charconv>
#include <chrono>
#include <cstring>
#include <format>
#include <map>
#include <numeric>
#include <optional>
#include <string_view>
#include <system_error>

#include "absl/log/log.h"
#include <dirent.h>
#include <sys/stat.h>

#include "x/cpp/breaker/breaker.h"

#include "driver/ethercat/coe/coe.h"
#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/esi/settings.h"
#include "driver/ethercat/igh/master.h"
#include "driver/ethercat/master/dc.h"
#include "driver/ethercat/sii/sii.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::igh {
namespace {
/// @brief where the IgH kernel module lists its masters, as EtherCAT<n>.
constexpr auto SYSFS_PATH = "/sys/class/EtherCAT";
/// @brief the device of the first IgH master.
constexpr auto DEVICE_PATH = "/dev/EtherCAT0";

/// @brief the CoE identity object and the sub-index that holds the serial number.
constexpr uint16_t IDENTITY_INDEX = 0x1018;
constexpr uint8_t SERIAL_SUB_INDEX = 4;
/// @brief the largest serial number that the Core and the Console store exactly. A
/// rounded serial number would match no saved device, so a larger one counts as none.
constexpr uint64_t MAX_SERIAL = (uint64_t{1} << 53) - 1;

/// @brief returns why an SDO transfer failed with ret, the negative error code of the
/// IgH call, and abort_code, which IgH gives only with EIO.
errors::SDOFailure sdo_failure(const int ret, const uint32_t abort_code) {
    using Kind = errors::SDOFailure::Kind;
    switch (-ret) {
        case EIO:
            // IgH also gives EIO with no abort code when the slave does not answer.
            if (abort_code == 0) return {.kind = Kind::NO_REPLY};
            return {.kind = Kind::ABORTED, .abort_code = abort_code};
        // A slave with no CoE gives SOEM a mailbox error, but IgH refuses the transfer
        // itself.
        case EPROTO:
        case EPROTONOSUPPORT:
            return {.kind = Kind::MAILBOX_ERROR};
        case ENOBUFS:
            return {.kind = Kind::REPLY_TOO_LARGE};
        // Any other code, such as EINVAL for a slave that left the bus, means that the
        // transfer did not reach the slave.
        default:
            return {.kind = Kind::NO_REPLY};
    }
}

/// @brief reads index:sub_index of the slave at ring position as an unsigned value of
/// at most sizeof(T) bytes. Returns why the read failed when it fails.
template<typename T>
std::pair<T, std::optional<errors::SDOFailure>> upload(
    const API &api,
    ec_master_t *master,
    const uint16_t position,
    const uint16_t index,
    const uint8_t sub_index
) {
    std::array<uint8_t, sizeof(T)> data{};
    size_t size = 0;
    uint32_t abort_code = 0;
    if (const int ret = api.master_sdo_upload(
            master,
            position,
            index,
            sub_index,
            data.data(),
            data.size(),
            &size,
            &abort_code
        );
        ret < 0)
        return {0, sdo_failure(ret, abort_code)};
    T value = 0;
    for (size_t i = 0; i < size; i++)
        value |= static_cast<T>(data[i]) << (8 * i);
    return {value, std::nullopt};
}

/// @brief returns a read of the object dictionary of the slave at position.
coe::Read sdo_reader(const API &api, ec_master_t *master, const uint16_t position) {
    return [&api, master, position](const uint16_t index, const uint8_t sub_index)
               -> std::pair<uint32_t, x::errors::Error> {
        const auto [value, failure] = upload<uint32_t>(
            api,
            master,
            position,
            index,
            sub_index
        );
        if (failure) return {0, coe::read_error(index, sub_index, *failure)};
        return {value, x::errors::NIL};
    };
}

/// @brief reads the CoE serial number of slave, at ring position, when its SII serial
/// number is 0, as IOLITE modules have. IgH reads only the SII serial number.
void read_serial(
    const API &api,
    ec_master_t *master,
    const uint16_t position,
    slave::DiscoveryResult &slave
) {
    if (slave.properties.serial != 0) return;
    // CiA 301 gives the serial number 4 bytes, but IOLITE modules give it 8.
    const auto [serial, failure] = upload<uint64_t>(
        api,
        master,
        position,
        IDENTITY_INDEX,
        SERIAL_SUB_INDEX
    );
    if (!failure) {
        if (serial <= MAX_SERIAL) slave.properties.serial = serial;
        return;
    }
    // Only a slave that refuses the read for good has no CoE serial number.
    if (failure->permanent()) return;
    const auto err = coe::read_error(IDENTITY_INDEX, SERIAL_SUB_INDEX, *failure);
    slave.status.serial_error = err.data;
}

/// @brief names each entry of slave with the name and data type that the CoE
/// dictionary of the slave at ring position gives. IgH reads the dictionary of
/// object_count objects in the background after its scan, so an entry that it has not
/// read yet keeps its name.
void fill_dictionary_entries(
    const API &api,
    const unsigned int master_index,
    const uint16_t position,
    const uint16_t object_count,
    slave::Properties &slave
) {
    std::vector<ObjectInfo> objects;
    for (uint16_t i = 0; i < object_count; ++i) {
        ObjectInfo object;
        if (api.object_info(master_index, position, i, object) != 0) break;
        objects.push_back(std::move(object));
    }
    // A lookup of an entry that IgH has not read writes an error to the kernel log. IgH
    // reads the name of an object, then its entries, then the next object. IgH gives no
    // sign that it has read the final object, so it counts once it has a name.
    std::map<uint16_t, uint8_t> read;
    for (size_t i = 0; i < objects.size(); ++i) {
        const bool next_named = i + 1 < objects.size() ? !objects[i + 1].name.empty()
                                                       : i + 1 == object_count;
        if (!objects[i].name.empty() && next_named)
            read[objects[i].index] = objects[i].max_sub_index;
    }
    const auto fill = [&](std::vector<pdo::Properties> &pdos) {
        for (auto &pdo: pdos) {
            const auto object = read.find(pdo.index);
            if (pdo.is_padding() || object == read.end() ||
                pdo.sub_index > object->second)
                continue;
            EntryInfo entry;
            if (api.entry_info(
                    master_index,
                    position,
                    pdo.index,
                    pdo.sub_index,
                    entry
                ) != 0 ||
                entry.name.empty())
                continue;
            pdo.name = std::move(entry.name);
            pdo.data_type = telem::map_ethercat_to_synnax(
                static_cast<telem::DataType>(entry.data_type),
                pdo.bit_length
            );
        }
    };
    fill(slave.input_pdos);
    fill(slave.output_pdos);
}

/// @brief reads the PDO layout that the slave at position assigns.
std::pair<std::vector<pdo::Sync>, x::errors::Error>
read_layout(const API &api, ec_master_t *master, const uint16_t position) {
    return coe::read_layout(
        sdo_reader(api, master, position),
        EC_MAX_SYNC_MANAGERS,
        coe::OnError::FAIL
    );
}

/// @brief returns the reason for ret, the negative error code of an IgH call.
std::string reason(const int ret) {
    return std::generic_category().message(-ret);
}

/// @brief how long the master waits for IgH to read more of a CoE dictionary. IgH
/// starts to read the dictionary of a slave 3 s after the slave reaches PRE-OP.
const auto DICTIONARY_TIMEOUT = x::telem::SECOND * 5;
/// @brief the time between reads of the dictionaries while IgH reads them.
const auto DICTIONARY_POLL_INTERVAL = x::telem::MILLISECOND * 100;
/// @brief the time between reads of the slave states while the slaves move to OP.
const auto OP_POLL_INTERVAL = x::telem::MILLISECOND * 10;
/// @brief how long the master waits for IgH to finish a scan of the bus.
const auto SCAN_TIMEOUT = x::telem::SECOND * 10;
/// @brief the time between reads of the master state while IgH scans the bus.
const auto SCAN_POLL_INTERVAL = x::telem::MILLISECOND * 10;
/// @brief how long a reset can take to clear the AL error flags, since IgH resets in
/// the background.
const auto RESET_GRACE = x::telem::MILLISECOND * 500;
/// @brief the waits between the resets that recover() asks for while a slave stays
/// faulted.
const x::breaker::Config RESET_RETRY{
    .name = "igh_reset",
    .base_interval = x::telem::SECOND,
    .scale = 2,
    .max_interval = x::telem::SECOND * 30,
};

/// @brief returns the state that a slave in state moves to next on its way to OP.
std::string next_state(const slave::State state) {
    switch (state) {
        case slave::State::SAFE_OP:
            return "OP";
        case slave::State::PRE_OP:
            return "SAFE-OP";
        default:
            return "PRE-OP";
    }
}

/// @brief returns the IgH ring position, which counts from 0, of the slave at position.
uint16_t ring_position(const uint16_t position) {
    return position - 1;
}

/// @brief returns whether a slave in state is below PRE-OP. IgH aborts each SDO
/// transfer of a slave in INIT.
bool starting(const slave::State state) {
    return state == slave::State::INIT || state == slave::State::BOOT ||
           state == slave::State::UNKNOWN;
}

/// @brief what the master reads from the SII of a slave.
struct SII {
    /// @brief the DC modes of the slave.
    std::vector<dc::OpMode> op_modes;
    /// @brief whether IgH reads the CoE dictionary of the slave.
    bool dictionary = false;
};

/// @brief reads the SII of the slave at ring position.
/// @returns the SII and 0, or the negative error code of a failed read.
std::pair<SII, int>
read_sii(const API &api, const unsigned int master_index, const uint16_t position) {
    std::map<uint16_t, std::vector<uint8_t>> categories;
    const auto read = [&](const uint32_t offset, std::span<uint16_t> words) {
        // IgH rejects a read of no words.
        if (words.empty()) return 0;
        // A read past the last word that an offset can address ends the SII, as a read
        // past the end of the SII does.
        if (offset + words.size() > UINT16_MAX) return -EINVAL;
        return api.sii_read(master_index, position, offset, words);
    };
    for (uint32_t offset = sii::CATEGORIES;;) {
        // The SII that IgH holds ends at the type word of the end marker, and IgH logs
        // a kernel error for each read past its end.
        uint16_t type = 0;
        uint16_t size = 0;
        int ret = read(offset, std::span(&type, 1));
        if (ret == 0 && type == sii::END) break;
        if (ret == 0) ret = read(offset + 1, std::span(&size, 1));
        std::vector<uint16_t> words;
        if (ret == 0 &&
            (type == sii::STRINGS || type == sii::DC || type == sii::GENERAL)) {
            words.resize(size);
            ret = read(offset + 2, words);
        }
        // IgH rejects a read past the end of the SII with EINVAL, so a SII with no end
        // marker ends at its last whole category.
        if (ret == -EINVAL) break;
        if (ret != 0) return {{}, ret};
        for (const auto word: words) {
            categories[type].push_back(static_cast<uint8_t>(word & 0xFF));
            categories[type].push_back(static_cast<uint8_t>(word >> 8));
        }
        offset += 2 + size;
    }
    std::array<uint16_t, 1> protocols{};
    const int ret = read(sii::MAILBOX_PROTOCOLS, protocols);
    // A SII too short to hold the protocols lists none.
    if (ret != 0 && ret != -EINVAL) return {{}, ret};
    const bool coe = ret == 0 && (protocols[0] & sii::COE) != 0;
    const auto general = categories.find(sii::GENERAL);
    return {
        {
            .op_modes = sii::parse_op_modes(
                categories[sii::DC],
                sii::parse_strings(categories[sii::STRINGS])
            ),
            // IgH reads the dictionary unless the general category turns off SDO
            // information.
            .dictionary = coe && (general == categories.end() ||
                                  sii::parse_sdo_info(general->second)),
        },
        0
    };
}

/// @brief returns errors::SLAVE_CONFIG_ERROR when IgH would start the SYNC0 events of
/// slave off the whole SYNC0 cycles after master::DC_EPOCH that the engine expects. IgH
/// starts SYNC0 on whole spans of the SYNC0 cycle plus the SYNC1 register.
x::errors::Error
check_sync1(const slave::Properties &slave, const dc::Registers &regs) {
    if (!regs.marks_data() || regs.sync1_cycle % regs.sync0_cycle == 0)
        return x::errors::NIL;
    return x::errors::Error(
        errors::SLAVE_CONFIG_ERROR,
        std::format(
            "module {} ({}) cannot run its DC mode on the IgH master: SYNC1 comes {} "
            "after SYNC0, which is not a whole number of SYNC0 cycles of {}",
            slave.position,
            slave.name,
            x::telem::TimeSpan(regs.sync1_cycle).to_string(),
            x::telem::TimeSpan(regs.sync0_cycle).to_string()
        )
    );
}

/// @brief returns errors::NO_SLAVES_FOUND for the master named name at index, telling a
/// port with no link from slaves that do not answer.
x::errors::Error
no_slaves_error(const std::string &name, const unsigned int index, const bool link_up) {
    const auto found = "No EtherCAT modules found on " + name + ". ";
    if (link_up)
        return x::errors::Error(
            errors::NO_SLAVES_FOUND,
            found + "Check the cable and the power of the first module."
        );
    return x::errors::Error(
        errors::NO_LINK,
        found + std::format(
                    "Its network port has no link. Check the cable to the first "
                    "module, and that MASTER{}_DEVICE in the IgH configuration names "
                    "the port.",
                    index
                )
    );
}

/// @brief returns errors::ACTIVATION_ERROR for ret, the error of a failed reservation
/// of the master named name at index.
x::errors::Error
reserve_error(const std::string &name, const unsigned int index, const int ret) {
    if (ret == -EBUSY)
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            "another program uses " + name + ". Stop it, then start the task again"
        );
    if (ret == -ENODEV)
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            std::format(
                "no network port is attached to {}. Set MASTER{}_DEVICE in the IgH "
                "configuration to the MAC address of the port, then restart IgH "
                "with 'sudo systemctl restart ethercat'",
                name,
                index
            )
        );
    return x::errors::Error(
        errors::ACTIVATION_ERROR,
        "failed to reserve " + name + ": " + reason(ret)
    );
}

/// @brief returns the latest application time at or before now that is a whole number
/// of SYNC0 cycles of each slave in dc_sync whose SYNC0 events mark data.
/// @returns errors::SLAVE_CONFIG_ERROR when the cycles have no common multiple at or
/// before now.
std::pair<uint64_t, x::errors::Error>
first_app_time(const uint64_t now, const std::vector<dc::Registers> &dc_sync) {
    uint64_t multiple = 1;
    for (const auto &regs: dc_sync) {
        if (!regs.marks_data()) continue;
        const uint64_t cycle = regs.sync0_cycle;
        const auto factor = multiple / std::gcd(multiple, cycle);
        if (factor > now / cycle)
            return {
                0,
                x::errors::Error(
                    errors::SLAVE_CONFIG_ERROR,
                    "the SYNC0 cycles of the modules have no common multiple that the "
                    "IgH master can start them on"
                )
            };
        multiple = factor * cycle;
    }
    return {now / multiple * multiple, x::errors::NIL};
}

/// @brief returns t in nanoseconds since the steady clock epoch.
int64_t nanoseconds(const hs_clock::time_point t) {
    return std::chrono::duration_cast<std::chrono::nanoseconds>(t.time_since_epoch())
        .count();
}

/// @brief returns s as a decimal number, or nullopt when s holds anything else.
std::optional<unsigned int> parse_index(const std::string_view s) {
    unsigned int index = 0;
    const auto [end, ec] = std::from_chars(s.data(), s.data() + s.size(), index);
    if (ec != std::errc{} || end != s.data() + s.size()) return std::nullopt;
    return index;
}
}

Master::Master(std::shared_ptr<API> api, const unsigned int master_index, Config cfg):
    api(std::move(api)),
    master_index(master_index),
    cfg(std::move(cfg)),
    ec_master(nullptr),
    initialized(false),
    activated(false) {}

Master::~Master() {
    this->deactivate();
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error> Master::scan() {
    if (!this->initialized) {
        if (auto err = this->initialize()) return {{}, err};
    } else
        this->read_dictionaries();
    return {this->cached_slaves, x::errors::NIL};
}

void Master::read_dictionaries() {
    for (auto &slave: this->cached_slaves) {
        auto &props = slave.properties;
        const auto pos = ring_position(props.position);
        ec_slave_info_t info{};
        if (this->api->master_get_slave(this->ec_master, pos, &info) != 0) continue;
        fill_dictionary_entries(
            *this->api,
            this->master_index,
            pos,
            info.sdo_count,
            props
        );
        esi::fill_known_entries(props);
    }
}

x::errors::Error Master::initialize() {
    if (auto err = this->api->check_device(this->master_index)) return err;
    // Only a reserved master stops IgH from scanning the bus again, so changed() sees
    // the slaves that come and go until activate() reserves it.
    this->ec_master = this->api->open_master(this->master_index);
    if (!this->ec_master)
        return x::errors::Error(
            errors::MASTER_INIT_ERROR,
            "failed to open " + this->name() +
                ". Check that libethercat comes from the EtherLab version of the "
                "ec_master kernel module"
        );

    const auto release = [this](const x::errors::Error &err) {
        this->api->release_master(this->ec_master);
        this->ec_master = nullptr;
        return err;
    };
    const auto fail = [&release](const std::string &msg) {
        return release(x::errors::Error(errors::MASTER_INIT_ERROR, msg));
    };

    ec_master_info_t master_info{};
    const auto deadline = this->cfg.clock.steady() + SCAN_TIMEOUT.chrono();
    while (true) {
        if (this->api->master(this->ec_master, &master_info) < 0)
            return fail("failed to get master info");
        if (master_info.scan_busy == 0) break;
        if (this->cfg.clock.steady() >= deadline)
            return fail(
                "IgH did not finish its scan of the bus within " +
                SCAN_TIMEOUT.to_string()
            );
        this->cfg.clock.sleep(SCAN_POLL_INTERVAL);
    }
    if (master_info.slave_count == 0)
        return release(
            no_slaves_error(this->name(), this->master_index, master_info.link_up != 0)
        );

    this->cached_slaves.clear();
    this->cached_slaves.reserve(master_info.slave_count);
    this->dictionaries.clear();
    this->reference_clock.reset();

    for (unsigned int i = 0; i < master_info.slave_count; ++i) {
        ec_slave_info_t slave_info;
        if (const int ret = this->api
                                ->master_get_slave(this->ec_master, i, &slave_info);
            ret != 0)
            return fail(
                std::format("failed to read module {}: {}", i + 1, reason(ret))
            );
        slave::DiscoveryResult result{};
        result.properties.network = "igh:" + std::to_string(this->master_index);
        result.properties.position = static_cast<uint16_t>(i + 1);
        result.properties.vendor_id = slave_info.vendor_id;
        result.properties.product_code = slave_info.product_code;
        result.properties.revision = slave_info.revision_number;
        result.properties.serial = slave_info.serial_number;
        result.properties.name = slave_info.name;
        result.status.state = slave::from_al_state(slave_info.al_state);
        result.status.starting = starting(result.status.state);
        // IgH reads DC receive times only from slaves with a distributed clock, and it
        // makes the first of them the reference clock.
        if (!this->reference_clock && slave_info.ports[0].receive_time != 0)
            this->reference_clock = result.properties.position;
        for (uint8_t sm = 0; sm < slave_info.sync_count; ++sm)
            if (const auto err = this->append_scanned_pdos(result.properties, sm)) {
                result.properties.input_pdos.clear();
                result.properties.output_pdos.clear();
                result.status.pdo_discovery_error = "failed to read the PDO layout: " +
                                                    err.data;
                break;
            }
        esi::fill(result);
        auto [sii, ret] = read_sii(
            *this->api,
            this->master_index,
            static_cast<uint16_t>(i)
        );
        if (ret != 0)
            return fail(
                std::format(
                    "failed to read the SII of module {}: {}",
                    i + 1,
                    reason(ret)
                )
            );
        result.properties.op_modes = std::move(sii.op_modes);
        if (sii.dictionary) this->dictionaries.insert(result.properties.position);
        this->cached_slaves.push_back(std::move(result));
    }
    std::vector<slave::DiscoveryResult *> started;
    for (auto &slave: this->cached_slaves)
        if (!slave.status.starting) started.push_back(&slave);
    this->read_coe(started);
    this->read_dictionaries();
    slave::clear_duplicate_serials(this->cached_slaves);

    this->initialized = true;
    VLOG(1) << "[ethercat.igh] master " << this->master_index << " initialized with "
            << this->cached_slaves.size() << " slaves";
    return x::errors::NIL;
}

void Master::await_dictionaries(const std::vector<slave::DiscoveryResult *> &slaves) {
    struct Progress {
        uint16_t position;
        uint16_t listed = 0;
        uint16_t described = 0;
        std::chrono::steady_clock::time_point next_lookup;
        x::telem::TimeSpan lookup_gap = DICTIONARY_POLL_INTERVAL;
    };
    std::vector<Progress> pending;
    for (const auto *slave: slaves)
        if (this->dictionaries.contains(slave->properties.position))
            pending.push_back({.position = ring_position(slave->properties.position)});
    auto deadline = this->cfg.clock.steady() + DICTIONARY_TIMEOUT.chrono();
    while (!pending.empty()) {
        const auto now = this->cfg.clock.steady();
        bool progressed = false;
        // IgH lists the objects, then reads the description and then the entries of
        // each. A description can hold no name, so an object counts as described once
        // it or a later object has a name or sub-indices.
        std::erase_if(pending, [&](Progress &read) {
            ec_slave_info_t info{};
            if (this->api->master_get_slave(this->ec_master, read.position, &info) != 0)
                return false;
            if (info.sdo_count != read.listed) {
                read = {.position = read.position, .listed = info.sdo_count};
                progressed = true;
            }
            ObjectInfo object;
            for (uint16_t i = read.described; i < read.listed; ++i)
                if (this->api->object_info(
                        this->master_index,
                        read.position,
                        i,
                        object
                    ) == 0 &&
                    (!object.name.empty() || object.max_sub_index != 0)) {
                    read.described = i + 1;
                    progressed = true;
                }
            if (read.listed == 0 || read.described < read.listed ||
                now < read.next_lookup)
                return false;
            // IgH gives no sign that it has read the final entry, and it logs each
            // lookup of an entry it has not read, so the lookups back off.
            EntryInfo entry;
            if (this->api->object_info(
                    this->master_index,
                    read.position,
                    read.listed - 1,
                    object
                ) == 0 &&
                this->api->entry_info(
                    this->master_index,
                    read.position,
                    object.index,
                    object.max_sub_index,
                    entry
                ) == 0) {
                progressed = true;
                return true;
            }
            read.next_lookup = now + read.lookup_gap.chrono();
            read.lookup_gap = read.lookup_gap * 2;
            return false;
        });
        if (pending.empty()) return;
        if (progressed) deadline = now + DICTIONARY_TIMEOUT.chrono();
        if (now >= deadline) {
            for (const auto &read: pending)
                LOG(WARNING) << "[ethercat.igh] IgH read no more of the CoE dictionary "
                             << "of module " << read.position + 1 << " for "
                             << DICTIONARY_TIMEOUT;
            return;
        }
        this->cfg.clock.sleep(DICTIONARY_POLL_INTERVAL);
    }
}

void Master::read_coe(const std::vector<slave::DiscoveryResult *> &slaves) {
    this->await_dictionaries(slaves);
    for (auto *slave: slaves) {
        auto &props = slave->properties;
        const auto pos = ring_position(props.position);
        read_serial(*this->api, this->ec_master, pos, *slave);
        auto [settings, err] = esi::read_settings(
            props.vendor_id,
            props.product_code,
            props.revision,
            props.op_modes,
            sdo_reader(*this->api, this->ec_master, pos)
        );
        slave->status.settings_error = err.data;
        props.settings = std::move(settings);
    }
}

void Master::read_starting() {
    std::vector<slave::DiscoveryResult *> pending;
    for (auto &slave: this->cached_slaves)
        if (slave.status.starting) pending.push_back(&slave);
    if (pending.empty()) return;
    // IgH moves each slave to PRE-OP in the background while the master is open.
    const auto deadline = this->cfg.clock.steady() + this->cfg.pre_op_timeout.chrono();
    std::vector<slave::DiscoveryResult *> ready;
    while (true) {
        std::vector<slave::DiscoveryResult *> still;
        for (auto *slave: pending) {
            ec_slave_info_t info{};
            const auto pos = ring_position(slave->properties.position);
            const bool read = this->api
                                  ->master_get_slave(this->ec_master, pos, &info) == 0;
            const auto state = slave::from_al_state(info.al_state);
            if (!read || starting(state)) {
                still.push_back(slave);
                continue;
            }
            slave->status.state = state;
            slave->status.starting = false;
            ready.push_back(slave);
        }
        pending = std::move(still);
        if (pending.empty() || this->cfg.clock.steady() >= deadline) break;
        this->cfg.clock.sleep(OP_POLL_INTERVAL);
    }
    this->read_coe(ready);
    slave::clear_duplicate_serials(this->cached_slaves);
}

uint64_t Master::app_time() const {
    return static_cast<uint64_t>(
        nanoseconds(this->cfg.clock.steady()) + this->app_offset
    );
}

std::pair<master::Change, x::errors::Error> Master::changed() {
    if (!this->initialized)
        return {{}, x::errors::Error(errors::MASTER_INIT_ERROR, "not initialized")};
    ec_master_info_t info;
    if (this->api->master(this->ec_master, &info) < 0)
        return {
            {},
            x::errors::Error(errors::MASTER_INIT_ERROR, "failed to get master info")
        };
    // IgH scans the bus again by itself when the slaves change, so its slave info is
    // current once the scan ends.
    if (info.scan_busy != 0 || info.slave_count != this->cached_slaves.size())
        return {master::Change{.stale = true}, x::errors::NIL};
    for (const auto &slave: this->cached_slaves) {
        const auto &props = slave.properties;
        ec_slave_info_t slave_info;
        if (const int ret = this->api->master_get_slave(
                this->ec_master,
                static_cast<uint16_t>(props.position - 1),
                &slave_info
            );
            ret != 0)
            return {
                {},
                x::errors::Error(
                    errors::MASTER_INIT_ERROR,
                    std::format(
                        "failed to read module {}: {}",
                        props.position,
                        reason(ret)
                    )
                )
            };
        if (slave_info.vendor_id != props.vendor_id ||
            slave_info.product_code != props.product_code ||
            slave_info.revision_number != props.revision)
            return {master::Change{.stale = true}, x::errors::NIL};
    }
    return {{}, x::errors::NIL};
}

x::errors::Error Master::download(
    const slave::Properties &slave,
    const slave::StartupParameter &param,
    const std::span<const uint8_t> data
) {
    uint32_t abort_code = 0;
    const int ret = this->api->master_sdo_download(
        this->ec_master,
        ring_position(slave.position),
        param.index,
        param.sub_index,
        data.data(),
        data.size(),
        &abort_code
    );
    if (ret >= 0) return x::errors::NIL;
    const auto failure = sdo_failure(ret, abort_code);
    const bool rejected = failure.kind == errors::SDOFailure::Kind::ABORTED;
    return x::errors::Error(
        errors::startup_parameter_error(abort_code),
        "module " + std::to_string(slave.position) + " (" + slave.name + ") " +
            (rejected ? "rejected" : "failed to write") + " startup parameter " +
            param.to_string() + ": " + failure.reason()
    );
}

x::errors::Error Master::write(
    const uint16_t position,
    const std::vector<slave::StartupParameter> &parameters
) {
    if (position == 0 || position > this->cached_slaves.size())
        return x::errors::Error(
            errors::STARTUP_PARAMETER_ERROR,
            "no module at position " + std::to_string(position)
        );
    const auto &slave = this->cached_slaves[position - 1].properties;
    if (!this->activated) {
        for (const auto &param: parameters)
            if (auto err = this->download(slave, param, param.bytes())) return err;
        return x::errors::NIL;
    }
    const auto config = std::ranges::find(
        this->slave_configs,
        position,
        &std::pair<uint16_t, ec_slave_config_t *>::first
    );
    const auto conflict = [&](const slave::StartupParameter &param) {
        return x::errors::Error(
            errors::SETTING_CONFLICT,
            "the IgH master cannot change " + param.to_string() + " of module " +
                std::to_string(position) + " (" + slave.name +
                ") while other tasks run the bus. Stop them, then start this task first"
        );
    };
    for (const auto &param: parameters) {
        const auto want = param.bytes();
        std::vector<uint8_t> held(want.size());
        size_t size = 0;
        uint32_t abort_code = 0;
        if (const int ret = this->api->master_sdo_upload(
                this->ec_master,
                ring_position(position),
                param.index,
                param.sub_index,
                held.data(),
                held.size(),
                &size,
                &abort_code
            );
            ret < 0)
            return x::errors::Error(
                errors::startup_parameter_error(abort_code),
                "module " + std::to_string(position) + " (" + slave.name +
                    ") failed to read startup parameter " + param.to_string() + ": " +
                    sdo_failure(ret, abort_code).reason()
            );
        held.resize(size);
        const bool holds = held == want;
        if (config == this->slave_configs.end()) {
            if (!holds) return conflict(param);
            continue;
        }
        auto &kept = this->startup_values[{position, param.index, param.sub_index}];
        auto &added = this->running_additions[position];
        if (kept != want && added >= RUNNING_ADDITION_LIMIT) return conflict(param);
        if (!holds)
            if (auto err = this->download(slave, param, want)) return err;
        if (kept == want) continue;
        // IgH documents config SDOs for use before activation, but IgH 1.6 takes one on
        // a running bus under the master lock and writes it when the slave restarts.
        if (this->api->slave_config_sdo(
                config->second,
                param.index,
                param.sub_index,
                want.data(),
                want.size()
            ) < 0) {
            if (!holds)
                if (auto err = this->download(slave, param, held)) return err;
            return conflict(param);
        }
        kept = want;
        added++;
    }
    return x::errors::NIL;
}

std::pair<uint32_t, x::errors::Error>
Master::read(const uint16_t position, const uint16_t index, const uint8_t sub_index) {
    return sdo_reader(*this->api, this->ec_master, ring_position(position))(
        index,
        sub_index
    );
}

x::errors::Error Master::add_startup_parameters(
    const slave::Properties &slave,
    ec_slave_config_t *sc,
    const std::vector<slave::StartupParameter> &parameters
) {
    for (const auto &param: parameters) {
        const auto data = param.bytes();
        // IgH reports a rejected config SDO only as a slave stuck in PRE-OP, so a
        // direct write first gets the abort code.
        if (auto err = this->download(slave, param, data)) return err;
        if (this->api->slave_config_sdo(
                sc,
                param.index,
                param.sub_index,
                data.data(),
                data.size()
            ) < 0)
            return x::errors::Error(
                errors::ACTIVATION_ERROR,
                "failed to add startup parameter " + param.to_string() + " to module " +
                    std::to_string(slave.position) + " (" + slave.name + ")"
            );
        this->startup_values[{slave.position, param.index, param.sub_index}] = data;
    }
    return x::errors::NIL;
}

master::Activation Master::activate(
    const master::Plan &plan,
    const std::function<void(const master::Image &)> &start_cycle
) {
    if (!this->initialized)
        return {.err = x::errors::Error(errors::ACTIVATION_ERROR, "not initialized")};
    if (this->activated)
        return {.err = x::errors::Error(errors::ACTIVATION_ERROR, "already activated")};
    const auto config_of = [&plan](const uint16_t pos) {
        const auto it = plan.slaves.find(pos);
        return it == plan.slaves.end() ? slave::Config{} : it->second;
    };
    // IgH has no identity to configure a slave by when it failed to read its EEPROM.
    auto identified = plan;
    std::map<uint16_t, slave::Refusal> unidentified;
    for (const auto &slave: this->cached_slaves) {
        const auto &props = slave.properties;
        if (!config_of(props.position).enabled || props.vendor_id != 0 ||
            props.product_code != 0)
            continue;
        unidentified[props.position] = {
            .reason = "gave no identity when IgH read its EEPROM"
        };
        identified.slaves[props.position].enabled = false;
    }
    const auto refuse_unidentified = [&] {
        std::string msg;
        for (const auto &[pos, refusal]: unidentified)
            msg += std::format(
                "{}module {} ({}) {}",
                msg.empty() ? "" : "; ",
                pos,
                this->cached_slaves[pos - 1].properties.name,
                refusal.reason
            );
        return master::Activation{
            .err = x::errors::Error(errors::ACTIVATION_ERROR, msg),
            .refusals = std::move(unidentified),
        };
    };

    auto [dc_setups, dc_err] = master::resolve_dc(this->cached_slaves, identified);
    if (dc_err) return {.err = dc_err};
    this->read_starting();

    if (const int ret = this->api->master_reserve(this->ec_master); ret != 0)
        return {.err = reserve_error(this->name(), this->master_index, ret)};
    // The slaves can change until the reservation, and IgH keeps its slave list from
    // then until activation.
    const auto [change, change_err] = this->changed();
    if (change_err) return {.err = change_err};
    if (change.stale)
        return {
            .err = x::errors::Error(
                errors::ACTIVATION_ERROR,
                "the modules on " + this->name() + " changed since the last scan"
            )
        };
    master::Image image;
    for (const auto &[pos, regs]: dc_setups) {
        if (auto err = check_sync1(this->cached_slaves[pos - 1].properties, regs))
            return {.err = err};
        if (regs.sync0_cycle > 0 && (regs.assign_activate & dc::PDI_SYNC_UNIT) == 0)
            image.dc_sync.push_back(regs);
    }
    this->app_offset = (this->cfg.clock.now() - master::DC_EPOCH).nanoseconds() -
                       nanoseconds(this->cfg.clock.steady());
    auto [first_app, app_err] = first_app_time(this->app_time(), image.dc_sync);
    if (app_err) return {.err = app_err};
    for (const auto &slave: this->cached_slaves) {
        const auto pos = slave.properties.position;
        const auto config = config_of(pos);
        if (!config.enabled || unidentified.contains(pos)) continue;
        auto *sc = this->api->master_slave_config(
            this->ec_master,
            0,
            ring_position(pos),
            slave.properties.vendor_id,
            slave.properties.product_code
        );
        if (sc == nullptr)
            return {
                .err = x::errors::Error(
                    errors::ACTIVATION_ERROR,
                    "failed to configure module " + std::to_string(pos) + " (" +
                        slave.properties.name + ")"
                )
            };
        this->slave_configs.emplace_back(pos, sc);
        const auto &params = config.startup_parameters;
        if (auto err = this->add_startup_parameters(slave.properties, sc, params))
            return {.err = err};
        if (!params.empty())
            if (auto err = this->configure_assigned_pdos(pos, sc)) return {.err = err};
        if (auto err = this->register_pdo_entries(slave.properties, sc, image))
            return {.err = err};
        const auto setup = std::ranges::find(
            dc_setups,
            pos,
            &master::DCSetup::position
        );
        if (setup == dc_setups.end()) continue;
        const auto &regs = setup->registers;
        if (const int ret = this->api->slave_config_dc(
                sc,
                regs.assign_activate,
                regs.sync0_cycle,
                regs.sync0_shift,
                regs.sync1_cycle,
                0
            );
            ret != 0)
            return {
                .err = x::errors::Error(
                    errors::ACTIVATION_ERROR,
                    std::format(
                        "failed to set up the DC sync of module {} ({}): {}",
                        pos,
                        slave.properties.name,
                        reason(ret)
                    )
                )
            };
    }
    this->dc_setups = std::move(dc_setups);
    // The first application time sets where IgH starts the SYNC0 events of each slave.
    if (!this->dc_setups.empty()) {
        if (const int ret = this->api
                                ->master_application_time(this->ec_master, first_app);
            ret != 0)
            return {
                .err = x::errors::Error(
                    errors::ACTIVATION_ERROR,
                    "failed to give the application time: " + reason(ret)
                )
            };
        image.reference_clock = this->reference_clock;
    }
    // The caller leaves out a refused slave only when the image maps another one.
    if (!unidentified.empty() && image.offsets.empty()) return refuse_unidentified();

    const size_t size = this->domains.empty() ? 0 : this->domains.back().end();
    VLOG(1) << "[ethercat.igh] activating master " << this->master_index << " with "
            << this->slave_configs.size() << " configured slaves and "
            << this->domains.size() << " domains of " << size << " bytes";

    if (this->api->master_activate(this->ec_master) < 0)
        return {
            .err = x::errors::Error(
                errors::ACTIVATION_ERROR,
                "ecrt_master_activate failed"
            )
        };

    for (auto &domain: this->domains)
        domain.data = this->api->domain_data(domain.handle);
    this->inputs.assign(size, 0);
    this->outputs.assign(size, 0);
    this->latest_inputless.reserve(this->domains.size());

    this->activated = true;
    LOG(INFO) << "[ethercat.igh] master " << this->master_index
              << " activated successfully";

    image.inputs = this->inputs;
    image.outputs = this->outputs;
    image.slaves = this->cached_slaves;
    // The IgH master moves the slaves to OP on its own while the caller cycles.
    start_cycle(image);
    if (!unidentified.empty()) return refuse_unidentified();
    return this->wait_for_op(plan);
}

master::Activation Master::wait_for_op(const master::Plan &plan) {
    auto deadline = this->cfg.clock.steady() + this->cfg.op_timeout.chrono();
    const size_t total = this->slave_configs.size();
    size_t waiting = total;
    std::optional<std::chrono::steady_clock::time_point> reset_at;
    while (true) {
        std::vector<uint16_t> pending;
        for (const auto &[pos, sc]: this->slave_configs) {
            ec_slave_config_state_t state{};
            if (this->api->slave_config_state(sc, &state) != 0 ||
                state.operational == 0)
                pending.push_back(pos);
        }
        const bool advanced = pending.size() < waiting;
        if (advanced) {
            waiting = pending.size();
            if (plan.progress) plan.progress(total - waiting, total);
        }
        if (pending.empty()) return {};
        if (plan.cancel != nullptr && !plan.cancel->running())
            return {
                .err = x::errors::Error(
                    x::errors::CANCELLED,
                    "the start stopped before the slaves reached OP"
                ),
            };
        const auto now = this->cfg.clock.steady();
        if (advanced) deadline = now + this->cfg.op_timeout.chrono();
        if (now >= deadline) return this->op_error(pending, true);
        if (this->refusing(pending)) {
            if (!reset_at) {
                // The reset also clears a flag set before activation.
                if (const int ret = this->api->master_reset(this->ec_master);
                    ret != 0) {
                    LOG(WARNING) << "[ethercat.igh] master " << this->master_index
                                 << " failed to reset: " << reason(ret);
                    return this->op_error(pending, false);
                }
                reset_at = now;
            } else if (now - *reset_at >= RESET_GRACE.chrono())
                return this->op_error(pending, false);
        }
        this->cfg.clock.sleep(OP_POLL_INTERVAL);
    }
}

bool Master::refusing(const std::vector<uint16_t> &positions) const {
    return std::ranges::all_of(positions, [this](const uint16_t pos) {
        ec_slave_info_t info{};
        return this->api->master_get_slave(
                   this->ec_master,
                   ring_position(pos),
                   &info
               ) == 0 &&
               info.error_flag != 0;
    });
}

master::Activation
Master::op_error(const std::vector<uint16_t> &pending, const bool timed_out) {
    master::Activation activation;
    std::string msg;
    for (const auto pos: pending) {
        const auto &name = this->cached_slaves[pos - 1].properties.name;
        ec_slave_info_t info{};
        if (const int ret = this->api->master_get_slave(
                this->ec_master,
                ring_position(pos),
                &info
            );
            ret != 0) {
            if (!msg.empty()) msg += "; ";
            msg += std::format(
                "module {} ({}): failed to read its state: {}",
                pos,
                name,
                reason(ret)
            );
            continue;
        }
        const auto state = slave::from_al_state(info.al_state);
        // The slave can reach OP after the last poll and before this read.
        if (state == slave::State::OP) continue;
        if (!msg.empty()) msg += "; ";
        msg += std::format(
            "module {} ({}) is in {}",
            pos,
            name,
            slave::to_string(state)
        );
        if (info.error_flag == 0) continue;
        msg += " with an AL error";
        activation.refusals[pos] = {.reason = "refused " + next_state(state)};
    }
    if (msg.empty()) return activation;
    activation.err = x::errors::Error(
        activation.refusals.empty() ? errors::STATE_CHANGE_ERROR
                                    : errors::STATE_CHANGE_REFUSED,
        "failed to reach OP" +
            (timed_out ? " (no module reached OP for " +
                             this->cfg.op_timeout.to_string() + ")"
                       : "") +
            ": " + msg
    );
    return activation;
}

void Master::deactivate() {
    if (!this->initialized) return;

    VLOG(1) << "[ethercat.igh] master " << this->master_index << " deactivating";
    if (this->activated)
        if (const int ret = this->api->master_deactivate(this->ec_master); ret != 0)
            LOG(WARNING) << "[ethercat.igh] master " << this->master_index
                         << " failed to deactivate: " << reason(ret);
    if (this->ec_master) {
        this->api->release_master(this->ec_master);
        this->ec_master = nullptr;
    }
    this->activated = false;
    this->initialized = false;
    this->slave_configs.clear();
    this->startup_values.clear();
    this->running_additions.clear();
    this->cached_slaves.clear();
    this->dictionaries.clear();
    this->domains.clear();
    this->inputs.clear();
    this->outputs.clear();
    this->latest_health.clear();
    this->latest_shortfall.reset();
    this->latest_inputless.clear();
    this->faulted = false;
    this->resets = 0;
    this->next_reset = {};
    this->reference_clock.reset();
    this->dc_setups.clear();
    this->clocks_aligned = false;
    this->latest_dc.reset();
}

const x::errors::Error &Master::receive() {
    this->latest_shortfall.reset();
    this->latest_inputless.clear();
    if (!this->activated)
        return this->received = x::errors::Error(errors::CYCLIC_ERROR, "not activated");

    if (const int ret = this->api->master_receive(this->ec_master); ret != 0)
        return this->received = x::errors::Error(
                   errors::CYCLIC_ERROR,
                   "failed to receive frames: " + reason(ret)
               );
    uint32_t working_counter = 0;
    uint32_t expected = 0;
    size_t silent = 0;
    for (const auto &domain: this->domains) {
        if (const int ret = this->api->domain_process(domain.handle); ret != 0)
            return this->received = x::errors::Error(
                       errors::CYCLIC_ERROR,
                       "failed to process a domain: " + reason(ret)
                   );
        ec_domain_state_t state{};
        if (const int ret = this->api->domain_state(domain.handle, &state); ret != 0)
            return this->received = x::errors::Error(
                       errors::CYCLIC_ERROR,
                       "failed to read a domain state: " + reason(ret)
                   );
        std::memcpy(this->inputs.data() + domain.offset, domain.data, domain.size);
        const uint32_t full = domain.share.inputs + domain.share.outputs;
        expected += full;
        // IgH splits a domain too large for one datagram, which can change the count it
        // expects, so its own state decides whether the slave took full part.
        if (state.wc_state == EC_WC_COMPLETE) {
            working_counter += full;
            continue;
        }
        working_counter += std::min<uint32_t>(state.working_counter, full);
        if (state.wc_state != EC_WC_ZERO) continue;
        silent++;
        this->latest_inputless.push_back(domain.position);
    }
    // IgH aligns the slave clocks to the application time before any slave reaches
    // SAFE-OP and gives a working counter.
    if (working_counter > 0) this->clocks_aligned = true;
    if (!this->dc_setups.empty() && this->clocks_aligned) {
        uint32_t dc = 0;
        const int ret = this->api->master_reference_clock_time(this->ec_master, &dc);
        if (ret == 0)
            this->latest_dc = master::DC_EPOCH +
                              x::telem::TimeSpan(
                                  static_cast<int64_t>(this->sent_at) +
                                  static_cast<int32_t>(
                                      dc - static_cast<uint32_t>(this->sent_at)
                                  )
                              );
        // A frame that did not reach the reference clock carries no time (EIO), and IgH
        // has no reference clock while it scans the bus (ENXIO).
        else if (ret == -EIO || ret == -ENXIO)
            this->latest_dc.reset();
        else
            return this->received = x::errors::Error(
                       errors::CYCLIC_ERROR,
                       "failed to read the reference clock: " + reason(ret)
                   );
    }
    if (working_counter == expected) return x::errors::NIL;
    this->latest_shortfall = expected - working_counter;
    if (silent == this->domains.size()) {
        this->latest_inputless.clear();
        return this->received = x::errors::Error(
                   errors::WORKING_COUNTER_ERROR,
                   "no modules responded"
               );
    }
    return this->received = x::errors::Error(
               errors::WORKING_COUNTER_ERROR,
               std::format(
                   "incomplete working counter: {} of {}",
                   working_counter,
                   expected
               )
           );
}

x::errors::Error Master::send() {
    if (!this->activated)
        return x::errors::Error(errors::CYCLIC_ERROR, "not activated");
    if (!this->dc_setups.empty()) {
        this->sent_at = this->app_time();
        if (const int ret = this->api->master_application_time(
                this->ec_master,
                this->sent_at
            );
            ret != 0)
            return x::errors::Error(
                errors::CYCLIC_ERROR,
                "failed to give the application time: " + reason(ret)
            );
        // IgH has no reference clock while it scans the bus.
        for (const int ret:
             {this->api->master_sync_reference_clock_to(this->ec_master, this->sent_at),
              this->api->master_sync_slave_clocks(this->ec_master)})
            if (ret != 0 && ret != -ENXIO)
                return x::errors::Error(
                    errors::CYCLIC_ERROR,
                    "failed to sync the distributed clocks: " + reason(ret)
                );
    }

    for (const auto &domain: this->domains) {
        std::memcpy(domain.data, this->outputs.data() + domain.offset, domain.size);
        if (const int ret = this->api->domain_queue(domain.handle); ret != 0)
            return x::errors::Error(
                errors::CYCLIC_ERROR,
                "failed to queue a domain: " + reason(ret)
            );
    }
    if (const int ret = this->api->master_send(this->ec_master); ret != 0)
        return x::errors::Error(
            errors::CYCLIC_ERROR,
            "failed to send frames: " + reason(ret)
        );
    return x::errors::NIL;
}

std::vector<slave::Health> Master::check(const x::breaker::Breaker &) {
    ec_master_info_t info{};
    const bool scanning = this->api->master(this->ec_master, &info) == 0 &&
                          info.scan_busy != 0;
    std::vector<slave::Health> health;
    health.reserve(this->slave_configs.size());
    for (const auto &[pos, sc]: this->slave_configs) {
        if (scanning && !this->silent(pos))
            if (const auto prev = std::ranges::find(
                    this->latest_health,
                    pos,
                    &slave::Health::position
                );
                prev != this->latest_health.end()) {
                health.push_back(*prev);
                continue;
            }
        ec_slave_config_state_t state{};
        ec_slave_info_t slave_info{};
        // A slave the master cannot read is not responding.
        const bool read = this->api->slave_config_state(sc, &state) == 0 &&
                          this->api->master_get_slave(
                              this->ec_master,
                              ring_position(pos),
                              &slave_info
                          ) == 0;
        health.push_back({
            .position = pos,
            .state = slave::from_al_state(state.al_state),
            .faulted = read && slave_info.error_flag != 0,
            .lost = !read || state.online == 0,
        });
    }
    this->latest_health = health;
    this->faulted = std::ranges::any_of(health, &slave::Health::faulted);
    // A reset clears the AL error flags while IgH configures the slaves again, so only
    // a bus back in OP ends the backoff.
    if (std::ranges::all_of(health, [](const slave::Health &h) {
            return h.state == slave::State::OP && !h.faulted && !h.lost;
        })) {
        this->resets = 0;
        this->next_reset = {};
    }
    return health;
}

bool Master::silent(const uint16_t position) const {
    const auto domain = std::ranges::find(this->domains, position, &Domain::position);
    if (domain == this->domains.end()) return false;
    ec_domain_state_t state{};
    return this->api->domain_state(domain->handle, &state) != 0 ||
           state.wc_state == EC_WC_ZERO;
}

void Master::recover(const x::breaker::Breaker &breaker) {
    const auto now = this->cfg.clock.steady();
    if (!this->faulted || !breaker.running() || now < this->next_reset) return;
    if (const int ret = this->api->master_reset(this->ec_master); ret != 0)
        LOG(WARNING) << "[ethercat.igh] master " << this->master_index
                     << " failed to reset: " << reason(ret);
    this->resets++;
    this->next_reset = now + RESET_RETRY.interval(this->resets).chrono();
}

x::errors::Error
Master::configure_assigned_pdos(const uint16_t position, ec_slave_config_t *sc) {
    auto props = this->cached_slaves[position - 1].properties;
    const auto slave_name = "module " + std::to_string(position) + " (" + props.name +
                            ")";
    const auto [syncs, err] = read_layout(
        *this->api,
        this->ec_master,
        ring_position(position)
    );
    if (err)
        return x::errors::Error(
            errors::PDO_DISCOVERY_ERROR,
            "failed to read the PDO layout of " + slave_name +
                " after its startup parameters: " + err.data
        );
    const auto assignable = [&](const bool is_input) {
        return std::ranges::any_of(syncs, [is_input](const pdo::Sync &s) {
            return !s.fixed && s.is_input == is_input;
        });
    };
    if (!assignable(true) && !assignable(false)) return x::errors::NIL;
    std::vector<std::vector<ec_pdo_entry_info_t>> entry_infos;
    std::vector<std::vector<ec_pdo_info_t>> pdo_infos;
    std::vector<ec_sync_info_t> sync_infos;
    std::vector<uint8_t> empty_syncs;
    std::vector<uint16_t> empty_pdos;
    // A direction with no assignable sync manager keeps the PDOs found at discovery.
    if (assignable(true)) props.input_pdos.clear();
    if (assignable(false)) props.output_pdos.clear();
    for (const auto &sync: syncs) {
        if (sync.fixed) {
            if (!assignable(sync.is_input)) continue;
            if (const auto scan_err = this->append_scanned_pdos(props, sync.index))
                return x::errors::Error(
                    errors::PDO_DISCOVERY_ERROR,
                    "failed to read the PDO layout of " + slave_name + ": " +
                        scan_err.data
                );
            continue;
        }
        auto &infos = pdo_infos.emplace_back();
        if (sync.pdos.empty()) empty_syncs.push_back(sync.index);
        for (const auto &pdo: sync.pdos) {
            // Moving a vector keeps its buffer, so IgH can hold the pointer.
            auto &entries = entry_infos.emplace_back();
            for (const auto &entry: pdo.entries)
                entries.push_back({
                    .index = entry.index,
                    .subindex = entry.sub_index,
                    .bit_length = entry.bit_length,
                });
            infos.push_back({
                .index = pdo.index,
                .n_entries = static_cast<unsigned int>(entries.size()),
                .entries = entries.data(),
            });
            if (pdo.entries.empty()) empty_pdos.push_back(pdo.index);
            pdo::append(
                sync.is_input ? props.input_pdos : props.output_pdos,
                sync.is_input,
                pdo
            );
        }
        sync_infos.push_back({
            .index = sync.index,
            .dir = sync.is_input ? EC_DIR_INPUT : EC_DIR_OUTPUT,
            .n_pdos = static_cast<unsigned int>(infos.size()),
            .pdos = infos.data(),
            .watchdog_mode = EC_WD_DEFAULT,
        });
    }
    if (this->api->slave_config_pdos(
            sc,
            static_cast<unsigned int>(sync_infos.size()),
            sync_infos.data()
        ) < 0)
        return x::errors::Error(
            errors::PDO_MAPPING_ERROR,
            "failed to configure the PDOs of " + slave_name
        );
    // ecrt_slave_config_pdos keeps the default assignment of a sync manager with no
    // PDOs and the default mapping of a PDO with no entries.
    for (const auto sync: empty_syncs)
        if (this->api->slave_config_pdo_assign_clear(sc, sync) < 0)
            return x::errors::Error(
                errors::PDO_MAPPING_ERROR,
                "failed to clear the PDO assignment of " + slave_name
            );
    for (const auto pdo: empty_pdos)
        if (this->api->slave_config_pdo_mapping_clear(sc, pdo) < 0)
            return x::errors::Error(
                errors::PDO_MAPPING_ERROR,
                "failed to clear the PDO mapping of " + slave_name
            );
    if (ec_slave_info_t info{};
        this->api->master_get_slave(this->ec_master, ring_position(position), &info) ==
        0)
        fill_dictionary_entries(
            *this->api,
            this->master_index,
            ring_position(position),
            info.sdo_count,
            props
        );
    esi::fill_known_entries(props);
    this->cached_slaves[position - 1].properties = std::move(props);
    return x::errors::NIL;
}

x::errors::Error Master::register_pdo_entries(
    const slave::Properties &slave,
    ec_slave_config_t *sc,
    master::Image &image
) {
    const auto mapped = [](const auto &pdos) {
        return std::ranges::any_of(pdos, [](const pdo::Properties &pdo) {
            return !pdo.is_padding();
        });
    };
    if (!mapped(slave.output_pdos) && !mapped(slave.input_pdos)) return x::errors::NIL;
    auto *handle = this->api->master_create_domain(this->ec_master);
    if (handle == nullptr)
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            std::format(
                "failed to create the domain of module {} ({})",
                slave.position,
                slave.name
            )
        );
    const size_t base = this->domains.empty() ? 0 : this->domains.back().end();
    bool has_outputs = false;
    bool has_inputs = false;
    const auto add = [&](const pdo::Properties &pdo) {
        if (pdo.is_padding()) return;
        const pdo::Key key{slave.position, pdo.index, pdo.sub_index, pdo.is_input};
        unsigned int bit = 0;
        const int offset = this->api->slave_config_reg_pdo_entry(
            sc,
            pdo.index,
            pdo.sub_index,
            handle,
            &bit
        );
        if (offset < 0) {
            image.errors.emplace(
                key,
                x::errors::Error(
                    errors::PDO_MAPPING_ERROR,
                    std::format(
                        "the IgH master rejected PDO {} of module {} ({}): {}",
                        telem::format_index_sub_index(pdo.index, pdo.sub_index),
                        slave.position,
                        slave.name,
                        reason(offset)
                    )
                )
            );
            return;
        }
        (pdo.is_input ? has_inputs : has_outputs) = true;
        image.offsets[key] = {
            base + static_cast<size_t>(offset),
            static_cast<uint8_t>(bit),
            pdo.bit_length
        };
    };
    for (const auto &pdo: slave.output_pdos)
        add(pdo);
    for (const auto &pdo: slave.input_pdos)
        add(pdo);
    if (!has_outputs && !has_inputs) return x::errors::NIL;
    // IgH reads and writes a domain with both directions in one LRW datagram, which a
    // slave adds 2 to when it takes its outputs. A domain with only outputs is an LWR
    // datagram, which the slave adds 1 to.
    master::WorkingCounterShare share{.inputs = has_inputs ? 1U : 0U};
    if (has_outputs) share.outputs = has_inputs ? 2U : 1U;
    this->domains.push_back({
        .position = slave.position,
        .handle = handle,
        .offset = base,
        .size = this->api->domain_size(handle),
        .share = share,
    });
    image.working_counter_shares[slave.position] = share;
    return x::errors::NIL;
}

x::errors::Error
Master::append_scanned_pdos(slave::Properties &props, const uint8_t sync_index) {
    const auto pos = ring_position(props.position);
    ec_sync_info_t sync_info{};
    if (const int ret = this->api->master_get_sync_manager(
            this->ec_master,
            pos,
            sync_index,
            &sync_info
        );
        ret != 0)
        return x::errors::Error(
            errors::PDO_DISCOVERY_ERROR,
            std::format("failed to read sync manager {}: {}", sync_index, reason(ret))
        );
    const bool is_input = sync_info.dir == EC_DIR_INPUT;
    for (unsigned int pdo_pos = 0; pdo_pos < sync_info.n_pdos; ++pdo_pos) {
        ec_pdo_info_t pdo_info{};
        if (const int ret = this->api->master_get_pdo(
                this->ec_master,
                pos,
                sync_index,
                static_cast<uint16_t>(pdo_pos),
                &pdo_info
            );
            ret != 0)
            return x::errors::Error(
                errors::PDO_DISCOVERY_ERROR,
                std::format(
                    "failed to read PDO {} of sync manager {}: {}",
                    pdo_pos,
                    sync_index,
                    reason(ret)
                )
            );
        pdo::PDO pdo{.index = pdo_info.index, .entries = {}};
        for (unsigned int entry_pos = 0; entry_pos < pdo_info.n_entries; ++entry_pos) {
            ec_pdo_entry_info_t entry{};
            if (const int ret = this->api->master_get_pdo_entry(
                    this->ec_master,
                    pos,
                    sync_index,
                    static_cast<uint16_t>(pdo_pos),
                    static_cast<uint16_t>(entry_pos),
                    &entry
                );
                ret != 0)
                return x::errors::Error(
                    errors::PDO_DISCOVERY_ERROR,
                    std::format(
                        "failed to read entry {} of PDO {:#06x}: {}",
                        entry_pos,
                        pdo_info.index,
                        reason(ret)
                    )
                );
            pdo.entries.push_back({
                .index = entry.index,
                .sub_index = entry.subindex,
                .bit_length = entry.bit_length,
            });
        }
        pdo::append(is_input ? props.input_pdos : props.output_pdos, is_input, pdo);
    }
    return x::errors::NIL;
}

std::pair<std::unique_ptr<Manager>, x::errors::Error> Manager::open() {
    struct stat st{};
    if (stat(DEVICE_PATH, &st) != 0)
        return {
            nullptr,
            x::errors::Error(errors::MASTER_INIT_ERROR, "IgH device not found")
        };
    auto [api, err] = API::load();
    if (err) return {nullptr, err};
    return {std::make_unique<Manager>(std::move(api)), x::errors::NIL};
}

std::vector<master::Info> Manager::enumerate() {
    constexpr std::string_view prefix = "EtherCAT";
    std::vector<master::Info> masters;
    DIR *dir = opendir(SYSFS_PATH);
    if (dir == nullptr) return masters;
    while (const dirent *entry = readdir(dir)) {
        const std::string_view name = entry->d_name;
        if (!name.starts_with(prefix)) continue;
        const auto index = parse_index(name.substr(prefix.size()));
        if (!index) continue;
        masters.push_back({
            .key = "igh:" + std::to_string(*index),
            .description = "IgH EtherCAT Master " + std::to_string(*index),
        });
    }
    closedir(dir);
    return masters;
}

std::pair<std::shared_ptr<master::Master>, x::errors::Error>
Manager::create(const std::string &key) {
    constexpr std::string_view prefix = "igh:";
    const auto index = key.starts_with(prefix)
                         ? parse_index(std::string_view(key).substr(prefix.size()))
                         : std::nullopt;
    if (!index)
        return {
            nullptr,
            x::errors::Error(
                errors::MASTER_INIT_ERROR,
                "invalid IgH master key '" + key + "': expected format 'igh:N'"
            )
        };
    return {std::make_shared<Master>(this->api, *index), x::errors::NIL};
}
}
