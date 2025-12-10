/*============================================================================*/
/*                       NI System Configuration API                          */
/*----------------------------------------------------------------------------*/
/*----------------------------------------------------------------------------*/
/*                                                                            */
/* Title:   nisyscfg.h                                                        */
/* Purpose: Include file to interact with the NI System Configuration API     */
/*                                                                            */
/*============================================================================*/

#ifndef ___nisysconfig_h___
#define ___nisysconfig_h___

#include <stddef.h>
#include <stdarg.h>
#include "nisyscfg_errors.h"

#if defined(_CVI_)
#pragma EnableLibraryRuntimeChecking
#endif

// All our public functions return a signed 32-bit status code.
#if defined(__WIN32__) || defined(__NT__) || defined(_WIN32) || defined(WIN32)
#define NISYSCFGCONV    __stdcall
#define NISYSCFGCDECL   NISysCfgStatus __cdecl
typedef unsigned __int64 NISysCfgUInt64;
#else
#define NISYSCFGCONV
#define NISYSCFGCDECL   NISysCfgStatus
typedef unsigned long long NISysCfgUInt64;
#endif
#define NISYSCFGCFUNC   NISysCfgStatus NISYSCFGCONV

#if defined(__cplusplus) || defined(__cplusplus__)
extern "C" {
#endif

// Functions that output a simple string require a user-supplied buffer of this size.
#define NISYSCFG_SIMPLE_STRING_LENGTH           1024

// Functions that auto-restart a target use a default timeout of 180 seconds (3 minutes).
// Call NISysCfgSetSystemProperty with NISysCfgSystemPropertyAutoRestartTimeout to change it.
#define NISYSCFG_REBOOT_DEFAULT_TIMEOUT_MSEC    180000

// Functions that operate on a remote target use a default timeout of 300 seconds (5 minutes).
// Call NISysCfgSetRemoteTimeout to change it.
#define NISYSCFG_REMOTE_DEFAULT_TIMEOUT_MSEC    300000

#if defined(_CVI_)
#pragma pack(push, 4)
#ifndef CVITime_DECLARED
#define CVITime_DECLARED
typedef struct CVITime { unsigned __int64 lsb; __int64 msb; } CVITime;
#endif
#ifndef CVIAbsoluteTime_DECLARED
#define CVIAbsoluteTime_DECLARED
typedef union CVIAbsoluteTime { CVITime cviTime; unsigned int u32Data[4]; } CVIAbsoluteTime;
#endif
typedef CVIAbsoluteTime NISysCfgTimestampUTC;
#pragma pack(pop)
#else
typedef struct {
    unsigned int u32Data[4];
} NISysCfgTimestampUTC;
#endif

////////////////////////////////////////////////////////////////////////////////
// Handles
////////////////////////////////////////////////////////////////////////////////

typedef void *NISysCfgSessionHandle;
typedef void *NISysCfgResourceHandle;
typedef void *NISysCfgFilterHandle;
typedef void *NISysCfgSoftwareSetHandle;
typedef void *NISysCfgEnumResourceHandle;
typedef void *NISysCfgEnumExpertHandle;
typedef void *NISysCfgEnumSystemHandle;
typedef void *NISysCfgEnumSoftwareFeedHandle;
typedef void *NISysCfgEnumSoftwareSetHandle;
typedef void *NISysCfgEnumDependencyHandle;
typedef void *NISysCfgEnumSoftwareComponentHandle;

////////////////////////////////////////////////////////////////////////////////
// Enumerations
////////////////////////////////////////////////////////////////////////////////

typedef enum {
    NISysCfgIncludeCachedResultsNone = 0,
    NISysCfgIncludeCachedResultsOnlyIfOnline = 1,
    NISysCfgIncludeCachedResultsAll = 3
} NISysCfgIncludeCachedResults;

// The initialization string may contain any combination of 1 or 2
// of the hostname, IP address, and/or MAC address.
typedef enum {
    NISysCfgSystemNameFormatHostname = 0x10,        // "hostname"
    NISysCfgSystemNameFormatHostnameIp = 0x12,        // "hostname (1.2.3.4)"
    NISysCfgSystemNameFormatHostnameMac = 0x13,        // "hostname (01:02:03:04:05:06)"
    NISysCfgSystemNameFormatIp = 0x20,        // "1.2.3.4"
    NISysCfgSystemNameFormatIpHostname = 0x21,        // "1.2.3.4 (hostname)"
    NISysCfgSystemNameFormatIpMac = 0x23,        // "1.2.3.4 (01:02:03:04:05:06)"
    NISysCfgSystemNameFormatMac = 0x30,        // "01:02:03:04:05:06"
    NISysCfgSystemNameFormatMacHostname = 0x31,        // "01:02:03:04:05:06 (hostname)"
    NISysCfgSystemNameFormatMacIp = 0x32         // "01:02:03:04:05:06 (1.2.3.4)"
} NISysCfgSystemNameFormat;

typedef enum {
    NISysCfgFileSystemDefault = 0x0000,
    NISysCfgFileSystemFat = 0x0001,
    NISysCfgFileSystemReliance = 0x0002,
    NISysCfgFileSystemUBIFS = 0x4000,
    NISysCfgFileSystemExt4 = 0x8000
} NISysCfgFileSystemMode;

typedef enum {
    NISysCfgResetPrimaryResetOthers = 0,
    NISysCfgPreservePrimaryResetOthers = 1,
    NISysCfgPreservePrimaryPreserveOthers = 2,
    NISysCfgPreservePrimaryApplyOthers = 3,
    NISysCfgApplyPrimaryResetOthers = 4,
    NISysCfgApplyPrimaryPreserveOthers = 5,
    NISysCfgApplyPrimaryApplyOthers = 6
} NISysCfgNetworkInterfaceSettings;

typedef enum {
    NISysCfgItemTypeStandard = 0,           // Standard visible component
    NISysCfgItemTypeHidden = 1,           // Hidden component
    NISysCfgItemTypeSystem = 2,           // Required system component (hidden package or base system image installed by the host)
    NISysCfgItemTypeUnknown = 3,           // Unknown component type
    NISysCfgItemTypeStartup = 4,           // Startup component
    NISysCfgItemTypeImage = 5,           // User-defined system image
    NISysCfgItemTypeEssential = 6,           // Required visible component
    NISysCfgItemTypeSystemPackage = 7            // Base system image installed using a package from feeds on ni.com
} NISysCfgComponentType;

typedef enum {
    NISysCfgIncludeItemsAllVisible = 0x0000,      // All visible (standard, startup, essential)
    NISysCfgIncludeItemsAllVisibleAndHidden = 0x0001,      // Visible and hidden
    NISysCfgIncludeItemsOnlyStandard = 0x0002,      // Only standard
    NISysCfgIncludeItemsOnlyStartup = 0x0003       // Only startup
} NISysCfgIncludeComponentTypes;

typedef enum {
    NISysCfgVersionSelectionHighest = 0,
    NISysCfgVersionSelectionExact = 1
} NISysCfgVersionSelectionMode;

typedef enum {
    NISysCfgImportMergeItems = 0,           // Source data "wins" in the case of overwrite conflicts
    NISysCfgImportDeleteConfigFirst = 0x100000,    // Delete product data at destination prior to copying
    NISysCfgImportPreserveConflictItems = 0x200000     // Destination data "wins" in the case of overwrite conflicts
} NISysCfgImportMode;

typedef enum {
    NISysCfgReportXml = 0,
    NISysCfgReportHtml = 1,
    NISysCfgReportTechnicalSupportZip = 2
} NISysCfgReportType;

typedef enum {
    NISysCfgBusTypeBuiltIn = 0,
    NISysCfgBusTypePciPxi = 1,
    NISysCfgBusTypeUsb = 2,
    NISysCfgBusTypeGpib = 3,
    NISysCfgBusTypeVxi = 4,
    NISysCfgBusTypeSerial = 5,
    NISysCfgBusTypeTcpIp = 6,
    NISysCfgBusTypeCompactRio = 7,
    NISysCfgBusTypeScxi = 8,
    NISysCfgBusTypeCompactDaq = 9,
    NISysCfgBusTypeSwitchBlock = 10,
    NISysCfgBusTypeScc = 11,
    NISysCfgBusTypeFireWire = 12,
    NISysCfgBusTypeAccessory = 13,
    NISysCfgBusTypeCan = 14,
    NISysCfgBusTypeSwitchBlockDevice = 15,
    NISysCfgBusTypeSlsc = 16
} NISysCfgBusType;

typedef enum {
    NISysCfgHasDriverTypeUnknown = -1,
    NISysCfgHasDriverTypeNotInstalled = 0,
    NISysCfgHasDriverTypeInstalled = 1
} NISysCfgHasDriverType;

typedef enum {
    NISysCfgIsPresentTypeInitializing = -2,
    NISysCfgIsPresentTypeUnknown = -1,
    NISysCfgIsPresentTypeNotPresent = 0,
    NISysCfgIsPresentTypePresent = 1
} NISysCfgIsPresentType;

typedef enum {
    NISysCfgIpAddressModeStatic = 1,
    NISysCfgIpAddressModeDhcpOrLinkLocal = 2,
    NISysCfgIpAddressModeLinkLocalOnly = 4,
    NISysCfgIpAddressModeDhcpOnly = 8
} NISysCfgIpAddressMode;

typedef enum {
    NISysCfgBoolFalse = 0,
    NISysCfgBoolTrue = 1
} NISysCfgBool;

typedef enum {
    NISysCfgLocaleDefault = 0,
    NISysCfgLocaleChineseSimplified = 2052,
    NISysCfgLocaleEnglish = 1033,
    NISysCfgLocaleFrench = 1036,
    NISysCfgLocaleGerman = 1031,
    NISysCfgLocaleJapanese = 1041,
    NISysCfgLocaleKorean = 1042
} NISysCfgLocale;

typedef enum {
    NISysCfgFilterModeMatchValuesAll = 1,
    NISysCfgFilterModeMatchValuesAny = 2,
    NISysCfgFilterModeMatchValuesNone = 3,
    NISysCfgFilterModeAllPropertiesExist = 4
} NISysCfgFilterMode;

typedef enum {
    NISysCfgServiceTypemDnsNiTcp = 0,
    NISysCfgServiceTypemDnsNiRealtime = 1,
    NISysCfgServiceTypemDnsNiSysapi = 2,
    NISysCfgServiceTypemDnsNiHttp = 3,
    NISysCfgServiceTypeLocalSystem = 4,
    NISysCfgServiceTypeLocalNetInterface = 5,
    NISysCfgServiceTypeLocalTimeKeeper = 6,
    NISysCfgServiceTypeLocalTimeSource = 7,
    NISysCfgServiceTypemDnsLxi = 8,
    NISysCfgServiceTypeLocalFpga = 9
} NISysCfgServiceType;

typedef enum {
    NISysCfgAdapterTypeEthernet = 1,
    NISysCfgAdapterTypeWlan = 2
} NISysCfgAdapterType;

typedef enum {
    NISysCfgAdapterModeDisabled = 1,
    NISysCfgAdapterModeTcpIpEthernet = 2,
    NISysCfgAdapterModeDeterministic = 4,
    NISysCfgAdapterModeEtherCat = 8,
    NISysCfgAdapterModeTcpIpWlan = 32,
    NISysCfgAdapterModeTcpIpAccessPoint = 64
} NISysCfgAdapterMode;

typedef enum {
    NISysCfgLinkSpeedNone = 0,
    NISysCfgLinkSpeedAuto = 1,
    NISysCfgLinkSpeed10mbHalf = 2,
    NISysCfgLinkSpeed10mbFull = 4,
    NISysCfgLinkSpeed100mbHalf = 8,
    NISysCfgLinkSpeed100mbFull = 16,
    NISysCfgLinkSpeedGigabitHalf = 32,
    NISysCfgLinkSpeedGigabitFull = 64,
    // Wireless 802.11 protocols (speeds)
    NISysCfgLinkSpeedWlan80211a = 131072,
    NISysCfgLinkSpeedWlan80211b = 262144,
    NISysCfgLinkSpeedWlan80211g = 524288,
    NISysCfgLinkSpeedWlan80211n = 1048576,
    NISysCfgLinkSpeedWlan80211n5GHz = 2097152
} NISysCfgLinkSpeed;

typedef enum {
    NISysCfgPacketDetectionNone = 0,
    NISysCfgPacketDetectionLineInterrupt = 1,
    NISysCfgPacketDetectionPolling = 2,
    NISysCfgPacketDetectionSignaledInterrupt = 4
} NISysCfgPacketDetection;

typedef enum {
    NISysCfgConnectionTypeNone = 0,
    NISysCfgConnectionTypeInfrastructure = 1,
    NISysCfgConnectionTypeAdHoc = 2
} NISysCfgConnectionType;

typedef enum {
    NISysCfgSecurityTypeNone = 0,
    NISysCfgSecurityTypeNotSupported = 1,
    NISysCfgSecurityTypeOpen = 2,
    NISysCfgSecurityTypeWep = 4,
    NISysCfgSecurityTypeWpaPsk = 8,
    NISysCfgSecurityTypeWpaEap = 16,
    NISysCfgSecurityTypeWpa2Psk = 32,
    NISysCfgSecurityTypeWpa2Eap = 64
} NISysCfgSecurityType;

typedef enum {
    NISysCfgEapTypeNone = 0,
    NISysCfgEapTypeEapTls = 1,
    NISysCfgEapTypeEapTtls = 2,
    NISysCfgEapTypeEapFast = 4,
    NISysCfgEapTypeLeap = 8,
    NISysCfgEapTypePeap = 16
} NISysCfgEapType;

// Negative firmware states are in-progress; the user should continue polling.
// Non-negative firmware states are terminal; no update operation is in progress.
typedef enum {
    NISysCfgFirmwareReadyPendingAutoRestart = -4,
    NISysCfgFirmwareVerifyingNewImage = -3,
    NISysCfgFirmwareWritingFlashingNewImage = -2,
    NISysCfgFirmwareUpdateModeWaitingForImage = -1,
    NISysCfgFirmwareCorruptCannotRun = 0,
    NISysCfgFirmwareNoneInstalled = 1,
    NISysCfgFirmwareInstalledNormalOperation = 2,
    NISysCfgFirmwareReadyPendingUserRestart = 3,
    NISysCfgFirmwareReadyPendingUserAction = 4,
    NISysCfgFirmwareUpdateAttemptFailed = 5
} NISysCfgFirmwareStatus;

typedef enum {
    NISysCfgValidateButDoNotDelete = -1,
    NISysCfgDeleteIfNoDependenciesExist = 0,
    NISysCfgDeleteItemAndAnyDependencies = 1,
    NISysCfgDeleteItemButKeepDependencies = 2
} NISysCfgDeleteValidationMode;

typedef enum {
    NISysCfgAccessTypeLocalOnly = 0,
    NISysCfgAccessTypeLocalAndRemote = 1
} NISysCfgAccessType;

typedef enum {
    NISysCfgLedStateOff = 0,
    NISysCfgLedStateSolidGreen = 1,
    NISysCfgLedStateSolidYellow = 2,
    NISysCfgLedStateBlinkingGreen = 4,
    NISysCfgLedStateBlinkingYellow = 8
} NISysCfgLedState;

typedef enum {
    NISysCfgSwitchStateDisabled = 0,
    NISysCfgSwitchStateEnabled = 1
} NISysCfgSwitchState;

typedef enum {
    NISysCfgFirmwareUpdateModeNone = 0,
    NISysCfgFirmwareUpdateModeManual = 1,
    NISysCfgFirmwareUpdateModeDriverManaged = 2
} NISysCfgFirmwareUpdateMode;

typedef enum {
    NISysCfgModuleProgramModeNone = 0,
    NISysCfgModuleProgramModeRealtimeCpu = 1,
    NISysCfgModuleProgramModeRealtimeScan = 2,
    NISysCfgModuleProgramModeLabVIEWFpga = 4
} NISysCfgModuleProgramMode;

typedef enum {
    NISysCfgFeatureActivationStateNone = 0,
    NISysCfgFeatureActivationStateUnactivated = 1,
    NISysCfgFeatureActivationStateActivated = 2
} NISysCfgFeatureActivationState;

// NOTE: For string properties, callers pass in a pointer to a buffer or array they have allocated.
typedef enum {
    // Read-only properties
    NISysCfgResourcePropertyIsDevice = 16781312,    // NISysCfgBool
    NISysCfgResourcePropertyIsChassis = 16941056,    // NISysCfgBool
    NISysCfgResourcePropertyConnectsToBusType = 16785408,    // NISysCfgBusType
    NISysCfgResourcePropertyVendorId = 16789504,    // unsigned int
    NISysCfgResourcePropertyVendorName = 16793600,    // char *
    NISysCfgResourcePropertyProductId = 16797696,    // unsigned int
    NISysCfgResourcePropertyProductName = 16801792,    // char *
    NISysCfgResourcePropertySerialNumber = 16805888,    // char *
    NISysCfgResourcePropertyFirmwareRevision = 16969728,    // char *
    NISysCfgResourcePropertyIsNIProduct = 16809984,    // NISysCfgBool
    NISysCfgResourcePropertyIsSimulated = 16814080,    // NISysCfgBool
    NISysCfgResourcePropertyConnectsToLinkName = 16818176,    // char *
    NISysCfgResourcePropertyHasDriver = 16920576,    // NISysCfgHasDriverType
    NISysCfgResourcePropertyIsPresent = 16924672,    // NISysCfgIsPresentType
    NISysCfgResourcePropertySlotNumber = 16822272,    // int
    NISysCfgResourcePropertySupportsInternalCalibration = 16842752,    // NISysCfgBool
    NISysCfgResourcePropertySupportsExternalCalibration = 16859136,    // NISysCfgBool
    NISysCfgResourcePropertyExternalCalibrationLastTemp = 16867328,    // double
    NISysCfgResourcePropertyCalibrationComments = 16961536,    // char *
    NISysCfgResourcePropertyInternalCalibrationLastLimited = 17420288,    // NISysCfgBool
    NISysCfgResourcePropertyExternalCalibrationChecksum = 17432576,    // char *
    NISysCfgResourcePropertyCurrentTemp = 16965632,    // double
    NISysCfgResourcePropertyPxiPciBusNumber = 16875520,    // unsigned int
    NISysCfgResourcePropertyPxiPciDeviceNumber = 16879616,    // unsigned int
    NISysCfgResourcePropertyPxiPciFunctionNumber = 16883712,    // unsigned int
    NISysCfgResourcePropertyPxiPciLinkWidth = 16973824,    // int
    NISysCfgResourcePropertyPxiPciMaxLinkWidth = 16977920,    // int
    NISysCfgResourcePropertyUsbInterface = 16887808,    // unsigned int
    NISysCfgResourcePropertyTcpHostName = 16928768,    // char *
    NISysCfgResourcePropertyTcpMacAddress = 16986112,    // char *
    NISysCfgResourcePropertyTcpIpAddress = 16957440,    // char *
    NISysCfgResourcePropertyTcpDeviceClass = 17022976,    // char *
    NISysCfgResourcePropertyGpibPrimaryAddress = 16994304,    // int
    NISysCfgResourcePropertyGpibSecondaryAddress = 16998400,    // int
    NISysCfgResourcePropertySerialPortBinding = 17076224,    // char *
    NISysCfgResourcePropertyProvidesBusType = 16932864,    // NISysCfgBusType
    NISysCfgResourcePropertyProvidesLinkName = 16936960,    // char *
    NISysCfgResourcePropertyNumberOfSlots = 16826368,    // int
    NISysCfgResourcePropertySupportsFirmwareUpdate = 17080320,    // NISysCfgBool
    NISysCfgResourcePropertyFirmwareFilePattern = 17084416,    // char *
    NISysCfgResourcePropertyRecommendedCalibrationInterval = 17207296,    // int
    NISysCfgResourcePropertySupportsCalibrationWrite = 17215488,    // NISysCfgBool
    NISysCfgResourcePropertyHardwareRevision = 17256448,    // char *
    NISysCfgResourcePropertyCpuModelName = 17313792,    // char *
    NISysCfgResourcePropertyCpuSteppingRevision = 17317888,    // int
    NISysCfgResourcePropertyModelNameNumber = 17436672,    // unsigned int
    NISysCfgResourcePropertyModuleProgramMode = 17440768,    // NISysCfgModuleProgramMode
    NISysCfgResourcePropertyConnectsToNumSlots = 17072128,    // int
    NISysCfgResourcePropertySlotOffsetLeft = 17276928,    // unsigned int
    NISysCfgResourcePropertyInternalCalibrationValuesInRange = 17489920,    // NISysCfgBool
    NISysCfgResourcePropertyNumberOfInternalCalibrationDetails = 17510400,    // int
    NISysCfgResourcePropertyFeatureActivationState = 17534976,    // NISysCfgFeatureActivationState

    // Read/Write firmware properties
    NISysCfgResourcePropertyFirmwareUpdateMode = 17354752,    // NISysCfgFirmwareUpdateMode

    // Read/Write calibration properties
    NISysCfgResourcePropertyExternalCalibrationLastTime = 16863232,    // NISysCfgTimestampUTC
    NISysCfgResourcePropertyExternalCalibrationLastAdjustTime = 17502208,    // NISysCfgTimestampUTC
    NISysCfgResourcePropertyRecommendedNextCalibrationTime = 16871424,    // NISysCfgTimestampUTC
    NISysCfgResourcePropertyExternalCalibrationLastLimited = 17428480,    // NISysCfgBool

    // Write-only calibration properties
    NISysCfgResourcePropertyCalibrationCurrentPassword = 17223680,    // char *
    NISysCfgResourcePropertyCalibrationNewPassword = 17227776,    // char *

    // Read/Write remote access properties
    NISysCfgResourcePropertySysCfgAccess = 219504640,   // NISysCfgAccessType

    // Read-only network adapter properties
    NISysCfgResourcePropertyAdapterType = 219332608,   // NISysCfgAdapterType
    NISysCfgResourcePropertyMacAddress = 219168768,   // char *

    // Read/Write network adapter properties
    NISysCfgResourcePropertyAdapterMode = 219160576,   // NISysCfgAdapterMode
    NISysCfgResourcePropertyTcpIpRequestMode = 219172864,   // NISysCfgIpAddressMode
    NISysCfgResourcePropertyTcpIpv4Address = 219181056,   // char *
    NISysCfgResourcePropertyTcpIpv4Subnet = 219189248,   // char *
    NISysCfgResourcePropertyTcpIpv4Gateway = 219193344,   // char *
    NISysCfgResourcePropertyTcpIpv4DnsServer = 219197440,   // char *
    NISysCfgResourcePropertyTcpPreferredLinkSpeed = 219213824,   // NISysCfgLinkSpeed
    NISysCfgResourcePropertyTcpCurrentLinkSpeed = 219222016,   // NISysCfgLinkSpeed
    NISysCfgResourcePropertyTcpPacketDetection = 219258880,   // NISysCfgPacketDetection
    NISysCfgResourcePropertyTcpPollingInterval = 219262976,   // unsigned int
    NISysCfgResourcePropertyIsPrimaryAdapter = 219308032,   // NISysCfgBool
    NISysCfgResourcePropertyEtherCatMasterId = 219250688,   // unsigned int
    NISysCfgResourcePropertyEtherCatMasterRedundancy = 219500544,   // NISysCfgBool

    // Read-only wireless network adapter properties
    NISysCfgResourcePropertyWlanBssid = 219398144,   // char *
    NISysCfgResourcePropertyWlanCurrentLinkQuality = 219394048,   // unsigned int

    // Read/Write wireless network adapter properties
    NISysCfgResourcePropertyWlanCurrentSsid = 219377664,   // char *
    NISysCfgResourcePropertyWlanCurrentConnectionType = 219381760,   // NISysCfgConnectionType
    NISysCfgResourcePropertyWlanCurrentSecurityType = 219385856,   // NISysCfgSecurityType
    NISysCfgResourcePropertyWlanCurrentEapType = 219389952,   // NISysCfgEapType
    NISysCfgResourcePropertyWlanCountryCode = 219406336,   // int
    NISysCfgResourcePropertyWlanChannelNumber = 219410432,   // unsigned int
    NISysCfgResourcePropertyWlanClientCertificate = 219422720,   // char *

    // Write-only wireless network adapter properties
    NISysCfgResourcePropertyWlanSecurityIdentity = 219414528,   // char *
    NISysCfgResourcePropertyWlanSecurityKey = 219418624,   // char *

    // Read-only time properties
    NISysCfgResourcePropertySystemStartTime = 17108992,     // NISysCfgTimestampUTC

    // Read/Write time properties
    NISysCfgResourcePropertyCurrentTime = 219279360,    // NISysCfgTimestampUTC
    NISysCfgResourcePropertyTimeZone = 219471872,    // char *

    // Read/Write startup settings properties
    NISysCfgResourcePropertyUserDirectedSafeModeSwitch = 219537408,    // NISysCfgBool
    NISysCfgResourcePropertyConsoleOutSwitch = 219541504,    // NISysCfgBool
    NISysCfgResourcePropertyIpResetSwitch = 219545600,    // NISysCfgBool

    // Read-only counts for indexed properties
    NISysCfgResourcePropertyNumberOfDiscoveredAccessPoints = 219365376,   // unsigned int
    NISysCfgResourcePropertyNumberOfExperts = 16891904,    // int
    NISysCfgResourcePropertyNumberOfServices = 17010688,    // int
    NISysCfgResourcePropertyNumberOfAvailableFirmwareVersions = 17088512,    // int
    NISysCfgResourcePropertyNumberOfCpuCores = 17506304,    // int
    NISysCfgResourcePropertyNumberOfCpuLogicalProcessors = 17137664,    // int
    NISysCfgResourcePropertyNumberOfFans = 17174528,    // int
    NISysCfgResourcePropertyNumberOfPowerSensors = 17448960,    // int
    NISysCfgResourcePropertyNumberOfTemperatureSensors = 17186816,    // int
    NISysCfgResourcePropertyNumberOfVoltageSensors = 17149952,    // int
    NISysCfgResourcePropertyNumberOfUserLedIndicators = 17281024,    // int
    NISysCfgResourcePropertyNumberOfUserSwitches = 17293312,    // int
    NISysCfgResourcePropertyNumberOfActivatedFeatures = 17518592     // unsigned int
} NISysCfgResourceProperty;

#define NISysCfgResourcePropertyNumberOfCpus                   NISysCfgResourcePropertyNumberOfCpuLogicalProcessors
#define NISysCfgResourcePropertyPxiPciSlotLinkWidth            (NISysCfgResourceProperty)16982016
#define NISysCfgResourcePropertyInternalCalibrationLastTime    (NISysCfgResourceProperty)NISysCfgIndexedPropertyInternalCalibrationLastTime
#define NISysCfgResourcePropertyInternalCalibrationLastTemp    (NISysCfgResourceProperty)NISysCfgIndexedPropertyInternalCalibrationLastTemp

#if defined(WIN32) && (_MSC_VER >= 1300)
#pragma deprecated("NISysCfgResourcePropertyNumberOfCpus")
#pragma deprecated("NISysCfgResourcePropertyPxiPciSlotLinkWidth")
#pragma deprecated("NISysCfgResourcePropertyInternalCalibrationLastTime")
#pragma deprecated("NISysCfgResourcePropertyInternalCalibrationLastTemp")
#endif

typedef enum {
    // Read-only properties
    NISysCfgIndexedPropertyServiceType = 17014784,    // NISysCfgServiceType
    NISysCfgIndexedPropertyAvailableFirmwareVersion = 17092608,    // char *

    // Read-only wireless network adapter properties
    NISysCfgIndexedPropertyWlanAvailableSsid = 219336704,   // char *
    NISysCfgIndexedPropertyWlanAvailableBssid = 219443200,   // char *
    NISysCfgIndexedPropertyWlanAvailableConnectionType = 219340800,   // NISysCfgConnectionType
    NISysCfgIndexedPropertyWlanAvailableSecurityType = 219344896,   // NISysCfgSecurityType
    NISysCfgIndexedPropertyWlanAvailableLinkQuality = 219353088,   // unsigned int
    NISysCfgIndexedPropertyWlanAvailableChannelNumber = 219357184,   // unsigned int
    NISysCfgIndexedPropertyWlanAvailableLinkSpeed = 219361280,   // NISysCfgLinkSpeed

    // Read-only properties
    NISysCfgIndexedPropertyCpuTotalLoad = 17141760,    // unsigned int
    NISysCfgIndexedPropertyCpuInterruptLoad = 17145856,    // unsigned int
    NISysCfgIndexedPropertyCpuSpeed = 17309696,    // unsigned int
    NISysCfgIndexedPropertyFanName = 17178624,    // char *
    NISysCfgIndexedPropertyFanReading = 17182720,    // unsigned int
    NISysCfgIndexedPropertyPowerName = 17453056,    // char *
    NISysCfgIndexedPropertyPowerReading = 17457152,    // double
    NISysCfgIndexedPropertyPowerUpperCritical = 17461248,    // double
    NISysCfgIndexedPropertyTemperatureName = 17190912,    // char *
    NISysCfgIndexedPropertyTemperatureReading = 16965632,    // double
    NISysCfgIndexedPropertyTemperatureLowerCritical = 17195008,    // double
    NISysCfgIndexedPropertyTemperatureUpperCritical = 17199104,    // double
    NISysCfgIndexedPropertyVoltageName = 17154048,    // char *
    NISysCfgIndexedPropertyVoltageReading = 17158144,    // double
    NISysCfgIndexedPropertyVoltageNominal = 17162240,    // double
    NISysCfgIndexedPropertyVoltageLowerCritical = 17166336,    // double
    NISysCfgIndexedPropertyVoltageUpperCritical = 17170432,    // double
    NISysCfgIndexedPropertyUserLedName = 17285120,    // char *
    NISysCfgIndexedPropertyUserSwitchName = 17297408,    // char *
    NISysCfgIndexedPropertyUserSwitchState = 17301504,    // NISysCfgSwitchState
    NISysCfgIndexedPropertyInternalCalibrationName = 17514496,    // char *
    NISysCfgIndexedPropertyInternalCalibrationLastTime = 16846848,    // NISysCfgTimestampUTC
    NISysCfgIndexedPropertyInternalCalibrationLastTemp = 16850944,    // double
    NISysCfgIndexedPropertyActivatedFeatureName = 17526784,    // char *
    NISysCfgIndexedPropertyActivatedFeatureID = 17522688,    // unsigned int

    // Read/Write properties
    NISysCfgIndexedPropertyUserLedState = 17289216,    // NISysCfgLedState

    // Read-only properties
    NISysCfgIndexedPropertyExpertName = 16900096,    // char *
    NISysCfgIndexedPropertyExpertResourceName = 16896000,    // char *
    NISysCfgIndexedPropertyExpertUserAlias = 16904192     // char *
} NISysCfgIndexedProperty;

typedef enum {
    // Read-only properties
    NISysCfgSystemPropertyDeviceClass = 16941057,    // char *
    NISysCfgSystemPropertyProductId = 16941058,    // int
    NISysCfgSystemPropertyFileSystem = 16941060,    // NISysCfgFileSystemMode
    NISysCfgSystemPropertyFirmwareRevision = 16941061,    // char *
    NISysCfgSystemPropertyIsFactoryResetSupported = 16941067,    // NISysCfgBool
    NISysCfgSystemPropertyIsFirmwareUpdateSupported = 16941068,    // NISysCfgBool
    NISysCfgSystemPropertyIsLocked = 16941069,    // NISysCfgBool
    NISysCfgSystemPropertyIsLockingSupported = 16941070,    // NISysCfgBool
    NISysCfgSystemPropertyIsOnLocalSubnet = 16941072,    // NISysCfgBool
    NISysCfgSystemPropertyIsRestartSupported = 16941076,    // NISysCfgBool
    NISysCfgSystemPropertyMacAddress = 16941077,    // char *
    NISysCfgSystemPropertyProductName = 16941078,    // char *
    NISysCfgSystemPropertyOperatingSystem = 16941079,    // char *
    NISysCfgSystemPropertyOperatingSystemVersion = 17100800,    // char *
    NISysCfgSystemPropertyOperatingSystemDescription = 17104896,    // char *
    NISysCfgSystemPropertySerialNumber = 16941080,    // char *
    NISysCfgSystemPropertySystemState = 16941082,    // char *
    NISysCfgSystemPropertyMemoryPhysTotal = 219480064,   // double
    NISysCfgSystemPropertyMemoryPhysFree = 219484160,   // double
    NISysCfgSystemPropertyMemoryLargestBlock = 219488256,   // double
    NISysCfgSystemPropertyMemoryVirtTotal = 219492352,   // double
    NISysCfgSystemPropertyMemoryVirtFree = 219496448,   // double
    NISysCfgSystemPropertyPrimaryDiskTotal = 219291648,   // double
    NISysCfgSystemPropertyPrimaryDiskFree = 219295744,   // double
    NISysCfgSystemPropertySystemResourceHandle = 16941086,    // NISysCfgResourceHandle
    NISysCfgSystemPropertyImageDescription = 219516928,   // char *
    NISysCfgSystemPropertyImageId = 219521024,   // char *
    NISysCfgSystemPropertyImageTitle = 219525120,   // char *
    NISysCfgSystemPropertyImageVersion = 219529216,   // char *
    NISysCfgSystemPropertyInstalledApiVersion = 16941087,    // char *

    // Read/Write properties
    NISysCfgSystemPropertyIsDst = 16941066,    // NISysCfgBool
    NISysCfgSystemPropertyIsRestartProtected = 16941073,    // NISysCfgBool
    NISysCfgSystemPropertyHaltOnError = 16941074,    // NISysCfgBool
    NISysCfgSystemPropertyRepositoryLocation = 16941084,    // char *
    NISysCfgSystemPropertySystemComment = 16941081,    // char *
    NISysCfgSystemPropertyAutoRestartTimeout = 16941085,    // unsigned int

    // Read/Write network adapter properties
    NISysCfgSystemPropertyDnsServer = 16941059,    // char *
    NISysCfgSystemPropertyGateway = 16941062,    // char *
    NISysCfgSystemPropertyHostname = 16941063,    // char *
    NISysCfgSystemPropertyIpAddress = 16941064,    // char *
    NISysCfgSystemPropertyIpAddressMode = 16941065,    // NISysCfgIpAddressMode
    NISysCfgSystemPropertySubnetMask = 16941083     // char *
} NISysCfgSystemProperty;

typedef enum {
    // Write-only properties
    NISysCfgFilterPropertyIsDevice = 16781312,    // NISysCfgBool
    NISysCfgFilterPropertyIsChassis = 16941056,    // NISysCfgBool
    NISysCfgFilterPropertyServiceType = 17014784,    // NISysCfgServiceType
    NISysCfgFilterPropertyConnectsToBusType = 16785408,    // NISysCfgBusType
    NISysCfgFilterPropertyConnectsToLinkName = 16818176,    // char *
    NISysCfgFilterPropertyProvidesBusType = 16932864,    // NISysCfgBusType
    NISysCfgFilterPropertyVendorId = 16789504,    // unsigned int
    NISysCfgFilterPropertyProductId = 16797696,    // unsigned int
    NISysCfgFilterPropertySerialNumber = 16805888,    // char *
    NISysCfgFilterPropertyIsNIProduct = 16809984,    // NISysCfgBool
    NISysCfgFilterPropertyIsSimulated = 16814080,    // NISysCfgBool
    NISysCfgFilterPropertySlotNumber = 16822272,    // int
    NISysCfgFilterPropertyHasDriver = 16920576,    // NISysCfgHasDriverType
    NISysCfgFilterPropertyIsPresent = 16924672,    // NISysCfgIsPresentType
    NISysCfgFilterPropertySupportsCalibration = 16908288,    // NISysCfgBool
    NISysCfgFilterPropertySupportsFirmwareUpdate = 17080320,    // NISysCfgBool
    NISysCfgFilterPropertyProvidesLinkName = 16936960,    // char *
    NISysCfgFilterPropertyExpertName = 16900096,    // char *
    NISysCfgFilterPropertyResourceName = 16896000,    // char *
    NISysCfgFilterPropertyUserAlias = 16904192     // char *
} NISysCfgFilterProperty;

typedef enum {
    NISysCfgPropertyTypeBool = 1,
    NISysCfgPropertyTypeInt = 2,
    NISysCfgPropertyTypeUnsignedInt = 3,
    NISysCfgPropertyTypeDouble = 4,
    NISysCfgPropertyTypeString = 6,
    NISysCfgPropertyTypeTimestamp = 7,
} NISysCfgPropertyType;

// These macros are provided for backward compatibility.
#define NISysCfgBusTypeFlexAdapter                             NISysCfgBusTypeAccessory
#define NISysCfgFilterModeAll                                  NISysCfgFilterModeMatchValuesAll
#define NISysCfgFilterModeAny                                  NISysCfgFilterModeMatchValuesAny
#define NISysCfgFilterModeNone                                 NISysCfgFilterModeMatchValuesNone
#define NISysCfgPacketDetectionInterrupt                       NISysCfgPacketDetectionLineInterrupt
#define NISysCfgResourcePropertyWlanAvailableCount             NISysCfgResourcePropertyNumberOfDiscoveredAccessPoints
#define NISysCfgResetPrimaryDisableOthers                      NISysCfgResetPrimaryResetOthers
#define NISysCfgPreservePrimaryDisableOthers                   NISysCfgPreservePrimaryResetOthers
#define NISysCfgApplyPrimaryDisableOthers                      NISysCfgApplyPrimaryResetOthers

////////////////////////////////////////////////////////////////////////////////
// System Configuration core functions
////////////////////////////////////////////////////////////////////////////////

NISYSCFGCFUNC NISysCfgInitializeSession(
        const char *targetName,                // NULL or "" => localhost
        const char *username,                  // NULL or "" => no credentials
        const char *password,                  // NULL or "" => no credentials
        NISysCfgLocale language,                  // LCID or 0 to indicate default.
        NISysCfgBool forcePropertyRefresh,
        unsigned int connectTimeoutMsec,
        NISysCfgEnumExpertHandle *expertEnumHandle,          // Can be NULL
        NISysCfgSessionHandle *sessionHandle
);

// This function is used to close a session or any other handle type returned by this API.
NISYSCFGCFUNC NISysCfgCloseHandle(
        void *syscfgHandle
);

NISYSCFGCFUNC NISysCfgGetSystemExperts(
        NISysCfgSessionHandle sessionHandle,
        const char *expertNames,               // NULL or "" => all experts
        NISysCfgEnumExpertHandle *expertEnumHandle
);

NISYSCFGCFUNC NISysCfgSetRemoteTimeout(
        NISysCfgSessionHandle sessionHandle,
        unsigned int remoteTimeoutMsec
);

NISYSCFGCFUNC NISysCfgFindHardware(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgFilterMode filterMode,                // Ignored if filter handle is NULL
        NISysCfgFilterHandle filterHandle,              // Can be NULL
        const char *expertNames,               // NULL or "" => all experts
        NISysCfgEnumResourceHandle *resourceEnumHandle
);

NISYSCFGCFUNC NISysCfgFindSystems(
        NISysCfgSessionHandle sessionHandle,             // Can be NULL or a session to "localhost"
        const char *deviceClass,               // NULL or "" => all classes
        NISysCfgBool detectOnlineSystems,
        NISysCfgIncludeCachedResults cacheMode,
        NISysCfgSystemNameFormat findOutputMode,
        unsigned int timeoutMsec,
        NISysCfgBool onlyInstallableSystems,
        NISysCfgEnumSystemHandle *systemEnumHandle
);

// Caller should free detailedResult using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgSelfTestHardware(
        NISysCfgResourceHandle resourceHandle,
        unsigned int mode,
        char **detailedResult
);

// Caller should free detailedResult using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgSelfCalibrateHardware(
        NISysCfgResourceHandle resourceHandle,
        char **detailedResult
);

NISYSCFGCFUNC NISysCfgResetHardware(
        NISysCfgResourceHandle resourceHandle,
        unsigned int mode
);

NISYSCFGCFUNC NISysCfgRenameResource(
        NISysCfgResourceHandle resourceHandle,
        const char *newName,
        NISysCfgBool overwriteConflict,
        NISysCfgBool updateDependencies,
        NISysCfgBool *nameAlreadyExisted,        // Can be NULL
        NISysCfgResourceHandle *overwrittenResourceHandle  // Can be NULL
);

// Caller should free detailedResult using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgDeleteResource(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgDeleteValidationMode mode,
        NISysCfgBool *dependentItemsDeleted,     // Can be NULL
        char **detailedResult
);

NISYSCFGCFUNC NISysCfgGetResourceProperty(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgResourceProperty propertyID,
        void *value
);

NISYSCFGCFUNC NISysCfgGetResourcePropertyType(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgResourceProperty propertyID,
        NISysCfgPropertyType *propertyType
);

NISYSCFGCDECL NISysCfgSetResourceProperty(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgResourceProperty propertyID,
        ...
);

NISYSCFGCDECL NISysCfgSetResourcePropertyWithType(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgResourceProperty propertyID,
        NISysCfgPropertyType propertyType,
        ...
);

NISYSCFGCFUNC NISysCfgSetResourcePropertyV(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgResourceProperty propertyID,
        va_list args
);

NISYSCFGCFUNC NISysCfgSetResourcePropertyWithTypeV(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgResourceProperty propertyID,
        NISysCfgPropertyType propertyType,
        va_list args
);

NISYSCFGCFUNC NISysCfgGetResourceIndexedProperty(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgIndexedProperty propertyID,
        unsigned int index,
        void *value
);

// Caller should free detailedResult using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgSaveResourceChanges(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgBool *changesRequireRestart,
        char **detailedResult
);

NISYSCFGCFUNC NISysCfgGetSystemProperty(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgSystemProperty propertyID,
        void *value
);

NISYSCFGCFUNC NISysCfgGetSystemPropertyType(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgSystemProperty propertyID,
        NISysCfgPropertyType *propertyType
);

NISYSCFGCDECL NISysCfgSetSystemProperty(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgSystemProperty propertyID,
        ...
);

NISYSCFGCFUNC NISysCfgSetSystemPropertyV(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgSystemProperty propertyID,
        va_list args
);

// Caller should free detailedResult using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgSaveSystemChanges(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool *changesRequireRestart,
        char **detailedResult
);

NISYSCFGCFUNC NISysCfgCreateFilter(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgFilterHandle *filterHandle
);

NISYSCFGCDECL NISysCfgSetFilterProperty(
        NISysCfgFilterHandle filterHandle,
        NISysCfgFilterProperty propertyID,
        ...
);

NISYSCFGCDECL NISysCfgSetFilterPropertyWithType(
        NISysCfgFilterHandle filterHandle,
        NISysCfgFilterProperty propertyID,
        NISysCfgPropertyType propertyType,
        ...
);

NISYSCFGCFUNC NISysCfgSetFilterPropertyV(
        NISysCfgFilterHandle filterHandle,
        NISysCfgFilterProperty propertyID,
        va_list args
);

NISYSCFGCFUNC NISysCfgSetFilterPropertyWithTypeV(
        NISysCfgFilterHandle filterHandle,
        NISysCfgFilterProperty propertyID,
        NISysCfgPropertyType propertyType,
        va_list args
);

////////////////////////////////////////////////////////////////////////////////
// System Configuration firmware modifit cation functions
////////////////////////////////////////////////////////////////////////////////

// Caller should free detailedResult using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgUpgradeFirmwareFromFile(
        NISysCfgResourceHandle resourceHandle,
        const char *firmwareFile,
        NISysCfgBool autoStopTasks,
        NISysCfgBool alwaysOverwrite,
        NISysCfgBool waitForOperationToFinish,
        NISysCfgFirmwareStatus *firmwareStatus,
        char **detailedResult
);

// Caller should free detailedResult using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgUpgradeFirmwareVersion(
        NISysCfgResourceHandle resourceHandle,
        const char *firmwareVersion,
        NISysCfgBool autoStopTasks,
        NISysCfgBool alwaysOverwrite,
        NISysCfgBool waitForOperationToFinish,
        NISysCfgFirmwareStatus *firmwareStatus,
        char **detailedResult
);

// Caller should free detailedResult using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgEraseFirmware(
        NISysCfgResourceHandle resourceHandle,
        NISysCfgBool autoStopTasks,
        NISysCfgFirmwareStatus *firmwareStatus,
        char **detailedResult
);

// Caller should free detailedResult using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgCheckFirmwareStatus(
        NISysCfgResourceHandle resourceHandle,
        int *percentComplete,
        NISysCfgFirmwareStatus *firmwareStatus,
        char **detailedResult
);

// Caller should free detailedResult using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgActivateFeature(
        NISysCfgResourceHandle resourceHandle,
        unsigned int featureID,
        const char *activationCode,
        char **detailedResult
);

////////////////////////////////////////////////////////////////////////////////
// System Configuration software and installation functions
////////////////////////////////////////////////////////////////////////////////

NISYSCFGCFUNC NISysCfgFormat(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool forceSafeMode,
        NISysCfgBool restartAfterFormat,
        NISysCfgFileSystemMode fileSystem,
        NISysCfgNetworkInterfaceSettings networkSettings,
        unsigned int timeoutMsec
);

NISYSCFGCFUNC NISysCfgFormatWithBaseSystemImage(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool autoRestart,
        NISysCfgFileSystemMode fileSystem,
        NISysCfgNetworkInterfaceSettings networkSettings,
        const char *systemImageID,
        const char *systemImageVersion,
        unsigned int timeoutMsec
);

NISYSCFGCFUNC NISysCfgRestart(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool waitForRestartToFinish,
        NISysCfgBool installMode,
        NISysCfgBool flushDNS,
        unsigned int timeoutMsec,
        char newIpAddress[]
);

NISYSCFGCFUNC NISysCfgGetAvailableSoftwareComponents(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgIncludeComponentTypes itemTypes,
        NISysCfgEnumSoftwareComponentHandle *componentEnumHandle
);

NISYSCFGCFUNC NISysCfgGetAvailableSoftwareSets(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgEnumSoftwareSetHandle *setEnumHandle
);

NISYSCFGCFUNC NISysCfgGetAvailableBaseSystemImages(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgEnumSoftwareComponentHandle *systemImageEnumHandle
);

NISYSCFGCFUNC NISysCfgGetFilteredSoftwareComponents(
        const char *repositoryPath,
        const char *deviceClass,
        const char *operatingSystem,
        unsigned int productID,
        NISysCfgIncludeComponentTypes itemTypes,
        NISysCfgEnumSoftwareComponentHandle *componentEnumHandle
);

NISYSCFGCFUNC NISysCfgGetFilteredSoftwareSets(
        const char *repositoryPath,
        const char *deviceClass,
        const char *operatingSystem,
        unsigned int productID,
        NISysCfgEnumSoftwareSetHandle *setEnumHandle
);

NISYSCFGCFUNC NISysCfgGetFilteredBaseSystemImages(
        const char *repositoryPath,
        const char *deviceClass,
        const char *operatingSystem,
        unsigned int productID,
        NISysCfgEnumSoftwareComponentHandle *systemImageEnumHandle
);

NISYSCFGCFUNC NISysCfgGetInstalledSoftwareComponents(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgIncludeComponentTypes itemTypes,
        NISysCfgBool cached,
        NISysCfgEnumSoftwareComponentHandle *componentEnumHandle
);

NISYSCFGCFUNC NISysCfgGetInstalledSoftwareSet(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool cached,
        NISysCfgSoftwareSetHandle *setHandle
);

NISYSCFGCFUNC NISysCfgGetSystemImageAsFolder(
        NISysCfgSessionHandle sessionHandle,
        const char *destinationFolder,
        const char *encryptionPassphrase,
        NISysCfgBool overwriteIfExists,
        NISysCfgBool installedSoftwareOnly,
        NISysCfgBool autoRestart
);

NISYSCFGCFUNC NISysCfgGetSystemImageAsFolder2(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool autoRestart,
        const char *destinationFolder,
        const char *encryptionPassphrase,
        unsigned int numBlacklistEntries,
        const char **blacklistFilesDirectories, // Can be NULL if numBlacklistEntries==0
        NISysCfgBool overwriteIfExists,
        NISysCfgBool installedSoftwareOnly
);

NISYSCFGCFUNC NISysCfgCreateSystemImageAsFolder(
        NISysCfgSessionHandle sessionHandle,
        const char *imageTitle,
        const char *imageID,
        const char *imageVersion,
        const char *imageDescription,
        NISysCfgBool autoRestart,
        const char *destinationFolder,
        const char *encryptionPassphrase,
        unsigned int numBlacklistEntries,
        const char **blacklistFilesDirectories, // Can be NULL if numBlacklistEntries==0
        NISysCfgBool overwriteIfExists
);

NISYSCFGCFUNC NISysCfgSetSystemImageFromFolder(
        NISysCfgSessionHandle sessionHandle,
        const char *sourceFolder,
        const char *encryptionPassphrase,
        NISysCfgBool autoRestart,
        NISysCfgBool originalSystemOnly
);

NISYSCFGCFUNC NISysCfgSetSystemImageFromFolder2(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool autoRestart,
        const char *sourceFolder,
        const char *encryptionPassphrase,
        unsigned int numBlacklistEntries,
        const char **blacklistFilesDirectories, // Can be NULL if numBlacklistEntries==0
        NISysCfgBool originalSystemOnly,
        NISysCfgNetworkInterfaceSettings networkSettings
);

NISYSCFGCFUNC NISysCfgInstallAll(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool autoRestart,
        NISysCfgBool deselectConflicts,
        NISysCfgEnumSoftwareComponentHandle *installedComponentEnumHandle,
        NISysCfgEnumDependencyHandle *brokenDependencyEnumHandle
);

NISYSCFGCFUNC NISysCfgInstallUninstallComponents(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool autoRestart,
        NISysCfgBool autoSelectDependencies,
        NISysCfgEnumSoftwareComponentHandle componentToInstallEnumHandle,
        unsigned int numComponentsToUninstall,
        const char **componentIDsToUninstall,   // Can be NULL if numComponentsToUninstall==0
        NISysCfgEnumDependencyHandle *brokenDependencyEnumHandle
);

NISYSCFGCFUNC NISysCfgInstallUninstallComponents2(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool autoRestart,
        NISysCfgBool autoSelectDependencies,
        NISysCfgBool autoSelectRecommends,
        NISysCfgEnumSoftwareComponentHandle componentToInstallEnumHandle,
        unsigned int numComponentsToUninstall,
        const char **componentIDsToUninstall,   // Can be NULL if numComponentsToUninstall==0
        NISysCfgEnumDependencyHandle *brokenDependencyEnumHandle
);

NISYSCFGCFUNC NISysCfgInstallSoftwareSet(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool autoRestart,
        const char *softwareSetID,
        const char *version,
        NISysCfgEnumSoftwareComponentHandle addonEnumHandle,
        NISysCfgEnumDependencyHandle *brokenDependencyEnumHandle
);

NISYSCFGCFUNC NISysCfgInstallStartup(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool autoRestart,
        NISysCfgEnumSoftwareComponentHandle startupEnumHandle,
        NISysCfgBool uninstallConflicts,
        NISysCfgEnumSoftwareComponentHandle *installedComponentEnumHandle,
        NISysCfgEnumSoftwareComponentHandle *uninstalledComponentEnumHandle,
        NISysCfgEnumDependencyHandle *brokenDependencyEnumHandle
);

NISYSCFGCFUNC NISysCfgUninstallAll(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgBool autoRestart
);

NISYSCFGCFUNC NISysCfgGetSoftwareFeeds(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgEnumSoftwareFeedHandle *feedEnumHandle
);

NISYSCFGCFUNC NISysCfgAddSoftwareFeed(
        NISysCfgSessionHandle sessionHandle,
        const char *feedName,
        const char *uri,
        NISysCfgBool enabled,
        NISysCfgBool trusted
);

NISYSCFGCFUNC NISysCfgModifySoftwareFeed(
        NISysCfgSessionHandle sessionHandle,
        const char *feedName,
        const char *newFeedName,
        const char *uri,
        NISysCfgBool enabled,
        NISysCfgBool trusted
);

NISYSCFGCFUNC NISysCfgRemoveSoftwareFeed(
        NISysCfgSessionHandle sessionHandle,
        const char *feedName
);

////////////////////////////////////////////////////////////////////////////////
// System Configuration enumerators and utility functions
////////////////////////////////////////////////////////////////////////////////

NISYSCFGCFUNC NISysCfgChangeAdministratorPassword(
        NISysCfgSessionHandle sessionHandle,
        const char *newPassword
);

NISYSCFGCFUNC NISysCfgExportConfiguration(
        NISysCfgSessionHandle sessionHandle,
        const char *destinationFile,
        const char *expertNames,               // NULL or "" => all experts
        NISysCfgBool overwriteIfExists
);

// Caller should free detailedResult using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgImportConfiguration(
        NISysCfgSessionHandle sessionHandle,
        const char *sourceFile,
        const char *expertNames,               // NULL or "" => all experts
        NISysCfgImportMode importMode,
        char **detailedResult
);

NISYSCFGCFUNC NISysCfgGenerateMAXReport(
        NISysCfgSessionHandle sessionHandle,
        const char *outputFilename,
        NISysCfgReportType reportType,
        NISysCfgBool overwriteIfExists
);

NISYSCFGCFUNC NISysCfgCreateComponentsEnum(
        NISysCfgEnumSoftwareComponentHandle *componentEnumHandle
);

NISYSCFGCFUNC NISysCfgAddComponentToEnum(
        NISysCfgEnumSoftwareComponentHandle componentEnumHandle,
        const char *ID,
        const char *version,
        NISysCfgVersionSelectionMode mode
);

NISYSCFGCFUNC NISysCfgFreeDetailedString(
        char str[]
);

NISYSCFGCFUNC NISysCfgNextResource(
        NISysCfgSessionHandle sessionHandle,
        NISysCfgEnumResourceHandle resourceEnumHandle,
        NISysCfgResourceHandle *resourceHandle
);

NISYSCFGCFUNC NISysCfgNextSystemInfo(
        NISysCfgEnumSystemHandle systemEnumHandle,
        char system[]
);

NISYSCFGCFUNC NISysCfgNextExpertInfo(
        NISysCfgEnumExpertHandle expertEnumHandle,
        char expertName[],
        char displayName[],
        char version[]
);

// Caller should free detailedDescription using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgNextComponentInfo(
        NISysCfgEnumSoftwareComponentHandle componentEnumHandle,
        char ID[],
        char version[],
        char title[],
        NISysCfgComponentType *itemType,
        char **detailedDescription
);

NISYSCFGCFUNC NISysCfgNextSoftwareSet(
        NISysCfgEnumSoftwareSetHandle setEnumHandle,
        NISysCfgSoftwareSetHandle *setHandle
);

// Caller should free detailedDescription using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgGetSoftwareSetInfo(
        NISysCfgSoftwareSetHandle setHandle,
        NISysCfgIncludeComponentTypes itemTypes,
        NISysCfgBool includeAddOnDeps,
        char ID[],
        char version[],
        char title[],
        NISysCfgComponentType *setType,
        char **detailedDescription,
        NISysCfgEnumSoftwareComponentHandle *addOnEnumHandle,
        NISysCfgEnumSoftwareComponentHandle *itemEnumHandle
);

// Caller should free dependerDetailedDescription and dependeeDetailedDescription using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgNextDependencyInfo(
        NISysCfgEnumDependencyHandle dependencyEnumHandle,
        char dependerID[],
        char dependerVersion[],
        char dependerTitle[],
        char **dependerDetailedDescription,
        char dependeeID[],
        char dependeeVersion[],
        char dependeeTitle[],
        char **dependeeDetailedDescription
);

NISYSCFGCFUNC NISysCfgNextSoftwareFeed(
        NISysCfgEnumSoftwareFeedHandle feedEnumHandle,
        char feedName[],
        char uri[],
        NISysCfgBool *enabled,
        NISysCfgBool *trusted
);

NISYSCFGCFUNC NISysCfgResetEnumeratorGetCount(
        void *enumHandle,
        unsigned int *count
);

// Helper method to get the status string for a given status code.
// Caller should free detailedResult using NISysCfgFreeDetailedString.
NISYSCFGCFUNC NISysCfgGetStatusDescription(
        NISysCfgSessionHandle sessionHandle,              // Can be NULL
        NISysCfgStatus status,
        char **detailedDescription
);

NISYSCFGCFUNC NISysCfgTimestampFromValues(
        NISysCfgUInt64 secondsSinceEpoch1970,
        double fractionalSeconds,
        NISysCfgTimestampUTC *timestamp
);

NISYSCFGCFUNC NISysCfgValuesFromTimestamp(
        const NISysCfgTimestampUTC *timestamp,
        NISysCfgUInt64 *secondsSinceEpoch1970,
        double *fractionalSeconds
);

#ifdef __cplusplus
}
#endif

#endif
