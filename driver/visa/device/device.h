// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <unordered_map>
#include <mutex>

#include "glog/logging.h"

#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/visa/errors.h"
#include "driver/visa/api/types.h"
#include "driver/visa/api/api.h"

namespace visa::device {

using namespace visa_types;

/// @brief RAII wrapper for a VISA session.
template<typename API_T>
struct SessionImpl {
    ViSession session;
    std::shared_ptr<API_T> api;

    SessionImpl(ViSession sess, std::shared_ptr<API_T> api_):
        session(sess), api(std::move(api_)) {}

    ~SessionImpl() {
        if (session != VI_NULL && api != nullptr) {
            (void)api->close(session);  // Ignore errors in destructor
        }
    }

    // Prevent copying
    SessionImpl(const SessionImpl &) = delete;
    SessionImpl &operator=(const SessionImpl &) = delete;

    /// @brief reads data from the instrument.
    /// @param buffer the buffer to read into.
    /// @param count the maximum number of bytes to read.
    /// @param actual the actual number of bytes read (output parameter).
    /// @returns xerrors::NIL if successful, any other error otherwise.
    xerrors::Error read(uint8_t *buffer, const size_t count, size_t &actual) const {
        if (api == nullptr) return xerrors::Error("VISA API not available");
        ViUInt32 retCount;
        const ViStatus status = api->read(session, buffer, count, &retCount);
        actual = retCount;
        return parse_visa_error(api, status, session);
    }

    /// @brief writes data to the instrument.
    /// @param data the data to write.
    /// @param count the number of bytes to write.
    /// @param actual the actual number of bytes written (output parameter).
    /// @returns xerrors::NIL if successful, any other error otherwise.
    xerrors::Error write(const uint8_t *data, const size_t count, size_t &actual) const {
        if (api == nullptr) return xerrors::Error("VISA API not available");
        ViUInt32 retCount;
        const ViStatus status = api->write(session, const_cast<ViBuf>(data), count, &retCount);
        actual = retCount;
        return parse_visa_error(api, status, session);
    }

    /// @brief sends a query (write + read) to the instrument.
    /// @param command the SCPI command to send.
    /// @param response the buffer to store the response.
    /// @param max_len the maximum response length.
    /// @returns xerrors::NIL if successful, any other error otherwise.
    xerrors::Error query(
        const char *command,
        char *response,
        const size_t max_len
    ) const {
        // Write command
        size_t written;
        if (const auto err = write(
                reinterpret_cast<const uint8_t *>(command),
                strlen(command),
                written
            ); err)
            return err;

        // Read response
        size_t read_count;
        return read(reinterpret_cast<uint8_t *>(response), max_len - 1, read_count);
    }

    /// @brief sets the timeout for I/O operations.
    /// @param timeout_ms the timeout in milliseconds.
    /// @returns xerrors::NIL if successful, any other error otherwise.
    xerrors::Error set_timeout(const uint32_t timeout_ms) const {
        if (api == nullptr) return xerrors::Error("VISA API not available");
        constexpr ViUInt32 VI_ATTR_TMO_VALUE = 0x3FFF001A;
        const ViStatus status = api->set_attribute(session, VI_ATTR_TMO_VALUE, timeout_ms);
        return parse_visa_error(api, status, session);
    }

    /// @brief sets the termination character.
    /// @param term_char the termination character.
    /// @param enabled whether the termination character is enabled.
    /// @returns xerrors::NIL if successful, any other error otherwise.
    xerrors::Error set_term_char(const char term_char, const bool enabled) const {
        if (api == nullptr) return xerrors::Error("VISA API not available");
        constexpr ViUInt32 VI_ATTR_TERMCHAR_EN = 0x3FFF0038;
        constexpr ViUInt32 VI_ATTR_TERMCHAR = 0x3FFF0018;
        constexpr ViUInt32 VI_TRUE = 1;
        constexpr ViUInt32 VI_FALSE = 0;

        if (const ViStatus status = api->set_attribute(
                session,
                VI_ATTR_TERMCHAR_EN,
                enabled ? VI_TRUE : VI_FALSE
            ); status < VI_SUCCESS)
            return parse_visa_error(api, status, session);

        if (enabled) {
            const ViStatus status = api->set_attribute(session, VI_ATTR_TERMCHAR, term_char);
            return parse_visa_error(api, status, session);
        }

        return xerrors::NIL;
    }
};

/// @brief Type alias for Session using the real API.
using Session = SessionImpl<visa_api::API>;

/// @brief Configuration for a VISA connection.
struct ConnectionConfig {
    /// @brief The VISA resource name (e.g., "TCPIP0::192.168.1.100::INSTR")
    std::string resource_name;
    /// @brief The I/O timeout in milliseconds (default: 5000ms)
    uint32_t timeout_ms = 5000;
    /// @brief The termination character (default: '\n')
    char term_char = '\n';
    /// @brief Whether the termination character is enabled (default: true)
    bool term_char_enabled = true;

    ConnectionConfig() = default;

    ConnectionConfig(
        std::string resource_name,
        const uint32_t timeout_ms = 5000,
        const char term_char = '\n',
        const bool term_char_enabled = true
    ):
        resource_name(std::move(resource_name)),
        timeout_ms(timeout_ms),
        term_char(term_char),
        term_char_enabled(term_char_enabled) {}

    /// @brief constructs a ConnectionConfig from a JSON object.
    explicit ConnectionConfig(xjson::Parser parser):
        resource_name(parser.required<std::string>("resource_name")),
        timeout_ms(parser.optional<uint32_t>("timeout_ms", 5000)),
        term_char(parser.optional<std::string>("term_char", "\n")[0]),
        term_char_enabled(parser.optional<bool>("term_char_enabled", true)) {}

    /// @brief returns the JSON representation of the configuration.
    [[nodiscard]] json to_json() const {
        return {
            {"resource_name", resource_name},
            {"timeout_ms", timeout_ms},
            {"term_char", std::string(1, term_char)},
            {"term_char_enabled", term_char_enabled}
        };
    }
};

/// @brief Manages VISA sessions and provides connection pooling.
template<typename API_T = visa_api::API>
class ManagerImpl {
    /// @brief mutex to protect shared state.
    std::mutex mu;
    /// @brief the VISA API wrapper.
    std::shared_ptr<API_T> api;
    /// @brief the VISA resource manager session.
    ViSession resource_manager = VI_NULL;
    /// @brief cache of active sessions (weak pointers to allow cleanup).
    std::unordered_map<std::string, std::weak_ptr<SessionImpl<API_T>>> sessions;
    /// @brief whether the resource manager has been initialized.
    bool rm_initialized = false;

    /// @brief ensures the resource manager is initialized.
    /// @returns xerrors::NIL if successful, error otherwise.
    /// @note Caller must hold lock on mu.
    xerrors::Error ensure_rm_initialized() {
        if (rm_initialized) return xerrors::NIL;
        if (api == nullptr) return xerrors::Error("VISA API not available");

        const ViStatus status = api->open_default_rm(&resource_manager);
        if (status < VI_SUCCESS) {
            return parse_visa_error(api, status);
        }
        rm_initialized = true;
        return xerrors::NIL;
    }

public:
    explicit ManagerImpl(const std::shared_ptr<API_T> &api_): api(api_) {}

    ~ManagerImpl() {
        std::lock_guard lock(mu);
        if (rm_initialized && api != nullptr) {
            (void)api->close(resource_manager);  // Ignore errors in destructor
        }
    }

    /// @brief acquires a session to the specified VISA resource.
    /// @param config the connection configuration.
    /// @returns a pair containing the session and any error that occurred.
    std::pair<std::shared_ptr<SessionImpl<API_T>>, xerrors::Error>
    acquire(const ConnectionConfig &config) {
        std::lock_guard lock(mu);

        // Ensure resource manager is initialized
        if (const auto err = ensure_rm_initialized(); err) {
            return {nullptr, err};
        }

        const std::string &id = config.resource_name;

        // Check cache for existing session
        const auto it = sessions.find(id);
        if (it != sessions.end()) {
            const auto existing = it->second.lock();
            if (existing != nullptr) {
                return {existing, xerrors::NIL};
            }
            // Weak pointer expired, remove from cache
            sessions.erase(it);
        }

        // Open new session
        ViSession sess;
        const ViStatus status = api->open(
            resource_manager,
            const_cast<ViRsrc>(config.resource_name.c_str()),
            VI_NULL,
            VI_NULL,
            &sess
        );

        if (status < VI_SUCCESS) {
            return {nullptr, parse_visa_error(api, status)};
        }

        auto session = std::make_shared<SessionImpl<API_T>>(sess, api);

        // Configure timeout
        if (const auto err = session->set_timeout(config.timeout_ms); err) {
            LOG(WARNING) << "[visa] failed to set timeout for " << id << ": "
                         << err.message();
        }

        // Configure termination character
        if (const auto err = session->set_term_char(
                config.term_char,
                config.term_char_enabled
            ); err) {
            LOG(WARNING) << "[visa] failed to set term char for " << id << ": "
                         << err.message();
        }

        // Cache the session
        sessions[id] = session;

        return {session, xerrors::NIL};
    }

    /// @brief finds VISA resources matching the given expression.
    /// @param expr the VISA resource expression (e.g., "?*::INSTR")
    /// @param resources output vector of discovered resource names
    /// @returns xerrors::NIL if successful, error otherwise
    xerrors::Error find_resources(
        const std::string &expr,
        std::vector<std::string> &resources
    ) {
        std::lock_guard lock(mu);

        // Ensure resource manager is initialized
        if (const auto err = ensure_rm_initialized(); err) {
            return err;
        }

        ViFindList find_list;
        ViUInt32 ret_count;
        ViChar desc[VI_FIND_BUFLEN];

        const ViStatus status = api->find_rsrc(
            resource_manager,
            const_cast<ViString>(expr.c_str()),
            &find_list,
            &ret_count,
            desc
        );

        if (status < VI_SUCCESS && status != VI_ERROR_RSRC_NFOUND) {
            return parse_visa_error(api, status);
        }

        if (ret_count == 0) {
            return xerrors::NIL; // No devices found, not an error
        }

        // Add first resource
        resources.emplace_back(desc);

        // Add remaining resources
        for (ViUInt32 i = 1; i < ret_count; i++) {
            if (const ViStatus s = api->find_next(find_list, desc); s < VI_SUCCESS) {
                break;
            }
            resources.emplace_back(desc);
        }

        (void)api->close(find_list);  // Ignore close errors - we have the data we need
        return xerrors::NIL;
    }

    /// @brief queries the *IDN? from a resource (for device identification).
    /// @param resource_name the VISA resource name
    /// @param idn output string for the device identification
    /// @returns xerrors::NIL if successful, error otherwise
    xerrors::Error query_idn(const std::string &resource_name, std::string &idn) {
        std::lock_guard lock(mu);

        // Ensure resource manager is initialized
        if (const auto err = ensure_rm_initialized(); err) {
            return err;
        }

        // Temporarily open the device
        ViSession sess;
        const ViStatus open_status = api->open(
            resource_manager,
            const_cast<ViRsrc>(resource_name.c_str()),
            VI_NULL,
            VI_NULL,
            &sess
        );

        if (open_status < VI_SUCCESS) {
            return parse_visa_error(api, open_status);
        }

        // Send *IDN?
        const char *cmd = "*IDN?\n";
        ViUInt32 ret_count;
        (void)api->write(  // Ignore errors - best effort attempt
            sess,
            reinterpret_cast<ViBuf>(const_cast<char *>(cmd)),
            strlen(cmd),
            &ret_count
        );

        // Read response
        char buffer[256] = {0};
        (void)api->read(  // Ignore errors - device might not support *IDN?
            sess,
            reinterpret_cast<ViBuf>(buffer),
            sizeof(buffer) - 1,
            &ret_count
        );

        (void)api->close(sess);  // Ignore close errors

        // If we got data, use it
        if (strlen(buffer) > 0) {
            idn = std::string(buffer);
        }

        return xerrors::NIL; // Don't error if *IDN? fails - device might not support it
    }
};

/// @brief Type alias for the standard Manager using the real API.
using Manager = ManagerImpl<visa_api::API>;

}
