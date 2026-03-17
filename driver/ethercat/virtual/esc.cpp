// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <arpa/inet.h>
#include <linux/if_packet.h>
#include <net/ethernet.h>
#include <net/if.h>
#include <poll.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <unistd.h>

#include "glog/logging.h"

#include "driver/ethercat/virtual/esc.h"

namespace ethercat::virtual_esc {
constexpr size_t MAX_FRAME_SIZE = 1518;
constexpr size_t REGISTER_SPACE_SIZE = 0x1000;
constexpr int POLL_TIMEOUT_MS = 100;

constexpr uint16_t SII_VENDOR_ID_OFFSET = 0x0008;
constexpr uint16_t SII_PRODUCT_CODE_OFFSET = 0x000A;
constexpr uint16_t SII_REVISION_OFFSET = 0x000C;
constexpr uint16_t SII_SERIAL_OFFSET = 0x000E;

constexpr uint16_t SII_CAT_NOP = 0;
constexpr uint16_t SII_CAT_STRINGS = 10;
constexpr uint16_t SII_CAT_GENERAL = 30;
constexpr uint16_t SII_CAT_FMMU = 40;
constexpr uint16_t SII_CAT_SYNCM = 41;
constexpr uint16_t SII_CAT_TXPDO = 50;
constexpr uint16_t SII_CAT_RXPDO = 51;
constexpr uint16_t SII_CAT_END = 0xFFFF;

VirtualESC::VirtualESC(Config config):
    config(std::move(config)),
    sii_address(0),
    running(false),
    frame_count(0),
    raw_socket(-1) {
    this->init_object_dictionary();
    this->init_sii_eeprom();
    this->init_registers();
    this->input_data.resize(this->config.tx_pdo_bytes(), 0);
    this->output_data.resize(this->config.rx_pdo_bytes(), 0);
}

VirtualESC::~VirtualESC() { this->stop(); }

xerrors::Error VirtualESC::start(const std::string& interface) {
    if (this->running.load()) return xerrors::NIL;
    this->iface_name = interface;
    this->raw_socket = socket(AF_PACKET, SOCK_RAW, htons(ETH_P_ALL));
    if (this->raw_socket < 0)
        return xerrors::Error(SOCKET_ERROR, "failed to create raw socket");
    struct ifreq ifr{};
    std::strncpy(ifr.ifr_name, interface.c_str(), IFNAMSIZ - 1);
    if (ioctl(this->raw_socket, SIOCGIFINDEX, &ifr) < 0) {
        close(this->raw_socket);
        this->raw_socket = -1;
        return xerrors::Error(BIND_ERROR, "failed to get interface index");
    }
    struct sockaddr_ll sll{};
    sll.sll_family = AF_PACKET;
    sll.sll_ifindex = ifr.ifr_ifindex;
    sll.sll_protocol = htons(ETH_P_ALL);
    if (bind(this->raw_socket, reinterpret_cast<struct sockaddr*>(&sll), sizeof(sll)) <
        0) {
        close(this->raw_socket);
        this->raw_socket = -1;
        return xerrors::Error(BIND_ERROR, "failed to bind to interface");
    }
    this->running.store(true);
    this->worker = std::thread(&VirtualESC::run, this);
    return xerrors::NIL;
}

void VirtualESC::stop() {
    if (!this->running.load()) return;
    this->running.store(false);
    if (this->worker.joinable()) this->worker.join();
    if (this->raw_socket >= 0) {
        close(this->raw_socket);
        this->raw_socket = -1;
    }
}

slave::State VirtualESC::current_state() const {
    std::lock_guard lock(this->mu);
    return this->state_machine.current_state();
}

bool VirtualESC::is_operational() const {
    std::lock_guard lock(this->mu);
    return this->state_machine.is_operational();
}

void VirtualESC::run() {
    std::vector<uint8_t> buffer(MAX_FRAME_SIZE);
    struct pollfd pfd{};
    pfd.fd = this->raw_socket;
    pfd.events = POLLIN;
    while (this->running.load()) {
        int ret = poll(&pfd, 1, POLL_TIMEOUT_MS);
        if (ret < 0) {
            if (errno == EINTR) continue;
            LOG(ERROR) << "poll failed: " << strerror(errno);
            break;
        }
        if (ret == 0) continue;
        ssize_t len = recv(this->raw_socket, buffer.data(), buffer.size(), 0);
        if (len < 0) {
            if (errno == EINTR || errno == EAGAIN) continue;
            LOG(ERROR) << "recv failed: " << strerror(errno);
            continue;
        }
        if (len < static_cast<ssize_t>(MIN_FRAME_SIZE)) continue;
        Frame frame;
        if (!frame.parse(std::span<const uint8_t>(buffer.data(), len))) continue;
        this->process_frame(frame);
        frame.swap_mac_addresses();
        ssize_t sent = send(
            this->raw_socket,
            frame.data().data(),
            frame.size(),
            0
        );
        if (sent < 0) LOG(WARNING) << "send failed: " << strerror(errno);
        this->frame_count.fetch_add(1);
    }
}

void VirtualESC::process_frame(Frame& frame) {
    auto dgrams = frame.datagrams();
    if (dgrams.empty()) return;
    size_t offset = 0;
    while (offset + MIN_DATAGRAM_SIZE <= dgrams.size()) {
        Datagram dgram(dgrams, offset);
        if (!dgram.valid()) break;
        this->handle_datagram(dgram);
        if (!dgram.more_follows()) break;
        offset = dgram.next_offset();
    }
}

void VirtualESC::handle_datagram(Datagram& dgram) {
    const auto cmd = dgram.command();
    switch (cmd) {
        case Command::BRD:
            this->handle_broadcast_read(dgram);
            break;
        case Command::BWR:
            this->handle_broadcast_write(dgram);
            break;
        case Command::APRD:
            this->handle_auto_increment_read(dgram);
            break;
        case Command::APWR:
            this->handle_auto_increment_write(dgram);
            break;
        case Command::FPRD:
            this->handle_configured_address_read(dgram);
            break;
        case Command::FPWR:
            this->handle_configured_address_write(dgram);
            break;
        case Command::LRD:
            this->handle_logical_read(dgram);
            break;
        case Command::LWR:
            this->handle_logical_write(dgram);
            break;
        case Command::LRW:
            this->handle_logical_read_write(dgram);
            break;
        default:
            break;
    }
}

void VirtualESC::handle_broadcast_read(Datagram& dgram) {
    std::lock_guard lock(this->mu);
    const uint16_t addr = dgram.ado();
    auto data = dgram.data();
    if (this->read_register(addr, data)) dgram.increment_wkc();
}

void VirtualESC::handle_broadcast_write(Datagram& dgram) {
    std::lock_guard lock(this->mu);
    const uint16_t addr = dgram.ado();
    auto data = dgram.data();
    if (this->write_register(addr, data)) dgram.increment_wkc();
}

void VirtualESC::handle_auto_increment_read(Datagram& dgram) {
    std::lock_guard lock(this->mu);
    const int16_t adp = dgram.adp();
    if (adp == 0) {
        const uint16_t addr = dgram.ado();
        auto data = dgram.data();
        if (this->read_register(addr, data)) dgram.increment_wkc();
    }
    dgram.decrement_adp();
}

void VirtualESC::handle_auto_increment_write(Datagram& dgram) {
    std::lock_guard lock(this->mu);
    const int16_t adp = dgram.adp();
    if (adp == 0) {
        const uint16_t addr = dgram.ado();
        auto data = dgram.data();
        if (this->write_register(addr, data)) dgram.increment_wkc();
    }
    dgram.decrement_adp();
}

void VirtualESC::handle_configured_address_read(Datagram& dgram) {
    std::lock_guard lock(this->mu);
    const uint16_t configured_addr = dgram.configured_address();
    if (configured_addr != this->config.station_address) return;
    const uint16_t addr = dgram.ado();
    auto data = dgram.data();
    if (this->read_register(addr, data)) dgram.increment_wkc();
}

void VirtualESC::handle_configured_address_write(Datagram& dgram) {
    std::lock_guard lock(this->mu);
    const uint16_t configured_addr = dgram.configured_address();
    if (configured_addr != this->config.station_address) return;
    const uint16_t addr = dgram.ado();
    auto data = dgram.data();
    if (this->write_register(addr, data)) dgram.increment_wkc();
}

void VirtualESC::handle_logical_read(Datagram& dgram) {
    std::lock_guard lock(this->mu);
    const auto state = this->state_machine.current_state();
    if (state != slave::State::SAFE_OP && state != slave::State::OP) return;
    auto data = dgram.data();
    const size_t copy_size = std::min(data.size(), this->input_data.size());
    std::memcpy(data.data(), this->input_data.data(), copy_size);
    dgram.increment_wkc();
}

void VirtualESC::handle_logical_write(Datagram& dgram) {
    std::lock_guard lock(this->mu);
    const auto state = this->state_machine.current_state();
    if (state != slave::State::SAFE_OP && state != slave::State::OP) return;
    auto data = dgram.data();
    if (state == slave::State::OP) {
        const size_t copy_size = std::min(data.size(), this->output_data.size());
        std::memcpy(this->output_data.data(), data.data(), copy_size);
    }
    dgram.increment_wkc();
}

void VirtualESC::handle_logical_read_write(Datagram& dgram) {
    std::lock_guard lock(this->mu);
    const auto state = this->state_machine.current_state();
    if (state != slave::State::SAFE_OP && state != slave::State::OP) return;
    auto data = dgram.data();
    if (state == slave::State::OP) {
        const size_t rx_size = std::min(data.size(), this->output_data.size());
        std::memcpy(this->output_data.data(), data.data(), rx_size);
    }
    const size_t tx_size = std::min(data.size(), this->input_data.size());
    std::memcpy(data.data(), this->input_data.data(), tx_size);
    dgram.increment_wkc_rw();
}

bool VirtualESC::read_register(const uint16_t addr, std::span<uint8_t> data) {
    if (addr == REG_AL_STATUS) {
        const uint16_t status = this->state_machine.al_status();
        const size_t copy_size = std::min(data.size(), sizeof(status));
        std::memcpy(data.data(), &status, copy_size);
        return true;
    }
    if (addr == REG_AL_STATUS_CODE) {
        const uint16_t code = static_cast<uint16_t>(this->state_machine.al_status_code());
        const size_t copy_size = std::min(data.size(), sizeof(code));
        std::memcpy(data.data(), &code, copy_size);
        return true;
    }
    if (addr == REG_STATION_ADDRESS) {
        const uint16_t station = this->config.station_address;
        const size_t copy_size = std::min(data.size(), sizeof(station));
        std::memcpy(data.data(), &station, copy_size);
        return true;
    }
    if (addr >= REG_SII_DATA && addr < REG_SII_DATA + 4) {
        const size_t byte_offset = addr - REG_SII_DATA;
        uint8_t sii_data[4] = {0, 0, 0, 0};
        const uint16_t sii_addr = static_cast<uint16_t>(
            this->registers[0x0504] | (this->registers[0x0505] << 8)
        );
        if (sii_addr < this->sii_eeprom.size()) {
            const uint16_t word0 = this->sii_eeprom[sii_addr];
            sii_data[0] = static_cast<uint8_t>(word0 & 0xFF);
            sii_data[1] = static_cast<uint8_t>((word0 >> 8) & 0xFF);
            if (static_cast<size_t>(sii_addr) + 1 < this->sii_eeprom.size()) {
                const uint16_t word1 = this->sii_eeprom[sii_addr + 1];
                sii_data[2] = static_cast<uint8_t>(word1 & 0xFF);
                sii_data[3] = static_cast<uint8_t>((word1 >> 8) & 0xFF);
            }
            if (sii_addr >= 0x54 && sii_addr <= 0x64) {
                LOG(INFO) << "SII SYNCM read: addr=0x" << std::hex << sii_addr
                          << " bytes=[" << static_cast<int>(sii_data[0])
                          << "," << static_cast<int>(sii_data[1])
                          << "," << static_cast<int>(sii_data[2])
                          << "," << static_cast<int>(sii_data[3]) << "]";
            }
        }
        const size_t avail = 4 - byte_offset;
        const size_t copy_size = std::min(data.size(), avail);
        std::memcpy(data.data(), sii_data + byte_offset, copy_size);
        return true;
    }
    if (addr == REG_SII_CONTROL) {
        const uint16_t status = 0x0000;
        const size_t copy_size = std::min(data.size(), sizeof(status));
        std::memcpy(data.data(), &status, copy_size);
        return true;
    }
    if (addr < REGISTER_SPACE_SIZE && addr + data.size() <= REGISTER_SPACE_SIZE) {
        std::memcpy(data.data(), this->registers.data() + addr, data.size());
        return true;
    }
    return false;
}

bool VirtualESC::write_register(const uint16_t addr, std::span<const uint8_t> data) {
    if (addr == REG_AL_CONTROL && data.size() >= 2) {
        const uint16_t requested = static_cast<uint16_t>(data[0] | (data[1] << 8));
        return this->state_machine.request_state(requested);
    }
    if (addr == REG_STATION_ADDRESS && data.size() >= 2) {
        this->config.station_address = static_cast<uint16_t>(data[0] | (data[1] << 8));
        return true;
    }
    if (addr == REG_SII_CONTROL || addr == 0x0504) {
        if (addr < REGISTER_SPACE_SIZE && addr + data.size() <= REGISTER_SPACE_SIZE)
            std::memcpy(this->registers.data() + addr, data.data(), data.size());
        return true;
    }
    if (addr < REGISTER_SPACE_SIZE && addr + data.size() <= REGISTER_SPACE_SIZE) {
        std::memcpy(this->registers.data() + addr, data.data(), data.size());
        return true;
    }
    return false;
}

void VirtualESC::init_object_dictionary() {
    this->od.set_identity(
        this->config.vendor_id,
        this->config.product_code,
        this->config.revision,
        this->config.serial
    );
    this->od.set_device_name(this->config.name);
    this->od.set_hw_version(this->config.hw_version);
    this->od.set_sw_version(this->config.sw_version);
    for (const auto& pdo : this->config.tx_pdos) this->od.add_tx_pdo(pdo);
    for (const auto& pdo : this->config.rx_pdos) this->od.add_rx_pdo(pdo);
}

void VirtualESC::init_sii_eeprom() {
    this->sii_eeprom.resize(512, 0);
    this->sii_eeprom[SII_VENDOR_ID_OFFSET] = static_cast<uint16_t>(this->config.vendor_id & 0xFFFF);
    this->sii_eeprom[SII_VENDOR_ID_OFFSET + 1] = static_cast<uint16_t>((this->config.vendor_id >> 16) & 0xFFFF);
    this->sii_eeprom[SII_PRODUCT_CODE_OFFSET] = static_cast<uint16_t>(this->config.product_code & 0xFFFF);
    this->sii_eeprom[SII_PRODUCT_CODE_OFFSET + 1] = static_cast<uint16_t>((this->config.product_code >> 16) & 0xFFFF);
    this->sii_eeprom[SII_REVISION_OFFSET] = static_cast<uint16_t>(this->config.revision & 0xFFFF);
    this->sii_eeprom[SII_REVISION_OFFSET + 1] = static_cast<uint16_t>((this->config.revision >> 16) & 0xFFFF);
    this->sii_eeprom[SII_SERIAL_OFFSET] = static_cast<uint16_t>(this->config.serial & 0xFFFF);
    this->sii_eeprom[SII_SERIAL_OFFSET + 1] = static_cast<uint16_t>((this->config.serial >> 16) & 0xFFFF);
    this->sii_eeprom[0x0018] = this->config.mbx_protocols;
    this->sii_eeprom[0x001C] = this->config.mbx_out_addr;
    this->sii_eeprom[0x001D] = this->config.mbx_out_size;
    this->sii_eeprom[0x001E] = this->config.mbx_in_addr;
    this->sii_eeprom[0x001F] = this->config.mbx_in_size;
    this->sii_eeprom[0x0020] = this->config.mbx_out_addr;
    this->sii_eeprom[0x0021] = this->config.mbx_out_size;
    this->sii_eeprom[0x0022] = this->config.mbx_in_addr;
    this->sii_eeprom[0x0023] = this->config.mbx_in_size;
    size_t cat_offset = 0x0040;
    this->sii_eeprom[cat_offset++] = SII_CAT_GENERAL;
    this->sii_eeprom[cat_offset++] = 16;
    const uint16_t general_start = cat_offset;
    this->sii_eeprom[cat_offset++] = 0x0000;
    this->sii_eeprom[cat_offset++] = 0x0000;
    this->sii_eeprom[cat_offset++] = 0x0001;
    this->sii_eeprom[cat_offset++] = 0x0000;
    const uint16_t input_bits = static_cast<uint16_t>(this->config.tx_pdo_bytes() * 8);
    const uint16_t output_bits = static_cast<uint16_t>(this->config.rx_pdo_bytes() * 8);
    this->sii_eeprom[general_start + 4] = input_bits;
    this->sii_eeprom[general_start + 5] = output_bits;
    cat_offset = general_start + 16;
    this->sii_eeprom[cat_offset++] = SII_CAT_SYNCM;
    this->sii_eeprom[cat_offset++] = 16;
    const uint16_t output_bytes = static_cast<uint16_t>(this->config.rx_pdo_bytes());
    const uint16_t input_bytes_sm = static_cast<uint16_t>(this->config.tx_pdo_bytes());
    this->sii_eeprom[cat_offset++] = this->config.mbx_out_addr;
    this->sii_eeprom[cat_offset++] = this->config.mbx_out_size;
    this->sii_eeprom[cat_offset++] = 0x0026;
    this->sii_eeprom[cat_offset++] = 0x0101;
    this->sii_eeprom[cat_offset++] = this->config.mbx_in_addr;
    this->sii_eeprom[cat_offset++] = this->config.mbx_in_size;
    this->sii_eeprom[cat_offset++] = 0x0022;
    this->sii_eeprom[cat_offset++] = 0x0201;
    this->sii_eeprom[cat_offset++] = 0x1C00;
    this->sii_eeprom[cat_offset++] = output_bytes;
    this->sii_eeprom[cat_offset++] = 0x0064;
    this->sii_eeprom[cat_offset++] = 0x0301;
    this->sii_eeprom[cat_offset++] = 0x1D00;
    this->sii_eeprom[cat_offset++] = input_bytes_sm;
    this->sii_eeprom[cat_offset++] = 0x0020;
    this->sii_eeprom[cat_offset++] = 0x0401;
    for (const auto& pdo : this->config.tx_pdos) {
        this->sii_eeprom[cat_offset++] = SII_CAT_TXPDO;
        const size_t len_offset = cat_offset++;
        const size_t entry_start = cat_offset;
        this->sii_eeprom[cat_offset++] = pdo.index;
        this->sii_eeprom[cat_offset++] = static_cast<uint16_t>(pdo.entries.size());
        this->sii_eeprom[cat_offset++] = 0x0300;
        this->sii_eeprom[cat_offset++] = 0x0000;
        for (const auto& entry : pdo.entries) {
            this->sii_eeprom[cat_offset++] = entry.index;
            this->sii_eeprom[cat_offset++] = static_cast<uint16_t>(entry.sub_index | (0x00 << 8));
            this->sii_eeprom[cat_offset++] = static_cast<uint16_t>(entry.bit_length | (0x00 << 8));
            this->sii_eeprom[cat_offset++] = 0x0000;
        }
        this->sii_eeprom[len_offset] = static_cast<uint16_t>(cat_offset - entry_start);
    }
    for (const auto& pdo : this->config.rx_pdos) {
        this->sii_eeprom[cat_offset++] = SII_CAT_RXPDO;
        const size_t len_offset = cat_offset++;
        const size_t entry_start = cat_offset;
        this->sii_eeprom[cat_offset++] = pdo.index;
        this->sii_eeprom[cat_offset++] = static_cast<uint16_t>(pdo.entries.size());
        this->sii_eeprom[cat_offset++] = 0x0200;
        this->sii_eeprom[cat_offset++] = 0x0000;
        for (const auto& entry : pdo.entries) {
            this->sii_eeprom[cat_offset++] = entry.index;
            this->sii_eeprom[cat_offset++] = static_cast<uint16_t>(entry.sub_index | (0x00 << 8));
            this->sii_eeprom[cat_offset++] = static_cast<uint16_t>(entry.bit_length | (0x00 << 8));
            this->sii_eeprom[cat_offset++] = 0x0000;
        }
        this->sii_eeprom[len_offset] = static_cast<uint16_t>(cat_offset - entry_start);
    }
    this->sii_eeprom[cat_offset++] = SII_CAT_END;
    this->sii_eeprom[cat_offset++] = 0x0000;

    LOG(INFO) << "SII SYNCM at 0x52: " << std::hex
              << "cat=" << this->sii_eeprom[0x52]
              << " len=" << this->sii_eeprom[0x53];
    LOG(INFO) << "  SM2: start=" << this->sii_eeprom[0x5c]
              << " len=" << this->sii_eeprom[0x5d]
              << " ctrl=" << this->sii_eeprom[0x5e]
              << " type=" << this->sii_eeprom[0x5f];
    LOG(INFO) << "  SM3: start=" << this->sii_eeprom[0x60]
              << " len=" << this->sii_eeprom[0x61]
              << " ctrl=" << this->sii_eeprom[0x62]
              << " type=" << this->sii_eeprom[0x63];
}

void VirtualESC::init_registers() {
    this->registers.resize(REGISTER_SPACE_SIZE, 0);
    this->registers[0x0000] = 0x11;
    this->registers[0x0001] = 0x00;
    const uint16_t input_bytes = static_cast<uint16_t>(this->config.tx_pdo_bytes());
    const uint16_t output_bytes = static_cast<uint16_t>(this->config.rx_pdo_bytes());
    this->registers[REG_DL_STATUS] = 0x05;
    this->registers[REG_DL_STATUS + 1] = 0x00;
    this->registers[REG_SM0_START] = static_cast<uint8_t>(this->config.mbx_out_addr & 0xFF);
    this->registers[REG_SM0_START + 1] = static_cast<uint8_t>((this->config.mbx_out_addr >> 8) & 0xFF);
    this->registers[REG_SM0_LENGTH] = static_cast<uint8_t>(this->config.mbx_out_size & 0xFF);
    this->registers[REG_SM0_LENGTH + 1] = static_cast<uint8_t>((this->config.mbx_out_size >> 8) & 0xFF);
    this->registers[REG_SM0_CONTROL] = 0x26;
    constexpr uint16_t sm1_start = REG_SM0_START + SM_REG_SIZE;
    this->registers[sm1_start] = static_cast<uint8_t>(this->config.mbx_in_addr & 0xFF);
    this->registers[sm1_start + 1] = static_cast<uint8_t>((this->config.mbx_in_addr >> 8) & 0xFF);
    this->registers[sm1_start + 2] = static_cast<uint8_t>(this->config.mbx_in_size & 0xFF);
    this->registers[sm1_start + 3] = static_cast<uint8_t>((this->config.mbx_in_size >> 8) & 0xFF);
    this->registers[sm1_start + 4] = 0x22;
    constexpr uint16_t sm2_start = REG_SM0_START + 2 * SM_REG_SIZE;
    this->registers[sm2_start] = 0x00;
    this->registers[sm2_start + 1] = 0x1C;
    this->registers[sm2_start + 2] = static_cast<uint8_t>(output_bytes & 0xFF);
    this->registers[sm2_start + 3] = static_cast<uint8_t>((output_bytes >> 8) & 0xFF);
    this->registers[sm2_start + 4] = 0x64;
    constexpr uint16_t sm3_start = REG_SM0_START + 3 * SM_REG_SIZE;
    this->registers[sm3_start] = 0x00;
    this->registers[sm3_start + 1] = 0x1D;
    this->registers[sm3_start + 2] = static_cast<uint8_t>(input_bytes & 0xFF);
    this->registers[sm3_start + 3] = static_cast<uint8_t>((input_bytes >> 8) & 0xFF);
    this->registers[sm3_start + 4] = 0x20;
}

}
