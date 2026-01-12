// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "api.h"
#include "driver/errors/errors.h"

namespace ljm {
static const std::unordered_map<std::string, std::string> ERROR_DESCRIPTIONS = {
    {"LJ_SUCCESS", ""},
    {"LJME_FRAMES_OMITTED_DUE_TO_PACKET_SIZE",
     "Some read/write operation(s) were not sent to the device because the "
     "Feedback command being created was too large for the device, given the "
     "current connection type."},
    {"LJME_DEBUG_LOG_FAILURE",
     "The debug log file could not be written to. Check the LJM_DEBUG_LOG_FILE "
     "and related LJM configurations."},
    {"LJME_USING_DEFAULT_CALIBRATION",
     "The device's calibration could not be read, so the nominal calibration "
     "values were used. Possible Cause: Incompatible microSD card installed. "
     "See SD Card section of T7 Datasheet."},
    {"LJME_DEBUG_LOG_FILE_NOT_OPEN",
     "The message could not be logged because the debug log file has not been "
     "opened."},
    {"FEATURE_NOT_IMPLEMENTED", ""},
    {"LJME_MODBUS_ERRORS_BEGIN",
     "This indicates where the Modbus error codes start is not a real Modbus "
     "error code."},
    {"LJME_MODBUS_ERRORS_END",
     "This indicates where the Modbus error codes end is not a real Modbus "
     "error code."},
    {"LJME_MBE1_ILLEGAL_FUNCTION", "The device received an invalid function code."},
    {"LJME_MBE2_ILLEGAL_DATA_ADDRESS",
     "The device received an invalid address. Alternatively, this error could "
     "indicate illegal access to a device register such as trying to write to "
     "a read only register."},
    {"LJME_MBE3_ILLEGAL_DATA_VALUE",
     "The device received a value that could not be written to the specified "
     "address."},
    {"LJME_MBE4_SLAVE_DEVICE_FAILURE", "The device encountered an error."},
    {"LJME_MBE5_ACKNOWLEDGE",
     "The device acknowledges the request, but will take some time to process."},
    {"LJME_MBE6_SLAVE_DEVICE_BUSY", "The device is busy and cannot respond currently."},
    {"LJME_MBE8_MEMORY_PARITY_ERROR", "The device detected a memory parity error."},
    {"LJME_MBE10_GATEWAY_PATH_UNAVAILABLE", "The requested route was not available."},
    {"LJME_MBE11_GATEWAY_TARGET_NO_RESPONSE",
     "The requested route was available but the device failed to respond."},
    {"LJME_LIBRARY_ERRORS_BEGIN",
     "This indicates where the LJM error codes start is not a real LJM error "
     "code."},
    {"LJME_LIBRARY_ERRORS_END",
     "This indicates where the LJM error codes end is not a real LJM error "
     "code."},
    {"LJME_UNKNOWN_ERROR",
     "LJM reached an unexpected state. Please contact LabJack support if "
     "you've received this error."},
    {"LJME_INVALID_DEVICE_TYPE",
     "LJM received an unsupported device type during device open or discovery."},
    {"LJME_INVALID_HANDLE",
     "LJM reached an invalid state due to an issue with an invalid handle "
     "reference."},
    {"LJME_DEVICE_NOT_OPEN",
     "The requested handle did not refer to an open device. Perhaps the Open "
     "call failed but you still tried to use the returned handle. Perhaps your "
     "code called Close somewhere and then after tried to use the handle."},
    {"LJME_STREAM_NOT_INITIALIZED",
     "The requested device does not have a stream initialized in LJM."},
    {"LJME_DEVICE_DISCONNECTED",
     "The device could not be contacted and LJM is not configured to heal "
     "device connections (see LJM config LJM_HEAL_CONNECTION_MODE)."},
    {"LJME_DEVICE_NOT_FOUND",
     "A device matching the requested device parameters could not be found and "
     "therefore could not be opened. This could indicate an invalid parameter "
     "to a device open function or that the desired connection is already "
     "claimed/unavailable."},
    {"LJME_APERIODIC_STREAM_OUT_NOT_INITIALIZED",
     "There was an attempt to modify an aperiodic stream-out prior to its "
     "initialization."},
    {"LJME_DEVICE_ALREADY_OPEN",
     "The device being registered was already registered. Please contact "
     "LabJack support if you've received this error."},
    {"LJME_DEVICE_CURRENTLY_CLAIMED_BY_ANOTHER_PROCESS",
     "At least one matching device was found, but cannot be claimed by this "
     "process because a different process has already claimed it. Either close "
     "the device handle in your other process (i.e. program or application) or "
     "close the other process. Then try opening the device again."},
    {"LJME_CANNOT_CONNECT",
     "At least one device matching the requested device parameters was found, "
     "but a connection could not be established. Typically indicates a TCP "
     "socket connection issue."},
    {"LJME_STREAM_OUT_INDEX_OUT_OF_RANGE",
     "An invalid index value was passed to a stream-out function."},
    {"LJME_SOCKET_LEVEL_ERROR", "A TCP socket-level error occurred."},
    {"LJME_SCAN_RATE_INCONSISTENT",
     "The scan rate passed to a stream out function did not match the stream "
     "scan rate"},
    {"LJME_CANNOT_OPEN_DEVICE", "The device could not be opened."},
    {"LJME_CANNOT_DISCONNECT", "An error occurred while attempting to disconnect."},
    {"LJME_WINSOCK_FAILURE", "A Windows WINSOCK error occurred."},
    {"LJME_RECONNECT_FAILED",
     "The device could not be contacted. LJM is configured to heal device "
     "connections and the reconnection attempt failed."},
    {"LJME_CONNECTION_HAS_YIELDED_RECONNECT_FAILED",
     "The device connection was yielded to another process and now cannot be "
     "contacted."},
    {"LJME_USB_FAILURE", "System USB communication failed fatally."},
    {"LJME_STREAM_FLUSH_TIMEOUT",
     "Stream flushing did not resolve before the timeout."},
    {"LJME_U3_NOT_SUPPORTED_BY_LJM",
     "LJM does not support UD-series LabJack devices; use the UD driver"},
    {"LJME_U6_NOT_SUPPORTED_BY_LJM",
     "LJM does not support UD-series LabJack devices; use the UD driver"},
    {"LJME_UE9_NOT_SUPPORTED_BY_LJM",
     "LJM does not support UD-series LabJack devices; use the UD driver"},
    {"LJME_INVALID_ADDRESS",
     "The requested address did not match any address known to LJM (included "
     "in ljm_constants.json)"},
    {"LJME_INVALID_CONNECTION_TYPE",
     "LJM received an unexpected connection type during device open or "
     "discovery."},
    {"LJME_INVALID_DIRECTION", "LJM received an unexpected read/write direction."},
    {"LJME_INVALID_FUNCTION", "LJM received an unexpected Modbus function."},
    {"LJME_INVALID_NUM_REGISTERS", "LJM received an unexpected number of registers."},
    {"LJME_INVALID_PARAMETER",
     "LJM received an unexpected parameter. Please check the LabJackM.h "
     "description of the function that returned this error for an explanation."},
    {"LJME_INVALID_PROTOCOL_ID", "LJM received an unexpected Modbus protocol ID."},
    {"LJME_INVALID_TRANSACTION_ID",
     "LJM received an unexpected Modbus transaction ID."},
    {"LJME_NUM_WRITES_LARGER_THAN_AVAILABLE_SPACE",
     "LJM received more values to stream out than are available in the LJM "
     "write queue."},
    {"LJME_UNKNOWN_VALUE_TYPE", "LJM received an unexpected data type."},
    {"LJME_MEMORY_ALLOCATION_FAILURE", "LJM was unable to allocate needed memory."},
    {"LJME_NO_COMMAND_BYTES_SENT",
     "No Modbus command bytes could be sent to the device."},
    {"LJME_INCORRECT_NUM_COMMAND_BYTES_SENT",
     "An incorrect number of Modbus command bytes were sent to the device, as "
     "reported by the underlying driver."},
    {"LJME_NO_RESPONSE_BYTES_RECEIVED",
     "No Modbus response bytes could be received from the device during the "
     "timeout period. Likely indicates some failure/unreliability with the "
     "communication method."},
    {"LJME_INCORRECT_NUM_RESPONSE_BYTES_RECEIVED",
     "An incorrect/unexpected number of Modbus response bytes were received "
     "from the device during the timeout period. Likely indicates some "
     "failure/unreliability with the communication method."},
    {"LJME_MIXED_FORMAT_IP_ADDRESS",
     "LJM received an IP identifier that seemed to be formatted in both "
     "hexadecimal and decimal."},
    {"LJME_UNKNOWN_IDENTIFIER",
     "LJM received an unknown identifier type during device open or discovery."},
    {"LJME_NOT_IMPLEMENTED",
     "An unimplemented functionality was attempted to be used. Please contact "
     "LabJack support if you've received this error so we can implement it for "
     "you!"},
    {"LJME_INVALID_INDEX",
     "LJM received an index/offset that was out-of-bounds. Please contact "
     "LabJack support if you've received this error."},
    {"LJME_INVALID_LENGTH",
     "LJM received a length that did not make sense. Please contact LabJack "
     "support if you've received this error."},
    {"LJME_ERROR_BIT_SET",
     "A Modbus response had the error bit set. Please contact LabJack support "
     "if you've received this error."},
    {"LJME_INVALID_MAXBYTESPERMBFB",
     "LJM_AddressesToMBFB received a value of MaxBytesPerMBFB that was too "
     "small."},
    {"LJME_NULL_POINTER",
     "LJM received a null pointer for a pointer parameter that is required to "
     "not be null."},
    {"LJME_NULL_OBJ",
     "LJM found a null object. Please contact LabJack support if you've "
     "received this error."},
    {"LJME_RESERVED_NAME", "The requested name was invalid/reserved."},
    {"LJME_UNPARSABLE_DEVICE_TYPE",
     "LJM received a device type string that was unparsable or unexpected."},
    {"LJME_UNPARSABLE_CONNECTION_TYPE",
     "LJM received a connection type string that was unparsable or unexpected."},
    {"LJME_UNPARSABLE_IDENTIFIER",
     "LJM received a identifier string that was unparsable or unexpected."},
    {"LJME_PACKET_SIZE_TOO_LARGE",
     "The Modbus packet to be sent or the expected Modbus packet to be "
     "received exceeded the maximum packet size for the device connection."},
    {"LJME_TRANSACTION_ID_ERR",
     "LJM received a mismatched Modbus transaction ID from the device."},
    {"LJME_PROTOCOL_ID_ERR",
     "LJM received a mismatched Modbus protocol ID from the device."},
    {"LJME_LENGTH_ERR",
     "LJM received a mismatched Modbus packet length from the device."},
    {"LJME_UNIT_ID_ERR", "LJM received a mismatched Modbus unit ID from the device."},
    {"LJME_FUNCTION_ERR",
     "LJM received a mismatched Modbus function ID from the device."},
    {"LJME_STARTING_REG_ERR",
     "LJM received a mismatched Modbus register address from the device."},
    {"LJME_NUM_REGS_ERR",
     "LJM received a mismatched Modbus number of registers from the device."},
    {"LJME_NUM_BYTES_ERR",
     "LJM received a mismatched Modbus number of bytes from the device."},
    {"LJME_CONFIG_FILE_NOT_FOUND",
     "The LJM configuration file could not be opened for reading."},
    {"LJME_CONFIG_PARSING_ERROR",
     "An error occurred while parsing the LJM configuration file."},
    {"LJME_INVALID_NUM_VALUES", "LJM received an invalid number of values."},
    {"LJME_CONSTANTS_FILE_NOT_FOUND",
     "The LJM error constants file and/or Modbus constants file could not be "
     "opened for reading."},
    {"LJME_INVALID_CONSTANTS_FILE",
     "The LJM error constants file and/or Modbus constants file could not be "
     "parsed."},
    {"LJME_INVALID_NAME",
     "The requested name was not found in the register loaded from the "
     "constants file (if the constants file is loaded)."},
    {"LJME_OVERSPECIFIED_PORT",
     "The requested connection type is ANY and port is specified."},
    {"LJME_INTENT_NOT_READY",
     "The internal operation object was not prepared correctly. Please contact "
     "LabJack support if you've received this error."},
    {"LJME_ATTR_LOAD_COMM_FAILURE",
     "A device matching the requested device parameters was found, claimed, "
     "and connected, but a communication error prevented device attributes "
     "from being loaded to LJM. The device was not opened. Likely indicates "
     "some failure/unreliability with the communication method."},
    {"LJME_INVALID_CONFIG_NAME", "LJM received an unknown configuration name."},
    {"LJME_ERROR_RETRIEVAL_FAILURE",
     "An error occurred on the device and LJM was unable to retrieve more "
     "information about that error. Please contact LabJack support if you've "
     "received this error."},
    {"LJME_LJM_BUFFER_FULL",
     "The LJM stream buffer was filled with stream data and stream was "
     "stopped. See "
     "https://labjack.com/support/software/api/ljm/"
     "streaming-lots-of-9999-values"},
    {"LJME_COULD_NOT_START_STREAM",
     "LJM could not start stream. Input may have incorrect or a different "
     "error may have occurred."},
    {"LJME_STREAM_NOT_RUNNING",
     "Stream data was attempted to be accessed when LJM was not streaming data "
     "to/from the device."},
    {"LJME_UNABLE_TO_STOP_STREAM", "The stream could not be stopped."},
    {"LJME_INVALID_VALUE", "LJM received a value that was not expected."},
    {"LJME_SYNCHRONIZATION_TIMEOUT",
     "LJM did not receive data from the device for longer than the timeout "
     "specified by LJM_STREAM_RECEIVE_TIMEOUT_MS. Often returned in relation "
     "to externally clocked stream modes. See the eStreamStart documentation "
     "in the LJM User's Guide."},
    {"LJME_OLD_FIRMWARE",
     "The current firmware version of the device was not sufficient to "
     "read/write a requested register, as according to the loaded LJM "
     "constants file."},
    {"LJME_CANNOT_READ_OUT_ONLY_STREAM",
     "The stream running is out-only and does not produce data values."},
    {"LJME_NO_SCANS_RETURNED",
     "The LJM configuration LJM_STREAM_SCANS_RETURN is set to "
     "LJM_STREAM_SCANS_RETURN_ALL_OR_NONE and the full ScansPerRead scans have "
     "not be received."},
    {"LJME_TEMPERATURE_OUT_OF_RANGE",
     "LJM_TCVoltsToTemp received a temperature that was out of range for the "
     "given thermocouple type."},
    {"LJME_VOLTAGE_OUT_OF_RANGE",
     "LJM_TCVoltsToTemp received a voltage that was out of range for the given "
     "thermocouple type."},
    {"LJME_FUNCTION_DOES_NOT_SUPPORT_THIS_TYPE",
     "The function does not support the given data type. For example, "
     "LJM_eReadName and LJM_eReadAddress do not support reading LJM_STRING "
     "values, which are too large."},
    {"LJME_INVALID_INFO_HANDLE",
     "The given info handle did not exist. (It may have been passed to "
     "LJM_CleanInfo already.)"},
    {"LJME_NO_DEVICES_FOUND",
     "An Open/OpenS call was called - with any device type, any connection "
     "type, and any identifier - but no devices were found."},
    {"LJME_AUTO_IPS_FILE_NOT_FOUND",
     "The LJM auto IPs file could not be found or read. LJM will attempt to "
     "create the auto IPs file, as needed."},
    {"LJME_AUTO_IPS_FILE_INVALID", "The LJM auto IPs file contents were not valid."},
    {"LJME_INVALID_INTERVAL_HANDLE", "The given interval handle did not exist."},
    {"LJME_NAMED_MUTEX_PERMISSION_DENIED",
     "There was a permission error while accessing a system-level named mutex. "
     "Try restarting your computer."},
    {"LJME_DIGITAL_AUTO_RECOVERY_ERROR_DETECTED",
     "During stream, the device buffer overflowed, causing auto-recovery to "
     "occur. See the digital stream information in the Stream Configs section "
     "of the LJM User's Guide."},
    {"LJME_NEGATIVE_RECEIVE_BUFFER_SIZE",
     "During stream, the receive buffer size was negative. This is probably "
     "because LJM_STREAM_TCP_RECEIVE_BUFFER_SIZE was set to too large a number "
     "to be represented by the signed data type being used, int. Solution: Use "
     "a smaller LJM_STREAM_TCP_RECEIVE_BUFFER_SIZE."},
    {"FEATURE_NOT_SUPPORTED", "Feature is not supported on this device."},
    {"MODBUS_RSP_OVERFLOW", "Response packet greater than max packet size."},
    {"MODBUS_CMD_OVERFLOW", "Command packet greater than max packet size"},
    {"MODBUS_STRING_CMD_TOO_BIG", ""},
    {"MODBUS_STRING_PARAM_TOO_BIG", ""},
    {"MODBUS_STRING_BAD_NUM_PARAMS", ""},
    {"MODBUS_INVALID_NUM_REGISTERS",
     "Register data types does not match the number of registers in the "
     "request."},
    {"MODBUS_READ_TOO_LARGE", ""},
    {"MODBUS_NUM_REGS_MUST_BE_EVEN",
     "Register or group of registers requires the access be in an even number "
     "of registers."},
    {"MODBUS_STRING_MISSING_NULL",
     "Strings must be terminated with a null (\\0, 0x00)."},
    {"UDP_DISCOVERY_ONLY_MODE_IS_ENABLED",
     "The UDP discovery socket has been set to discovery only. Only a few "
     "registers may be read while in this mode."},
    {"STARTUP_CONFIG_INVALID_CODE", ""},
    {"STARTUP_CONFIG_INVALID_READ",
     "An attempt was made to read beyond the configuration structure."},
    {"USER_RAM_FIFO_MUST_BE_EMPTY",
     "The FIFO can not contain any data when data size is being changed."},
    {"USER_RAM_FIFO_INSUFFICIENT_VALUES",
     "The FIFO contains fewer values than requested."},
    {"USER_RAM_FIFO_INSUFFICIENT_SPACE",
     "FIFO does not have enough free space to hold the requested write. No "
     "data was added to the FIFO."},
    {"USER_RAM_FIFO_SIZE_MUST_BE_EVEN",
     "The number of bytes allocated to the FIFO must be even."},
    {"INTFLASH_ADD_INVALID", ""},
    {"INTFLASH_CODE_INVALID", ""},
    {"INTFLASH_OP_PROHIBITED",
     "Attempted to read or write a section that is not allowed."},
    {"INTFLASH_SECTION_OVERWRITE",
     "Attempted to write or read beyond the currently selected section."},
    {"INTFLASH_KEY_INVALID", "Specified Key and Address mismatch."},
    {"FLASH_VERIFICATION_FAILED",
     "A write to flash failed to set one or more bits to the desired values."},
    {"FLASH_ERASE_FAILED",
     "One or more bits failed to set during a flash erase operation."},
    {"INTFLASH_UNAVAILABLE",
     "Flash can not be accessed due to the WiFi module booting up."},
    {"FILEIO_UNAVAILABLE",
     "The file system can not be accessed due to the WiFi module booting up."},
    {"AIN_RANGE_INVALID", "Specified range not available on this device."},
    {"AIN_SETTLING_INVALID", "Specified settling is greater than device max."},
    {"AIN_SET_TO_BINARY",
     "Analog input system currently set to binary. Some operations, such as "
     "AIN_EF, will fail."},
    {"AIN_NEGATIVE_CHANNEL_INVALID",
     "For channel range 0-13: Negative channel must be even channel number + "
     "1. For extended channel range 48-127: Negative channel must be channel "
     "number + 8."},
    {"AIN_ALL_ZERO_ONLY", "Only a value of zero may be written to this address."},
    {"AIN_RESOLUTION_INVALID",
     "Selected resolution is invalid. Valid range is 0-5 for T4 and 0-12 for "
     "T7."},
    {"AIN_RATE_INVALID",
     "The selected AIN rate configuration is invalid for the desired "
     "resolution index or vise-versa."},
    {"AIN_CHN_DISABLED", "The AIN channel has been disabled"},
    {"LUA_VM_STATE_NO_CHANGE", ""},
    {"LUA_INITIALIZATION_ERROR", ""},
    {"LUA_INVALID_NUM_IO_FLOATS",
     "Requested more than the max possible number of IO floats."},
    {"LUA_IO_FLOATMEM_OVERFLOW",
     "Attempt to read/write beyond the currently allocated IO space."},
    {"LUA_INVALID_MODE", ""},
    {"LUA_IS_RUNNING", "A running script is preventing the requested operation."},
    {"LUA_CODE_BUFFER_EMPTY", "Attempted to run a program that is not present."},
    {"LUA_DEBUG_IS_DISABLED",
     "Attempted to read from debug buffer while debug is disabled."},
    {"LUA_TABLE_SMALLER_THAN_SPECIFIED_SIZE",
     "The Lua table provided is too small to process the request."},
    {"LUA_IS_CLOSING", "The Lua VM is being closed, usually takes less than 100 ms."},
    {"SYSTEM_MEMORY_BEREFT",
     "Insufficient RAM to perform the requested action. Often occurs when a "
     "loaded Lua script is too large."},
    {"SYSTEM_MEMORY_OVERWRITE", "Attempted to overwrite a buffer."},
    {"SYSTEM_REBOOT_CODE_INVALID", "Invalid code supplied when issuing a reboot."},
    {"SYSTEM_READ_OVERRUN", ""},
    {"SYSTEM_INVALID_PIN", ""},
    {"SYSTEM_NVRAM_UNAVAILABLE", "NVRAM is not available on this device."},
    {"SYSTEM_NVRAM_INVALID_ADDRESS",
     "The specified NVRAM location is not available on this device."},
    {"SYSTEM_WAIT_TOO_LONG", "The requested wait time is beyond the max allowed."},
    {"SYSTEM_INCOMPATIBLE_FIRMWARE_VERSION",
     "The firmware image is not compatible with this device. Typically due to "
     "flash chip incompatibility."},
    {"DEVICE_NAME_MUST_BE_ALPHANUM",
     "Attempted to write a device name with invalid characters."},
    {"POWER_INVALID_SETTING", "Unknown value specified."},
    {"POWER_USB_NEEDS_20MHZ_OR_MORE",
     "Core must be running at 20MHz minimum for USB to operate."},
    {"POWER_ETH_NEEDS_20MHZ_OR_MORE",
     "Core must be running at 20MHz minimum for Ethernet to operate."},
    {"POWER_AIN_NEEDS_20MHZ_OR_MORE",
     "Core must be running at 20MHz minimum for AIN to operate."},
    {"POWER_STREAM_NEEDS_20MHZ_OR_MORE",
     "Core must be running at 20MHz minimum to use stream mode."},
    {"POWER_SD_NEEDS_20MHZ_OR_MORE",
     "Core must be running at 20MHz minimum to use the SD card."},
    {"POWER_CAN_NOT_CHANGE_USED_CONNECTION",
     "Can not change the power level of the connected medium."},
    {"POWER_NO_CHANGE", "The written power mode is the same as the current setting."},
    {"POWER_ANALOG_OFF", "Analog input system is powered down."},
    {"WIFI_NOT_ASSOCIATED",
     "WiFi needs to be connected to a network before the requested action can "
     "be performed."},
    {"HW_DIO_NOT_AVAILABLE", ""},
    {"DIO_SET_TO_ANALOG",
     "The digital line addressed is set to analog. Digital operations cannot "
     "be performed."},
    {"HW_CNTRA_NOT_AVAILABLE",
     "Counter A is being used by another system. Typically this is due to a "
     "high speed counter being enabled while trying to enable a DIO_EF clock "
     "that requires the same resource or vise-versa"},
    {"HW_CNTRB_NOT_AVAILABLE",
     "Counter B is being used by another system. Typically this is due to a "
     "high speed counter being enabled while trying to enable a DIO_EF clock "
     "that requires the same resource or vise-versa"},
    {"HW_CNTRC_NOT_AVAILABLE", "Counter C is being used by another system."},
    {"HW_CNTRD_NOT_AVAILABLE",
     "Counter D is being used by another system. Typically this is due to a "
     "high speed counter that shares a resource with the stream clock being "
     "enabled while trying to enable stream mode or vise-versa"},
    {"HW_CIO0_NOT_AVAILABLE", ""},
    {"HW_CIO1_NOT_AVAILABLE", ""},
    {"HW_DAC1_NOT_AVAILABLE",
     "DAC1 is an active stream out target and therefore unavailable."},
    {"HW_LEDS_NOT_AVAILABLE",
     "The LEDs cannot be controlled when the LED power mode is not set to "
     "manual."},
    {"EF_DIO_HAS_NO_TNC_FEATURES",
     "The DIO accessed does not support any extended features."},
    {"EF_INVALID_TYPE", "The selected type is not recognized."},
    {"EF_TYPE_NOT_SUPPORTED", "The selected type is not recognized."},
    {"EF_PIN_TYPE_MISMATCH", "The requested type is not supported on this DIO pin."},
    {"EF_CLOCK_SOURCE_NOT_ENABLED",
     "Attempted to disable a clock source that is not running."},
    {"EF_32BIT_DATA_INTO_16BIT_REG",
     "A number greater than 16-bits was written to a clock source configured "
     "for 16-bits. Commonly occurs when an incorrect value is applied to "
     "CONFIG_A or CONFIG_B for an output feature such as PWM or pulse output."},
    {"EF_SET_TO_32BIT", ""},
    {"EF_SMOOTH_VALUE_OUT_OF_RANGE", ""},
    {"EF_32BIT_REQUIRES_BOTH_CLOCK0and1",
     "Both Clock_Source 1 and Clock_Source 2 must be disabled before enabling "
     "Clock_Source 0."},
    {"EF_DIVISOR_VALUE_INVALID",
     "The specified divisor value is not supported. Supported values are: "
     "1,2,4,8,16,32,64,256. This code was formerly named "
     "EF_PRESCALE_VALUE_INVALID"},
    {"EF_PIN_RESERVED", "The pin is already in use by another system."},
    {"EF_INVALID_DIO_NUMBER",
     "The specified DIO_EF address is not supported on this device."},
    {"EF_LINE_MUST_BE_LOW_BEFORE_STARTING",
     "The DIO line must be set to output low to ensure proper signal "
     "generation."},
    {"EF_INVALID_DIVISOR", ""},
    {"EF_VALUE_GREATER_THAN_PERIOD",
     "The DIO_EF_CONFIG value written determines a DIO line transition point "
     "relative to the count of a ClockSource. To prevent signal glitches, the "
     "value must be less than the selected ClockSource's roll value."},
    {"EF_CAN_NOT_CHANGE_INDEX_WHILE_ENABLED",
     "The index of a DIO_EF can not be changed while that DIO_EF is enabled."},
    {"AIN_EF_INVALID_TYPE", "The specified type index is not supported."},
    {"AIN_EF_INVALID_NUM_SAMPLES", "Too many samples specified."},
    {"AIN_EF_CALCULATION_ERROR", ""},
    {"AIN_EF_CHANNEL_INACTIVE",
     "The AIN_EF channel has not been initialized. To initialize, set the "
     "index configuration to a non-zero value."},
    {"AIN_EF_CALCULATION_OUT_OF_RANGE",
     "The final value calculated for the AIN_EF is outside of the range the "
     "feature supports. This often indicates that there is some configuration "
     "issue."},
    {"AIN_EF_INVALID_CHANNEL",
     "AIN_EF is not supported on the specified channel, or the specified "
     "channel does not exist."},
    {"AIN_EF_INVALID_CJC_REGISTER",
     "The register address specified for CJC measurement is not supported."},
    {"AIN_EF_STREAM_START_FAILURE", "Could not start the data collection stream."},
    {"AIN_EF_COULD_NOT_FIND_PERIOD",
     "Failed to detect a period to perform calculations over."},
    {"AIN_EF_MUST_BE_DIFFERENTIAL", "The selected AIN must be set to differential."},
    {"AIN_EF_SCAN_TIME_TOO_LONG",
     "The data collection time (number of samples / scan_rate) is too big. "
     "Limit is set to 180 ms."},
    {"AIN_EF_INVALID_EXCITATION_INDEX",
     "The excitation circuit index specified is not valid, use a different "
     "excitation circuit."},
    {"STREAM_NEED_AT_LEAST_ONE_CHN", "The list of channels to stream is empty."},
    {"STREAM_CLOCK_BASE_NOT_WRITABLE", "The stream clock base is read only."},
    {"STREAM_EXTCLK_AND_GATE_MX", ""},
    {"STREAM_IN_SPONTANEOUS_MODE",
     "Stream data can not be read with commands while in spontaneous mode."},
    {"STREAM_USB_PKT_OVERFLOW",
     "STREAM_SAMPLES_PER_PACKET is set to a value greater than the USB "
     "interface can support."},
    {"STREAM_IS_ACTIVE",
     "The requested operation can not be performed while stream is running."},
    {"STREAM_CONFIG_INVALID", "Stream resolution can not be greater than 8."},
    {"STREAM_CHN_LIST_INVALID", "The channel list contains an unstreamable address."},
    {"STREAM_SCAN_RATE_INVALID",
     "The scan rate times the number of channels per scan is too great for "
     "this device."},
    {"STREAM_OUT_BUFF_TOO_BIG", ""},
    {"STREAM_OUT_NUM_INVALID", "An invalid stream out number was specified."},
    {"STREAM_DATA_TYPE_INVALID", "An unsupported data types was specified."},
    {"STREAM_TARGET_CONFIG_INVALID",
     "Stream must be set to either spontaneous or command-response."},
    {"STREAM_OUT_BUFF_FULL",
     "Attempted to write more data than the buffer can hold. Extra data was "
     "discarded."},
    {"STREAM_OUT_TARGET_INVALID",
     "The specified address cannot be a stream out target."},
    {"STREAM_BUFF_SIZE_INVALID",
     "The specified buffer was either too large or was not a power of 2."},
    {"STREAM_OUT_BUFF_LOOP_OVERWRITE", ""},
    {"STREAM_OUT_BUFF_DNE",
     "The buffer size must be set before data can be written to it."},
    {"STREAM_SAMPLES_PER_PKT_INVALID",
     "The specified number of samples per packet is too large."},
    {"STREAM_BUFFER_DNE", ""},
    {"STREAM_NOT_RUNNING", "Stream was already disabled."},
    {"STREAM_SETTLING_INVALID",
     "Specified settling time is greater than the max possible."},
    {"STREAM_OUT_LOOP_TOO_BIG",
     "The loop size is too big for the current buffer size."},
    {"STREAM_OUT_DATA_TRGT_MISSMATCH",
     "There is a mismatch between the stream out buffer type and target "
     "register."},
    {"STREAM_INVALID_DIVISOR", "Selected divisor can not be used."},
    {"STREAM_CHN_CAN_NOT_BE_STREAMED", "The requested channel can not be streamed."},
    {"STREAM_OUT_DAC_IN_USE",
     "The high resolution converter cannot be used while stream out is used to "
     "update a DAC."},
    {"STREAM_OUT_NEEDS_TO_BE_ENABLED",
     "The STREAM_OUT#_ENABLE register must be set to 1 before writing "
     "STREAM_OUT#_LOOP_NUM_VALUES, STREAM_OUT#_SET_LOOP, or any of the "
     "STREAM_OUT#_BUFFER_ registers."},
    {"CANNOT_STREAM_FAST_WITH_DAC1_FREQUENCY_OUT_ENABLED",
     "DAC1_FREQUENCY_OUT_ENABLE should be disabled before streaming at rates "
     "above 10kHz due to poor frequency output performance at higher stream "
     "rates."},
    {"CANNOT_ENABLE_DAC1_FREQUENCY_OUT_WITH_FAST_STREAM",
     "DAC1_FREQUENCY_OUT_ENABLE should not be enabled while streaming at rates "
     "above 10kHz due to poor frequency output performance at higher stream "
     "rates."},
    {"STREAM_TRIGGER_INDEX_INVALID",
     "The stream trigger index is not valid for this device"},
    {"STREAM_RATE_INVALID_FOR_CJC",
     "Stream can only be run at a max of 250Hz when streaming TEMPERATURE(0:7) "
     "registers"},
    {"SWDT_ROLLT_INVALID", "The specified Software Watchdog timeout is too short."},
    {"SWDT_ENABLED",
     "The watchdog must be disabled before the requested operation can be "
     "performed."},
    {"SWDT_DIO_SETTINGS_INVALID",
     "The specified Software Watchdog DIO configurations are not valid."},
    {"SWDT_DAC0_SETTINGS_INVALID",
     "The specified Software Watchdog DAC0 configurations are not valid."},
    {"SWDT_DAC1_SETTINGS_INVALID",
     "The specified Software Watchdog DAC1 configurations are not valid."},
    {"RTC_TIME_INVALID", ""},
    {"RTC_SNTP_TIME_INVALID", "The specified SNTP update interval is too short."},
    {"RTC_NOT_PRESENT",
     "The requested operation can not be performed on units without a "
     "real-time-clock."},
    {"SPI_MODE_INVALID", "Valid modes are 0-3."},
    {"SPI_NO_DATA_AVAILABLE", "SPI RX data is not available in the RX buffer."},
    {"SPI_CS_PIN_INVALID", "Attempted to set an invalid pin."},
    {"SPI_CLK_PIN_INVALID", "Attempted to set an invalid pin."},
    {"SPI_MISO_PIN_INVALID", "Attempted to set an invalid pin."},
    {"SPI_MOSI_PIN_INVALID", "Attempted to set an invalid pin."},
    {"SPI_CS_PIN_RESERVED", "Selected pin is not available."},
    {"SPI_CLK_PIN_RESERVED", "Selected pin is not available."},
    {"SPI_MISO_PIN_RESERVED", "Selected pin is not available."},
    {"SPI_MOSI_PIN_RESERVED", "Selected pin is not available."},
    {"SPI_TRANSFER_SIZE_TOO_LARGE", "The specified TX buffer size is too large."},
    {"I2C_BUS_BUSY",
     "One or both of the I2C lines are held low. Check hardware and reset the "
     "bus."},
    {"I2C_NO_DATA_AVAILABLE", "Attempted to read from an empty buffer."},
    {"I2C_SDA_PIN_INVALID", "Attempted to set an invalid pin."},
    {"I2C_SCL_PIN_INVALID", "Attempted to set an invalid pin."},
    {"I2C_SDA_PIN_RESERVED", "The selected pin is not available."},
    {"I2C_SCL_PIN_RESERVED", "The selected pin is not available."},
    {"I2C_TX_SIZE_TOO_LARGE", "The specified TX buffer size is too large."},
    {"I2C_RX_SIZE_TOO_LARGE", "The specified RX buffer size is too large."},
    {"I2C_BUFFER_OVERRUN",
     "The data that was attempted to be written to the I2C TX buffer exceeded "
     "the maximum TX buffer size. Only data up to the maximum buffer size was "
     "placed in the TX buffer."},
    {"I2C_SPEED_TOO_LOW",
     "The throttle setting is too low, watchdog may fire. Minimum value = 46000"},
    {"SBUS_COMM_TIME_OUT", "Slave device did not respond."},
    {"SBUS_NO_ACK", "Slave device did not acknowledge the data transfer."},
    {"SBUS_CUSTOM_MODE_INVALID", ""},
    {"SBUS_INVALID_DIO_NUM", "Attempted to set an invalid pin."},
    {"SBUS_BACKGROUND_SERVICE_ON",
     "Command-response reads can not be used while the background service is "
     "running."},
    {"SBUS_CHECKSUM_ERROR", "SHT communication checksum failed."},
    {"TDAC_SDA_SCL_INVALID", "SCL must be even and SDA must be SCL+1."},
    {"TDAC_SCL_INVALID", "Attempted to set an invalid pin."},
    {"TDAC_INVALID_CHANNEL", "The specified channel not supported on this device."},
    {"TDAC_CAL_READ_FAILURE", "Failed to read the TDAC calibration."},
    {"TDAC_NOT_FOUND", "The TDAC did not respond to communication attempts."},
    {"TDAC_NOT_INITIALIZED",
     "A TDAC has not been initialized, try writing to the the TDAC# register."},
    {"ONEWIRE_UNSUPPORTED_FUNCTION", "Unknown function specified."},
    {"ONEWIRE_NO_PRESENCE_PULSE", "Unable to detect any devices on the bus."},
    {"ASYNCH_NUM_DATA_BITS_INVALID",
     "The specified number of data bits is not supported."},
    {"ASYNCH_NUM_TO_WRITE_INVALID", "The number of bytes to send is invalid."},
    {"ASYNCH_READ_BUFF_SIZE_INVALID",
     "The specified buffer size is invalid. Max is 2048."},
    {"ASYNCH_BAUD_TOO_HIGH", "The baud rate is too high for this device."},
    {"ASYNCH_IS_ENABLED",
     "The specified operation can not be performed while enabled."},
    {"ASYNCH_IS_NOT_ENABLED",
     "The specified operation can not be performed while disabled."},
    {"ASYNCH_TX_BUFFER_FULL", "The transmit buffer is full."},
    {"ASYNCH_TX_TIMEOUT",
     "Transmission timed out. Do not write more than 100 ms at a time."},
    {"ASYNCH_BAUD_ZERO", "The baud rate is zero. Please specify a baud rate."},
    {"FILE_IO_DISK_ERROR", "A hard error occurred in the low level disk I/O layer."},
    {"FILE_IO_INTERNAL_ERROR", "Assertion failed."},
    {"FILE_IO_NOT_READY", "The physical drive cannot work."},
    {"FILE_IO_NO_FILE", "Could not find the file."},
    {"FILE_IO_NO_PATH", "Could not find the path."},
    {"FILE_IO_PATH_NAME_INVALID", "The path name format is invalid."},
    {"FILE_IO_DENIED",
     "Access denied due to prohibited access or the directory is full."},
    {"FILE_IO_EXIST", "Access denied due to prohibited access."},
    {"FILE_IO_INVALID_OBJECT",
     "The file/directory object is invalid. In the context of performing an "
     "\"ls\" command, this indicates that there are no more files."},
    {"FILE_IO_WRITE_PROTECTED", "The physical drive is write protected."},
    {"FILE_IO_INVALID_DRIVE", "The logical drive number is invalid."},
    {"FILE_IO_NOT_ENABLED", "The volume has no work area."},
    {"FILE_IO_NO_FILESYSTEM", "There is no valid FAT12, FAT16, or FAT32 volume."},
    {"FILE_IO_MKFS_ABORTED",
     "The f_mkfs() function aborted due to any parameter error."},
    {"FILE_IO_TIMEOUT",
     "Could not get granted access to the volume within the defined timeout "
     "period."},
    {"FILE_IO_LOCKED",
     "The operation is rejected according to the file sharing policy."},
    {"FILE_IO_NOT_ENOUGH_CORE", "LFN working buffer could not be allocated."},
    {"FILE_IO_TOO_MANY_OPEN_FILES",
     "The number of open files is greater than the allowable limit (files > "
     "_FS_SHARE)."},
    {"FILE_IO_INVALID_PARAMETER", "The given parameter is invalid."},
    {"WIFI_ASSOCIATED", "The WiFi module associated to the network."},
    {"WIFI_ASSOCIATING", "The WiFi module is attempting to associate to the network."},
    {"WIFI_ASSOCIATION_FAILED", "The WiFi module failed to associate to the network."},
    {"WIFI_UNPOWERED", "The WiFi module is not currently powered."},
    {"WIFI_BOOTING_UP", "The WiFi module is initializing."},
    {"WIFI_COULD_NOT_START", "The WiFi module was unable to properly initialize."},
    {"WIFI_APPLYING_SETTINGS",
     "The WiFi module is attempting to apply the desired network settings."},
    {"WIFI_DHCP_STARTED",
     "The WiFi module has begun the process of claiming a DHCP lease."},
    {"WIFI_OTHER", "The WiFi module is in an unspecified state."},
    {"WIFI_UPDATE_CONFIG", "The WiFi module is getting ready to start an update."},
    {"WIFI_UPDATE_IN_PROG", "The WiFi module is in the process of updating."},
    {"WIFI_UPDATE_REBOOT",
     "The WiFi module has updated and will restart to apply the new changes."},
    {"WIFI_UPDATE_SUCCESS", "The WiFi module was successfully updated."},
    {"WIFI_UPDATE_FAILED", "The WiFi module was not successfully updated."},
    {"STREAM_AUTO_RECOVER_ACTIVE",
     "The stream buffer reached its capacity and auto-recovery has begun to "
     "avoid an overflow. No new samples will be saved until there is free "
     "space in the buffer."},
    {"STREAM_AUTO_RECOVER_END",
     "There is space available in the stream buffer for new samples again and "
     "auto-recovery has ended."},
    {"STREAM_SCAN_OVERLAP",
     "A new scan started before the previous scan finished. Generally occurs "
     "because ScanRate > MaxSampleRate/NumChannels. Note that MaxSampleRate is "
     "impacted by Range, ResolutionIndex, and Settling. Try adding commands "
     "right before StreamStart to set AIN_ALL_RANGE=10, "
     "STREAM_RESOLUTION_INDEX=0, and STREAM_SETTLING_US=0."},
    {"STREAM_AUTO_RECOVER_END_OVERFLOW",
     "During stream auto-recovery, the variable tracking the number of skipped "
     "scans has reached an overflow condition."},
    {"STREAM_BURST_COMPLETE",
     "The specified number of stream scans have been acquired, stream will be "
     "stopped automatically."},
    {"STREAM_BUFFER_FULL",
     "The stream buffer reached capacity while auto-recovery was disabled. "
     "Stream has been stopped."},
    {"SELFDIAG_MAIN_OSC_FAIL", ""},
    {"FILE_IO_NOT_FOUND", "The requested file was not found."},
    {"FILE_IO_NO_DISK", "No SD card present or SC card could not be initialized."},
    {"FILE_IO_INVALID_NAME", "The file name is invalid."},
    {"FILE_IO_FILE_NOT_OPEN",
     "An open file is required to perform the requested operation."},
    {"FILE_IO_TOO_MANY_OPEN", "There are too many files open."},
    {"FILE_IO_SD_CARD_NOT_FOUND",
     "Failed to mount the SD card. Card may be bad or incompatible."},
    {"FILE_IO_END_OF_CWD", "There are no more files in the current working directory."}
};

const xerrors::Error CRITICAL_ERROR = driver::CRITICAL_HARDWARE_ERROR.sub("labjack");
const xerrors::Error TEMPORARY_ERROR = driver::TEMPORARY_HARDWARE_ERROR.sub("labjack");
const xerrors::Error RECONNECT_FAILED = CRITICAL_ERROR.sub("LJME_RECONNECT_FAILED");
const xerrors::Error NO_RESPONSE_BYTES_RECEIVED = CRITICAL_ERROR.sub(
    "LJME_NO_RESPONSE_BYTES_RECEIVED"
);
const xerrors::Error STREAM_NOT_INITIALIZED = CRITICAL_ERROR.sub(
    "LJME_STREAM_NOT_INITIALIZED"
);
const xerrors::Error SYNCHRONIZATION_TIMEOUT = CRITICAL_ERROR.sub(
    "LJME_SYNCHRONIZATION_TIMEOUT"
);
const xerrors::Error LJME_AUTO_IPS_FILE_NOT_FOUND = CRITICAL_ERROR.sub(
    "LJME_AUTO_IPS_FILE_NOT_FOUND"
);
const auto TEMPORARILY_UNREACHABLE = xerrors::Error(
    TEMPORARY_ERROR.sub("unreachable"),
    "The device is temporarily unreachable. Will keep trying"
);

inline xerrors::Error parse_error(const std::shared_ptr<ljm::API> &ljm, const int err) {
    if (err == 0) return xerrors::NIL;

    char err_msg[LJM_MAX_NAME_SIZE];
    ljm->err_to_string(err, err_msg);

    std::string description;
    if (const auto it = ERROR_DESCRIPTIONS.find(err_msg);
        it != ERROR_DESCRIPTIONS.end())
        description = it->second;
    return xerrors::Error(CRITICAL_ERROR.sub(err_msg), description);
}
}
