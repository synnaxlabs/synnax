// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <unordered_map>
#include <vector>
#include <cstring>

#include "driver/visa/api/types.h"

namespace visa_api {

using namespace visa_types;

/// @brief Mock VISA API for testing without real hardware.
class MockAPI {
public:
    /// @brief Configurable responses for device queries.
    struct DeviceResponse {
        std::string idn = "Mock Vendor,Mock Model,SN12345,1.0.0";
        std::unordered_map<std::string, std::string> command_responses;
    };

    /// @brief Configuration for mock behavior.
    struct Config {
        bool fail_open_rm = false;
        bool fail_open_session = false;
        bool fail_find_resources = false;
        ViStatus open_rm_status = VI_SUCCESS;
        ViStatus open_session_status = VI_SUCCESS;
        ViStatus timeout_on_read = false;
        std::vector<std::string> resources = {"TCPIP0::192.168.1.100::INSTR"};
        std::unordered_map<std::string, DeviceResponse> devices;
    };

private:
    Config cfg;
    ViSession next_session = 1000;
    ViSession resource_manager = 0;
    std::unordered_map<ViSession, std::string> session_to_resource;
    std::unordered_map<ViSession, std::vector<uint8_t>> pending_reads;

public:
    MockAPI(): cfg({}) {}
    explicit MockAPI(Config cfg): cfg(std::move(cfg)) {}

    /// @brief Mock viOpenDefaultRM.
    ViStatus open_default_rm(ViSession *rm) {
        if (cfg.fail_open_rm) return cfg.open_rm_status;
        resource_manager = next_session++;
        *rm = resource_manager;
        return VI_SUCCESS;
    }

    /// @brief Mock viOpen.
    ViStatus open(
        ViSession rm,
        ViRsrc resource_name,
        ViUInt32,
        ViUInt32,
        ViSession *session
    ) {
        if (cfg.fail_open_session) return cfg.open_session_status;
        *session = next_session++;
        session_to_resource[*session] = std::string(resource_name);
        return VI_SUCCESS;
    }

    /// @brief Mock viClose.
    ViStatus close(ViSession session) {
        session_to_resource.erase(session);
        pending_reads.erase(session);
        return VI_SUCCESS;
    }

    /// @brief Mock viWrite.
    ViStatus write(
        ViSession session,
        ViBuf buf,
        ViUInt32 count,
        ViUInt32 *ret_count
    ) {
        const std::string command(reinterpret_cast<const char *>(buf), count);
        *ret_count = count;

        // Handle *IDN? specially
        if (command.find("*IDN?") != std::string::npos) {
            const auto &resource = session_to_resource[session];
            auto it = cfg.devices.find(resource);
            if (it != cfg.devices.end()) {
                const auto &response = it->second.idn + "\n";
                pending_reads[session] = std::vector<uint8_t>(
                    response.begin(),
                    response.end()
                );
            }
            return VI_SUCCESS;
        }

        // Handle custom command responses
        const auto &resource = session_to_resource[session];
        auto dev_it = cfg.devices.find(resource);
        if (dev_it != cfg.devices.end()) {
            const auto &device = dev_it->second;
            // Try to find a matching response
            for (const auto &[cmd, resp]: device.command_responses) {
                if (command.find(cmd) != std::string::npos) {
                    pending_reads[session] = std::vector<uint8_t>(
                        resp.begin(),
                        resp.end()
                    );
                    return VI_SUCCESS;
                }
            }
        }

        // Default response for unknown commands
        const std::string default_resp = "0.0\n";
        pending_reads[session] = std::vector<uint8_t>(
            default_resp.begin(),
            default_resp.end()
        );
        return VI_SUCCESS;
    }

    /// @brief Mock viRead.
    ViStatus read(
        ViSession session,
        ViBuf buf,
        ViUInt32 count,
        ViUInt32 *ret_count
    ) {
        if (cfg.timeout_on_read) return VI_ERROR_TMO;

        auto it = pending_reads.find(session);
        if (it == pending_reads.end()) {
            *ret_count = 0;
            return VI_SUCCESS;
        }

        const auto &data = it->second;
        const size_t copy_len = std::min(
            static_cast<size_t>(count),
            data.size()
        );
        std::memcpy(buf, data.data(), copy_len);
        *ret_count = copy_len;
        buf[copy_len] = '\0'; // Null-terminate
        pending_reads.erase(it);
        return VI_SUCCESS;
    }

    /// @brief Mock viSetAttribute.
    ViStatus set_attribute(
        ViSession,
        ViUInt32,
        ViUInt32
    ) {
        return VI_SUCCESS;
    }

    /// @brief Mock viStatusDesc.
    ViStatus status_desc(
        ViSession,
        ViStatus status,
        ViChar desc[]
    ) {
        if (status == VI_ERROR_TMO) {
            strcpy(desc, "Timeout expired before operation completed.");
        } else if (status == VI_ERROR_CONN_LOST) {
            strcpy(desc, "Connection to device lost.");
        } else if (status == VI_ERROR_IO) {
            strcpy(desc, "I/O error occurred.");
        } else {
            snprintf(desc, 256, "Unknown error: 0x%08X", status);
        }
        return VI_SUCCESS;
    }

    /// @brief Mock viFindRsrc.
    ViStatus find_rsrc(
        ViSession,
        ViString,
        ViFindList *find_list,
        ViUInt32 *ret_count,
        ViChar desc[]
    ) {
        if (cfg.fail_find_resources) return VI_ERROR_RSRC_NFOUND;
        if (cfg.resources.empty()) {
            *ret_count = 0;
            return VI_ERROR_RSRC_NFOUND;
        }

        *ret_count = cfg.resources.size();
        *find_list = next_session++; // Use session as find list handle
        strcpy(desc, cfg.resources[0].c_str());
        return VI_SUCCESS;
    }

    /// @brief Mock viFindNext.
    ViStatus find_next(ViFindList, ViChar desc[]) {
        // For simplicity, just return error (all resources returned in first call)
        return VI_ERROR_RSRC_NFOUND;
    }
};

/// @brief Creates a mock API that matches the API interface but doesn't inherit.
class MockAPIWrapper {
    std::shared_ptr<MockAPI> mock;

public:
    explicit MockAPIWrapper(const MockAPI::Config &cfg):
        mock(std::make_shared<MockAPI>(cfg)) {}

    ViStatus open_default_rm(ViSession *rm) {
        return mock->open_default_rm(rm);
    }

    ViStatus open(
        ViSession rm,
        ViRsrc resource_name,
        ViUInt32 mode,
        ViUInt32 timeout,
        ViSession *session
    ) {
        return mock->open(rm, resource_name, mode, timeout, session);
    }

    ViStatus close(ViSession session) {
        return mock->close(session);
    }

    ViStatus write(
        ViSession session,
        ViBuf buf,
        ViUInt32 count,
        ViUInt32 *ret_count
    ) {
        return mock->write(session, buf, count, ret_count);
    }

    ViStatus read(
        ViSession session,
        ViBuf buf,
        ViUInt32 count,
        ViUInt32 *ret_count
    ) {
        return mock->read(session, buf, count, ret_count);
    }

    ViStatus set_attribute(
        ViSession session,
        ViUInt32 attr,
        ViUInt32 value
    ) {
        return mock->set_attribute(session, attr, value);
    }

    ViStatus status_desc(
        ViSession session,
        ViStatus status,
        ViChar desc[]
    ) {
        return mock->status_desc(session, status, desc);
    }

    ViStatus find_rsrc(
        ViSession rm,
        ViString expr,
        ViFindList *find_list,
        ViUInt32 *ret_count,
        ViChar desc[]
    ) {
        return mock->find_rsrc(rm, expr, find_list, ret_count, desc);
    }

    ViStatus find_next(ViFindList find_list, ViChar desc[]) {
        return mock->find_next(find_list, desc);
    }
};

}
