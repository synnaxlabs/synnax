// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cerrno>
#include <string>
#include <system_error>

#include <fcntl.h>
#include <unistd.h>

#include "x/cpp/lib/lib.h"

#include "driver/errors/errors.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/igh/ecrt.h"

namespace driver::ethercat::igh {
/// @brief library path for the IgH EtherCAT master shared library.
const std::string IGH_LIBRARY_NAME = "libethercat.so.1";

/// @brief library information for error messages.
const driver::LibraryInfo IGH_LIB_INFO = {
    "IgH EtherCAT Master",
    "https://gitlab.com/etherlab.org/ethercat"
};

/// @brief error returned when the IgH library cannot be loaded.
const auto LOAD_ERROR = driver::errors::missing_lib(IGH_LIB_INFO);

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
        decltype(&ecrt_slave_config_reg_pdo_entry) slave_config_reg_pdo_entry;
        decltype(&ecrt_slave_config_sdo) slave_config_sdo;
        decltype(&ecrt_slave_config_pdos) slave_config_pdos;
        decltype(&ecrt_slave_config_pdo_assign_clear) slave_config_pdo_assign_clear;
        decltype(&ecrt_slave_config_pdo_mapping_clear) slave_config_pdo_mapping_clear;
        decltype(&ecrt_master_sdo_download) master_sdo_download;
        decltype(&ecrt_master_sdo_upload) master_sdo_upload;
        decltype(&ecrt_master_get_sync_manager) master_get_sync_manager;
        decltype(&ecrt_master_get_pdo) master_get_pdo;
        decltype(&ecrt_master_get_pdo_entry) master_get_pdo_entry;
    };

    /// @brief shared library handle.
    std::unique_ptr<x::lib::Shared> lib;
    /// @brief function pointers loaded from the library.
    FunctionPointers func_ptrs{};

    API(std::unique_ptr<x::lib::Shared> lib, const FunctionPointers &func_ptrs):
        lib(std::move(lib)), func_ptrs(func_ptrs) {}

protected:
    /// @brief tag type for constructing without loading shared library (testing only).
    struct MockTag {};

    /// @brief constructs an API without loading a shared library (for mock subclasses).
    explicit API(MockTag) {}

public:
    virtual ~API() = default;

    /// @brief loads the IgH EtherCAT library.
    /// @returns x::lib::LOAD_ERROR when the library is not installed, or when it lacks
    /// a function that the Driver calls.
    static std::pair<std::shared_ptr<API>, x::errors::Error> load() {
        auto lib = std::make_unique<x::lib::Shared>(IGH_LIBRARY_NAME);
        if (!lib->load()) return {nullptr, LOAD_ERROR};
        FunctionPointers f{};
        std::string missing;
        const auto resolve = [&]<typename F>(F &fn, const char *name) {
            fn = reinterpret_cast<F>(const_cast<void *>(lib->get_func_ptr(name)));
            if (fn == nullptr && missing.empty()) missing = name;
        };
        resolve(f.request_master, "ecrt_request_master");
        resolve(f.release_master, "ecrt_release_master");
        resolve(f.master_activate, "ecrt_master_activate");
        resolve(f.master_deactivate, "ecrt_master_deactivate");
        resolve(f.master_create_domain, "ecrt_master_create_domain");
        resolve(f.domain_size, "ecrt_domain_size");
        resolve(f.domain_data, "ecrt_domain_data");
        resolve(f.domain_process, "ecrt_domain_process");
        resolve(f.domain_queue, "ecrt_domain_queue");
        resolve(f.domain_state, "ecrt_domain_state");
        resolve(f.master_send, "ecrt_master_send");
        resolve(f.master_receive, "ecrt_master_receive");
        resolve(f.master, "ecrt_master");
        resolve(f.master_get_slave, "ecrt_master_get_slave");
        resolve(f.master_slave_config, "ecrt_master_slave_config");
        resolve(f.slave_config_state, "ecrt_slave_config_state");
        resolve(f.slave_config_reg_pdo_entry, "ecrt_slave_config_reg_pdo_entry");
        resolve(f.slave_config_sdo, "ecrt_slave_config_sdo");
        resolve(f.slave_config_pdos, "ecrt_slave_config_pdos");
        resolve(f.slave_config_pdo_assign_clear, "ecrt_slave_config_pdo_assign_clear");
        resolve(
            f.slave_config_pdo_mapping_clear,
            "ecrt_slave_config_pdo_mapping_clear"
        );
        resolve(f.master_sdo_download, "ecrt_master_sdo_download");
        resolve(f.master_sdo_upload, "ecrt_master_sdo_upload");
        resolve(f.master_get_sync_manager, "ecrt_master_get_sync_manager");
        resolve(f.master_get_pdo, "ecrt_master_get_pdo");
        resolve(f.master_get_pdo_entry, "ecrt_master_get_pdo_entry");
        if (!missing.empty())
            return {
                nullptr,
                x::errors::Error(
                    x::lib::LOAD_ERROR,
                    IGH_LIB_INFO.name + " library " + IGH_LIBRARY_NAME +
                        " has no function " + missing +
                        ". Install a newer version: " + IGH_LIB_INFO.url + "."
                )
            };
        return {std::shared_ptr<API>(new API(std::move(lib), f)), x::errors::NIL};
    }

    /// @brief checks that the process can open the device of a master.
    /// @param master_index the index of the master, whose device is
    /// /dev/EtherCAT<master_index>.
    /// @returns errors::PERMISSION_DENIED when the process lacks permission to open the
    /// device, or errors::MASTER_INIT_ERROR when it fails to open for another reason.
    [[nodiscard]] virtual x::errors::Error
    check_device(unsigned int master_index) const {
        const std::string path = "/dev/EtherCAT" + std::to_string(master_index);
        const int fd = ::open(path.c_str(), O_RDWR);
        if (fd >= 0) {
            ::close(fd);
            return x::errors::NIL;
        }
        const int err = errno;
        if (err == EACCES)
            return x::errors::Error(
                errors::PERMISSION_DENIED,
                "permission denied opening " + path +
                    ": reinstall the Driver service with 'sudo synnax-driver install', "
                    "or run the Core or the Driver as root"
            );
        return x::errors::Error(
            errors::MASTER_INIT_ERROR,
            "failed to open " + path + ": " + std::generic_category().message(err)
        );
    }

    /// @brief requests an EtherCAT master for realtime operation.
    [[nodiscard]] virtual ec_master_t *request_master(unsigned int master_index) const {
        return this->func_ptrs.request_master(master_index);
    }

    /// @brief releases a requested EtherCAT master.
    virtual void release_master(ec_master_t *master) const {
        this->func_ptrs.release_master(master);
    }

    /// @brief activates the master.
    [[nodiscard]] virtual int master_activate(ec_master_t *master) const {
        return this->func_ptrs.master_activate(master);
    }

    /// @brief deactivates the master.
    virtual int master_deactivate(ec_master_t *master) const {
        return this->func_ptrs.master_deactivate(master);
    }

    /// @brief creates a new process data domain.
    [[nodiscard]] virtual ec_domain_t *master_create_domain(ec_master_t *master) const {
        return this->func_ptrs.master_create_domain(master);
    }

    /// @brief returns the size of the domain's process data.
    [[nodiscard]] virtual size_t domain_size(const ec_domain_t *domain) const {
        return this->func_ptrs.domain_size(domain);
    }

    /// @brief returns a pointer to the domain's process data.
    [[nodiscard]] virtual uint8_t *domain_data(const ec_domain_t *domain) const {
        return this->func_ptrs.domain_data(domain);
    }

    /// @brief processes received datagrams.
    virtual int domain_process(ec_domain_t *domain) const {
        return this->func_ptrs.domain_process(domain);
    }

    /// @brief queues domain datagrams for sending.
    virtual int domain_queue(ec_domain_t *domain) const {
        return this->func_ptrs.domain_queue(domain);
    }

    /// @brief returns the current domain state.
    virtual int
    domain_state(const ec_domain_t *domain, ec_domain_state_t *state) const {
        return this->func_ptrs.domain_state(domain, state);
    }

    /// @brief sends all queued datagrams.
    virtual int master_send(ec_master_t *master) const {
        return this->func_ptrs.master_send(master);
    }

    /// @brief fetches received frames from the hardware.
    virtual int master_receive(ec_master_t *master) const {
        return this->func_ptrs.master_receive(master);
    }

    /// @brief obtains master information.
    [[nodiscard]] virtual int
    master(ec_master_t *master, ec_master_info_t *master_info) const {
        return this->func_ptrs.master(master, master_info);
    }

    /// @brief obtains slave information.
    [[nodiscard]] virtual int master_get_slave(
        ec_master_t *master,
        uint16_t slave_position,
        ec_slave_info_t *slave_info
    ) const {
        return this->func_ptrs.master_get_slave(master, slave_position, slave_info);
    }

    /// @brief obtains a slave configuration.
    [[nodiscard]] virtual ec_slave_config_t *master_slave_config(
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
    virtual int slave_config_state(
        const ec_slave_config_t *sc,
        ec_slave_config_state_t *state
    ) const {
        return this->func_ptrs.slave_config_state(sc, state);
    }

    /// @brief registers a PDO entry for process data exchange.
    [[nodiscard]] virtual int slave_config_reg_pdo_entry(
        ec_slave_config_t *sc,
        uint16_t entry_index,
        uint8_t entry_sub_index,
        ec_domain_t *domain,
        unsigned int *bit_position
    ) const {
        return this->func_ptrs.slave_config_reg_pdo_entry(
            sc,
            entry_index,
            entry_sub_index,
            domain,
            bit_position
        );
    }

    /// @brief adds an SDO the master writes each time it moves the slave to SAFE-OP.
    [[nodiscard]] virtual int slave_config_sdo(
        ec_slave_config_t *sc,
        uint16_t index,
        uint8_t subindex,
        const uint8_t *data,
        size_t size
    ) const {
        return this->func_ptrs.slave_config_sdo(sc, index, subindex, data, size);
    }

    /// @brief sets the PDO assignment and mapping the master configures on the slave.
    [[nodiscard]] virtual int slave_config_pdos(
        ec_slave_config_t *sc,
        unsigned int n_syncs,
        const ec_sync_info_t syncs[]
    ) const {
        return this->func_ptrs.slave_config_pdos(sc, n_syncs, syncs);
    }

    /// @brief makes the master assign no PDOs to a sync manager of the slave.
    [[nodiscard]] virtual int
    slave_config_pdo_assign_clear(ec_slave_config_t *sc, uint8_t sync_index) const {
        return this->func_ptrs.slave_config_pdo_assign_clear(sc, sync_index);
    }

    /// @brief makes the master map no entries to a PDO of the slave.
    [[nodiscard]] virtual int
    slave_config_pdo_mapping_clear(ec_slave_config_t *sc, uint16_t pdo_index) const {
        return this->func_ptrs.slave_config_pdo_mapping_clear(sc, pdo_index);
    }

    /// @brief writes an SDO to a slave and blocks until it answers, setting abort_code
    /// when the slave rejects the write.
    [[nodiscard]] virtual int master_sdo_download(
        ec_master_t *master,
        uint16_t slave_position,
        uint16_t index,
        uint8_t subindex,
        const uint8_t *data,
        size_t data_size,
        uint32_t *abort_code
    ) const {
        return this->func_ptrs.master_sdo_download(
            master,
            slave_position,
            index,
            subindex,
            data,
            data_size,
            abort_code
        );
    }

    /// @brief reads an SDO from a slave and blocks until it answers, setting abort_code
    /// when the slave rejects the read.
    [[nodiscard]] virtual int master_sdo_upload(
        ec_master_t *master,
        uint16_t slave_position,
        uint16_t index,
        uint8_t subindex,
        uint8_t *target,
        size_t target_size,
        size_t *result_size,
        uint32_t *abort_code
    ) const {
        return this->func_ptrs.master_sdo_upload(
            master,
            slave_position,
            index,
            subindex,
            target,
            target_size,
            result_size,
            abort_code
        );
    }

    /// @brief obtains sync manager information.
    [[nodiscard]] virtual int master_get_sync_manager(
        ec_master_t *master,
        uint16_t slave_position,
        uint8_t sync_index,
        ec_sync_info_t *sync
    ) const {
        return this->func_ptrs
            .master_get_sync_manager(master, slave_position, sync_index, sync);
    }

    /// @brief obtains PDO information.
    [[nodiscard]] virtual int master_get_pdo(
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
    [[nodiscard]] virtual int master_get_pdo_entry(
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
