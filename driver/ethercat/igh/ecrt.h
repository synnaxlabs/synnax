/*
 * EtherCAT master application interface (ecrt.h)
 *
 * Copyright (C) 2006-2024 Florian Pose, Ingenieurgemeinschaft IgH
 *
 * This file is part of the IgH EtherCAT master userspace library.
 *
 * The IgH EtherCAT master userspace library is free software; you can
 * redistribute it and/or modify it under the terms of the GNU Lesser General
 * Public License as published by the Free Software Foundation; version 2.1
 * of the License.
 *
 * The IgH EtherCAT master userspace library is distributed in the hope that
 * it will be useful, but WITHOUT ANY WARRANTY; without even the implied
 * warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the IgH EtherCAT master userspace library. If not, see
 * <http://www.gnu.org/licenses/>.
 *
 * ---
 *
 * The license mentioned above concerns the source code only. Using the
 * EtherCAT technology and brand is only permitted in compliance with the
 * industrial property and similar rights of Beckhoff Automation GmbH.
 *
 * ---
 *
 * vim: expandtab sw=4 sts=4 tw=78
 *
 * Vendored from IgH EtherCAT Master stable-1.6 branch.
 * Source: https://gitlab.com/etherlab.org/ethercat
 */

#ifndef __ECRT_H__
#define __ECRT_H__

#ifdef __KERNEL__
#include <asm/byteorder.h>
#include <linux/in.h>
#include <linux/time.h>
#include <linux/types.h>
#else
#include <netinet/in.h>
#include <stdint.h>
#include <stdlib.h>
#include <sys/time.h>
#endif

/******************************************************************************
 * Global definitions
 *****************************************************************************/

/** EtherCAT master version magic.
 *
 * This must match between userspace library and kernel module.
 */
#define ECRT_VER_MAJOR 1
#define ECRT_VER_MINOR 6

/** Convenience macro for version calculation. */
#define ECRT_VERSION(a, b) (((a) << 8) + (b))

/** Combined version magic number. */
#define ECRT_VERSION_MAGIC ECRT_VERSION(ECRT_VER_MAJOR, ECRT_VER_MINOR)

/** Feature flag: Master redundancy available. */
#define EC_HAVE_REDUNDANCY

/** Feature flag: Emergency ring available. */
#define EC_HAVE_EMERGENCY

/** Feature flag: Register access available. */
#define EC_HAVE_REG_ACCESS

/** Feature flag: Reference clock selection available. */
#define EC_HAVE_SELECT_REF_CLOCK

/** Feature flag: Reference clock time available. */
#define EC_HAVE_REF_CLOCK_TIME

/** Feature flag: Position-based register access available. */
#define EC_HAVE_REG_BY_POS

/** Feature flag: Sync-to time available. */
#define EC_HAVE_SYNC_TO

/** Feature flag: Slave config flags available. */
#define EC_HAVE_FLAGS

/** Feature flag: SoE requests available. */
#define EC_HAVE_SOE_REQUESTS

/** Feature flag: Scan progress available. */
#define EC_HAVE_SCAN_PROGRESS

/** Feature flag: IP address setting available. */
#define EC_HAVE_SET_IP

/** Feature flag: State timeout configuration available. */
#define EC_HAVE_STATE_TIMEOUT

#ifndef EC_PUBLIC_API
#if defined(ethercat_EXPORTS) && !defined(__KERNEL__)
#define EC_PUBLIC_API __attribute__((visibility("default")))
#else
#define EC_PUBLIC_API
#endif
#endif

/** End of list marker for PDO entry registration. */
#define EC_END ~0U

/** Maximum number of sync managers per slave. */
#define EC_MAX_SYNC_MANAGERS 16

/** Maximum string length for slave names. */
#define EC_MAX_STRING_LENGTH 64

/** Maximum number of ports per slave. */
#define EC_MAX_PORTS 4

/** Convert struct timeval to nanoseconds since 2000-01-01. */
#define EC_TIMEVAL2NANO(TV)                                                            \
    (((TV).tv_sec - 946684800ULL) * 1000000000ULL + (TV).tv_usec * 1000ULL)

/** Size of an emergency message in bytes. */
#define EC_COE_EMERGENCY_MSG_SIZE 8

/******************************************************************************
 * Data types
 *****************************************************************************/

struct ec_master;
typedef struct ec_master ec_master_t; /**< Master. */

struct ec_slave_config;
typedef struct ec_slave_config ec_slave_config_t; /**< Slave configuration. */

struct ec_domain;
typedef struct ec_domain ec_domain_t; /**< Domain. */

struct ec_sdo_request;
typedef struct ec_sdo_request ec_sdo_request_t; /**< SDO request. */

struct ec_soe_request;
typedef struct ec_soe_request ec_soe_request_t; /**< SoE request. */

struct ec_voe_handler;
typedef struct ec_voe_handler ec_voe_handler_t; /**< VoE handler. */

struct ec_reg_request;
typedef struct ec_reg_request ec_reg_request_t; /**< Register request. */

/******************************************************************************
 * Master state
 *****************************************************************************/

/** Master state.
 *
 * This structure is used to query the master state via ecrt_master_state().
 */
typedef struct {
    unsigned int slaves_responding; /**< Number of slaves responding. */
    unsigned int al_states : 4; /**< Application-layer states of all slaves. */
    unsigned int link_up : 1; /**< true if at least one link is up. */
} ec_master_state_t;

/******************************************************************************
 * Master link state
 *****************************************************************************/

/** Master link state.
 *
 * This structure is used to query the link state of a specific device via
 * ecrt_master_link_state().
 */
typedef struct {
    unsigned int slaves_responding; /**< Number of slaves responding. */
    unsigned int al_states : 4; /**< Application-layer states of all slaves. */
    unsigned int link_up : 1; /**< true if link is up. */
} ec_master_link_state_t;

/******************************************************************************
 * Slave configuration state
 *****************************************************************************/

/** Slave configuration state.
 *
 * This structure is used to query the state of a slave configuration via
 * ecrt_slave_config_state().
 */
typedef struct {
    unsigned int online : 1; /**< true if slave is online. */
    unsigned int operational : 1; /**< true if slave is in OP state. */
    unsigned int al_state : 4; /**< Current application-layer state. */
} ec_slave_config_state_t;

/******************************************************************************
 * Master information
 *****************************************************************************/

/** Master information.
 *
 * This structure is used to query master information via ecrt_master().
 */
typedef struct {
    unsigned int slave_count; /**< Number of slaves on the bus. */
    unsigned int link_up : 1; /**< true if at least one link is up. */
    uint8_t scan_busy; /**< true if bus scan is in progress. */
    uint64_t app_time; /**< Application time. */
} ec_master_info_t;

/******************************************************************************
 * Master scan progress
 *****************************************************************************/

/** Master scan progress.
 *
 * This structure is used to query the bus scan progress.
 */
typedef struct {
    unsigned int slave_count; /**< Total number of slaves to scan. */
    unsigned int scan_index; /**< Current scan index. */
} ec_master_scan_progress_t;

/******************************************************************************
 * Slave port descriptor
 *****************************************************************************/

/** Slave port type. */
typedef enum {
    EC_PORT_NOT_IMPLEMENTED, /**< Port not implemented. */
    EC_PORT_NOT_CONFIGURED, /**< Port not configured. */
    EC_PORT_EBUS, /**< EBUS port. */
    EC_PORT_MII /**< MII port. */
} ec_slave_port_desc_t;

/******************************************************************************
 * Slave port link state
 *****************************************************************************/

/** Slave port link state. */
typedef struct {
    uint8_t link_up; /**< Link detected. */
    uint8_t loop_closed; /**< Loop closed. */
    uint8_t signal_detected; /**< Signal detected. */
} ec_slave_port_link_t;

/******************************************************************************
 * Slave information
 *****************************************************************************/

/** Slave information.
 *
 * This structure is used to query slave information via ecrt_master_get_slave().
 */
typedef struct {
    uint16_t position; /**< Position on the bus. */
    uint32_t vendor_id; /**< Vendor ID. */
    uint32_t product_code; /**< Product code. */
    uint32_t revision_number; /**< Revision number. */
    uint32_t serial_number; /**< Serial number. */
    uint16_t alias; /**< Station alias. */
    int16_t current_on_ebus; /**< Current consumption in mA. */
    struct {
        ec_slave_port_desc_t desc; /**< Port type. */
        ec_slave_port_link_t link; /**< Link state. */
        uint32_t receive_time; /**< Receive time. */
        uint16_t next_slave; /**< Next slave on port. */
        uint32_t delay_to_next_dc; /**< Delay to next slave [ns]. */
    } ports[EC_MAX_PORTS]; /**< Port information. */
    uint8_t al_state; /**< Current application-layer state. */
    uint8_t error_flag; /**< Error flag. */
    uint8_t sync_count; /**< Number of sync managers. */
    uint16_t sdo_count; /**< Number of SDOs in dictionary. */
    char name[EC_MAX_STRING_LENGTH]; /**< Slave name from SII. */
} ec_slave_info_t;

/******************************************************************************
 * Working counter state
 *****************************************************************************/

/** Working counter interpretation. */
typedef enum {
    EC_WC_ZERO = 0, /**< No registered process data was exchanged. */
    EC_WC_INCOMPLETE, /**< Some of the registered process data was exchanged. */
    EC_WC_COMPLETE /**< All registered process data was exchanged. */
} ec_wc_state_t;

/******************************************************************************
 * Domain state
 *****************************************************************************/

/** Domain state.
 *
 * This structure is used to query the domain state via ecrt_domain_state().
 */
typedef struct {
    unsigned int working_counter; /**< Working counter value. */
    ec_wc_state_t wc_state; /**< Working counter interpretation. */
    unsigned int redundancy_active; /**< Redundancy is active. */
} ec_domain_state_t;

/******************************************************************************
 * Direction
 *****************************************************************************/

/** PDO direction. */
typedef enum {
    EC_DIR_INVALID, /**< Invalid direction. */
    EC_DIR_OUTPUT, /**< Output (master to slave). */
    EC_DIR_INPUT, /**< Input (slave to master). */
    EC_DIR_COUNT /**< Number of directions. */
} ec_direction_t;

/******************************************************************************
 * Watchdog mode
 *****************************************************************************/

/** Watchdog mode. */
typedef enum {
    EC_WD_DEFAULT, /**< Use default watchdog behavior. */
    EC_WD_ENABLE, /**< Enable watchdog. */
    EC_WD_DISABLE /**< Disable watchdog. */
} ec_watchdog_mode_t;

/******************************************************************************
 * PDO entry information
 *****************************************************************************/

/** PDO entry information.
 *
 * This structure describes a single PDO entry (mapped object).
 */
typedef struct {
    uint16_t index; /**< Index of the mapped object. */
    uint8_t subindex; /**< Subindex of the mapped object. */
    uint8_t bit_length; /**< Size in bits. */
} ec_pdo_entry_info_t;

/******************************************************************************
 * PDO information
 *****************************************************************************/

/** PDO information.
 *
 * This structure describes a PDO.
 */
typedef struct {
    uint16_t index; /**< PDO index. */
    unsigned int n_entries; /**< Number of entries in the PDO. */
    ec_pdo_entry_info_t const *entries; /**< Array of PDO entries. */
} ec_pdo_info_t;

/******************************************************************************
 * Sync manager information
 *****************************************************************************/

/** Sync manager configuration information.
 *
 * This structure is used to configure sync managers including their PDOs
 * via ecrt_slave_config_pdos().
 */
typedef struct {
    uint8_t index; /**< Sync manager index. */
    ec_direction_t dir; /**< Sync manager direction. */
    unsigned int n_pdos; /**< Number of PDOs in this sync manager. */
    ec_pdo_info_t const *pdos; /**< Array of PDOs. */
    ec_watchdog_mode_t watchdog_mode; /**< Watchdog mode. */
} ec_sync_info_t;

/******************************************************************************
 * PDO entry registration
 *****************************************************************************/

/** List entry for PDO registration.
 *
 * This structure is used for registering multiple PDO entries at once via
 * ecrt_domain_reg_pdo_entry_list().
 */
typedef struct {
    uint16_t alias; /**< Slave alias address. */
    uint16_t position; /**< Slave position. */
    uint32_t vendor_id; /**< Expected vendor ID. */
    uint32_t product_code; /**< Expected product code. */
    uint16_t index; /**< PDO entry index. */
    uint8_t subindex; /**< PDO entry subindex. */
    unsigned int *offset; /**< Pointer to store byte offset. */
    unsigned int *bit_position; /**< Pointer to store bit position. */
} ec_pdo_entry_reg_t;

/******************************************************************************
 * Request state
 *****************************************************************************/

/** Request state.
 *
 * This is used to represent the current state of a request (SDO, SoE, etc.).
 */
typedef enum {
    EC_REQUEST_UNUSED, /**< Request not in use. */
    EC_REQUEST_BUSY, /**< Request is being processed. */
    EC_REQUEST_SUCCESS, /**< Request was successful. */
    EC_REQUEST_ERROR /**< Request failed. */
} ec_request_state_t;

/******************************************************************************
 * Application-layer state
 *****************************************************************************/

/** Application-layer state. */
typedef enum {
    EC_AL_STATE_INIT = 1, /**< Init state. */
    EC_AL_STATE_PREOP = 2, /**< Pre-operational state. */
    EC_AL_STATE_SAFEOP = 4, /**< Safe-operational state. */
    EC_AL_STATE_OP = 8 /**< Operational state. */
} ec_al_state_t;

#ifdef __cplusplus
extern "C" {
#endif

/******************************************************************************
 * Global functions
 *****************************************************************************/

/** Returns the version magic of the realtime interface.
 *
 * \return Version magic (ECRT_VERSION_MAGIC).
 */
EC_PUBLIC_API unsigned int ecrt_version_magic(void);

/** Requests an EtherCAT master for realtime operation.
 *
 * \param master_index Index of the master to request (0 for first master).
 * \return Pointer to the master, or NULL on error.
 */
EC_PUBLIC_API ec_master_t *ecrt_request_master(unsigned int master_index);

#ifndef __KERNEL__
/** Opens an EtherCAT master for userspace access.
 *
 * \param master_index Index of the master to open.
 * \return Pointer to the master, or NULL on error.
 */
EC_PUBLIC_API ec_master_t *ecrt_open_master(unsigned int master_index);
#endif

/** Releases a requested/opened EtherCAT master.
 *
 * \param master Master to release.
 */
EC_PUBLIC_API void ecrt_release_master(ec_master_t *master);

/******************************************************************************
 * Master methods
 *****************************************************************************/

#ifndef __KERNEL__
/** Reserves an EtherCAT master for exclusive use.
 *
 * \param master Master to reserve.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_reserve(ec_master_t *master);
#endif

#ifdef __KERNEL__
/** Set callbacks for sending and receiving frames.
 *
 * \param master Master.
 * \param send_cb Send callback.
 * \param receive_cb Receive callback.
 * \param cb_data Callback data.
 */
void ecrt_master_callbacks(
    ec_master_t *master,
    void (*send_cb)(void *),
    void (*receive_cb)(void *),
    void *cb_data
);
#endif

/** Creates a new process data domain.
 *
 * \param master Master.
 * \return Pointer to the new domain, or NULL on error.
 */
EC_PUBLIC_API ec_domain_t *ecrt_master_create_domain(ec_master_t *master);

/** Obtains a slave configuration.
 *
 * \param master Master.
 * \param alias Slave alias (0 if not used).
 * \param position Slave position on the bus.
 * \param vendor_id Expected vendor ID.
 * \param product_code Expected product code.
 * \return Pointer to the slave configuration, or NULL on error.
 */
EC_PUBLIC_API ec_slave_config_t *ecrt_master_slave_config(
    ec_master_t *master,
    uint16_t alias,
    uint16_t position,
    uint32_t vendor_id,
    uint32_t product_code
);

/** Selects the reference clock slave.
 *
 * \param master Master.
 * \param sc Slave configuration to use as reference clock.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_master_select_reference_clock(ec_master_t *master, ec_slave_config_t *sc);

/** Obtains master information.
 *
 * \param master Master.
 * \param master_info Pointer to store master information.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master(ec_master_t *master, ec_master_info_t *master_info);

/** Obtains bus scan progress.
 *
 * \param master Master.
 * \param progress Pointer to store scan progress.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_master_scan_progress(ec_master_t *master, ec_master_scan_progress_t *progress);

/** Obtains slave information.
 *
 * \param master Master.
 * \param slave_position Position of the slave on the bus.
 * \param slave_info Pointer to store slave information.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_get_slave(
    ec_master_t *master,
    uint16_t slave_position,
    ec_slave_info_t *slave_info
);

#ifndef __KERNEL__
/** Obtains sync manager information.
 *
 * \param master Master.
 * \param slave_position Position of the slave on the bus.
 * \param sync_index Sync manager index.
 * \param sync Pointer to store sync manager information.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_get_sync_manager(
    ec_master_t *master,
    uint16_t slave_position,
    uint8_t sync_index,
    ec_sync_info_t *sync
);

/** Obtains PDO information.
 *
 * \param master Master.
 * \param slave_position Position of the slave on the bus.
 * \param sync_index Sync manager index.
 * \param pos PDO position in sync manager.
 * \param pdo Pointer to store PDO information.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_get_pdo(
    ec_master_t *master,
    uint16_t slave_position,
    uint8_t sync_index,
    uint16_t pos,
    ec_pdo_info_t *pdo
);

/** Obtains PDO entry information.
 *
 * \param master Master.
 * \param slave_position Position of the slave on the bus.
 * \param sync_index Sync manager index.
 * \param pdo_pos PDO position in sync manager.
 * \param entry_pos Entry position in PDO.
 * \param entry Pointer to store PDO entry information.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_get_pdo_entry(
    ec_master_t *master,
    uint16_t slave_position,
    uint8_t sync_index,
    uint16_t pdo_pos,
    uint16_t entry_pos,
    ec_pdo_entry_info_t *entry
);
#endif

/** Downloads an SDO to a slave.
 *
 * \param master Master.
 * \param slave_position Position of the slave on the bus.
 * \param index SDO index.
 * \param subindex SDO subindex.
 * \param data Data to download.
 * \param data_size Size of data in bytes.
 * \param abort_code Pointer to store abort code on error.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_sdo_download(
    ec_master_t *master,
    uint16_t slave_position,
    uint16_t index,
    uint8_t subindex,
    const uint8_t *data,
    size_t data_size,
    uint32_t *abort_code
);

/** Downloads a complete SDO to a slave.
 *
 * \param master Master.
 * \param slave_position Position of the slave on the bus.
 * \param index SDO index.
 * \param data Data to download.
 * \param data_size Size of data in bytes.
 * \param abort_code Pointer to store abort code on error.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_sdo_download_complete(
    ec_master_t *master,
    uint16_t slave_position,
    uint16_t index,
    const uint8_t *data,
    size_t data_size,
    uint32_t *abort_code
);

/** Uploads an SDO from a slave.
 *
 * \param master Master.
 * \param slave_position Position of the slave on the bus.
 * \param index SDO index.
 * \param subindex SDO subindex.
 * \param target Buffer to store uploaded data.
 * \param target_size Size of target buffer in bytes.
 * \param result_size Pointer to store actual data size.
 * \param abort_code Pointer to store abort code on error.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_sdo_upload(
    ec_master_t *master,
    uint16_t slave_position,
    uint16_t index,
    uint8_t subindex,
    uint8_t *target,
    size_t target_size,
    size_t *result_size,
    uint32_t *abort_code
);

/** Writes an SoE IDN to a slave.
 *
 * \param master Master.
 * \param slave_position Position of the slave on the bus.
 * \param drive_no Drive number.
 * \param idn IDN to write.
 * \param data Data to write.
 * \param data_size Size of data in bytes.
 * \param error_code Pointer to store error code.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_write_idn(
    ec_master_t *master,
    uint16_t slave_position,
    uint8_t drive_no,
    uint16_t idn,
    const uint8_t *data,
    size_t data_size,
    uint16_t *error_code
);

/** Reads an SoE IDN from a slave.
 *
 * \param master Master.
 * \param slave_position Position of the slave on the bus.
 * \param drive_no Drive number.
 * \param idn IDN to read.
 * \param target Buffer to store read data.
 * \param target_size Size of target buffer in bytes.
 * \param result_size Pointer to store actual data size.
 * \param error_code Pointer to store error code.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_read_idn(
    ec_master_t *master,
    uint16_t slave_position,
    uint8_t drive_no,
    uint16_t idn,
    uint8_t *target,
    size_t target_size,
    size_t *result_size,
    uint16_t *error_code
);

/** Activates the master.
 *
 * \param master Master.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_activate(ec_master_t *master);

/** Deactivates the master.
 *
 * \param master Master.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_deactivate(ec_master_t *master);

/** Sets the send interval for cyclic operation.
 *
 * \param master Master.
 * \param send_interval Send interval in microseconds.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_master_set_send_interval(ec_master_t *master, size_t send_interval);

/** Sends all queued datagrams.
 *
 * \param master Master.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_send(ec_master_t *master);

/** Fetches received frames from the hardware.
 *
 * \param master Master.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_receive(ec_master_t *master);

#ifdef __KERNEL__
/** Sends all queued datagrams with external memory.
 *
 * \param master Master.
 * \return 0 on success, otherwise negative error code.
 */
int ecrt_master_send_ext(ec_master_t *master);
#endif

/** Returns the current master state.
 *
 * \param master Master.
 * \param state Pointer to store master state.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_master_state(const ec_master_t *master, ec_master_state_t *state);

/** Returns the state of a specific link.
 *
 * \param master Master.
 * \param dev_idx Device index (0 = main, 1 = backup).
 * \param state Pointer to store link state.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_link_state(
    const ec_master_t *master,
    unsigned int dev_idx,
    ec_master_link_state_t *state
);

/** Sets the application time.
 *
 * \param master Master.
 * \param app_time Application time in nanoseconds.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_application_time(ec_master_t *master, uint64_t app_time);

/** Queues the DC reference clock drift compensation datagram.
 *
 * \param master Master.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_sync_reference_clock(ec_master_t *master);

/** Queues the DC reference clock drift compensation datagram with custom time.
 *
 * \param master Master.
 * \param sync_time Sync time in nanoseconds.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_master_sync_reference_clock_to(ec_master_t *master, uint64_t sync_time);

/** Queues the DC clock drift compensation datagram for all slaves.
 *
 * \param master Master.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_sync_slave_clocks(ec_master_t *master);

/** Returns the reference clock time.
 *
 * \param master Master.
 * \param time Pointer to store reference clock time.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_master_reference_clock_time(const ec_master_t *master, uint32_t *time);

/** Queues the DC sync monitoring datagram.
 *
 * \param master Master.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_sync_monitor_queue(ec_master_t *master);

/** Processes the DC sync monitoring datagram.
 *
 * \param master Master.
 * \return Sync monitor value.
 */
EC_PUBLIC_API uint32_t ecrt_master_sync_monitor_process(const ec_master_t *master);

/** Resets the master.
 *
 * \param master Master.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_master_reset(ec_master_t *master);

/******************************************************************************
 * Slave configuration methods
 *****************************************************************************/

/** Configures a sync manager.
 *
 * \param sc Slave configuration.
 * \param sync_index Sync manager index.
 * \param direction Sync manager direction.
 * \param watchdog_mode Watchdog mode.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_sync_manager(
    ec_slave_config_t *sc,
    uint8_t sync_index,
    ec_direction_t direction,
    ec_watchdog_mode_t watchdog_mode
);

/** Configures the watchdog times.
 *
 * \param sc Slave configuration.
 * \param watchdog_divider Watchdog divider.
 * \param watchdog_intervals Watchdog intervals.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_watchdog(
    ec_slave_config_t *sc,
    uint16_t watchdog_divider,
    uint16_t watchdog_intervals
);

/** Adds a PDO to the sync manager's PDO assignment.
 *
 * \param sc Slave configuration.
 * \param sync_index Sync manager index.
 * \param index PDO index to add.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_pdo_assign_add(
    ec_slave_config_t *sc,
    uint8_t sync_index,
    uint16_t index
);

/** Clears a sync manager's PDO assignment.
 *
 * \param sc Slave configuration.
 * \param sync_index Sync manager index.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_slave_config_pdo_assign_clear(ec_slave_config_t *sc, uint8_t sync_index);

/** Adds an entry to a PDO's mapping.
 *
 * \param sc Slave configuration.
 * \param pdo_index PDO index.
 * \param entry_index Entry index.
 * \param entry_subindex Entry subindex.
 * \param entry_bit_length Entry bit length.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_pdo_mapping_add(
    ec_slave_config_t *sc,
    uint16_t pdo_index,
    uint16_t entry_index,
    uint8_t entry_subindex,
    uint8_t entry_bit_length
);

/** Clears a PDO's mapping.
 *
 * \param sc Slave configuration.
 * \param pdo_index PDO index.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_slave_config_pdo_mapping_clear(ec_slave_config_t *sc, uint16_t pdo_index);

/** Configures PDOs using sync info structures.
 *
 * \param sc Slave configuration.
 * \param n_syncs Number of sync info structures.
 * \param syncs Array of sync info structures.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_pdos(
    ec_slave_config_t *sc,
    unsigned int n_syncs,
    const ec_sync_info_t syncs[]
);

/** Registers a PDO entry for process data exchange.
 *
 * \param sc Slave configuration.
 * \param entry_index PDO entry index.
 * \param entry_subindex PDO entry subindex.
 * \param domain Domain to register in.
 * \param bit_position Pointer to store bit position (can be NULL).
 * \return Byte offset on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_reg_pdo_entry(
    ec_slave_config_t *sc,
    uint16_t entry_index,
    uint8_t entry_subindex,
    ec_domain_t *domain,
    unsigned int *bit_position
);

/** Registers a PDO entry by position.
 *
 * \param sc Slave configuration.
 * \param sync_index Sync manager index.
 * \param pdo_pos PDO position.
 * \param entry_pos Entry position.
 * \param domain Domain to register in.
 * \param bit_position Pointer to store bit position (can be NULL).
 * \return Byte offset on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_reg_pdo_entry_pos(
    ec_slave_config_t *sc,
    uint8_t sync_index,
    unsigned int pdo_pos,
    unsigned int entry_pos,
    ec_domain_t *domain,
    unsigned int *bit_position
);

/** Configures distributed clocks for a slave.
 *
 * \param sc Slave configuration.
 * \param assign_activate AssignActivate word.
 * \param sync0_cycle SYNC0 cycle time in ns.
 * \param sync0_shift SYNC0 shift time in ns.
 * \param sync1_cycle SYNC1 cycle time in ns.
 * \param sync1_shift SYNC1 shift time in ns.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_dc(
    ec_slave_config_t *sc,
    uint16_t assign_activate,
    uint32_t sync0_cycle,
    int32_t sync0_shift,
    uint32_t sync1_cycle,
    int32_t sync1_shift
);

/** Adds an SDO configuration.
 *
 * \param sc Slave configuration.
 * \param index SDO index.
 * \param subindex SDO subindex.
 * \param data SDO data.
 * \param size Size of data in bytes.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_sdo(
    ec_slave_config_t *sc,
    uint16_t index,
    uint8_t subindex,
    const uint8_t *data,
    size_t size
);

/** Adds an 8-bit SDO configuration.
 *
 * \param sc Slave configuration.
 * \param sdo_index SDO index.
 * \param sdo_subindex SDO subindex.
 * \param value 8-bit value.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_sdo8(
    ec_slave_config_t *sc,
    uint16_t sdo_index,
    uint8_t sdo_subindex,
    uint8_t value
);

/** Adds a 16-bit SDO configuration.
 *
 * \param sc Slave configuration.
 * \param sdo_index SDO index.
 * \param sdo_subindex SDO subindex.
 * \param value 16-bit value.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_sdo16(
    ec_slave_config_t *sc,
    uint16_t sdo_index,
    uint8_t sdo_subindex,
    uint16_t value
);

/** Adds a 32-bit SDO configuration.
 *
 * \param sc Slave configuration.
 * \param sdo_index SDO index.
 * \param sdo_subindex SDO subindex.
 * \param value 32-bit value.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_sdo32(
    ec_slave_config_t *sc,
    uint16_t sdo_index,
    uint8_t sdo_subindex,
    uint32_t value
);

/** Adds a complete SDO configuration.
 *
 * \param sc Slave configuration.
 * \param index SDO index.
 * \param data SDO data.
 * \param size Size of data in bytes.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_complete_sdo(
    ec_slave_config_t *sc,
    uint16_t index,
    const uint8_t *data,
    size_t size
);

/** Configures the emergency ring buffer size.
 *
 * \param sc Slave configuration.
 * \param elements Number of elements in ring buffer.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_emerg_size(ec_slave_config_t *sc, size_t elements);

/** Pops an emergency message from the ring buffer.
 *
 * \param sc Slave configuration.
 * \param target Buffer to store emergency message (8 bytes).
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_emerg_pop(ec_slave_config_t *sc, uint8_t *target);

/** Clears the emergency ring buffer.
 *
 * \param sc Slave configuration.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_emerg_clear(ec_slave_config_t *sc);

/** Returns the number of emergency overruns.
 *
 * \param sc Slave configuration.
 * \return Number of overruns.
 */
EC_PUBLIC_API int ecrt_slave_config_emerg_overruns(const ec_slave_config_t *sc);

/** Creates an SDO request.
 *
 * \param sc Slave configuration.
 * \param index SDO index.
 * \param subindex SDO subindex.
 * \param size Maximum data size.
 * \return Pointer to SDO request, or NULL on error.
 */
EC_PUBLIC_API ec_sdo_request_t *ecrt_slave_config_create_sdo_request(
    ec_slave_config_t *sc,
    uint16_t index,
    uint8_t subindex,
    size_t size
);

/** Creates an SoE request.
 *
 * \param sc Slave configuration.
 * \param drive_no Drive number.
 * \param idn IDN.
 * \param size Maximum data size.
 * \return Pointer to SoE request, or NULL on error.
 */
EC_PUBLIC_API ec_soe_request_t *ecrt_slave_config_create_soe_request(
    ec_slave_config_t *sc,
    uint8_t drive_no,
    uint16_t idn,
    size_t size
);

/** Creates a VoE handler.
 *
 * \param sc Slave configuration.
 * \param size Maximum data size.
 * \return Pointer to VoE handler, or NULL on error.
 */
EC_PUBLIC_API ec_voe_handler_t *
ecrt_slave_config_create_voe_handler(ec_slave_config_t *sc, size_t size);

/** Creates a register request.
 *
 * \param sc Slave configuration.
 * \param size Maximum data size.
 * \return Pointer to register request, or NULL on error.
 */
EC_PUBLIC_API ec_reg_request_t *
ecrt_slave_config_create_reg_request(ec_slave_config_t *sc, size_t size);

/** Returns the state of a slave configuration.
 *
 * \param sc Slave configuration.
 * \param state Pointer to store configuration state.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_slave_config_state(const ec_slave_config_t *sc, ec_slave_config_state_t *state);

/** Adds an SoE IDN configuration.
 *
 * \param sc Slave configuration.
 * \param drive_no Drive number.
 * \param idn IDN.
 * \param state Application-layer state for configuration.
 * \param data IDN data.
 * \param size Size of data in bytes.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_idn(
    ec_slave_config_t *sc,
    uint8_t drive_no,
    uint16_t idn,
    ec_al_state_t state,
    const uint8_t *data,
    size_t size
);

/** Sets a configuration flag.
 *
 * \param sc Slave configuration.
 * \param key Flag key.
 * \param value Flag value.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_slave_config_flag(ec_slave_config_t *sc, const char *key, int32_t value);

/** Sets the EoE MAC address.
 *
 * \param sc Slave configuration.
 * \param mac_address MAC address (6 bytes).
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_eoe_mac_address(
    ec_slave_config_t *sc,
    const unsigned char *mac_address
);

/** Sets the EoE IP address.
 *
 * \param sc Slave configuration.
 * \param ip_address IP address.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_slave_config_eoe_ip_address(ec_slave_config_t *sc, struct in_addr ip_address);

/** Sets the EoE subnet mask.
 *
 * \param sc Slave configuration.
 * \param subnet_mask Subnet mask.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_slave_config_eoe_subnet_mask(ec_slave_config_t *sc, struct in_addr subnet_mask);

/** Sets the EoE default gateway.
 *
 * \param sc Slave configuration.
 * \param gateway_address Gateway address.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_eoe_default_gateway(
    ec_slave_config_t *sc,
    struct in_addr gateway_address
);

/** Sets the EoE DNS address.
 *
 * \param sc Slave configuration.
 * \param dns_address DNS address.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_slave_config_eoe_dns_address(ec_slave_config_t *sc, struct in_addr dns_address);

/** Sets the EoE hostname.
 *
 * \param sc Slave configuration.
 * \param name Hostname.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_slave_config_eoe_hostname(ec_slave_config_t *sc, const char *name);

/** Configures state change timeout.
 *
 * \param sc Slave configuration.
 * \param from_state Source state.
 * \param to_state Target state.
 * \param timeout_ms Timeout in milliseconds.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_slave_config_state_timeout(
    ec_slave_config_t *sc,
    ec_al_state_t from_state,
    ec_al_state_t to_state,
    unsigned int timeout_ms
);

/******************************************************************************
 * Domain methods
 *****************************************************************************/

/** Registers a list of PDO entries for process data exchange.
 *
 * \param domain Domain.
 * \param pdo_entry_regs Array of PDO entry registrations (terminated by {}).
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_domain_reg_pdo_entry_list(
    ec_domain_t *domain,
    const ec_pdo_entry_reg_t *pdo_entry_regs
);

/** Returns the size of the domain's process data.
 *
 * \param domain Domain.
 * \return Size in bytes.
 */
EC_PUBLIC_API size_t ecrt_domain_size(const ec_domain_t *domain);

#ifdef __KERNEL__
/** Provides external memory for domain process data.
 *
 * \param domain Domain.
 * \param memory External memory buffer.
 */
void ecrt_domain_external_memory(ec_domain_t *domain, uint8_t *memory);
#endif

/** Returns a pointer to the domain's process data.
 *
 * \param domain Domain.
 * \return Pointer to process data.
 */
EC_PUBLIC_API uint8_t *ecrt_domain_data(const ec_domain_t *domain);

/** Processes received datagrams.
 *
 * \param domain Domain.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_domain_process(ec_domain_t *domain);

/** Queues domain datagrams for sending.
 *
 * \param domain Domain.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_domain_queue(ec_domain_t *domain);

/** Returns the current domain state.
 *
 * \param domain Domain.
 * \param state Pointer to store domain state.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_domain_state(const ec_domain_t *domain, ec_domain_state_t *state);

/******************************************************************************
 * SDO request methods
 *****************************************************************************/

/** Sets the SDO index and subindex.
 *
 * \param req SDO request.
 * \param index SDO index.
 * \param subindex SDO subindex.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_sdo_request_index(ec_sdo_request_t *req, uint16_t index, uint8_t subindex);

/** Sets the SDO request timeout.
 *
 * \param req SDO request.
 * \param timeout Timeout in milliseconds.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_sdo_request_timeout(ec_sdo_request_t *req, uint32_t timeout);

/** Returns a pointer to the SDO request data.
 *
 * \param req SDO request.
 * \return Pointer to data.
 */
EC_PUBLIC_API uint8_t *ecrt_sdo_request_data(const ec_sdo_request_t *req);

/** Returns the size of the SDO request data.
 *
 * \param req SDO request.
 * \return Data size in bytes.
 */
EC_PUBLIC_API size_t ecrt_sdo_request_data_size(const ec_sdo_request_t *req);

/** Returns the current state of the SDO request.
 *
 * \param req SDO request.
 * \return Request state.
 */
EC_PUBLIC_API ec_request_state_t ecrt_sdo_request_state(
#ifdef __KERNEL__
    const
#endif
    ec_sdo_request_t *req
);

/** Schedules an SDO write operation.
 *
 * \param req SDO request.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_sdo_request_write(ec_sdo_request_t *req);

/** Schedules an SDO read operation.
 *
 * \param req SDO request.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_sdo_request_read(ec_sdo_request_t *req);

/******************************************************************************
 * SoE request methods
 *****************************************************************************/

/** Sets the SoE request IDN.
 *
 * \param req SoE request.
 * \param drive_no Drive number.
 * \param idn IDN.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_soe_request_idn(ec_soe_request_t *req, uint8_t drive_no, uint16_t idn);

/** Sets the SoE request timeout.
 *
 * \param req SoE request.
 * \param timeout Timeout in milliseconds.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_soe_request_timeout(ec_soe_request_t *req, uint32_t timeout);

/** Returns a pointer to the SoE request data.
 *
 * \param req SoE request.
 * \return Pointer to data.
 */
EC_PUBLIC_API uint8_t *ecrt_soe_request_data(const ec_soe_request_t *req);

/** Returns the size of the SoE request data.
 *
 * \param req SoE request.
 * \return Data size in bytes.
 */
EC_PUBLIC_API size_t ecrt_soe_request_data_size(const ec_soe_request_t *req);

/** Returns the current state of the SoE request.
 *
 * \param req SoE request.
 * \return Request state.
 */
EC_PUBLIC_API ec_request_state_t ecrt_soe_request_state(
#ifdef __KERNEL__
    const
#endif
    ec_soe_request_t *req
);

/** Schedules an SoE write operation.
 *
 * \param req SoE request.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_soe_request_write(ec_soe_request_t *req);

/** Schedules an SoE read operation.
 *
 * \param req SoE request.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_soe_request_read(ec_soe_request_t *req);

/******************************************************************************
 * VoE handler methods
 *****************************************************************************/

/** Sets the VoE send header.
 *
 * \param voe VoE handler.
 * \param vendor_id Vendor ID.
 * \param vendor_type Vendor type.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_voe_handler_send_header(
    ec_voe_handler_t *voe,
    uint32_t vendor_id,
    uint16_t vendor_type
);

/** Returns the VoE received header.
 *
 * \param voe VoE handler.
 * \param vendor_id Pointer to store vendor ID.
 * \param vendor_type Pointer to store vendor type.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_voe_handler_received_header(
    const ec_voe_handler_t *voe,
    uint32_t *vendor_id,
    uint16_t *vendor_type
);

/** Returns a pointer to the VoE data.
 *
 * \param voe VoE handler.
 * \return Pointer to data.
 */
EC_PUBLIC_API uint8_t *ecrt_voe_handler_data(const ec_voe_handler_t *voe);

/** Returns the size of the VoE data.
 *
 * \param voe VoE handler.
 * \return Data size in bytes.
 */
EC_PUBLIC_API size_t ecrt_voe_handler_data_size(const ec_voe_handler_t *voe);

/** Schedules a VoE write operation.
 *
 * \param voe VoE handler.
 * \param size Size of data to send.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_voe_handler_write(ec_voe_handler_t *voe, size_t size);

/** Schedules a VoE read operation.
 *
 * \param voe VoE handler.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_voe_handler_read(ec_voe_handler_t *voe);

/** Schedules a VoE read operation without sync.
 *
 * \param voe VoE handler.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int ecrt_voe_handler_read_nosync(ec_voe_handler_t *voe);

/** Executes the VoE handler state machine.
 *
 * \param voe VoE handler.
 * \return Request state.
 */
EC_PUBLIC_API ec_request_state_t ecrt_voe_handler_execute(ec_voe_handler_t *voe);

/******************************************************************************
 * Register request methods
 *****************************************************************************/

/** Returns a pointer to the register request data.
 *
 * \param req Register request.
 * \return Pointer to data.
 */
EC_PUBLIC_API uint8_t *ecrt_reg_request_data(const ec_reg_request_t *req);

/** Returns the current state of the register request.
 *
 * \param req Register request.
 * \return Request state.
 */
EC_PUBLIC_API ec_request_state_t ecrt_reg_request_state(const ec_reg_request_t *req);

/** Schedules a register write operation.
 *
 * \param req Register request.
 * \param address Register address.
 * \param size Size of data to write.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_reg_request_write(ec_reg_request_t *req, uint16_t address, size_t size);

/** Schedules a register read operation.
 *
 * \param req Register request.
 * \param address Register address.
 * \param size Size of data to read.
 * \return 0 on success, otherwise negative error code.
 */
EC_PUBLIC_API int
ecrt_reg_request_read(ec_reg_request_t *req, uint16_t address, size_t size);

/******************************************************************************
 * Bitwise read/write macros
 *****************************************************************************/

/** Read a bit from process data.
 *
 * \param DATA Pointer to data.
 * \param POS Bit position (0-7).
 */
#define EC_READ_BIT(DATA, POS) ((*((uint8_t *) (DATA)) >> (POS)) & 0x01)

/** Write a bit to process data.
 *
 * \param DATA Pointer to data.
 * \param POS Bit position (0-7).
 * \param VAL Bit value (0 or 1).
 */
#define EC_WRITE_BIT(DATA, POS, VAL)                                                   \
    do {                                                                               \
        if (VAL)                                                                       \
            *((uint8_t *) (DATA)) |= (1 << (POS));                                     \
        else                                                                           \
            *((uint8_t *) (DATA)) &= ~(1 << (POS));                                    \
    } while (0)

/******************************************************************************
 * Byte-swapping for userspace
 *****************************************************************************/

#ifndef __KERNEL__

#if __BYTE_ORDER == __LITTLE_ENDIAN

#define le16_to_cpu(x) (x)
#define le32_to_cpu(x) (x)
#define le64_to_cpu(x) (x)

#define cpu_to_le16(x) (x)
#define cpu_to_le32(x) (x)
#define cpu_to_le64(x) (x)

#elif __BYTE_ORDER == __BIG_ENDIAN

#define swap16(x)                                                                      \
    ((uint16_t) ((((uint16_t) (x) & 0x00ffU) << 8) | (((uint16_t) (x) & 0xff00U) >> 8)))

#define swap32(x)                                                                      \
    ((uint32_t) ((((uint32_t) (x) & 0x000000ffUL) << 24) |                             \
                 (((uint32_t) (x) & 0x0000ff00UL) << 8) |                              \
                 (((uint32_t) (x) & 0x00ff0000UL) >> 8) |                              \
                 (((uint32_t) (x) & 0xff000000UL) >> 24)))

#define swap64(x)                                                                      \
    ((uint64_t) ((((uint64_t) (x) & 0x00000000000000ffULL) << 56) |                    \
                 (((uint64_t) (x) & 0x000000000000ff00ULL) << 40) |                    \
                 (((uint64_t) (x) & 0x0000000000ff0000ULL) << 24) |                    \
                 (((uint64_t) (x) & 0x00000000ff000000ULL) << 8) |                     \
                 (((uint64_t) (x) & 0x000000ff00000000ULL) >> 8) |                     \
                 (((uint64_t) (x) & 0x0000ff0000000000ULL) >> 24) |                    \
                 (((uint64_t) (x) & 0x00ff000000000000ULL) >> 40) |                    \
                 (((uint64_t) (x) & 0xff00000000000000ULL) >> 56)))

#define le16_to_cpu(x) swap16(x)
#define le32_to_cpu(x) swap32(x)
#define le64_to_cpu(x) swap64(x)

#define cpu_to_le16(x) swap16(x)
#define cpu_to_le32(x) swap32(x)
#define cpu_to_le64(x) swap64(x)

#endif /* __BYTE_ORDER */

#define le16_to_cpup(x) le16_to_cpu(*((uint16_t *) (x)))
#define le32_to_cpup(x) le32_to_cpu(*((uint32_t *) (x)))
#define le64_to_cpup(x) le64_to_cpu(*((uint64_t *) (x)))

#endif /* __KERNEL__ */

/******************************************************************************
 * Read macros
 *****************************************************************************/

/** Read an unsigned 8-bit value from process data. */
#define EC_READ_U8(DATA) ((uint8_t) *((uint8_t *) (DATA)))

/** Read a signed 8-bit value from process data. */
#define EC_READ_S8(DATA) ((int8_t) *((uint8_t *) (DATA)))

/** Read an unsigned 16-bit value from process data. */
#define EC_READ_U16(DATA) ((uint16_t) le16_to_cpup((void *) (DATA)))

/** Read a signed 16-bit value from process data. */
#define EC_READ_S16(DATA) ((int16_t) le16_to_cpup((void *) (DATA)))

/** Read an unsigned 32-bit value from process data. */
#define EC_READ_U32(DATA) ((uint32_t) le32_to_cpup((void *) (DATA)))

/** Read a signed 32-bit value from process data. */
#define EC_READ_S32(DATA) ((int32_t) le32_to_cpup((void *) (DATA)))

/** Read an unsigned 64-bit value from process data. */
#define EC_READ_U64(DATA) ((uint64_t) le64_to_cpup((void *) (DATA)))

/** Read a signed 64-bit value from process data. */
#define EC_READ_S64(DATA) ((int64_t) le64_to_cpup((void *) (DATA)))

#ifndef __KERNEL__
/** Read a 32-bit float from process data. */
EC_PUBLIC_API float ecrt_read_real(const void *data);

#define EC_READ_REAL(DATA) ecrt_read_real(DATA)

/** Read a 64-bit double from process data. */
EC_PUBLIC_API double ecrt_read_lreal(const void *data);

#define EC_READ_LREAL(DATA) ecrt_read_lreal(DATA)
#endif

/******************************************************************************
 * Write macros
 *****************************************************************************/

/** Write an unsigned 8-bit value to process data. */
#define EC_WRITE_U8(DATA, VAL)                                                         \
    do {                                                                               \
        *((uint8_t *) (DATA)) = ((uint8_t) (VAL));                                     \
    } while (0)

/** Write a signed 8-bit value to process data. */
#define EC_WRITE_S8(DATA, VAL) EC_WRITE_U8(DATA, VAL)

/** Write an unsigned 16-bit value to process data. */
#define EC_WRITE_U16(DATA, VAL)                                                        \
    do {                                                                               \
        *((uint16_t *) (DATA)) = cpu_to_le16((uint16_t) (VAL));                        \
    } while (0)

/** Write a signed 16-bit value to process data. */
#define EC_WRITE_S16(DATA, VAL) EC_WRITE_U16(DATA, VAL)

/** Write an unsigned 32-bit value to process data. */
#define EC_WRITE_U32(DATA, VAL)                                                        \
    do {                                                                               \
        *((uint32_t *) (DATA)) = cpu_to_le32((uint32_t) (VAL));                        \
    } while (0)

/** Write a signed 32-bit value to process data. */
#define EC_WRITE_S32(DATA, VAL) EC_WRITE_U32(DATA, VAL)

/** Write an unsigned 64-bit value to process data. */
#define EC_WRITE_U64(DATA, VAL)                                                        \
    do {                                                                               \
        *((uint64_t *) (DATA)) = cpu_to_le64((uint64_t) (VAL));                        \
    } while (0)

/** Write a signed 64-bit value to process data. */
#define EC_WRITE_S64(DATA, VAL) EC_WRITE_U64(DATA, VAL)

#ifndef __KERNEL__
/** Write a 32-bit float to process data. */
EC_PUBLIC_API void ecrt_write_real(void *data, float value);

#define EC_WRITE_REAL(DATA, VAL) ecrt_write_real(DATA, VAL)

/** Write a 64-bit double to process data. */
EC_PUBLIC_API void ecrt_write_lreal(void *data, double value);

#define EC_WRITE_LREAL(DATA, VAL) ecrt_write_lreal(DATA, VAL)
#endif

#ifdef __cplusplus
}
#endif

#endif /* __ECRT_H__ */
