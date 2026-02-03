// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/xlib/xlib.h"

#include "driver/errors/errors.h"
#include "driver/ethercat/igh/ecrt.h"

namespace ethercat::igh {
/// @brief library path for the IgH EtherCAT master shared library.
const std::string IGH_LIBRARY_NAME = "libethercat.so.1";

/// @brief library information for error messages.
const driver::LibraryInfo IGH_LIB_INFO = {
    "IgH EtherCAT Master",
    "https://gitlab.com/etherlab.org/ethercat"
};

/// @brief error returned when the IgH library cannot be loaded.
const auto LOAD_ERROR = driver::missing_lib(IGH_LIB_INFO);

/// @brief API wrapper for IgH EtherCAT master library functions with dynamic loading.
class API {
    /// @brief function pointers to the IgH EtherCAT library functions.
    struct FunctionPointers {
        decltype(&ecrt_request_master) request_master;
        decltype(&ecrt_release_master) release_master;
        decltype(&ecrt_master_activate) master_activate;
        decltype(&ecrt_master_deactivate) master_deactivate;
        decltype(&ecrt_master_create_domain) master_create_domain;
        decltype(&ecrt_domain_size) domain_size;
        decltype(&ecrt_domain_data) domain_data;
        decltype(&ecrt_domain_process) domain_process;
        decltype(&ecrt_domain_queue) domain_queue;
        decltype(&ecrt_domain_state) domain_state;
        decltype(&ecrt_master_send) master_send;
        decltype(&ecrt_master_receive) master_receive;
        decltype(&ecrt_master) master;
        decltype(&ecrt_master_get_slave) master_get_slave;
        decltype(&ecrt_master_slave_config) master_slave_config;
        decltype(&ecrt_slave_config_state) slave_config_state;
        decltype(&ecrt_slave_config_pdos) slave_config_pdos;
        decltype(&ecrt_slave_config_reg_pdo_entry) slave_config_reg_pdo_entry;
        decltype(&ecrt_master_get_sync_manager) master_get_sync_manager;
        decltype(&ecrt_master_get_pdo) master_get_pdo;
        decltype(&ecrt_master_get_pdo_entry) master_get_pdo_entry;
    };

    /// @brief shared library handle.
    std::unique_ptr<xlib::SharedLib> lib;
    /// @brief function pointers loaded from the library.
    FunctionPointers func_ptrs;

public:
    explicit API(std::unique_ptr<xlib::SharedLib> lib): lib(std::move(lib)) {
        memset(&this->func_ptrs, 0, sizeof(this->func_ptrs));
        this->func_ptrs
            .request_master = reinterpret_cast<decltype(&ecrt_request_master)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_request_master"))
        );
        this->func_ptrs
            .release_master = reinterpret_cast<decltype(&ecrt_release_master)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_release_master"))
        );
        this->func_ptrs
            .master_activate = reinterpret_cast<decltype(&ecrt_master_activate)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_master_activate"))
        );
        this->func_ptrs
            .master_deactivate = reinterpret_cast<decltype(&ecrt_master_deactivate)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_master_deactivate"))
        );
        this->func_ptrs.master_create_domain = reinterpret_cast<
            decltype(&ecrt_master_create_domain)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_master_create_domain"))
        );
        this->func_ptrs.domain_size = reinterpret_cast<decltype(&ecrt_domain_size)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_domain_size"))
        );
        this->func_ptrs.domain_data = reinterpret_cast<decltype(&ecrt_domain_data)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_domain_data"))
        );
        this->func_ptrs
            .domain_process = reinterpret_cast<decltype(&ecrt_domain_process)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_domain_process"))
        );
        this->func_ptrs.domain_queue = reinterpret_cast<decltype(&ecrt_domain_queue)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_domain_queue"))
        );
        this->func_ptrs.domain_state = reinterpret_cast<decltype(&ecrt_domain_state)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_domain_state"))
        );
        this->func_ptrs.master_send = reinterpret_cast<decltype(&ecrt_master_send)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_master_send"))
        );
        this->func_ptrs
            .master_receive = reinterpret_cast<decltype(&ecrt_master_receive)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_master_receive"))
        );
        this->func_ptrs.master = reinterpret_cast<decltype(&ecrt_master)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_master"))
        );
        this->func_ptrs
            .master_get_slave = reinterpret_cast<decltype(&ecrt_master_get_slave)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_master_get_slave"))
        );
        this->func_ptrs.master_slave_config = reinterpret_cast<
            decltype(&ecrt_master_slave_config)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_master_slave_config"))
        );
        this->func_ptrs
            .slave_config_state = reinterpret_cast<decltype(&ecrt_slave_config_state)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_slave_config_state"))
        );
        this->func_ptrs
            .slave_config_pdos = reinterpret_cast<decltype(&ecrt_slave_config_pdos)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_slave_config_pdos"))
        );
        this->func_ptrs.slave_config_reg_pdo_entry = reinterpret_cast<
            decltype(&ecrt_slave_config_reg_pdo_entry)>(const_cast<void *>(
            this->lib->get_func_ptr("ecrt_slave_config_reg_pdo_entry")
        ));
        this->func_ptrs.master_get_sync_manager = reinterpret_cast<
            decltype(&ecrt_master_get_sync_manager)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_master_get_sync_manager"))
        );
        this->func_ptrs
            .master_get_pdo = reinterpret_cast<decltype(&ecrt_master_get_pdo)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_master_get_pdo"))
        );
        this->func_ptrs.master_get_pdo_entry = reinterpret_cast<
            decltype(&ecrt_master_get_pdo_entry)>(
            const_cast<void *>(this->lib->get_func_ptr("ecrt_master_get_pdo_entry"))
        );
    }

    /// @brief loads the IgH EtherCAT library and returns an API instance.
    /// @return pair of API instance and error (nil on success).
    static std::pair<std::shared_ptr<API>, xerrors::Error> load() {
        auto lib = std::make_unique<xlib::SharedLib>(IGH_LIBRARY_NAME);
        if (!lib->load()) return {nullptr, LOAD_ERROR};
        return {std::make_shared<API>(std::move(lib)), xerrors::NIL};
    }

    /// @brief requests an EtherCAT master for realtime operation.
    [[nodiscard]] ec_master_t *request_master(unsigned int master_index) const {
        return this->func_ptrs.request_master(master_index);
    }

    /// @brief releases a requested EtherCAT master.
    void release_master(ec_master_t *master) const {
        this->func_ptrs.release_master(master);
    }

    /// @brief activates the master.
    [[nodiscard]] int master_activate(ec_master_t *master) const {
        return this->func_ptrs.master_activate(master);
    }

    /// @brief deactivates the master.
    /// No [[nodiscard]]: called during cleanup/error recovery where failure cannot be
    /// meaningfully handled - we're already shutting down.
    int master_deactivate(ec_master_t *master) const {
        return this->func_ptrs.master_deactivate(master);
    }

    /// @brief creates a new process data domain.
    [[nodiscard]] ec_domain_t *master_create_domain(ec_master_t *master) const {
        return this->func_ptrs.master_create_domain(master);
    }

    /// @brief returns the size of the domain's process data.
    [[nodiscard]] size_t domain_size(const ec_domain_t *domain) const {
        return this->func_ptrs.domain_size(domain);
    }

    /// @brief returns a pointer to the domain's process data.
    [[nodiscard]] uint8_t *domain_data(const ec_domain_t *domain) const {
        return this->func_ptrs.domain_data(domain);
    }

    /// @brief processes received datagrams.
    /// No [[nodiscard]]: cyclic function - working counter state (checked via
    /// domain_state) is the proper error detection mechanism for communication issues.
    int domain_process(ec_domain_t *domain) const {
        return this->func_ptrs.domain_process(domain);
    }

    /// @brief queues domain datagrams for sending.
    /// No [[nodiscard]]: cyclic function - working counter state is the proper error
    /// detection mechanism for communication issues.
    int domain_queue(ec_domain_t *domain) const {
        return this->func_ptrs.domain_queue(domain);
    }

    /// @brief returns the current domain state.
    /// No [[nodiscard]]: populates state struct which is the primary output; return
    /// value is secondary and state is checked directly after call.
    int domain_state(const ec_domain_t *domain, ec_domain_state_t *state) const {
        return this->func_ptrs.domain_state(domain, state);
    }

    /// @brief sends all queued datagrams.
    /// No [[nodiscard]]: cyclic function - working counter state is the proper error
    /// detection mechanism for communication issues.
    int master_send(ec_master_t *master) const {
        return this->func_ptrs.master_send(master);
    }

    /// @brief fetches received frames from the hardware.
    /// No [[nodiscard]]: cyclic function - working counter state is the proper error
    /// detection mechanism for communication issues.
    int master_receive(ec_master_t *master) const {
        return this->func_ptrs.master_receive(master);
    }

    /// @brief obtains master information.
    [[nodiscard]] int master(ec_master_t *master, ec_master_info_t *master_info) const {
        return this->func_ptrs.master(master, master_info);
    }

    /// @brief obtains slave information.
    [[nodiscard]] int master_get_slave(
        ec_master_t *master,
        uint16_t slave_position,
        ec_slave_info_t *slave_info
    ) const {
        return this->func_ptrs.master_get_slave(master, slave_position, slave_info);
    }

    /// @brief obtains a slave configuration.
    [[nodiscard]] ec_slave_config_t *master_slave_config(
        ec_master_t *master,
        uint16_t alias,
        uint16_t position,
        uint32_t vendor_id,
        uint32_t product_code
    ) const {
        return this->func_ptrs
            .master_slave_config(master, alias, position, vendor_id, product_code);
    }

    /// @brief returns the state of a slave configuration.
    /// No [[nodiscard]]: informational query used for logging/monitoring. Populates
    /// state struct which is the primary output; failure is non-critical.
    int slave_config_state(
        const ec_slave_config_t *sc,
        ec_slave_config_state_t *state
    ) const {
        return this->func_ptrs.slave_config_state(sc, state);
    }

    /// @brief configures PDOs using sync info structures.
    [[nodiscard]] int slave_config_pdos(
        ec_slave_config_t *sc,
        unsigned int n_syncs,
        const ec_sync_info_t syncs[]
    ) const {
        return this->func_ptrs.slave_config_pdos(sc, n_syncs, syncs);
    }

    /// @brief registers a PDO entry for process data exchange.
    [[nodiscard]] int slave_config_reg_pdo_entry(
        ec_slave_config_t *sc,
        uint16_t entry_index,
        uint8_t entry_subindex,
        ec_domain_t *domain,
        unsigned int *bit_position
    ) const {
        return this->func_ptrs.slave_config_reg_pdo_entry(
            sc,
            entry_index,
            entry_subindex,
            domain,
            bit_position
        );
    }

    /// @brief obtains sync manager information.
    [[nodiscard]] int master_get_sync_manager(
        ec_master_t *master,
        uint16_t slave_position,
        uint8_t sync_index,
        ec_sync_info_t *sync
    ) const {
        return this->func_ptrs
            .master_get_sync_manager(master, slave_position, sync_index, sync);
    }

    /// @brief obtains PDO information.
    [[nodiscard]] int master_get_pdo(
        ec_master_t *master,
        uint16_t slave_position,
        uint8_t sync_index,
        uint16_t pos,
        ec_pdo_info_t *pdo
    ) const {
        return this->func_ptrs
            .master_get_pdo(master, slave_position, sync_index, pos, pdo);
    }

    /// @brief obtains PDO entry information.
    [[nodiscard]] int master_get_pdo_entry(
        ec_master_t *master,
        uint16_t slave_position,
        uint8_t sync_index,
        uint16_t pdo_pos,
        uint16_t entry_pos,
        ec_pdo_entry_info_t *entry
    ) const {
        return this->func_ptrs.master_get_pdo_entry(
            master,
            slave_position,
            sync_index,
            pdo_pos,
            entry_pos,
            entry
        );
    }
};
}
