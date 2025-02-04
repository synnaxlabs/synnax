// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <string>

namespace ni {
// Enumerated types for Category and Action Required
enum class Category {
    Configuration,
    Communication,
    Hardware,
    Software,
    Other
};

enum class ActionRequired {
    Warning,
    ImmediateAction,
    Stall
};

// Struct to hold error information
struct ErrorInfo {
    Category category;
    ActionRequired actionRequired;
    std::string description;
};

// Map of error codes to error information
const std::map<int, ErrorInfo> errors = {
    {
        -209886,
        {
            Category::Configuration, ActionRequired::Warning,
            "Setting the specified property does not allow multiple task configuration of the sample clock."
        }
    },
    {
        -209885,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "A property is set to a value that is not supported by multiple task configuration of the sample clock."
        }
    },
    {
        -209884,
        {
            Category::Hardware, ActionRequired::Warning,
            "A common sample rate for all specified tasks could not be found. If your application contains both fast- and slow-sampled devices, consider enabling allow repeated samples."
        }
    },
    {
        -209883,
        {
            Category::Other, ActionRequired::Warning,
            "A common sample rate for all specified tasks could not be found."
        }
    },
    {
        -209882,
        {
            Category::Hardware, ActionRequired::Warning,
            "The task contains multiple devices or chassis. Multiple task configuration only supports tasks with a single device or chassis."
        }
    },
    {
        -209881,
        {
            Category::Other, ActionRequired::Warning,
            "The task does not support configuring its sample clock rate via multiple task configuration."
        }
    },
    {
        -209880,
        {
            Category::Other, ActionRequired::Warning,
            "Specified operation cannot be performed because the debug session's target task may have a registered timing source. The debug session does not support debugging tasks with registered timing sources."
        }
    },
    {
        -209879,
        {
            Category::Other, ActionRequired::Warning,
            "Specified operation cannot be performed because the debug session's target task has logging enabled. The debug session does not support debugging tasks with logging enabled. Turn off logging for the target task and try again."
        }
    },
    {
        -209878,
        {
            Category::Other, ActionRequired::Warning,
            "Specified operation cannot be performed because the debug session's target task has registered events. The debug session does not support debugging tasks with registered events. Unregister the events for the target task and try again."
        }
    },
    {
        -209877,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified operation cannot be performed because the debug session's target task is invalid or does not exist."
        }
    },
    {
        -209876,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Requested function is not supported by the device."
        }
    },
    {
        -209875,
        {
            Category::Other, ActionRequired::Warning,
            "Multiple tasks were found that match the debug session target settings.The debug session does not support debugging more than one task at a time. Change the target settings to match a specific task and try again."
        }
    },
    {
        -209874,
        {
            Category::Other, ActionRequired::Warning,
            "A task was not found that matches the debug session target settings."
        }
    },
    {
        -209873,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified operation cannot be performed because it is not supported in a debug session."
        }
    },
    {
        -209872,
        {
            Category::Other, ActionRequired::Warning,
            "Specified operation cannot be performed because it is not permitted in monitor mode.Make sure the debug session is in control mode before requesting this operation."
        }
    },
    {
        -209871,
        {
            Category::Configuration, ActionRequired::Warning,
            "Attempt to get property failed because your device contains multiple banks, and the property has different values for different banks. Get this property one bank at a time by changing the active device name to specify each individual bank."
        }
    },
    {
        -209870,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Timing source creation has failed because another timing source has already been registered for this task."
        }
    },
    {
        -209869,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Requested value is not a supported value for this hardware revision. However, it may be supported on later revisions. Visit ni.com/info, and enter the info code \"fielddaqfilter\" for additional information."
        }
    },
    {
        -209868,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "To set the sensor power supply voltage level, you must specify the sensor power configuration."
        }
    },
    {
        -209867,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "To enable the sensor power supply, you must specify the sensor power voltage level."
        }
    },
    {
        -209866,
        {
            Category::Hardware, ActionRequired::Warning,
            "The hardware cannot acquire data from the configured scanlist. Please enter the info code that follows at ni.com/info for additional information."
        }
    },
    {
        -209865,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "All internal routes have been reserved. If you are using time-based synchronization, please refer to info code \"TimeTriggerLimits\" at ni.com/info for additional information."
        }
    },
    {
        -209864,
        {
            Category::Other, ActionRequired::Warning,
            "The requested Reset Delay cannot be set based on your task configuration. If you are using C Series modules from different chassis in the same task, it is possible that the sample rate and module types cannot be synchronized effectively."
        }
    },
    {
        -209863,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Exceeded total number of time triggers available. Try disabling time triggers that are enabled on one or more DAQ tasks in your system. If no more time triggers can be disabled, try disabling other features that require internal routing resources. Please refer to info code \"TimeTriggerLimits\" at ni.com/info for more information."
        }
    },
    {
        -209862,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Exceeded total number of timestamps available. Try disabling timestamps that are user-defined or enabled by default on one or more DAQ tasks in your system."
        }
    },
    {
        -209861,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "One or more devices in your task are not running the IEEE 802.1AS or IEEE 1588 synchronization protocol. Use the NI-Sync LabVIEW API to enable 802.1AS or 1588. Refer to the NI-Sync help or LabVIEW examples for more information."
        }
    },
    {
        -209860,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "One or more devices could not be added to your task because they have existing coherency requirements that conflict with the new requirements of your current task. Stop the task(s) that use devices in this task and try running it again."
        }
    },
    {
        -209859,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "The devices in your task must be synchronized to one another using the IEEE 802.1AS or IEEE 1588 synchronization protocol. Connect the devices, either directly or via an 802.1AS or 1588-capable switch."
        }
    },
    {
        -209858,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "FieldDAQ bank name specified is invalid. FieldDAQ bank names are of the form <FieldDAQ name>-Bank<Bank number>. For example, FieldDAQ1-Bank1."
        }
    },
    {
        -209857,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Task contains physical channels on one or more devices that do not support the DAQmx hardware-timed single-point sample mode."
        }
    },
    {
        -209856,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Bank type in the source storage does not match the bank type in the destination."
        }
    },
    {
        -209855,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "FieldDAQ device does not support the specified number of banks. The bank number specified may be too large. Change the bank number to be a valid bank number."
        }
    },
    {
        -209854,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "The simulated FieldDAQ bank is not supported on this simulated FieldDAQ."
        }
    },
    {
        -209853,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "The IsSimulated flags for FieldDAQ and banks must match. Change the IsSimulated flags in the import file so that they match."
        }
    },
    {
        -209852,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "The specified device is no longer supported within the NI-DAQmx API."
        }
    },
    {
        -209851,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "The requested Sample Timing Engine does not support Use Only Onboard Memory."
        }
    },
    {
        -209850,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Task name specified conflicts with an existing task name in another project."
        }
    },
    {
        -209849,
        {
            Category::Other, ActionRequired::Warning,
            "A time Arm Start Trigger and time Start Trigger are configured. Ensure the Arm Start Trigger comes first."
        }
    },
    {
        -209848,
        {
            Category::Hardware, ActionRequired::Warning,
            "The time trigger cannot be configured because your device has lost synchronization lock to the grand master."
        }
    },
    {
        -209847,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Specified value is not supported for this property. Supported values are -1 (infinite window) or non-negative numbers up to 15.768000e9."
        }
    },
    {
        -209846,
        {
            Category::Configuration, ActionRequired::Warning,
            "The property cannot be queried before or during a data acquisition. You must explicitly commit, run, and stop your task before attempting to read this property."
        }
    },
    {
        -209845,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "The Sample Clock Timebase property you have requested is not supported by this task because it contains a reference clock device. Use the Reference Clock properties instead."
        }
    },
    {
        -209844,
        {
            Category::Other, ActionRequired::Warning,
            "Attempted to query a timestamp before it is available. Use the DAQmx Wait for Valid Timestamp VI/function to wait until the desired timestamp is available."
        }
    },
    {
        -209843,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Time operations are not supported by this device or task type."
        }
    },
    {
        -209842,
        {
            Category::Other, ActionRequired::Warning,
            "Timestamp requested is not enabled for this task."
        }
    },
    {
        -209841,
        {
            Category::Other, ActionRequired::Warning,
            "A time sync pulse and time start trigger are configured. Ensure the sync pulse comes first and the difference between them is larger than the task sync time."
        }
    },
    {
        -209840,
        {
            Category::Other, ActionRequired::Warning,
            "The configured time trigger is in the past."
        }
    },
    {
        -209839,
        {
            Category::Hardware, ActionRequired::Warning,
            "A time trigger configured is too far in the future. Use I/O Device Timescale instead of Host Timescale, configure the trigger at a point in the future closer to the trigger time, or schedule the trigger closer in time to the current Host Timescale time."
        }
    },
    {
        -209838,
        {
            Category::Configuration, ActionRequired::Warning,
            "Synchronization lock was lost during operation. If the occasional loss of synchronization is acceptable, change the Synchronization Unlock Behavior property to ignore sync loss. Otherwise, go to ni.com for more information about sync loss management."
        }
    },
    {
        -209837,
        {
            Category::Other, ActionRequired::Warning,
            "The timescales for time start trigger and time sync pulse must be the same."
        }
    },
    {
        -209836,
        {
            Category::Hardware, ActionRequired::Warning,
            "The devices in your task cannot be synchronized. This may be because there are no available synchronization mechanisms between the devices."
        }
    },
    {
        -209835,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "The specified property requires all associated channels to use the same range as specified by the Maximum and Minimum properties. Try setting the same Maximum and Minimum Values on each channel or change the conflicting property."
        }
    },
    {
        -209834,
        {
            Category::Configuration, ActionRequired::Warning,
            "Sample rate exceeds the maximum sample rate for the number of channels and property values specified. Reduce the sample rate or number of channels. Changing the specified property values may also increase the maximum possible sample rate."
        }
    },
    {
        -209833,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "DAQmx Wait for Valid Timestamp is not supported by one or more devices in this task."
        }
    },
    {
        -209832,
        {
            Category::Other, ActionRequired::Warning,
            "Specified Trigger Window has elapsed without a trigger being detected."
        }
    },
    {
        -209831,
        {
            Category::Hardware, ActionRequired::Warning,
            "Not all devices in the task support the specified Trigger configuration."
        }
    },
    {
        -209830,
        {
            Category::Hardware, ActionRequired::Warning,
            "Not all devices in the task support the specified Data Transfer Mechanism."
        }
    },
    {
        -209829,
        {
            Category::Hardware, ActionRequired::Warning,
            "Onboard device memory overflow. Because of system and/or bus-bandwidth limitations, the driver could not read data from the device fast enough to keep up with the data transfer throughput. Reduce the maximum data transfer rate of the device. You can also increase bandwidth by reducing the number of programs your computer is executing."
        }
    },
    {
        -209828,
        {
            Category::Hardware, ActionRequired::Warning,
            "All channels used for the analog trigger source must be on the same device. Remove trigger configurations from the task until all of the sources are channels from the same device."
        }
    },
    {
        -209827,
        {
            Category::Hardware, ActionRequired::Warning,
            "The device does not support using more than one trigger type at a time with the specified task configuration. Configure the device to use only one trigger type or to use a configuration that supports using multiple trigger types."
        }
    },
    {
        -209826,
        {
            Category::Other, ActionRequired::Warning,
            "Multiple source triggering requires the same number of values for all user-specified trigger configurations."
        }
    },
    {
        -209825,
        {
            Category::Configuration, ActionRequired::Warning,
            "The specified AO DAC range must be consistent with all reserved tasks on this device. Ensure that all reserved tasks on this device use the same value for the AO.DAC.Ref.Val channel attribute/property."
        }
    },
    {
        -209824,
        {
            Category::Other, ActionRequired::Warning,
            "The waveforms passed to DAQmx Write claim to be sampled at inconsistent rates. Check to make sure the data is sampled at the correct rate and either re-sample the data or correct the dt element of the waveforms."
        }
    },
    {
        -209823,
        {
            Category::Other, ActionRequired::Warning,
            "The function called is no longer supported by DAQmx."
        }
    },
    {
        -209822,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Negative duration values other than -1 are not supported. -1 indicates to read all samples."
        }
    },
    {
        -209821,
        {
            Category::Other, ActionRequired::Warning,
            "No samples were acquired within the specified duration. Increase the duration or sample rate so that at least one sample is acquired."
        }
    },
    {
        -209820,
        {
            Category::Other, ActionRequired::Warning,
            "The specified duration is too long. Specify a shorter duration."
        }
    },
    {
        -209819,
        {
            Category::Other, ActionRequired::Warning,
            "Duration-based reads are supported only in sample clock timing mode."
        }
    },
    {
        -209818,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "The selected LED state is invalid."
        }
    },
    {
        -209817,
        {
            Category::Hardware, ActionRequired::Warning,
            "A device in your watchdog task does not support different output state types. Set all channels to the same output state type or remove the channels from the task."
        }
    },
    {
        -209816,
        {
            Category::Hardware, ActionRequired::Warning,
            "Self-test of the device has failed because the measured power supply voltage is outside of tolerance. Please contact National Instruments technical support."
        }
    },
    {
        -209815,
        {
            Category::Hardware, ActionRequired::Warning,
            "DAQmx Write does not support multiple samples in Hardware-Timed Single-Point tasks. Specify a single sample."
        }
    },
    {
        -209814,
        {
            Category::Configuration, ActionRequired::Warning,
            "You cannot use onboard regeneration for a task with this many channels. Reduce the number of channels in the task, use fewer modules with more the 16-bits of precision, or set the AO.UseOnlyOnBrdMem attribute/property to false."
        }
    },
    {
        -209813,
        {
            Category::Hardware, ActionRequired::Warning,
            "To create watchdog task on this device, you must specify expiration states for all lines. Specify states for the missing channels."
        }
    },
    {
        -209812,
        {
            Category::Hardware, ActionRequired::Warning,
            "The selected shunt source option for the shunt calibration is not valid for this device."
        }
    },
    {
        -209811,
        {
            Category::Hardware, ActionRequired::Warning,
            "The selected shunt select option for the shunt calibration is not valid for this device."
        }
    },
    {
        -209810,
        {
            Category::Hardware, ActionRequired::Warning,
            "This device does not support shunt calibration for the requested configuration of shunt select and shunt source. Refer to the shunt calibration documentation for your application development environment for more information."
        }
    },
    {
        -209809,
        {
            Category::Other, ActionRequired::Warning,
            "The selected channels do not support buffered operations when alone in a task. Add an additional channel which supports buffered operations to the task or reconfigure the task to use On-Demand timing."
        }
    },
    {
        -209808,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "The specified feature is not supported on the attached accessory. Refer to your accessory documentation for accessories that support the requested feature."
        }
    },
    {
        -209807,
        {
            Category::Other, ActionRequired::Warning,
            "The threshold voltage value must be consistent when using the same terminal with RSE terminal configuration as a source for multiple inputs. For example, do not set 2.0 V on PFI0 for one input, and 3.0 V on PFI0 for another input."
        }
    },
    {
        -209806,
        {
            Category::Software, ActionRequired::Warning,
            "NI-DAQmx is not installed on the target system, is incompatible, or the installation is corrupt. Install or repair the driver software."
        }
    },
    {
        -209804,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Your device has lost synchronization lock to the grand master. Any timestamp values for the task may be invalid, but the task was not stopped because the SyncUnlockBehavior property is set to \"Ignore Lost Sync Lock.\" To query the status of your device, read either the ReadSyncUnlockedChansExist or WriteSyncUnlockedChansExist property."
        }
    },
    {
        -209803,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "The Onboard Memory Empty value of the AO.DataXferReqCond attribute/property is not supported on this device. The value of this attribute/property is coerced to a different value after Output.BufSize is set. You can emulate the behavior of the Onboard Memory Empty value by setting Output.BufSize to a lower value. The minimum value for this attribute/property is 2."
        }
    },
    {
        -209802,
        {
            Category::Other, ActionRequired::Warning,
            "DAQmx Wait for Next Sample Clock detected one or more missed sample clocks since the last call to Wait for Next Sample Clock which indicates that your program is not keeping up with the sample clock. To remove this warning, slow down the sample clock, or else change your application so that it can keep up with the sample clock."
        }
    },
    {
        -209801,
        {
            Category::Hardware, ActionRequired::Warning,
            "DAQmx Write did not complete before the arrival of the next sample clock which indicates that your program is not keeping up with the hardware clock. To remove this warning, slow down the hardware clock, or else change your application so that it can keep up with the hardware clock."
        }
    },
    {
        -209800,
        {
            Category::Hardware, ActionRequired::Warning,
            "DAQmx Read did not complete before the arrival of the next sample clock or change detection event, which indicates that your program is not keeping up with the hardware clock or the external change event. For tasks using sample clock timing, slow down the hardware clock or else change your application so that it can keep up with the hardware clock. For tasks using change detection timing, decrease the frequency of your event or else change your application so that it can keep up with the change event."
        }
    },
    {
        -201510,
        {
            Category::Other, ActionRequired::Warning,
            "The digital filtering properties must be consistent when using the same terminal as a source for multiple inputs. For example, digital filtering can either be enabled with the same minimum pulse width configuration whenever PFI0 is used, or disabled for all cases."
        }
    },
    {
        -201509,
        {
            Category::Other, ActionRequired::Warning,
            "The logic level behavior must be consistent when using the same terminal as a source for multiple inputs. For example, do not enable the pull-up on PFI0 for one input, and set it to none on PFI0 for another input."
        }
    },
    {
        -201508,
        {
            Category::Other, ActionRequired::Warning,
            "The terminal configuration must be consistent when using the same terminal as a source for multiple inputs. For example, do not set PFI0 to differential for one input, and PFI0 to RSE for another input."
        }
    },
    {
        -201507,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "A fatal hardware clocking error has occurred. The device is unusable until you restart it. If the problem persists, please contact National Instruments Technical Support."
        }
    },
    {
        -201506,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "A hardware clocking error has occurred. Try the operation again. If the problem persists, please contact National Instruments Technical Support."
        }
    },
    {
        -201505,
        {
            Category::Hardware, ActionRequired::Warning,
            "The chassis requires at least one DAQ device directly cabled to it. Edit the chassis and assign a device."
        }
    },
    {
        -201504,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "A fatal hardware clocking error has occurred. The device is unusable until you restart it. If you are using an external reference clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -201503,
        {
            Category::Software, ActionRequired::Warning,
            "The AI timing engine cannot be used for counter tasks. You can select a different timing engine or let the driver automatically select one."
        }
    },
    {
        -200078,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Using USB DAQ devices on Windows XP might result in corrupt data or other errors. Visit ni.com/info and enter info code \"WindowsXPUSBHotfix\" to obtain a patch."
        }
    },
    {
        -200077,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXIe-5611 is not configured properly and needs to be associated with an AWG and LO. Right-click on the NI PXIe-5611 and select 'Configure' to associate the AWG and LO. For more information, refer to NI-RFSG documentation, RF Signal Generators Getting Started Guide."
        }
    },
    {
        -200076,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXIe-5611 is not configured and needs to be associated with an AWG and LO. Right-click on the NI PXIe-5611 and select 'Configure' to associate the AWG and LO. For more information, refer to NI-RFSG documentation, RF Signal Generators Getting Started Guide."
        }
    },
    {
        -200075,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXI-5610 is not configured properly and needs to be associated with an AWG. Right-click on the NI PXI-5610 and select 'Configure' to associate the AWG. For more information, refer to NI-RFSG documentation, RF Signal Generators Getting Started Guide."
        }
    },
    {
        -200074,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXI-5610 is not configured and needs to be associated with an AWG. Right-click on the NI PXI-5610 and select 'Configure' to associate the AWG. For more information, refer to NI-RFSG documentation, RF Signal Generators Getting Started Guide."
        }
    },
    {
        -200073,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXI-5665 is not configured properly. Right-click on the NI PXIe-5606 and select 'Configure' to associate the digitizer and LO."
        }
    },
    {
        -200072,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXIe-5606 is not associated with a digitizer or an LO. Right-click on the NI PXIe-5606 and select 'Configure' to associate the digitizer and LO."
        }
    },
    {
        -200071,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXIe-5667 (7 GHz) is not configured properly. Right-click on the NI PXIe-5605 and select 'Configure' to associate the digitizer and LO. You can also associate the NI PXIe-5605 with additional IF conditioning and RF conditioning modules."
        }
    },
    {
        -200070,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXI-5665 (14 GHz) is not configured properly. Right-click on the NI PXIe-5605 and select 'Configure' to associate the digitizer and LO. You can also associate the NI PXIe-5605 with additional IF conditioning and RF conditioning modules."
        }
    },
    {
        -200069,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXIe-5605 is not associated with a digitizer or an LO. Right-click on the NI PXIe-5605 and select 'Configure' to associate the digitizer and LO. You can also associate the NI PXIe-5605 with additional IF conditioning and RF conditioning modules."
        }
    },
    {
        -200068,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXIe-5667 (3.6 GHz) is not configured properly. Right-click on the NI PXIe-5603 and select 'Configure' to associate the digitizer and LO. You can also associate the NI PXIe-5603 with additional IF conditioning and RF conditioning modules."
        }
    },
    {
        -200067,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXIe-5665 (3.6 GHz) is not configured properly. Right-click on the NI PXIe-5603 and select 'Configure' to associate the digitizer and LO. You can also associate the NI PXIe-5603 with additional IF conditioning and RF conditioning modules."
        }
    },
    {
        -200066,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXIe-5603 is not associated with a digitizer or an LO. Right-click on the NI PXIe-5603 and select 'Configure' to associate the digitizer and LO. You can also associate the NI PXIe-5603 with additional IF conditioning and RF conditioning modules."
        }
    },
    {
        -200065,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXIe-5663E is not configured properly. Right-click on the NI PXIe-5601 and select 'Configure' to associate the digitizer and LO."
        }
    },
    {
        -200064,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXIe-5663 is not configured properly. Right-click on the NI PXIe-5601 and select 'Configure' to associate the digitizer and LO."
        }
    },
    {
        -200063,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXIe-5601 is not associated with a digitizer or an LO. Right-click on the NI PXIe-5601 and select 'Configure' to associate the digitizer and LO."
        }
    },
    {
        -200062,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXI-5661 is not configured properly. Right-click on the NI PXI-5600 and select 'Configure' to associate the digitizer."
        }
    },
    {
        -200061,
        {
            Category::Other, ActionRequired::Warning,
            "The NI PXI-5600 is not associated with a digitizer. Right-click on the NI PXI-5600 and select 'Configure' to associate the digitizer."
        }
    },
    {
        -200060,
        {
            Category::Other, ActionRequired::Warning,
            "The same data is being read repetitively."
        }
    },
    {
        -200059,
        {
            Category::Other, ActionRequired::Warning,
            "The selected ports are not connected, so there is nothing to disconnect."
        }
    },
    {
        -200058,
        {
            Category::Communication, ActionRequired::ImmediateAction,
            "The connection count stored on the EEPROM is invalid. Write a value to Accessory Connection Count to fix the problem. Contact National Instruments if the problem persists."
        }
    },
    {
        -200057,
        {
            Category::Hardware, ActionRequired::Warning,
            "The network device already exists in the system."
        }
    },
    {
        -200056,
        {
            Category::Communication, ActionRequired::Warning,
            "The connection count of the attached accessory has exceeded the recommended limit. Contact National Instruments if the accessory appears to be functioning improperly."
        }
    },
    {
        -200055,
        {
            Category::Hardware, ActionRequired::Warning,
            "Power-up state section of the device EEPROM appears to be corrupt. Reconfigure the digital power-up states and perform a self-calibration."
        }
    },
    {
        -200054,
        {
            Category::Hardware, ActionRequired::Warning,
            "EEPROM of the device appears to be corrupt. Contact National Instruments if the device appears to be functioning improperly."
        }
    },
    {
        -200053,
        {
            Category::Hardware, ActionRequired::Warning,
            "Sample Rate specified may exceed device capabilities for some devices in the task. Specify a slower sample rate, decrease the number of channels, or use a separate task for some of the devices in the task."
        }
    },
    {
        -200052,
        {
            Category::Other, ActionRequired::Warning,
            "Buffer size specified is not evenly divisible by 8 times the sector size. For optimal performance, use a buffer size that is a multiple of 8 times the sector size. Refer to the NI-DAQmx Help for more information."
        }
    },
    {
        -200051,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Input voltage limits exceeded. Protection circuity disabled the inputs, however proper voltage levels are now present, and the error state has been cleared."
        }
    },
    {
        -200050,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Output generation aborted by the reverse power protection circuitry of the device. Either the output signal exceeded the output power limit, or power was driven back into the output of the device by an external source. Error state has been cleared."
        }
    },
    {
        -200049,
        {
            Category::Hardware, ActionRequired::Warning,
            "Calibration changed the gain calibration constants only and not the offset calibration constants because the necessary offset calibration data was not available. This device needs a reference signal of 0.0 Volts at gains of 1, 15, 20, and 310 in order to perform an offset calibration."
        }
    },
    {
        -200048,
        {
            Category::Configuration, ActionRequired::Warning,
            "Requested property value exceeds device specification limits. Device performance is not guaranteed. Use values within device specifications, or set the Allow Out of Specification User Settings property to true."
        }
    },
    {
        -200047,
        {
            Category::Hardware, ActionRequired::Warning,
            "Self-calibration section of the EEPROM on the device appears to be corrupt. Perform a self-calibration on the device."
        }
    },
    {
        -200046,
        {
            Category::Hardware, ActionRequired::Warning,
            "External calibration section of the EEPROM on the device appears to be corrupt. Perform an external calibration on the device."
        }
    },
    {
        -200045,
        {
            Category::Hardware, ActionRequired::Warning,
            "EEPROM of the device appears to be corrupt. Contact National Instruments if the device appears to be functioning improperly."
        }
    },
    {
        -200044,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid enumeration value was encountered during export. The exported file will require modification in order to successfully import."
        }
    },
    {
        -200043,
        {
            Category::Configuration, ActionRequired::Warning,
            "Date specified by the Channel Calibration Expiration Date property has expired. The channel calibration is applied in spite of this because the Apply Calibration if Expired property was set to true. To eliminate this warning, update the channel calibration, including the Expiration Date."
        }
    },
    {
        -200042,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Calibration constants stored in the EEPROM produced an invalid value for one or more analog output channels. The device will continue to function, but the accuracy of the generated signals may be compromised. An incorrect calibration might have been performed, or the calibration data in the EEPROM might have been corrupted. Perform an external calibration, perform a self-calibration, or contact National Instruments Technical Support."
        }
    },
    {
        -200041,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Calibration constants stored in EEPROM produced an invalid value for one or more analog input channels. The device will continue to function, but the accuracy of the measurements may be compromised. An incorrect calibration might have been performed, or the calibration data in the EEPROM might have been corrupted. Perform an external calibration, perform a self-calibration, or contact National Instruments Technical Support."
        }
    },
    {
        -200040,
        {
            Category::Other, ActionRequired::Warning,
            "Sample clock rate specified is so high that it violates the settling time requirements for the generation. Reduce the sample clock rate, or the accuracy of the generated signal might be compromised."
        }
    },
    {
        -200039,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Data may be invalid because the settling time of the enabled filter exceeds the period between two conversions on the analog-to-digital converter (ADC) for a task with more than one channel. Disable the filter by setting AI Lowpass Enable to false, acquire data from only one channel in the task, or increase the time between two ADC conversions by reducing the Sample Clock Rate, the Sample Clock Delay, and/or the number of channels in the task."
        }
    },
    {
        -200038,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Data may be invalid because the settling time of the enabled filter exceeds the period between two conversions on the analog-to-digital converter (ADC) for a task with more than one channel. Disable the filter by setting AI Lowpass Enable to false, increase the time between two ADC conversions by reducing the AI Convert Rate, or acquire data from only one channel in the task."
        }
    },
    {
        -200037,
        {
            Category::Other, ActionRequired::Warning,
            "Settings requested through a previous DAQmx Write were overwritten before they could be applied. This occurred either because DAQmx Write was invoked more than once between two consecutive sample clocks or because the frequency of the generated pulse train is lower than the sample clock rate (it takes more than one sample clock period to generate one period of the pulse train). The settings requested by the most recent DAQmx Write will be applied on the next sample clock. To avoid this warning, make sure that DAQmx Write is invoked exactly once between two consecutive sample clocks and that the frequency of the generated pulse train is higher than the sample clock rate."
        }
    },
    {
        -200036,
        {
            Category::Hardware, ActionRequired::Warning,
            "Requested Sample Clock Rate is higher than the maximum supported per device specifications. Reduce the Sample Clock Rate or use a device which supports the requested Sample Clock Rate."
        }
    },
    {
        -200035,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Clock rate specified is less than the minimum conversion rate of the ADC. Your data may be invalid."
        }
    },
    {
        -200034,
        {
            Category::Hardware, ActionRequired::Warning,
            "A sensor on the device detected a temperature approaching the device's maximum recommended operating temperature. The device will shut down if its temperature exceeds the maximum recommended operating temperature. To avoid the shutdown, ensure the device temperature does not get above the maximum recommended operating temperature. You can often prevent the shutdown by periodically cleaning fan filters. Refer to user documentation for more information."
        }
    },
    {
        -200033,
        {
            Category::Other, ActionRequired::Warning,
            "DAQmx Write was invoked more than once between two consecutive sample clocks. Only the last DAQmx Write took effect. To eliminate this warning, invoke DAQmx Write only once between two consecutive sample clocks."
        }
    },
    {
        -200032,
        {
            Category::Other, ActionRequired::Warning,
            "Output gain was coerced to the nearest acceptable value, because the original value was too high given the RF Frequency."
        }
    },
    {
        -200031,
        {
            Category::Other, ActionRequired::Warning,
            "Output gain was coerced to the nearest acceptable value, because the original value was too low given the RF Frequency."
        }
    },
    {
        -200030,
        {
            Category::Hardware, ActionRequired::Warning,
            "A sensor on the device detected a temperature approaching the device's maximum recommended operating temperature. The device will shut down if its temperature exceeds the maximum recommended operating temperature. To avoid the shutdown, ensure the device temperature does not get above the maximum recommended operating temperature. You can often prevent the shutdown by periodically cleaning fan filters. Refer to user documentation for more information."
        }
    },
    {
        -200029,
        {
            Category::Other, ActionRequired::Warning,
            "RIS acquisition was completed, but some of the bins were not filled with a sufficient number of samples to perform the requested RIS averaging. Data for those bins was computed from the available samples. Consider increasing the timeout for the operation. Refer to documentation for details about RIS acquisitions."
        }
    },
    {
        -200028,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Some of the last samples acquired during the finite DMA acquisition are possibly invalid due to counter limitations. Get the Number of Possibly Invalid Samples property to see how many samples might be invalid. Use continuous sample mode for DMA acquisitions with this type of counter."
        }
    },
    {
        -200027,
        {
            Category::Other, ActionRequired::Warning,
            "Sample clock rate has been coerced to the minimum supported value because the specified value was too low. For lower sample clock rates, use an external sample clock or an external sample clock timebase."
        }
    },
    {
        -200026,
        {
            Category::Other, ActionRequired::Warning,
            "Requested string could not fit into the given buffer. Only the first part of the string was copied into the buffer. To allow for the terminating NULL character, the number of characters copied into the buffer is equal to the size of the buffer minus one. Call the function twice. Call the function initially to determine the string size. Use the second function call to get the full string value. In the first function call, pass NULL for the buffer and zero for the buffer size. The positive return value of the function is the string size (which includes the terminating NULL). Use this value to allocate a buffer of sufficient size, then use this buffer in the second function call."
        }
    },
    {
        -200025,
        {
            Category::Other, ActionRequired::Warning,
            "User-defined information to be stored in the EEPROM is too long. Only the leading portion was saved. Refer to documentation for information about the maximum length allowed for user-defined information."
        }
    },
    {
        -200024,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "One of more of the properties saved with a later version of NI-DAQ are not supported by the installed version of NI-DAQ and are ignored. Upgrade the installed version of NI-DAQ to a version compatible with the version used when saving the properties to take advantage of all the saved properties."
        }
    },
    {
        -200022,
        {
            Category::Software, ActionRequired::Warning,
            "Attempted writing a sample value that was too small. The driver automatically coerced it to the minimum supported value."
        }
    },
    {
        -200021,
        {
            Category::Software, ActionRequired::Warning,
            "Attempted writing a sample value that was too large. The driver automatically coerced it to the maximum supported value."
        }
    },
    {
        -200020,
        {
            Category::Other, ActionRequired::Warning,
            "Requested number of pretrigger samples per channel could not be configured, so it was coerced to the minimum supported value."
        }
    },
    {
        -200019,
        {
            Category::Other, ActionRequired::Warning,
            "Requested read offset could not be configured, so the offset was coerced to the minimum supported value."
        }
    },
    {
        -200018,
        {
            Category::Other, ActionRequired::Warning,
            "Specified convert rate is too low to be generated using the onboard AI convert clock with the given timebase. The rate was coerced to the slowest possible convert rate. For slower rates, you must use an external convert clock or an external convert clock timebase."
        }
    },
    {
        -200017,
        {
            Category::Other, ActionRequired::Warning,
            "Specified sample rate is lower than the lowest rate that can be generated using the onboard clock. The rate was coerced to the slowest possible sample rate. For slower rates, use an external sample clock or an external sample clock timebase."
        }
    },
    {
        -200016,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "An attempt has been made to query the date/time of the last self calibration of a device that has never been self-calibrated using the NI-DAQmx API, so the date/time is invalid. Self-calibrate the board using the NI-DAQmx API."
        }
    },
    {
        -200015,
        {
            Category::Hardware, ActionRequired::Warning,
            "While writing to the buffer during a regeneration, the actual data generated might have alternated between old data and new data. That is, while the driver was replacing the old pattern in the buffer with the new pattern, the device might have generated a portion of new data, then a portion of old data, and then a portion of new data again. Reduce the sample rate, use a larger buffer, or refer to documentation about DAQmx Write for information about other ways to avoid this warning."
        }
    },
    {
        -200014,
        {
            Category::Other, ActionRequired::Warning,
            "The combination of sample rate and buffer size settings could result in a large number of interrupts, causing the system to hang. Decrease your sample rate, or increase your buffer size. For acquisitions without a reference trigger, you can disallow buffer overwrites. For generations, you can disable the regeneration of old samples."
        }
    },
    {
        -200013,
        {
            Category::Other, ActionRequired::Warning,
            "User-defined information string entered exceeds the maximum allowable string length. The string will be truncated to its maximum allowable length."
        }
    },
    {
        -200012,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Clock rate specified exceeds the maximum conversion rate of the ADC. ADC overrun errors are likely."
        }
    },
    {
        -200011,
        {
            Category::Other, ActionRequired::Warning,
            "Clock rate specified is so high that it violates the settling time requirements for the acquisition. Reduce the clock rate, or the accuracy of the measurement might be compromised."
        }
    },
    {
        -200010,
        {
            Category::Other, ActionRequired::Warning,
            "Finite acquisition or generation has been stopped before the requested number of samples were acquired or generated."
        }
    },
    {
        -200009,
        {
            Category::Other, ActionRequired::Warning,
            "Counter 1 DMA acquisition started while starting, committing, stopping, or uncommitting an analog output (AO) task. This could cause the counter acquisition to stop. If possible, use counter 0 instead of counter 1. Otherwise, start/commit the AO task before starting the counter 1 DMA acquisition, and stop/uncommit the AO task after stopping the counter 1 DMA acquisition."
        }
    },
    {
        -200008,
        {
            Category::Other, ActionRequired::Warning,
            "Counter 0 DMA acquisition started while starting, committing, stopping, or uncommitting an analog input (AI) task. This could cause the counter acquisition to stop. If possible, use counter 1 instead of counter 0. Otherwise, start/commit the AI task before starting the counter 0 DMA acquisition, and stop/uncommit the AI task after stopping the counter 0 DMA acquisition."
        }
    },
    {
        -200007,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "PLL was unlocked. Your data might be invalid."
        }
    },
    {
        -200005,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "ADC for one or more channels was overloaded. Your data might be invalid."
        }
    },
    {
        -200004,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Input termination resistor for one or more channels was overloaded. Your data might be invalid."
        }
    },
    {
        -200003,
        {
            Category::Other, ActionRequired::Warning,
            "Absolute timestamp counter has rolled over."
        }
    },
    {
        -200003,
        {
            Category::Other, ActionRequired::Warning,
            "Channel cannot be used more than once inside a list of channels. If you need to use the same physical channel more than once inside your list of channels, refer to that physical channel under different names."
        }
    },
    {
        -200004,
        {
            Category::Hardware, ActionRequired::Warning,
            "Data requested has been overwritten in the device memory."
        }
    },
    {
        -200005,
        {
            Category::Other, ActionRequired::Warning,
            "Data requested has not been acquired yet."
        }
    },
    {
        -200006,
        {
            Category::Hardware, ActionRequired::Warning,
            "Record requested has been overwritten in the device memory."
        }
    },
    {
        -200007,
        {
            Category::Other, ActionRequired::Warning,
            "Record requested has not been acquired yet ."
        }
    },
    {
        -200008,
        {Category::Other, ActionRequired::Warning, "Stop trigger has not occurred yet."}
    },
    {
        -200009,
        {
            Category::Other, ActionRequired::Warning,
            "Timestamps have been overwritten. You can no longer read any data."
        }
    },
    {
        -200010,
        {
            Category::Hardware, ActionRequired::Warning,
            "Onboard device memory overflow. Because of system and/or bus-bandwidth limitations, the driver could not read data from the device fast enough to keep up with the device throughput. Reduce your sample rate. If your data transfer method is interrupts, try using DMA or USB Bulk. You can also use a product with more onboard memory or reduce the number of programs your computer is executing concurrently."
        }
    },
    {
        -200011,
        {
            Category::Other, ActionRequired::Warning,
            "Task cannot contain both input and output channels. Either use channels of one direction in a task or make two separate tasks."
        }
    },
    {
        -200012,
        {
            Category::Hardware, ActionRequired::Warning,
            "Specified physical channel does not support digital output. Change the direction of the task, use another terminal, or use another device."
        }
    },
    {
        -200014,
        {
            Category::Other, ActionRequired::Warning,
            "Terminal cannot appear multiple times within a single digital input or output task."
        }
    },
    {
        -200015,
        {
            Category::Communication, ActionRequired::Warning,
            "Communication with SCXI failed. The communication cable to the SCXI hardware might have been disconnected or exposed to excessive noise."
        }
    },
    {
        -200016,
        {
            Category::Hardware, ActionRequired::Warning,
            "Onboard device memory underflow. Because of system and/or bus-bandwidth limitations, the driver could not write data to the device fast enough to keep up with the device output rate. Reduce your sample rate. If your data transfer method is interrupts, try using DMA or USB Bulk. You can also use a product with more onboard memory or reduce the number of programs your computer is executing concurrently."
        }
    },
    {
        -200017,
        {
            Category::Hardware, ActionRequired::Warning,
            "Onboard device memory underflow. Not enough new data has been sampled since the last read or the start of the measurement. Increase the sample rate, increase the timeout value, or decrease the number of samples to read."
        }
    },
    {
        -200018,
        {
            Category::Other, ActionRequired::Warning,
            "DAC conversion attempted before data to be converted was available. Decrease the output frequency to increase the period between DAC conversions, or reduce the size of your output buffer in order to write data more often. If you are using an external clock, check your signal for the presence of noise or glitches."
        }
    },
    {
        -200019,
        {
            Category::Other, ActionRequired::Warning,
            "ADC conversion attempted before the prior conversion was complete. Increase the period between ADC conversions. If you are using an external clock, check your signal for the presence of noise or glitches."
        }
    },
    {
        -200020,
        {
            Category::Hardware, ActionRequired::Warning,
            "Self-test of the device has failed."
        }
    },
    {
        -200022,
        {
            Category::Other, ActionRequired::Warning,
            "Resource requested by this task has already been reserved by a different task."
        }
    },
    {
        -200023,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Script contains an invalid character or symbol. Replace the invalid character with a valid symbol or alphanumeric character."
        }
    },
    {
        -200024,
        {
            Category::Other, ActionRequired::Warning,
            "Valid identifier expected but not found in script. The identifier should specify a valid waveform or script name. Identifiers cannot start with a number."
        }
    },
    {
        -200025,
        {
            Category::Other, ActionRequired::Warning,
            "Script name was expected, but not found in the script."
        }
    },
    {
        -200026,
        {
            Category::Other, ActionRequired::Warning,
            "Waveform name was expected, but not found in the script."
        }
    },
    {
        -200027,
        {
            Category::Other, ActionRequired::Warning,
            "Keyword was expected, but not found in the script."
        }
    },
    {
        -200028,
        {
            Category::Hardware, ActionRequired::Warning,
            "Waveform referenced in the script was not found in onboard memory. Write the waveform to the device before writing the script."
        }
    },
    {
        -200029,
        {
            Category::Other, ActionRequired::Warning,
            "Marker specified in a generate instruction exceeds the waveform boundaries. Change the marker position or positions to fit within the waveform, or increase the size of the waveform."
        }
    },
    {
        -200030,
        {
            Category::Other, ActionRequired::Warning,
            "Subset specified in a generate instruction exceeds the waveform boundaries. Change the subset start offset and/or subset length so the subset fits within the waveform, or increase the size of the waveform."
        }
    },
    {
        -200031,
        {
            Category::Other, ActionRequired::Warning,
            "Marker position specified is not a multiple of alignment quantum."
        }
    },
    {
        -200032,
        {
            Category::Other, ActionRequired::Warning,
            "Subset length specified is not valid. Change the subset length to be longer than zero samples and a multiple of the alignment quantum."
        }
    },
    {
        -200033,
        {
            Category::Other, ActionRequired::Warning,
            "Start offset of the subset is not a multiple of the alignment quantum."
        }
    },
    {
        -200034,
        {
            Category::Other, ActionRequired::Warning,
            "Marker position exceeds the length of the subset."
        }
    },
    {
        -200035,
        {
            Category::Other, ActionRequired::Warning,
            "Repeat loop is contained within too many levels of nested repeat loops. Unroll one of the \"repeat\" loops if possible, or change the script and run it several times. To unroll a loop, remove the \"repeat\" and \"end repeat\" instructions and explicitly replicate the instructions of the removed loop the desired number of times."
        }
    },
    {
        -200036,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Number of iterations specified for a finite \"repeat\" loop is invalid."
        }
    },
    {
        -200037,
        {
            Category::Other, ActionRequired::Warning,
            "Clear trigger instruction cannot be the last instruction of a \"repeat\" loop."
        }
    },
    {
        -200038,
        {
            Category::Other, ActionRequired::Warning,
            "Wait instruction cannot be the last instruction of a \"repeat until\" loop."
        }
    },
    {
        -200039,
        {
            Category::Hardware, ActionRequired::Warning,
            "Routing information associated with your device cannot be found."
        }
    },
    {
        -200040,
        {
            Category::Hardware, ActionRequired::Warning,
            "Source terminal to be routed could not be found on the device. Make sure the terminal name is valid for the specified device. Refer to Measurement & Automation Explorer for valid terminal names."
        }
    },
    {
        -200041,
        {
            Category::Hardware, ActionRequired::Warning,
            "Destination terminal to be routed could not be found on the device. Make sure the terminal name is valid for the specified device. Refer to Measurement & Automation Explorer or your hardware documentation for valid terminal names."
        }
    },
    {
        -200042,
        {
            Category::Hardware, ActionRequired::Warning,
            "Inversion requested is not possible. Either the hardware between the source and destination terminals does not support the inversion, or other routes in the task might be interfering with this route."
        }
    },
    {
        -200043,
        {
            Category::Hardware, ActionRequired::Warning,
            "Hardware necessary for this route is in use by another task or tasks."
        }
    },
    {
        -200044,
        {
            Category::Hardware, ActionRequired::Warning,
            "Route cannot be made between the source and destination terminals. Either the hardware does not support this route or other routes might be interfering with this route."
        }
    },
    {
        -200045,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device was removed or powered down between task verification and reservation. Ensure that the device is not being reset."
        }
    },
    {
        -200046,
        {
            Category::Communication, ActionRequired::ImmediateAction,
            "Invalid identifier at the end of the switch action. A connection separator, sequence separator, or valid switch action terminator must follow a switch action."
        }
    },
    {
        -200047,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Invalid identifier after the device identifier in the list entry."
        }
    },
    {
        -200048,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid trigger line in the \"<sac>\" or \"<wfa>\" statement in the list entry. Refer to the documentation for valid trigger lines."
        }
    },
    {
        -200049,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid value in the \"<repeat>\" statement in the list entry. The syntax for a repeat statement is \"<repeat integer>\". Refer to the documentation for valid integer values."
        }
    },
    {
        -200050,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Invalid channel name in the list entry. Refer to the documentation for valid channel names for the device in use."
        }
    },
    {
        -200051,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid identifier after a separator in the list entry."
        }
    },
    {
        -200052,
        {
            Category::Communication, ActionRequired::ImmediateAction,
            "Invalid identifier instead of an expected connection operator, \"->\", in the list entry. Refer to the documentation for proper syntax for connections involving channel ranges."
        }
    },
    {
        -200053,
        {
            Category::Hardware, ActionRequired::Warning,
            "Channels in switch actions cannot span different devices."
        }
    },
    {
        -200054,
        {
            Category::Communication, ActionRequired::Warning,
            "Semicolon or a semicolon modifier must follow a connection range statement. Refer to the documentation for information on connection ranges and semicolon modifiers."
        }
    },
    {
        -200055,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device identifier not specified in the list entry."
        }
    },
    {
        -200056,
        {
            Category::Other, ActionRequired::Warning,
            "Channel name not specified in the list entry."
        }
    },
    {
        -200057,
        {
            Category::Hardware, ActionRequired::Warning,
            "Duplicate device identifier in the device list. This is not allowed when waiting for devices to settle."
        }
    },
    {
        -200058,
        {
            Category::Other, ActionRequired::Warning,
            "Identifier in the list entry is too long."
        }
    },
    {
        -200059,
        {
            Category::Communication, ActionRequired::Warning,
            "List cannot end with the connection separator \"&\"."
        }
    },
    {
        -200060,
        {
            Category::Communication, ActionRequired::Warning,
            "Fully specified path cannot contain a connection range."
        }
    },
    {
        -200061,
        {
            Category::Communication, ActionRequired::ImmediateAction,
            "Invalid identifier in the list entry. The connection separator \"&\" or sequence separator \"&&\" was expected."
        }
    },
    {
        -200062,
        {
            Category::Communication, ActionRequired::ImmediateAction,
            "Invalid identifier instead of an expected connection operator \"->\" in the list entry."
        }
    },
    {
        -200063,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid identifier instead of an expected terminator in the list entry."
        }
    },
    {
        -200064,
        {
            Category::Communication, ActionRequired::Warning,
            "Unexpected connection separator \"&\" or sequence separator \"&&\" in the list entry."
        }
    },
    {
        -200065,
        {
            Category::Hardware, ActionRequired::Warning,
            "Action at the designated position in the scanlist is not valid for the device."
        }
    },
    {
        -200066,
        {
            Category::Communication, ActionRequired::ImmediateAction,
            "Connection operator is invalid at the designated point in the list entry."
        }
    },
    {
        -200067,
        {
            Category::Hardware, ActionRequired::Warning,
            "Settling time constraints for the device could not be satisfied. Refer to the documentation for details about settling time constraints."
        }
    },
    {
        -200068,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Scanning is not supported by the specified device."
        }
    },
    {
        -200069,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device specified is not a valid switch device."
        }
    },
    {
        -200070,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Advance trigger type specified is not supported by the device."
        }
    },
    {
        -200071,
        {
            Category::Other, ActionRequired::Warning,
            "Number of physical channels is too large."
        }
    },
    {
        -200072,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Duplicate channels in the list of physical channels are not supported by this device."
        }
    },
    {
        -200073,
        {
            Category::Hardware, ActionRequired::Warning,
            "SCXI module specified in the hardware configuration was not found. Make sure that the SCXI chassis is powered on, the SCXI cable is properly connected between the chassis communicator and the SCXI module, and that the cabled module specified in the hardware configuration is present in the specified slot."
        }
    },
    {
        -200074,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device unable to store calibration constants. Make sure that your hardware is properly installed, and test the regular operation of the device."
        }
    },
    {
        -200075,
        {
            Category::Other, ActionRequired::Warning,
            "Voltage data supplied is outside of the specified range. Change the range or the data. Refer to the documentation for more information about possible ranges."
        }
    },
    {
        -200076,
        {
            Category::Other, ActionRequired::Warning,
            "Current data supplied is outside of the specified range."
        }
    },
    {
        -200077,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Requested value is not a supported value for this property. The property value may be invalid because it conflicts with another property."
        }
    },
    {
        -200078,
        {
            Category::Other, ActionRequired::Warning,
            "Analog input (AI) task started or committed during a counter 0 DMA acquisition. If possible, use counter 1 instead of counter 0. Otherwise, start/commit the AI task before starting the counter 0 DMA acquisition."
        }
    },
    {
        -200079,
        {
            Category::Other, ActionRequired::Warning,
            "Analog output (AO) task started or committed during a counter 1 DMA acquisition. If possible, use counter 0 instead of counter 1. Otherwise, commit the AO task before starting the counter 1 DMA acquisition."
        }
    },
    {
        -200081,
        {
            Category::Other, ActionRequired::Warning,
            "Sample rate exceeds the maximum sample rate for the number of channels specified. Reduce the sample rate or the number of channels. Increasing the convert rate or reducing the sample delay might also alleviate the problem, if you set either of them."
        }
    },
    {
        -200082,
        {
            Category::Other, ActionRequired::Warning,
            "Minimum is greater than or equal to the maximum. Ensure the maximum value is greater than the minimum value. If using a custom scale, ensure that the scaled maximum is greater than the scaled minimum."
        }
    },
    {
        -200086,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Physical channel range syntax in the input string is invalid because multiple devices were listed in the string."
        }
    },
    {
        -200087,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Channel is not in the task, and the channel is not a valid global channel. Make sure that the channel is in the task or that the channel is a valid global channel. If you explicitly named the virtual channel in DAQmx Create Channel, you must use the name assigned to that channel. Also, check for typing errors."
        }
    },
    {
        -200088,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Task specified is invalid or does not exist."
        }
    },
    {
        -200089,
        {
            Category::Other, ActionRequired::Warning,
            "Task name specified conflicts with an existing task name."
        }
    },
    {
        -200090,
        {
            Category::Software, ActionRequired::ImmediateAction,
            "Shared library was not found. This error might be the result of an inadvertent deletion of an NI-DAQmx component. Reinstall NI-DAQmx, or download the latest version of the driver from the National Instruments website at ni.com. If the error is still returned, contact NI Technical Support."
        }
    },
    {
        -200091,
        {
            Category::Software, ActionRequired::ImmediateAction,
            "Shared library version installed is incorrect. This error might be the result of an incorrect installation of NI-DAQmx or a related software package. Reinstall NI-DAQmx, or download the latest version of the driver from the National Instruments website at ni.com. If the error is still returned, contact NI Technical Support."
        }
    },
    {
        -200092,
        {
            Category::Other, ActionRequired::Warning,
            "Function supported for channel-based tasks only."
        }
    },
    {
        -200093,
        {
            Category::Configuration, ActionRequired::Warning,
            "Attempted to retrieve channel properties from a multichannel task without selecting a specific channel. Use the Active Channel property to select a specific channel from which to retrieve properties."
        }
    },
    {
        -200094,
        {
            Category::Other, ActionRequired::Warning,
            "Digital waveform expected as input."
        }
    },
    {
        -200095,
        {Category::Other, ActionRequired::Warning, "Analog waveform expected as input."}
    },
    {
        -200096,
        {
            Category::Other, ActionRequired::Warning,
            "Number of samples to read must be -1 or greater."
        }
    },
    {
        -200097,
        {
            Category::Configuration, ActionRequired::Warning,
            "Attempted to retrieve channel properties from a multichannel task with more than one channel selected. You must select an individual channel to retrieve channel properties. If you are programming with LabVIEW, use the Active Channel property to specify the individual channel."
        }
    },
    {
        -200098,
        {
            Category::Other, ActionRequired::Warning,
            "Number of terminals requested cannot be greater than 1."
        }
    },
    {
        -200099,
        {Category::Other, ActionRequired::Warning, "Physical channel not specified."}
    },
    {
        -200100,
        {
            Category::Other, ActionRequired::Warning,
            "Specified DAQmx Read only can be used to read from a single channel. Use the multichannel DAQmx Read."
        }
    },
    {
        -200101,
        {
            Category::Other, ActionRequired::Warning,
            "Number of channels in data to write does not match the number of channels in the task."
        }
    },
    {
        -200102,
        {
            Category::Other, ActionRequired::Warning,
            "Pattern width specified does not match the number of lines in the digital channel."
        }
    },
    {
        -200103,
        {
            Category::Other, ActionRequired::Warning,
            "Number of samples to write must be the same for every channel."
        }
    },
    {
        -200104,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Bracket character (\"[\" or \"]\") at the specified position in the list is invalid. Matching bracket cannot be found. Check for nested fully specified paths or incorrectly paired brackets."
        }
    },
    {
        -200105,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Channel is invalid for the excitation mode of the SCXI-1122. Disable multiplexed excitation, or use one of the physical channels between ai0 and ai7."
        }
    },
    {
        -200106,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property must have the same value for all channels on this device."
        }
    },
    {
        -200107,
        {
            Category::Hardware, ActionRequired::Warning,
            "Module specified in the hardware configuration is not the module found. Make sure that the module specified in the hardware configuration is present in the specified slot."
        }
    },
    {
        -200108,
        {
            Category::Hardware, ActionRequired::Warning,
            "Calibration session is already open on this device. You can have only one open calibration session for each device. Use the handle obtained when the calibration session for this device was originally opened."
        }
    },
    {
        -200109,
        {
            Category::Other, ActionRequired::Warning,
            "Password is longer than four characters."
        }
    },
    {
        -200110,
        {Category::Other, ActionRequired::Warning, "Password supplied is incorrect."}
    },
    {
        -200111,
        {
            Category::Other, ActionRequired::Warning,
            "Password is required for this operation."
        }
    },
    {
        -200112,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Calibration handle is invalid. Open a calibration session to get a valid calibration handle. Use the valid calibration handle obtained when the calibration session was opened."
        }
    },
    {
        -200113,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device temperature is outside of the required range for calibration."
        }
    },
    {
        -200116,
        {
            Category::Hardware, ActionRequired::Warning,
            "Lines on the 8255 chip for this device are configured for output. Cannot tristate these lines at this time. Read values using an input task on another port."
        }
    },
    {
        -200117,
        {
            Category::Other, ActionRequired::Warning,
            "Port C cannot be used for data input/output in a handshaking task."
        }
    },
    {
        -200118,
        {
            Category::Other, ActionRequired::Warning,
            "Port reserved for handshaking. Cannot reserve this port or any of its lines for another task at this time."
        }
    },
    {
        -200119,
        {
            Category::Other, ActionRequired::Warning,
            "Port is configured for static digital operations by another task. Cannot configure this port or any of its lines for handshaking at this time."
        }
    },
    {
        -200120,
        {
            Category::Other, ActionRequired::Warning,
            "Port is configured for input. Cannot configure this port or any of its lines for output at this time."
        }
    },
    {
        -200121,
        {
            Category::Other, ActionRequired::Warning,
            "Port is configured for output. Cannot configure this port or any of its lines for input at this time."
        }
    },
    {
        -200122,
        {
            Category::Other, ActionRequired::Warning,
            "Lines 0 through 3 of this port are configured for input. Cannot configure these lines for output at this time."
        }
    },
    {
        -200123,
        {
            Category::Other, ActionRequired::Warning,
            "Lines 0 through 3 of this port are configured for output. Cannot configure these lines for input at this time."
        }
    },
    {
        -200124,
        {
            Category::Other, ActionRequired::Warning,
            "Lines 4 to 7 of this port are configured for input. Cannot configure these lines for output at this time."
        }
    },
    {
        -200125,
        {
            Category::Other, ActionRequired::Warning,
            "Lines 4 through 7 of this port are configured for output. Cannot configure these lines for input at this time."
        }
    },
    {
        -200126,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Lines on port C cannot be used for both handshaking control and static digital operations on an 8255 chip. Handshaking tasks automatically reserve some lines on port C as control lines. These lines cannot be reserved for static digital operations when the device is configured for handshaking. There are two likely causes for this error: 1. An attempt was made to reserve the lines for static digital operations when a handshaking task was previously configured. 2. An attempt was made to create a handshaking task when the lines were previously reserved for static digital operations. Refer to the documentation for information about which lines on port C are not available when the 8255 chip is in handshaking mode."
        }
    },
    {
        -200127,
        {
            Category::Hardware, ActionRequired::Warning,
            "Port 0 or any of its lines cannot be used to create a handshaking task. Use port 1 or port 2 of the 8255 chip on this device for handshaking."
        }
    },
    {
        -200128,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property must have the same value for all repeated physical channels. Set the same property value for all of the channels."
        }
    },
    {
        -200130,
        {
            Category::Other, ActionRequired::Warning,
            "Timebase divisor cannot be set for an external clock. You cannot divide down an externally supplied clock. If you want to divide down an external clock, specifiy an external timebase source instead and set the clock source to be internal."
        }
    },
    {
        -200131,
        {
            Category::Other, ActionRequired::Warning,
            "Analog trigger source must be the first channel in the acquisition or a valid analog trigger terminal. If you explicitly named the virtual channel in DAQmx Create Channel, you must use the name assigned to that channel."
        }
    },
    {
        -200132,
        {
            Category::Other, ActionRequired::Warning,
            "External timebase rate must be specified to translate the derived clock or timebase rate into ticks. Set the external timebase rate, or set the divisor instead of the clock or timebase rate."
        }
    },
    {
        -200133,
        {
            Category::Other, ActionRequired::Warning,
            "Counter timebase source, counter timebase rate, master timebase divisor, and master timebase rate settings are inconsistent with one another. The conflicting properties must satisfy the following constraint: Master Timebase Rate / Master Timebase Divisor = Counter Timebase Rate"
        }
    },
    {
        -200134,
        {
            Category::Other, ActionRequired::Warning,
            "Counter timebase source and counter timebase rate settings are inconsistent with one another. For internal counter timebase source selections, if the counter timebase rate is set, its value must match the rate corresponding to the counter timebase source. For example, 20 MHz corresponds to a rate of 20,000,000 Hz."
        }
    },
    {
        -200135,
        {
            Category::Other, ActionRequired::Warning,
            "Counter timebase source and counter timebase master timebase divisor settings are inconsistent with one another. If the divisor is specified, the following must apply: Master Timebase Rate / Counter Timebase Master Timebase Divisor = Rate corresponding to Counter Timebase Source."
        }
    },
    {
        -200136,
        {
            Category::Configuration, ActionRequired::Warning,
            "Frequency and Initial Delay property values are inconsistent with one or more counter timebase properties. The conflicting properties must satisfy the following constraint: Counter Timebase Rate / Counter Maximum Count <= X <= Counter Timebase Rate / 2 where X = Frequency and 1 / Initial Delay, and where Counter Timebase Rate = Master Timebase Rate / Counter Timebase Master Timebase Divisor or is inferred from the Counter Timebase Source selection."
        }
    },
    {
        -200137,
        {
            Category::Configuration, ActionRequired::Warning,
            "Initial Delay, High Time, and Low Time property values are inconsistent with one or more counter timebase properties. The conflicting properties must satisfy the following restraint: 2 / Counter Timebase Rate <= X <= Counter Maximum Count/ Counter Timebase Rate where X = Initial Delay, High Time, and Low Time, and where Counter Timebase Rate = Master Timebase Rate / Counter Timebase Master Timebase Divisor or is inferred from the Counter Timebase Source selection."
        }
    },
    {
        -200138,
        {
            Category::Other, ActionRequired::Warning,
            "A timebase could not be selected that covers the entire range specified in the Maximum and Minimum properties. The conflicting properties must satisfy the following constraints: Maximum <= Counter Maximum Count / Counter Timebase Rate Minimum >= 2 / Counter Timebase Rate."
        }
    },
    {
        -200139,
        {
            Category::Other, ActionRequired::Warning,
            "A timebase could not be selected that covers the entire range specified in the Maximum and Minimum properties. The conflicting properties must satisfy the following constraints: Maximum <= Counter Timebase Rate / 2 Minimum >= Counter Timebase Rate / Counter Maximum Count."
        }
    },
    {
        -200140,
        {
            Category::Other, ActionRequired::Warning,
            "Two consecutive active edges of the input signal occurred without a counter timebase edge. Use a faster counter timebase."
        }
    },
    {
        -200141,
        {
            Category::Other, ActionRequired::Warning,
            "Data was overwritten before it could be read by the system. If Data Transfer Mechanism is Interrupts, try using DMA or USB Bulk. Otherwise, divide the input signal before taking the measurement."
        }
    },
    {
        -200142,
        {
            Category::Configuration, ActionRequired::Warning,
            "Internal timebase could not be found that matches the rate specified in the Counter Timebase Rate property."
        }
    },
    {
        -200143,
        {
            Category::Configuration, ActionRequired::Warning,
            "Counter timebase rate must be specified for external counter timebase sources in order for frequency and/or time calculations to be made correctly. Set the Counter Timebase Rate property to the appropriate value for your external source."
        }
    },
    {
        -200144,
        {
            Category::Other, ActionRequired::Warning,
            "Pause trigger is only valid for continuous pulse generations. Change the sample mode to continuous, or do not use the pause trigger."
        }
    },
    {
        -200145,
        {
            Category::Other, ActionRequired::Warning,
            "Pause trigger is only valid for event counting if sample clock is not used."
        }
    },
    {
        -200146,
        {
            Category::Other, ActionRequired::Warning,
            "Pause and start triggers cannot both be active in this task."
        }
    },
    {
        -200147,
        {
            Category::Other, ActionRequired::Warning,
            "There cannot be multiple counters in the same task for input operations. Use a separate task for each counter."
        }
    },
    {
        -200148,
        {
            Category::Other, ActionRequired::Warning,
            "FREQOUT counter cannot generate the desired frequency. The FREQOUT counter is a 4-bit counter that can divide either the master timebase rate / 2 or the master timebase rate / 200 by a number between one and 16. Choose a frequency within this range."
        }
    },
    {
        -200149,
        {
            Category::Other, ActionRequired::Warning,
            "External timebase rate must be specified to translate the delay into ticks. Set the external timebase rate, or set the delay in units of ticks."
        }
    },
    {
        -200150,
        {
            Category::Other, ActionRequired::Warning,
            "Channel is not available when the module is in parallel mode."
        }
    },
    {
        -200151,
        {
            Category::Other, ActionRequired::Warning,
            "Your SCXI system is not set up for performing analog input operations on given channels. The SCXI module cabled to your digitizer cannot route analog signals from other modules to the digitizer, or is not configured to route them. To perform the desired operation with multiple SCXI modules and one digitizer, cable the digitizer to one of the analog input modules. The module your channels are on is one such module. Then, update the chassis configuration in MAX to reflect cabling change, and ensure that the cabled module is in multiplexed mode. Alternatively, you can use multiple digitizers and SCXI modules in parallel mode. For detailed information about cabling, refer to documentation."
        }
    },
    {
        -200152,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Data read from the EEPROM on the device is invalid. Verify that any accessories configured with this device are connected. If the problem continues, contact National Instruments Technical Support. The device might need to be recalibrated or repaired by NI."
        }
    },
    {
        -200153,
        {
            Category::Hardware, ActionRequired::Warning,
            "Reference voltage applied for calibration is outside the range defined for calibration of this device. Ensure that the reference voltage falls within the range specified for this device."
        }
    },
    {
        -200154,
        {
            Category::Hardware, ActionRequired::Warning,
            "Reference current applied for calibration is outside the range defined for calibration of this device. Ensure that the reference current falls within the range specified for this device."
        }
    },
    {
        -200155,
        {
            Category::Hardware, ActionRequired::Warning,
            "Reference resistance applied for calibration is outside the range defined for calibration of this device. Ensure that the reference resistance falls within the range specified for this device."
        }
    },
    {
        -200156,
        {
            Category::Hardware, ActionRequired::Warning,
            "Reference frequency applied for calibration is outside the range defined for calibration of this device. Ensure that the reference frequency falls within the range specified for this device."
        }
    },
    {
        -200157,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device could not complete the calibration operation. Calibration could fail for the following reasons: 1. The actual reference signal applied for calibration was different from the value you specified. Ensure that the reference signal applied is the same as the values that were input. 2. The reference signal was not stable over the period of time that the hardware was being calibrated. Ensure that the reference signal specified is free of noise and does not drift over the duration of the calibration. 3. The device is not functioning properly."
        }
    },
    {
        -200158,
        {
            Category::Hardware, ActionRequired::Warning,
            "Requested operation could not be performed because the necessary digital lines could not be reserved by SCXI. Another task might have reserved these lines previously. For example, E Series devices use lines 0, 1, 2, and 4 on port 0 to communicate with the SCXI module."
        }
    },
    {
        -200159,
        {
            Category::Communication, ActionRequired::Warning,
            "Requested operation could not be performed because the digital lines are being used for communication with SCXI or a TEDS carrier. For example, E Series devices use lines 0, 1, 2, and 4 on port 0 to communicate with a SCXI module. Therefore, you cannot use lines 0, 1, 2, and 4 for regular digital I/O."
        }
    },
    {
        -200160,
        {
            Category::Other, ActionRequired::Warning,
            "Channel could not be created. All channels must be created before the task is verified. Before I/O can be performed or properties can be retrieved, tasks are verified. Channels must be created before these actions can occur."
        }
    },
    {
        -200161,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device to which the sensor is attached does not have an available internal excitation source. Select another device with an available internal excitation source or supply external excitation."
        }
    },
    {
        -200162,
        {
            Category::Other, ActionRequired::Warning,
            "2-wire resistance configuration is incompatible with voltage excitation."
        }
    },
    {
        -200163,
        {
            Category::Other, ActionRequired::Warning,
            "Completion resistance value, R1, cannot be zero if the circuit uses voltage excitation."
        }
    },
    {
        -200166,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Output buffer underwrite. Your application was unable to write samples to the background buffer fast enough for the device to get new samples at the specified sample rate. To avoid this error, you can do any the following: 1. Increase the size of the background buffer by configuring the buffer. 2. Increase the number of samples you write each time you invoke a write operation. 3. Write samples more often. 4. Reduce your sample rate. 5. Change the data transfer mechanism from interrupts to DMA. 6. Initially write a sufficient number of samples to satisfy the specified data transfer request condition. 7. Reduce the number of applications that your computer is executing concurrently. In addition, if you do not need to ensure that each sample is generated once and only once, you can set the regeneration mode to allow regeneration."
        }
    },
    {
        -200167,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device cannot acquire from _cjTemp and other channels in the same task. Create one task for reading _cjTemp and another task for the other channels."
        }
    },
    {
        -200168,
        {
            Category::Hardware, ActionRequired::Warning,
            "Number of channels to acquire exceeds the device maximum. Reduce the number of channels. In some cases, you can access a large number of channels if they are identically configured and created consecutively. Refer to the documentation for more information."
        }
    },
    {
        -200169,
        {
            Category::Other, ActionRequired::Warning,
            "Memory mapping can be enabled only if Data Transfer Mechanism is Programmed IO. Enable memory mapping only when Data Transfer Mechanism is Programmed IO."
        }
    },
    {
        -200170,
        {
            Category::Hardware, ActionRequired::Warning,
            "Physical channel specified does not exist on this device. Refer to the documentation for channels available on this device."
        }
    },
    {
        -200171,
        {
            Category::Other, ActionRequired::Warning,
            "Virtual channel cannot be created. Another virtual channel with this name already exists."
        }
    },
    {
        -200172,
        {
            Category::Other, ActionRequired::Warning,
            "Buffer size must be zero when Data Transfer Mechanism is Programmed IO. Set buffer size to zero or Data Transfer Mechanism to something other than Programmed IO."
        }
    },
    {
        -200173,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "The combination of Sample Timebase Rate and Master Timebase Rate you specified is invalid. The driver computed the Sample Timebase Source Divisor by dividing the Master Timebase Rate by the Sample Timebase Rate. The resulting value for the Sample Timebase Source Divisor is not supported by your device. Refer to the documentation for more information about these three properties."
        }
    },
    {
        -200175,
        {
            Category::Hardware, ActionRequired::Warning,
            "Hardware is not responding. Ensure your hardware is powered on and all cables are properly connected."
        }
    },
    {
        -200176,
        {
            Category::Hardware, ActionRequired::Warning,
            "Operation is not permitted while the switch device is scanning."
        }
    },
    {
        -200177,
        {
            Category::Hardware, ActionRequired::Warning,
            "Operation is permitted only while the switch device is scanning."
        }
    },
    {
        -200178,
        {
            Category::Other, ActionRequired::Warning,
            "Task was created with a settling time different from the current settling time. When scanning, you must use the original settling time specified when the task was created."
        }
    },
    {
        -200179,
        {
            Category::Communication, ActionRequired::Warning,
            "Explicit connection between the channels already exists. You can make only one connection between these channels."
        }
    },
    {
        -200180,
        {
            Category::Other, ActionRequired::Warning,
            "Path between two switch channels is not available."
        }
    },
    {
        -200181,
        {
            Category::Hardware, ActionRequired::Warning,
            "Channel name specified is not valid for the switch device."
        }
    },
    {
        -200182,
        {
            Category::Other, ActionRequired::Warning,
            "Switch channels cannot be disconnected because there is no explicit path between them."
        }
    },
    {
        -200183,
        {
            Category::Other, ActionRequired::Warning,
            "Switch channel names cannot be duplicated in the path string."
        }
    },
    {
        -200184,
        {
            Category::Other, ActionRequired::Warning,
            "Leg in path cannot contain two channels that are already directly connected."
        }
    },
    {
        -200185,
        {
            Category::Other, ActionRequired::Warning,
            "Path contains a leg with two channels that cannot be directly connected."
        }
    },
    {
        -200186,
        {
            Category::Communication, ActionRequired::Warning,
            "Channels used to make the connection between two endpoints must be reserved for routing."
        }
    },
    {
        -200187,
        {
            Category::Other, ActionRequired::Warning,
            "Channel cannot be connected to itself."
        }
    },
    {
        -200188,
        {
            Category::Communication, ActionRequired::Warning,
            "Connection cannot be made between the specified channels because they are connected to different source channels."
        }
    },
    {
        -200189,
        {
            Category::Communication, ActionRequired::Warning,
            "Explicit connection cannot be made to a switch channel that is reserved for routing."
        }
    },
    {
        -200190,
        {
            Category::Communication, ActionRequired::Warning,
            "Disconnection path is not the same as the existing path. You can programmatically find out the existing path. Refer to your documentation for details."
        }
    },
    {
        -200191,
        {
            Category::Other, ActionRequired::Warning,
            "Task was created with a topology different from the current topology. When scanning, you must use the original topology specified when the task was created."
        }
    },
    {
        -200192,
        {
            Category::Hardware, ActionRequired::Warning,
            "Switch device supports continuous scanning only."
        }
    },
    {
        -200193,
        {
            Category::Hardware, ActionRequired::Warning,
            "Switch device does not support this operation."
        }
    },
    {
        -200194,
        {
            Category::Hardware, ActionRequired::Warning,
            "Hardware was unexpectedly powered off and back on. To recover, reset the device (either programmatically or by using Measurement & Automation Explorer)."
        }
    },
    {
        -200195,
        {
            Category::Hardware, ActionRequired::Warning,
            "Switch configuration has caused the switch device to exceed its power limit because there were too many closed relays. The switch was disabled. Reset it by doing one of the following: 1. Call DAQmx Switch Set Topology And Reset. 2. Call DAQmx Device Reset. 3. Use Measurement & Automation Explorer."
        }
    },
    {
        -200196,
        {
            Category::Hardware, ActionRequired::Warning,
            "Action at the end of the scan list is not valid for this device."
        }
    },
    {
        -200197,
        {
            Category::Configuration, ActionRequired::Warning,
            "Device does not support this property."
        }
    },
    {
        -200198,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Topology specified is invalid. Make sure the spelling of the topology is correct and that the switch supports that topology."
        }
    },
    {
        -200199,
        {
            Category::Hardware, ActionRequired::Warning,
            "Switch device must be reset before scanning. Reset the device by doing one of the following: 1. Call DAQmx Switch Set Topology And Reset. 2. Call DAQmx Device Reset. 3. Use Measurement & Automation Explorer."
        }
    },
    {
        -200200,
        {
            Category::Communication, ActionRequired::Warning,
            "Switch channel is already in exclusive use within another connection."
        }
    },
    {
        -200201,
        {
            Category::Hardware, ActionRequired::Warning,
            "Switch scan list is too large to fit in the onboard memory of the device."
        }
    },
    {
        -200202,
        {Category::Other, ActionRequired::ImmediateAction, "Relay name is invalid."}
    },
    {
        -200203,
        {
            Category::Hardware, ActionRequired::Warning,
            "Switch hardware is incapable of driving multiple trigger lines simultaneously."
        }
    },
    {
        -200204,
        {
            Category::Other, ActionRequired::Warning,
            "Unexpected identifier within the fully-specified path in the list."
        }
    },
    {
        -200205,
        {
            Category::Other, ActionRequired::Warning,
            "Topology does not support scanning."
        }
    },
    {
        -200206,
        {
            Category::Hardware, ActionRequired::Warning,
            "Advance trigger and Advance Complete event must use the same polarity in this particular switch device."
        }
    },
    {
        -200207,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Device identifier in the list entry is invalid."
        }
    },
    {
        -200208,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Range statement in the list entry contains an invalid character sequence."
        }
    },
    {
        -200209,
        {
            Category::Configuration, ActionRequired::Warning,
            "Duplicate device identifier found in the terminal list when trying to set the property. Only one instance of the device identifier is permitted."
        }
    },
    {
        -200210,
        {
            Category::Hardware, ActionRequired::Warning,
            "Multiple device identifiers from one chassis are not allowed in the terminal list."
        }
    },
    {
        -200211,
        {
            Category::Other, ActionRequired::Warning,
            "Multiple relay names were specified for a single relay operation."
        }
    },
    {
        -200212,
        {
            Category::Other, ActionRequired::Warning,
            "Measurement units specified for the channel are not valid for the Measurement Type of the channel."
        }
    },
    {
        -200213,
        {
            Category::Other, ActionRequired::Warning,
            "Pretrigger Samples per Channel requested plus minimum number of posttrigger samples exceed the requested Number of Samples per Channel. Decrease the number of Pretrigger Samples per Channel, or increase Number of Samples per Channel."
        }
    },
    {
        -200214,
        {
            Category::Hardware, ActionRequired::Warning,
            "Analog trigger circuitry unavailable on the device. Select a non-analog trigger type, or use a device with analog triggering hardware."
        }
    },
    {
        -200215,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Memory Mapping is not supported for buffered operations. Turn Memory Mapping off, set Buffer Size to 0, or do not configure the buffer for the operation."
        }
    },
    {
        -200216,
        {
            Category::Hardware, ActionRequired::Warning,
            "Buffered operations cannot use a Data Transfer Mechanism of Programmed I/O for this device and Channel Type. Non-buffered operations cannot use a Data Transfer Mechanism of Interrupts or DMA for this device and Channel Type."
        }
    },
    {
        -200217,
        {
            Category::Other, ActionRequired::Warning,
            "Buffered operations cannot use On Demand for Sample Timing Type. Set your buffer size to 0 for On Demand sample timing. Otherwise, configure your sample clock, or change your sample timing type for buffered operations."
        }
    },
    {
        -200218,
        {
            Category::Hardware, ActionRequired::Warning,
            "Data Transfer Mechanism must be Programmed I/O when not using hardware timing. Set Data Transfer Mechanism to Programmed I/O, configure your sample clock timing, or set Sample Timing Type to Sample Clock."
        }
    },
    {
        -200219,
        {
            Category::Hardware, ActionRequired::Warning,
            "Analog output virtual channels cannot be created out of order with respect to their physical channel numbers for the type of device you are using. For example, a virtual channel using physical channel ao0 must be created before a virtual channel with physical channel ao1."
        }
    },
    {
        -200220,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Device identifier is invalid."
        }
    },
    {
        -200221,
        {
            Category::Other, ActionRequired::Warning,
            "Amount of time allocated to perform this operation was exceeded."
        }
    },
    {
        -200222,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Acquisition has been stopped to prevent an input buffer overwrite. Your application was unable to read samples from the buffer fast enough to prevent new samples from overwriting unread data. To avoid this error, you can do any of the following: 1. Increase the size of the buffer. 2. Increase the number of samples you read each time you invoke a read operation. 3. Read samples more often. 4. Reduce the sample rate. 5. If your data transfer method is interrupts, try using DMA or USB Bulk. 6. Reduce the number of applications your computer is running concurrently. In addition, if you do not need to read every sample that is acquired, you can configure the overwrite mode to overwrite unread data, and then use the Relative To and Offset properties to read the desired samples."
        }
    },
    {
        -200223,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Specified threshold and hysteresis values for this channel create a triggering range that is not supported by your device. On the SCXI-1126, threshold minus hysteresis must be between -0.5 and 4.48."
        }
    },
    {
        -200224,
        {
            Category::Hardware, ActionRequired::Warning,
            "No registered trigger lines could be found between the devices in the route. If you have a PXI chassis, identify the chassis correctly in MAX, and make sure it has been configured properly. If you are using PCI devices, make sure they are connected with a RTSI cable and that the RTSI cable is registered in MAX. Otherwise, make sure there is an available trigger line on the trigger bus shared between the devices."
        }
    },
    {
        -200225,
        {
            Category::Other, ActionRequired::Warning,
            "Trigger line requested could not be reserved because it is already in use."
        }
    },
    {
        -200226,
        {
            Category::Hardware, ActionRequired::Warning,
            "Trigger bus to which the device is connected does not have any free trigger lines for the driver to choose. To free up trigger lines, you can do any of the following: 1. Stop other tasks that are connected to the same trigger bus as this device. 2. Use DAQmx Disconnect Route to stop any immediate routes that span this trigger bus. 3. Make more trigger lines on this trigger bus available to the driver."
        }
    },
    {
        -200227,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device does not have any free trigger lines for the device driver to choose. Although there might be trigger lines available on the respective trigger bus, the device cannot use the trigger bus because the device does not have enough free resources to do so. To free up trigger lines, you can do any of the following: 1. Stop other tasks that are connected to this device. 2. Use DAQmx Disconnect Route to stop any immediate routes that span this trigger bus and device."
        }
    },
    {
        -200228,
        {
            Category::Other, ActionRequired::Warning,
            "Buffer is too small to fit the string."
        }
    },
    {
        -200229,
        {
            Category::Other, ActionRequired::Warning,
            "Buffer is too small to fit read data."
        }
    },
    {
        -200230,
        {
            Category::Other, ActionRequired::Warning,
            "NULL pointer was passed for a required parameter."
        }
    },
    {
        -200231,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property requested cannot be set."
        }
    },
    {
        -200232,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property requested cannot be read."
        }
    },
    {
        -200233,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property specified is not valid for this function."
        }
    },
    {
        -200234,
        {
            Category::Other, ActionRequired::Warning,
            "Buffer is too small for requested samples to be written."
        }
    },
    {
        -200235,
        {
            Category::Other, ActionRequired::Warning,
            "Explanation could not be found for the requested status code. Verify that the requested status code is correct."
        }
    },
    {
        -200236,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property requested cannot be reset."
        }
    },
    {
        -200237,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external reference clock, make sure it is connected and within the jitter and voltage level specifications, and its rate is correctly specified. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200238,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external reference clock, make sure it is connected and within the jitter and voltage level specifications at all times, and its rate is correctly specified. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200239,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200240,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200241,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200242,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200243,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200244,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200245,
        {
            Category::Other, ActionRequired::Warning,
            "PLL could not phase-lock to the external reference clock. Make sure your reference clock is connected and that it is within the jitter and voltage specifications. Also, make sure the reference clock rate is correctly specified."
        }
    },
    {
        -200246,
        {
            Category::Other, ActionRequired::Warning,
            "PLL has lost phase-lock to the external reference clock. Make sure your reference clock is connected and that it is within the jitter and voltage level specifications at all times. Also, make sure the reference clock rate is correctly specified at all times."
        }
    },
    {
        -200247,
        {
            Category::Other, ActionRequired::Warning,
            "Integer was expected but not found in the script. Insert an appropriate integer at this location in the script."
        }
    },
    {
        -200248,
        {
            Category::Other, ActionRequired::Warning,
            "Specified marker position is too close to the end of the last \"generate\" statement in a \"repeat until\" loop. Move the marker position farther away from the end of the last \"generate\" statement in the \"repeat until\" loop."
        }
    },
    {
        -200249,
        {
            Category::Other, ActionRequired::Warning,
            "Length of waveform subset is too small for the last \"generate\" statement in a \"repeat until\" loop."
        }
    },
    {
        -200250,
        {
            Category::Other, ActionRequired::Warning,
            "Length of waveform is too small for the last \"generate\" statement in a \"repeat until\" loop."
        }
    },
    {
        -200251,
        {
            Category::Other, ActionRequired::Warning,
            "No DMA channels or USB Bulk Endpoints are available. Either shut down other tasks that might be using these resources or consider changing your data transfer mechanism to Interrupts if supported."
        }
    },
    {
        -200252,
        {
            Category::Other, ActionRequired::Warning,
            "Terminal cannot be tristated because it is busy. Disconnect any routes spanning this terminal, or stop any tasks using this terminal."
        }
    },
    {
        -200253,
        {
            Category::Hardware, ActionRequired::Warning,
            "Terminal could not be tristated because the hardware cannot tristate this terminal."
        }
    },
    {
        -200254,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Terminal for the device is invalid."
        }
    },
    {
        -200255,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Built-in temperature sensor is not supported on this channel. This channel is not configured to support a built-in temperature sensor. Make sure the accessory specified in the hardware configuration is correct and that the hardware supports a built-in temperature sensor on this channel."
        }
    },
    {
        -200256,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified topology cannot be used to reset the switch, because that topology is not supported by the connected terminal block. Refer to the documentation for supported topologies for the given terminal block, or disconnect the terminal block from the switch."
        }
    },
    {
        -200257,
        {
            Category::Configuration, ActionRequired::Warning,
            "Excitation property must be the same for related physical channels. Refer to the documentation for information about setting excitation across related physical channels."
        }
    },
    {
        -200258,
        {
            Category::Hardware, ActionRequired::Warning,
            "Gain value conflicts with specified AI Minimum and AI Maximum properties. The specified gain and AI Minimum and/or AI Maximum would cause the device to exceed the hardware limit. Lower the gain, or adjust AI Minimum and/or AI Maximum."
        }
    },
    {
        -200259,
        {
            Category::Configuration, ActionRequired::Warning,
            "Value selected for this jumper-controlled property must match the value specified in Measurement & Automation Explorer. Make sure the value specified in Measurement & Automation Explorer matches the value in your program and that the value corresponds to the selection made using the jumper on the device."
        }
    },
    {
        -200260,
        {
            Category::Other, ActionRequired::Warning,
            "Memory mapping has been enabled, and the sample clock has been configured; but the buffer size has not been set, and the data transfer mechanism has either not been set or was set to something other than Programmed I/O. Set the buffer size to 0, and/or change the data transfer mechanism to Programmed I/O."
        }
    },
    {
        -200261,
        {
            Category::Other, ActionRequired::Warning,
            "An attempt has been made to use an analog trigger in multiple situations with differing properties. Change the analog trigger properties so they are the same, or do not use an analog trigger for all situations."
        }
    },
    {
        -200262,
        {
            Category::Other, ActionRequired::Warning,
            "An attempt has been made to configure a trigger without configuring the appropriate sample clock properties or when Sample Timing Type was set to On Demand. Configure the sample clock type to something other than On Demand to use a trigger."
        }
    },
    {
        -200263,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device supports an analog channel as the source of an analog pause trigger only when it is the only channel in the task. Remove all of the channels currently in the task except the channel that will be used as the analog trigger source, or change the analog trigger source to a terminal."
        }
    },
    {
        -200264,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device supports an analog channel as the source of an analog reference trigger only when it is the only channel in the task. Remove all of the channels currently in the task except the channel that will be used as the analog trigger source, or change the analog trigger source to a terminal."
        }
    },
    {
        -200265,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "An attempt has been made to use an invalid analog trigger source. Ensure that the trigger source you specify matches the name of the virtual channel in the task or matches the name of a non-scannable terminal that the device can use as an analog trigger source."
        }
    },
    {
        -200266,
        {
            Category::Other, ActionRequired::Warning,
            "Minimum and maximum values for the channel are not symmetric."
        }
    },
    {
        -200267,
        {
            Category::Hardware, ActionRequired::Warning,
            "Product of AO channel properties Maximum Value and Gain exceeds the maximum voltage for the device."
        }
    },
    {
        -200268,
        {
            Category::Other, ActionRequired::Warning,
            "Specified Offset is too small given AO Gain and Minimum Value. The following constraint must hold: Offset > (Gain * Minimum Value / 2)"
        }
    },
    {
        -200269,
        {
            Category::Other, ActionRequired::Warning,
            "Specified Offset is too large for the given AO Gain and Maximum Value. The following constraint must hold: Offset < (Gain * Maximum Value / 2)"
        }
    },
    {
        -200270,
        {
            Category::Other, ActionRequired::Warning,
            "Interpolation rate specified is not possible for the given sample rate."
        }
    },
    {
        -200271,
        {
            Category::Hardware, ActionRequired::Warning,
            "Product of AO Channel properties Minimum Value and Gain exceeds the minimum voltage for the device."
        }
    },
    {
        -200272,
        {
            Category::Other, ActionRequired::Warning,
            "Sample clock rate requested is too low for the selected divide-down clock. Use the high resolution clock, or increase your sample rate."
        }
    },
    {
        -200273,
        {
            Category::Configuration, ActionRequired::Warning,
            "Sample clock rate and the sample clock divisor values are inconsistent with one another. Consider settting either the sample clock rate or the sample clock divisor, but not both. This allows the driver to automatically select an appropriate value for the other property. Alternatively, make sure the sample clock rate and sample clock divisor satisfy the following constraint: rate = timebase / divisor"
        }
    },
    {
        -200274,
        {
            Category::Other, ActionRequired::Warning,
            "Sample clock rate desired is too high for an external clock being brought in through the backplane. Bring in your external sample clock through one of the higher-frequency front panel connectors, or use a lower sample rate."
        }
    },
    {
        -200275,
        {
            Category::Other, ActionRequired::Warning,
            "Sample rate desired is too low for an external clock being brought in through the ClkIn connector. Change the sample rate so it is within limits, or use DDC_ClkIn to bring in your sample clock."
        }
    },
    {
        -200276,
        {
            Category::Other, ActionRequired::Warning,
            "Reference clock source and sample clock source cannot be the same. Use different terminals to bring in your reference clock and sample clock, or use only one of them at a time."
        }
    },
    {
        -200277,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid combination of position and offset. The position and offset specified a sample prior to the first sample acquired (sample 0). Make sure any negative read offset specified will select a valid sample when combined with the read position."
        }
    },
    {
        -200278,
        {
            Category::Other, ActionRequired::Warning,
            "Attempted to read a sample beyond the final sample acquired. The acquisition has stopped, therefore the sample specified by the combination of position and offset will never be available. Specify a position and offset which selects a sample up to, but not beyond, the final sample acquired. The final sample acquired can be determined by querying the total samples acquired after an acquisition has stopped."
        }
    },
    {
        -200279,
        {
            Category::Hardware, ActionRequired::Warning,
            "The application is not able to keep up with the hardware acquisition. Increasing the buffer size, reading the data more frequently, or specifying a fixed number of samples to read instead of reading all available samples might correct the problem."
        }
    },
    {
        -200281,
        {
            Category::Hardware, ActionRequired::Warning,
            "Reading relative to the reference trigger or relative to the start of pretrigger samples position before the acquisition is complete. Wait for the acquisition to complete before reading, or increase your read timeout. Also, make sure the hardware is set up and wired correctly, the signal for the reference trigger is correct, and that the reference trigger occurs while the device is acquiring data."
        }
    },
    {
        -200282,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Reading relative to the reference trigger or relative to the start of a pretrigger sample is not supported with the current task configuration. If you have not configured a reference trigger or if one of your devices is utilizing an onboard buffer to transfer data after an acquisition has completed, reading relative to reference trigger or relative to the first pretrigger sample is not supported."
        }
    },
    {
        -200283,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Acquisition has stopped to prevent the intermediate buffer from overflowing. The background was running too fast for the application to keep up, and the application was unable to read samples from the intermediate buffer fast enough to prevent losing samples. To avoid this error, you might reduce the sample rate, reduce the number of applications your computer is executing concurrently, or not read any samples until the acquisition is complete."
        }
    },
    {
        -200284,
        {
            Category::Other, ActionRequired::Warning,
            "Some or all of the samples requested have not yet been acquired. To wait for the samples to become available use a longer read timeout or read later in your program. To make the samples available sooner, increase the sample rate. If your task uses a start trigger, make sure that your start trigger is configured correctly. It is also possible that you configured the task for external timing, and no clock was supplied. If this is the case, supply an external clock."
        }
    },
    {
        -200286,
        {
            Category::Other, ActionRequired::Warning,
            "No data is available to read, because no acquisition has been started. Start the acquisition before attempting to read data, either explicitly or by enabling auto start and stop."
        }
    },
    {
        -200287,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Attempted to write to an invalid combination of position and offset. The position and offset specified a sample prior to the first sample generated (sample 0). Make sure any negative write offset specified will select a valid sample when combined with the write position."
        }
    },
    {
        -200288,
        {
            Category::Other, ActionRequired::Warning,
            "Attempted to write a sample beyond the final sample generated. The generation has stopped, therefore the sample specified by the combination of position and offset will never be available. Specify a position and offset which selects a sample up to, but not beyond, the final sample generated. The final sample generated can be determined by querying the total samples generated after a generation has stopped."
        }
    },
    {
        -200289,
        {
            Category::Hardware, ActionRequired::Warning,
            "Attempted to write samples that have already been generated or have already been sent to the device for generation. Increasing the buffer size or writing the data more frequently might correct the problem."
        }
    },
    {
        -200290,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "The generation has stopped to prevent the regeneration of old samples. Your application was unable to write samples to the background buffer fast enough to prevent old samples from being regenerated. To avoid this error, you can do any of the following: 1. Increase the size of the background buffer by configuring the buffer. 2. Increase the number of samples you write each time you invoke a write operation. 3. Write samples more often. 4. Reduce the sample rate. 5. If your data transfer method is interrupts, try using DMA or USB Bulk. 6. Reduce the number of applications your computer is executing concurrently. In addition, if you do not need to write every sample that is generated, you can configure the regeneration mode to allow regeneration, and then use the Position and Offset attributes to write the desired samples."
        }
    },
    {
        -200291,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "The generation has stopped because an intermediate buffer overflowed. The background was running too fast for the application to keep up, and the application was unable to write samples to the intermediate buffer fast enough to prevent regenerating old samples. To avoid this error, you can reduce the sample rate, reduce the number of applications your computer is executing concurrently, or write all samples before the generation starts."
        }
    },
    {
        -200292,
        {
            Category::Other, ActionRequired::Warning,
            "Some or all of the samples to write could not be written to the buffer yet. More space will free up as samples currently in the buffer are generated. To wait for more space to become available, use a longer write timeout. To make the space available sooner, increase the sample rate."
        }
    },
    {
        -200293,
        {
            Category::Other, ActionRequired::Warning,
            "The generation is not yet started, and not enough space is available in the buffer. Configure a larger buffer, or start the generation before writing more data than will fit in the buffer."
        }
    },
    {
        -200294,
        {
            Category::Other, ActionRequired::Warning,
            "Not enough samples were written to satisfy the initial data transfer request condition. To successfully start a generation, increase the number of samples initially written to the buffer before starting. Alternatively, decrease the number of samples required to start by changing the data transfer request condition."
        }
    },
    {
        -200295,
        {
            Category::Hardware, ActionRequired::Warning,
            "Attempt was made to write samples after start of generation where only onboard memory was used. In this case, all samples must be written to the device before the start of generation. No samples may be updated once the generation has started. If you wish to modify samples in the generation after the start of the generation, do not enable the onboard memory."
        }
    },
    {
        -200297,
        {
            Category::Configuration, ActionRequired::Warning,
            "This property is unavailable when using onboard memory."
        }
    },
    {
        -200300,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid timing type for this channel."
        }
    },
    {
        -200301,
        {
            Category::Configuration, ActionRequired::Warning,
            "Cannot update the Pulse Generation property. The pulse generation with previous property settings must complete a full cycle before the property can be updated."
        }
    },
    {
        -200302,
        {
            Category::Other, ActionRequired::Warning,
            "Signal being measured is slower than the specified measurement time. Increase the measurement time, or use a different measurement method."
        }
    },
    {
        -200303,
        {
            Category::Other, ActionRequired::Warning,
            "External sample clock source must be specified for this application."
        }
    },
    {
        -200304,
        {
            Category::Other, ActionRequired::Warning,
            "External master timebase rate must be specified for this channel given the selected measurement units. Specify the master timebase rate, or use ticks as the measurements units."
        }
    },
    {
        -200305,
        {
            Category::Other, ActionRequired::Warning,
            "Desired finite pulse train generation is not possible. Change the number of samples to be generated, increase the rate of the pulse train, or choose a different timebase source. Refer to the documentation for more details."
        }
    },
    {
        -200306,
        {
            Category::Configuration, ActionRequired::Warning,
            "An attempt was made to set the Samples per Channel property to a value greater than the maximum supported number."
        }
    },
    {
        -200307,
        {
            Category::Software, ActionRequired::Warning,
            "Specified master timebase rate does not match specified master timebase source. Do not set the master timebase rate when you are using an internal master timebase source. In this case, the driver sets the master timebase rate for you."
        }
    },
    {
        -200308,
        {
            Category::Software, ActionRequired::Warning,
            "Specified sample timebase rate does not match specified sample timebase source. Do not set the sample timebase rate when you are using an internal sample timebase source. In this case, the driver sets the sample timebase rate for you."
        }
    },
    {
        -200309,
        {
            Category::Software, ActionRequired::Warning,
            "Specified master timebase divisor (belonging to sample clock timebase) is not appropriate for the specified sample timebase source. Do not set the master timebase divisor when you are using an internal sample timebase source. In this case, the driver sets the master timebase divisor for you."
        }
    },
    {
        -200310,
        {
            Category::Hardware, ActionRequired::Warning,
            "The waveform you are trying to allocate on the device has been previously allocated. Make sure you are not trying to allocate the same waveform twice, or delete the existing waveform before allocating it again."
        }
    },
    {
        -200311,
        {
            Category::Other, ActionRequired::Warning,
            "You cannot write data outside the boundaries of your waveform. Make sure you are not trying to write more data than your waveform can accomodate and that your write location and write offset are set correctly."
        }
    },
    {
        -200312,
        {
            Category::Hardware, ActionRequired::Warning,
            "Waveform is not in the device memory. Make sure you are referring to a previously allocated and/or written waveform by its correct name. Also, make sure that the waveform was not deleted."
        }
    },
    {
        -200313,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200314,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200315,
        {
            Category::Hardware, ActionRequired::Warning,
            "There is not enough free device memory for your waveform. Delete waveforms or scripts not in use to free memory. If you have deleted multiple waveforms or scripts, the memory might have become fragmented. To avoid fragmentation, you can change the order in which you write/delete your waveforms and scripts."
        }
    },
    {
        -200316,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device data underflow. The device was not able to move data fast enough to keep up with the sample rate for the active script. Run the operation at a lower sample rate, or look for the following in the active script: markers might be too close together, waveforms might be too small, waits might be too short, or subsets might be too small. If you are using an external clock, the provided clock might have gone away during your generation."
        }
    },
    {
        -200317,
        {
            Category::Hardware, ActionRequired::Warning,
            "There is not enough free device memory for your script. Delete waveforms or scripts not in use to free memory. If you have deleted multiple waveforms or scripts, the memory might have become fragmented. To avoid fragmentation, you can change the order in which you write/delete your waveforms and scripts."
        }
    },
    {
        -200318,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid excitation value specified to be used for scaling with full bridge configuration. Change the excitation value if you want it to be used for scaling with full bridge configuration. Alternatively, change the bridge configuration, or do not use excitation value for scaling."
        }
    },
    {
        -200319,
        {
            Category::Other, ActionRequired::Warning,
            "Your SCXI system is not set up to perform the analog input operation on given channels. The SCXI module cabled to your digitizer cannot route AI Convert Clock from the digitizer to the other modules. To perform the desired operation with multiple SCXI modules and one digitizer, cable the digitizer to one of the modules that can route the AI Convert signal, such as the module your channels are on. After cabling the digitizer to the module, update the chassis configuration in Measurement & Automation Explorer to reflect the cabling change. For detailed information about cabling, refer to the documentation."
        }
    },
    {
        -200320,
        {
            Category::Other, ActionRequired::Warning,
            "Your SCXI system is not set up for analog input with simultaneous sample and hold on the given channels. The SCXI module cabled to your digitizer cannot route the signal needed for simultaneous sample and hold from the digitizer to the other modules. To perform the desired operation with multiple SCXI modules and one digitizer, cable the digitizer to one of the modules that can route the signal needed for simultaneous sample and hold. The simultaneous sample and hold module in your chassis is one such module. After cabling the digitizer to the module, update the chassis configuration in Measurement & Automation Explorer to reflect the cabling change. For detailed information about cabling, refer to the documentation."
        }
    },
    {
        -200321,
        {
            Category::Hardware, ActionRequired::Warning,
            "Attenuation Value conflicts with the specified AI Minimum and AI Maximum properties. The specified attenuation and AI Minimum and/or AI Maximum would cause the device to exceed the hardware limit. You should increase the Attenuation Value or adjust the AI Minimum and/or AI Maximum."
        }
    },
    {
        -200322,
        {
            Category::Other, ActionRequired::Warning,
            "Data transfer has been stopped to prevent the computer from becoming completely unresponsive. Could not transfer enough data to satisfy the data transfer requirements with Interrupts as the Data Transfer Mechanism. Reduce your Sample Clock Rate, use DMA as your Data Transfer Mechanism, or use a different Data Transfer Request Condition."
        }
    },
    {
        -200323,
        {
            Category::Hardware, ActionRequired::Warning,
            "Cannot perform a multidevice scan with Advance Trigger Type set to None. Without the advance trigger, the devices in the scan list cannot be synchronized."
        }
    },
    {
        -200324,
        {
            Category::Hardware, ActionRequired::Warning,
            "NI-DAQmx is unable to communicate with the device. Make sure the device is present in and accessible to the system, is not currently being reset, and is not reserved by another driver such as Traditional NI-DAQ (Legacy)."
        }
    },
    {
        -200325,
        {
            Category::Other, ActionRequired::Warning,
            "Reverse coefficients must be specified to scale your data using the polynomial scale."
        }
    },
    {
        -200326,
        {
            Category::Other, ActionRequired::Warning,
            "An attempt has been made to perform a route when the source and the destination are the same terminal. In many cases, such as when configuring an external clock or a counter source, you must select a PFI, PXI Trigger, or RTSI line as the source terminal."
        }
    },
    {
        -200327,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "You have specified an invalid value for dt in the waveform cluster. The value for dt must be greater than zero."
        }
    },
    {
        -200328,
        {
            Category::Hardware, ActionRequired::Warning,
            "Switch driver cannot open the topology configuration file for the switch device. A switch device cannot function without its configuration file. The configuration file is installed with the driver. The file might have been removed, renamed, or corrupted after installation. Make sure the configuration file is available to the driver at the expected location, or reinstall the product, as that will reinstall the configuration file."
        }
    },
    {
        -200329,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "An error has occurred while attempting to configure the device for an analog input acquisition. If an external master timebase is being used, make sure the source is connected and generating an appropriate clock. Otherwise, contact National Instruments Technical Support."
        }
    },
    {
        -200330,
        {
            Category::Hardware, ActionRequired::Warning,
            "An attempt has been made to use the PFI0 terminal of the device for both an analog and digital source. Use a terminal other than PFI0 as the source of your digital signal."
        }
    },
    {
        -200331,
        {
            Category::Other, ActionRequired::Warning,
            "Specified sample rate is lower than the lowest rate that can be generated using the onboard clock. The rate has been coerced to the slowest possible sample rate. For slower rates, use an external sample clock or an external sample clock timebase."
        }
    },
    {
        -200332,
        {
            Category::Hardware, ActionRequired::Warning,
            "Specified sample rate is higher than the fastest rate supported by the device."
        }
    },
    {
        -200333,
        {
            Category::Other, ActionRequired::Warning,
            "Delay from the start trigger is shorter than the shortest delay that can be generated using the onboard clock with a timebase suitable for generating the sample clock. For shorter delays, use a sample clock timebase with a higher rate, if applicable."
        }
    },
    {
        -200334,
        {
            Category::Other, ActionRequired::Warning,
            "Delay from start trigger is longer than the longest delay that can be generated using the onboard clock with a timebase suitable for generating the sample clock. For longer delays, use a slower sample clock timebase rate, if applicable."
        }
    },
    {
        -200335,
        {
            Category::Other, ActionRequired::Warning,
            "Specified AI convert rate is higher than the fastest rate possible with the current timebase."
        }
    },
    {
        -200336,
        {
            Category::Other, ActionRequired::Warning,
            "Delay from the sample clock is shorter than the shortest delay that can be generated using the onboard clock with a timebase suitable for generating the convert clock. For shorter delays, use a faster convert clock timebase rate, if applicable."
        }
    },
    {
        -200337,
        {
            Category::Other, ActionRequired::Warning,
            "Delay from the sample clock is longer than the longest delay that can be generated using the onboard clock with a timebase suitable for generating the convert clock. For longer delays, use a slower convert clock timebase rate, if applicable."
        }
    },
    {
        -200338,
        {
            Category::Hardware, ActionRequired::Warning,
            "An attempt has been made to read the calibration temperature for a device without an internal temperature sensor."
        }
    },
    {
        -200339,
        {
            Category::Other, ActionRequired::Warning,
            "Pulse width measurement was started while the input signal was active, and no additional pulses were received, which caused the measurement not to complete during the specified timeout. When measuring a single pulse width, make sure the measurement counter is started before the pulse to be measured is active, or provide a timeout sufficient for at least one additional pulse to be measured."
        }
    },
    {
        -200340,
        {
            Category::Other, ActionRequired::Warning,
            "By setting Number of Samples per Channel to -1, you indicated that all available data should be read. This is not valid for acquisitions without a buffer. Specify a value greater than or equal to zero for Number of Samples per Channel. Do not specify a value of zero for Buffer Size when configuring the input buffer."
        }
    },
    {
        -200341,
        {
            Category::Configuration, ActionRequired::Warning,
            "Generation was configured to use only onboard memory, but the corresponding buffer is larger than onboard memory. Buffer size is provided implicitly when data is written or explicitly when the buffer is configured. Configure the generation so that the Use Only Onboard Memory property is false. Alternatively, you can make sure the number of samples written and/or the size of the configured buffer do not exceed the onboard memory size."
        }
    },
    {
        -200342,
        {
            Category::Hardware, ActionRequired::Warning,
            "Script is not in the device memory. Make sure you are referring to a previously written script by its correct name. Also, make sure the script has not been deleted."
        }
    },
    {
        -200343,
        {
            Category::Software, ActionRequired::Warning,
            "Driver cannot determine the number of samples to read for a continuous task that has not yet started. Start the task explicitly, or specify the number of samples to read in DAQmx Read."
        }
    },
    {
        -200344,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested number of samples per channel is invalid. The number of samples per channel must be an integer multiple of the number of samples per channel increment."
        }
    },
    {
        -200345,
        {
            Category::Other, ActionRequired::Warning,
            "Event delay is outside of the legal range. Change the value of the delay, and/or verify that the units are correct."
        }
    },
    {
        -200346,
        {
            Category::Other, ActionRequired::Warning,
            "Event pulse width is outside of the legal range. Change the value of the pulse width, and/or verify that the units are correct."
        }
    },
    {
        -200347,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid intermediate buffer size. The size of the intermediate buffer must be an integer multiple of the intermediate buffer size increment."
        }
    },
    {
        -200348,
        {
            Category::Other, ActionRequired::Warning,
            "Scaled Values must be specified for the table scale."
        }
    },
    {
        -200349,
        {
            Category::Other, ActionRequired::Warning,
            "Prescaled Values must be specified for the table scale."
        }
    },
    {
        -200350,
        {
            Category::Other, ActionRequired::Warning,
            "Number of Prescaled Values needs to be equal to the number of Scaled Values in the table scale."
        }
    },
    {
        -200351,
        {
            Category::Other, ActionRequired::Warning,
            "Forward coefficients must be specified for the polynomial scale."
        }
    },
    {
        -200352,
        {
            Category::Other, ActionRequired::Warning,
            "Physical channel corresponding to the virtual channel specified for cold-junction compensation is already being used for a thermocouple measurement, so it cannot be used as the cold-junction compensation channel."
        }
    },
    {
        -200353,
        {
            Category::Configuration, ActionRequired::Warning,
            "Specified property value cannot be used, because it requires resources that are currently in use."
        }
    },
    {
        -200354,
        {
            Category::Configuration, ActionRequired::Warning,
            "Specified property value is not a valid terminal name."
        }
    },
    {
        -200355,
        {
            Category::Configuration, ActionRequired::Warning,
            "Specified property value cannot be used, because the hardware does not support it."
        }
    },
    {
        -200356,
        {
            Category::Other, ActionRequired::Warning,
            "Custom scale cannot be created. A saved scale with this name already exists."
        }
    },
    {
        -200357,
        {
            Category::Hardware, ActionRequired::Warning,
            "Measurement device cannot acquire data from the sensor in its current configuration. The voltage output range of your sensor does not overlap with the voltage input range of your measurement device. If your measurement device supports different gains or input ranges, try using a lower gain or a wider input range. If the device has a fixed gain/range, you might need to change sensor attribute settings such as Excitation Value or use a measurement device that supports a wider voltage input range."
        }
    },
    {
        -200358,
        {
            Category::Other, ActionRequired::Warning,
            "An attempt has been made to configure a reference trigger when the sample mode of the sample clock has been configured for continuous sampling. Reference trigger is only applicable for finite sampling. Change the sample mode to finite to use a reference trigger, or do not configure a reference trigger."
        }
    },
    {
        -200359,
        {
            Category::Other, ActionRequired::Warning,
            "Counter signals cannot be exported, because there is more than one counter channel in the task. Create separate tasks for each counter channel."
        }
    },
    {
        -200360,
        {
            Category::Other, ActionRequired::Warning,
            "CJC Source has been set to Channel, while the CJC channel has not been specified. Specify the CJC channel, or set CJC Source to a value other than Channel."
        }
    },
    {
        -200361,
        {
            Category::Hardware, ActionRequired::Warning,
            "Onboard device memory overflow. Because of system and/or bus-bandwidth limitations, the driver could not read data from the device fast enough to keep up with the device throughput. Reduce your sample rate. If your data transfer method is interrupts, try using DMA or USB Bulk. You can also use a product with more onboard memory or reduce the number of programs your computer is executing concurrently."
        }
    },
    {
        -200362,
        {
            Category::Configuration, ActionRequired::Warning,
            "The Overloaded Channels Exist property was not read prior to reading the specified property. The driver retrieves the overload state from the hardware when the application reads the Overloaded Channels Exist property. After the Overloaded Channels Exist property has been read, other information about overloaded channels may be read, such as which channels are overloaded."
        }
    },
    {
        -200363,
        {
            Category::Hardware, ActionRequired::Warning,
            "Specified inversion cannot be satisfied, because the hardware does not support it."
        }
    },
    {
        -200364,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Specified polarity is not supported by the hardware."
        }
    },
    {
        -200365,
        {
            Category::Other, ActionRequired::Warning,
            "Specified inversion cannot be satisfied, because it requires resources that are currently in use by another route."
        }
    },
    {
        -200366,
        {
            Category::Other, ActionRequired::Warning,
            "Specified inversion cannot be satisfied, because it requires resources that are currently in use by another route within this task."
        }
    },
    {
        -200367,
        {
            Category::Other, ActionRequired::Warning,
            "Specified polarity cannot be satisfied, because it requires resources that are currently in use by another route within this task."
        }
    },
    {
        -200368,
        {
            Category::Hardware, ActionRequired::Warning,
            "Specified route cannot be satisfied, because the hardware does not support it."
        }
    },
    {
        -200369,
        {
            Category::Other, ActionRequired::Warning,
            "Specified route cannot be satisfied, because it requires resources that are currently in use by another route."
        }
    },
    {
        -200370,
        {
            Category::Other, ActionRequired::Warning,
            "Specified route cannot be satisfied, because it requires resources that are currently in use by another route within this task."
        }
    },
    {
        -200371,
        {
            Category::Other, ActionRequired::Warning,
            "Requested multiple virtual channels that correspond to the same physical channel within a task. A task cannot contain multiple physical channels of a specified type. Use different physical channels for each virtual channel."
        }
    },
    {
        -200372,
        {
            Category::Software, ActionRequired::ImmediateAction,
            "Trigger type requested to be sent as the software trigger is invalid."
        }
    },
    {
        -200373,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Trigger type requested to be sent as software trigger is not supported for the specified task running on the given device."
        }
    },
    {
        -200374,
        {
            Category::Other, ActionRequired::Warning,
            "Signal type requested to be exported is not valid."
        }
    },
    {
        -200375,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Signal type requested to be exported is not supported for the specified task running on the given device."
        }
    },
    {
        -200376,
        {
            Category::Other, ActionRequired::Warning,
            "Requested creation of a separate channel for each line is not possible when a digital port is specified as the physical channel. Specify a range of digital lines, such as \"Dev1/port0/line0:7\", as the physical channel."
        }
    },
    {
        -200377,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Requested operation is not supported by the device during a scan. The device only supports operations on front-end channels (for example, ch0, ch1, ... or cjtemp) while scanning. Other operations, such as operations on analog bus channels (such as ab0 and ab1), are not supported by the device during a scan. Make sure your scan list contains only supported operations."
        }
    },
    {
        -200378,
        {
            Category::Other, ActionRequired::Warning,
            "Custom scale specified does not exist."
        }
    },
    {
        -200379,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "External clock frequency and external clock divisor values result in an invalid cutoff frequency for this device. The relationship between cutoff frequency, external clock frequency, and external clock divisor is: cutoffFreq = extClkFreq / (100 * extClkDiv) Change your external clock frequency or external clock divisor."
        }
    },
    {
        -200380,
        {
            Category::Communication, ActionRequired::Warning,
            "Strain gage calibration procedure has failed. Make sure the strain gages are connected to all the specified strain channels, the strain gage connections are appropriate for their bridge type configurations, the shunt resistor location is specified correctly, and your hardware jumpers (if any) are set up correctly."
        }
    },
    {
        -200381,
        {
            Category::Configuration, ActionRequired::Warning,
            "Unable to configure requested delay property given the current clock rate. Make sure the sample clock rate is greater than or equal to the phase shift DMC threshold of your device, or do not configure the delay. Consult your documentation for more information."
        }
    },
    {
        -200382,
        {
            Category::Other, ActionRequired::Warning,
            "Specified trigger type for pattern match mode could not be configured, because all pattern matchers are already in use."
        }
    },
    {
        -200383,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Reference Clock Source specified is invalid, given the Sample Clock Source. When Sample Clock Source is anything other than \"OnboardClock\", you must set Reference Clock Source to \"None\", and you cannot export the Reference Clock."
        }
    },
    {
        -200384,
        {
            Category::Other, ActionRequired::Warning,
            "Static output cannot be performed, because some data lines have already been reserved for a dynamic output."
        }
    },
    {
        -200385,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200386,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200387,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200388,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200389,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200390,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200391,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock or an external reference clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200392,
        {
            Category::Other, ActionRequired::Warning,
            "Specified output voltage is not valid with the given sample clock rate. Make sure your output voltage level is compatible with your sample clock rate by altering the output voltage level or the sample clock rate. Consult your documentation for more information."
        }
    },
    {
        -200393,
        {
            Category::Other, ActionRequired::Warning,
            "You are attempting to write to a read-only register."
        }
    },
    {
        -200394,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Requested value for the property is invalid, because it is not an unsigned integer. Even though the datatype of the property is a floating point number, the value must be an unsigned integer less than or equal to 9,007,199,254,740,992 (2^53)."
        }
    },
    {
        -200395,
        {
            Category::Configuration, ActionRequired::Warning,
            "There are no shared trigger lines between the two devices which are acceptable to both devices. While each of these two devices support some shared trigger lines, none of these shared trigger lines work for both devices for the specified property and corresponding value. Consider routing the signal through the I/O connectors of the two devices, if applicable."
        }
    },
    {
        -200396,
        {
            Category::Hardware, ActionRequired::Warning,
            "There are no shared trigger lines between the two devices that are acceptable to both devices. While each of the two devices support some shared trigger lines, none of the shared trigger lines work for both devices for the specified source and destination terminals. Consider routing the signal through the I/O connectors of the two devices, if applicable."
        }
    },
    {
        -200397,
        {
            Category::Other, ActionRequired::Warning,
            "Unable to load NI-DAQmx dynamic link library NICAIU.DLL. Make sure that NI-DAQmx is installed on your computer."
        }
    },
    {
        -200398,
        {
            Category::Other, ActionRequired::Warning,
            "Unable to find function in NI-DAQmx dynamic link library NICAIU.DLL. The DLL exists on your computer, but the version is incorrect. Install the correct version of the DLL on your computer."
        }
    },
    {
        -200399,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "No extended error information is available for the last error code. It is possible that there was a problem initializing the internal errors database. Please contact National Instruments Technical Support."
        }
    },
    {
        -200400,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested waveform length is invalid, because the number of samples is not an integer multiple of the waveform length increment."
        }
    },
    {
        -200401,
        {
            Category::Other, ActionRequired::Warning,
            "Number of points to compute over the range of x values is not positive. Specify a value greater than 0 for this input."
        }
    },
    {
        -200402,
        {
            Category::Other, ActionRequired::Warning,
            "Order of the reverse polynomial to compute is not positive. Specify a value greater than 0 for this input."
        }
    },
    {
        -200403,
        {
            Category::Other, ActionRequired::Warning,
            "Order of the reverse polynomial to compute is less than or equal to the number of points to compute over the range of x values. Reduce the order of the reverse polynomial or increase the number of points to compute over the range of x values."
        }
    },
    {
        -200404,
        {
            Category::Other, ActionRequired::Warning,
            "Forward and Reverse Coefficients for a polynomial scale are not specified. Each of these two sets of coefficients must contain at least one term. If only one set of coefficients is available, use the Compute Reverse Polynomial Coefficient utility to calculate the other set of coefficients."
        }
    },
    {
        -200405,
        {
            Category::Other, ActionRequired::Warning,
            "Forward Coefficients for a polynomial scale are not specified. This set of coefficients must contain at least one term. If the coefficients are not available, you can pass the supplied Reverse Coefficients to the Compute Reverse Polynomial Coefficients utility to calculate the required Forward Coefficients."
        }
    },
    {
        -200406,
        {
            Category::Other, ActionRequired::Warning,
            "Reverse Coefficients for a polynomial scale are not specified. This set of coefficients must contain at least one term. If the coefficients are not available, use the Compute Reverse Polynomial Coefficient utility to calculate the required coefficients from the supplied Forward Coefficients."
        }
    },
    {
        -200407,
        {
            Category::Other, ActionRequired::Warning,
            "Forward Coefficients for a polynomial scale are all set to zero. At least one of these coefficients must be non-zero."
        }
    },
    {
        -200408,
        {
            Category::Other, ActionRequired::Warning,
            "Reverse Coefficients for a polynomial scale are all set to zero. At least one of these coefficients must be non-zero."
        }
    },
    {
        -200409,
        {
            Category::Other, ActionRequired::Warning,
            "Slope for a linear scale is set to zero. The slope must be non-zero."
        }
    },
    {
        -200410,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Requested record number is invalid. Use the Records Done property to find out how many records are available. Record numbers start at 0. Use -1 for all available records."
        }
    },
    {
        -200411,
        {
            Category::Hardware, ActionRequired::Warning,
            "AC coupling is not allowed with 50 Ohm impedance for this device. Use DC coupling, or configure a different impedance setting."
        }
    },
    {
        -200412,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested analog input attenuation is invalid."
        }
    },
    {
        -200413,
        {
            Category::Other, ActionRequired::Warning,
            "Insufficient onboard memory for requested Number of Records and Samples per Channel combination. Reduce the Number of Records and/or Samples per Channel."
        }
    },
    {
        -200414,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested sample clock source is invalid."
        }
    },
    {
        -200415,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested reference clock source is invalid."
        }
    },
    {
        -200416,
        {
            Category::Other, ActionRequired::Warning,
            "Multiple records are not available with RIS."
        }
    },
    {
        -200417,
        {
            Category::Other, ActionRequired::Warning,
            "TDC is not enabled during RIS mode. TDC must be enabled when the digitizer is in the RIS mode. Enable TDC, or do not use RIS."
        }
    },
    {
        -200418,
        {
            Category::Other, ActionRequired::Warning,
            "Requested immediate trigger type while in RIS mode. Immediate triggering is not compatible with the RIS mode. Select a different trigger type, or do not use RIS."
        }
    },
    {
        -200419,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested Read Position is invalid in RIS mode."
        }
    },
    {
        -200420,
        {
            Category::Other, ActionRequired::Warning,
            "Requested sample rate exceeds maximum real-time sample rate. If you want a higher sampling rate and have a repetitive signal, enable RIS."
        }
    },
    {
        -200421,
        {
            Category::Other, ActionRequired::Warning,
            "Requested Hysteresis is not valid with the configured Trigger Level and AI Minimum. Configure the task so the following formula is satisfied: Trigger Level - Hysteresis > AI Minimum"
        }
    },
    {
        -200422,
        {
            Category::Other, ActionRequired::Warning,
            "Requested Trigger Level is not valid with the configured AI Minimum and AI Maximum. Configure the instrument so the following formula is satisfied: AI Minimum < Trigger Level < AI Maximum"
        }
    },
    {
        -200423,
        {
            Category::Other, ActionRequired::Warning,
            "Requested window trigger level is not valid with the configured AI Minimum and AI Maximum. Configure the instrument so the following formula is satisfied: AI Minimum < Window Trigger Level < AI Maximum"
        }
    },
    {
        -200424,
        {
            Category::Other, ActionRequired::Warning,
            "Requested video trigger line number is incompatible with the chosen video signal format."
        }
    },
    {
        -200425,
        {
            Category::Other, ActionRequired::Warning,
            "Requested hysteresis is not valid with the configured trigger level and AI Maximum. Configure the task so the following formula is satisfied: Trigger Level + Hysteresis < AI Maximum"
        }
    },
    {
        -200426,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested impedance for the external trigger is invalid. Specify an impedance that is appropriate for the external trigger, or choose a different trigger source."
        }
    },
    {
        -200427,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Configured reference clock rate is invalid. The reference clock rate must be within the valid range and a multiple of the increment value."
        }
    },
    {
        -200428,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Value passed to the Task/Channels In control is invalid. The value must refer to a valid task or valid virtual channels."
        }
    },
    {
        -200429,
        {
            Category::Other, ActionRequired::Warning,
            "Value passed to the Task/Channels In control is an empty string (or I/O control). The value must refer to a valid task or valid channels."
        }
    },
    {
        -200430,
        {
            Category::Other, ActionRequired::Warning,
            "I/O type of the physical channel does not match the I/O type required for the virtual channel you are creating."
        }
    },
    {
        -200431,
        {
            Category::Other, ActionRequired::Warning,
            "Selected physical channel does not support the measurement type required by the virtual channel you are creating. Create a channel of a measurement type that is supported by the physical channel, or select a physical channel that supports the measurement type."
        }
    },
    {
        -200432,
        {
            Category::Other, ActionRequired::Warning,
            "Selected physical channel does not support the output type required by the virtual channel you are creating. Create a channel of an output type that is supported by the physical channel, or select a physical channel that supports the output type."
        }
    },
    {
        -200433,
        {
            Category::Other, ActionRequired::Warning,
            "Scaled Values for a table scale must contain at least two values."
        }
    },
    {
        -200434,
        {
            Category::Other, ActionRequired::Warning,
            "Prescaled Values for a table scale must contain at least two values."
        }
    },
    {
        -200435,
        {
            Category::Other, ActionRequired::Warning,
            "Delay from sample clock is not available when an external convert source is specified. Change the convert source to onboard clock, or do not configure the delay from sample clock."
        }
    },
    {
        -200436,
        {
            Category::Other, ActionRequired::Warning,
            "Start trigger delay is not available when an external sample clock source is specified. Change the sample clock to onboard clock, or do not configure the start trigger delay."
        }
    },
    {
        -200437,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "External calibration constants are invalid. Perform an external calibration. Contact National Instruments Technical Support if you need additional information."
        }
    },
    {
        -200438,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid calibration area selected. Select self-calibration or external calibration."
        }
    },
    {
        -200439,
        {
            Category::Other, ActionRequired::Warning,
            "Requested operation only can be used during an external calibration session."
        }
    },
    {
        -200440,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested calibration close action is invalid. Select Store or Abort."
        }
    },
    {
        -200441,
        {
            Category::Other, ActionRequired::Warning,
            "Unable to detect the external stimulus frequency. Verify that the external stimulus is properly connected and has the correct frequency and amplitude."
        }
    },
    {
        -200442,
        {
            Category::Other, ActionRequired::Warning,
            "Unable to synchronize to the external stimulus frequency. Verify that the external stimulus has the correct frequency, amplitude, and stability. Consult the documentation for the calibration procedure for valid ranges."
        }
    },
    {
        -200443,
        {
            Category::Other, ActionRequired::Warning,
            "Attempt to store calibration constants without completing all the necessary external calibration steps. Refer to the documentation for the calibration procedure. Verify that all necessary steps are performed before closing the external calibration session."
        }
    },
    {
        -200444,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid physical channel selected for calibration."
        }
    },
    {
        -200445,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Requested calibration function is not supported by the device."
        }
    },
    {
        -200446,
        {
            Category::Other, ActionRequired::Warning,
            "External stimulus voltage read was outside the expected range. Verify that the external stimulus voltage is properly connected and has the correct amplitude."
        }
    },
    {
        -200447,
        {
            Category::Other, ActionRequired::Warning,
            "Units for the channel must be set to From Custom Scale when a custom scale is used with a channel."
        }
    },
    {
        -200448,
        {
            Category::Configuration, ActionRequired::Warning,
            "DAC Range High is not equal to the DAC Reference Voltage Value. When you do not set the DAC Range High property, the driver always makes sure it is equal to the DAC Reference Voltage value. If you do set the DAC Range High property make sure DAC Range High and DAC Reference Voltage Value are equal."
        }
    },
    {
        -200449,
        {
            Category::Configuration, ActionRequired::Warning,
            "DAC Range Low must be equal to either the negative DAC Reference Voltage Value or to zero. If you do not set the DAC Range Low property, the driver sets it for you. Otherwise, make sure DAC Range Low is equal to either the negative DAC Reference Voltage Value or to zero."
        }
    },
    {
        -200450,
        {
            Category::Configuration, ActionRequired::Warning,
            "Specified property cannot be set while the task is running."
        }
    },
    {
        -200451,
        {
            Category::Configuration, ActionRequired::Warning,
            "You can get the specified property only while the task is committed or running."
        }
    },
    {
        -200452,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Specified property is not supported by the device or is not applicable to the task."
        }
    },
    {
        -200453,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified timeout value is not supported. Supported timeout values are 0 (try or check once and return), positive numbers up to 4294967, and -1 (try or check until success or error)."
        }
    },
    {
        -200454,
        {
            Category::Configuration, ActionRequired::Warning,
            "You cannot get the specified property, because the task is not a buffered output task."
        }
    },
    {
        -200455,
        {
            Category::Configuration, ActionRequired::Warning,
            "You cannot get the specified property, because the task is not a buffered input task."
        }
    },
    {
        -200456,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Specified property is not supported, because the task is not an output task."
        }
    },
    {
        -200457,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Specified property is not supported, because the task is not an input task."
        }
    },
    {
        -200459,
        {
            Category::Other, ActionRequired::Warning,
            "Write failed, because there are no output channels in this task to which data can be written."
        }
    },
    {
        -200460,
        {
            Category::Other, ActionRequired::Warning,
            "Read failed, because there are no channels in this task from which data can be read."
        }
    },
    {
        -200461,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified channel name is invalid."
        }
    },
    {
        -200462,
        {
            Category::Configuration, ActionRequired::Warning,
            "Generation cannot be started because the output buffer is empty. Write data before starting a buffered generation. The following actions can empty the buffer: changing the size of the buffer, unreserving a task, setting the Regeneration Mode property, changing the Sample Mode, or configuring retriggering."
        }
    },
    {
        -200463,
        {
            Category::Other, ActionRequired::Warning,
            "Specified read or write operation failed, because the number of lines in the data for a channel does not match the number of lines in the channel. If you are using the Digital Waveform datatype, make sure the number of lines in the digital waveform matches the number of lines in the channel. If you are using boolean data, make sure the array dimension for lines in the data matches the number of lines in the channel."
        }
    },
    {
        -200464,
        {
            Category::Other, ActionRequired::Warning,
            "If performing a Write operation, the operation cannot be performed, because the number of channels in the specified data does not match the number of channels in this task. Adjust the data to match the number of channels in this task. If you are performing a read operation, the operation cannot be performed because this DAQmx Read only returns data from a single channel, and there are multiple channels in this task. Use the multichannel DAQmx Read."
        }
    },
    {
        -200465,
        {
            Category::Other, ActionRequired::Warning,
            "Specified digital channel contains more bits than what is supported by DAQmx Port Write. With the U8 version, channels must contain 8 bits or less; while for the U32 version, channels must contain 32 bits or less."
        }
    },
    {
        -200466,
        {
            Category::Other, ActionRequired::Warning,
            "Specified digital channel contains more bits than what is supported by DAQmx Port Read. With the U8 version, channels must contain 8 bits or less; while for the U32 version, channels must contain 32 bits or less."
        }
    },
    {
        -200467,
        {
            Category::Other, ActionRequired::Warning,
            "Specified string is not valid, because it is empty."
        }
    },
    {
        -200469,
        {
            Category::Other, ActionRequired::Warning,
            "Specified channel cannot be loaded, because it was saved with an incompatible, more recent version of NI-DAQ. Create the channel again, or upgrade NI-DAQ to a version compatible with the version used to save the channel. Consult the documentation for the version of NI-DAQ used to create the channel for more details."
        }
    },
    {
        -200470,
        {
            Category::Other, ActionRequired::Warning,
            "Specified task cannot be loaded, because it was saved with an incompatible, more recent version of NI-DAQ. Create the task again, or upgrade NI-DAQ to a version compatible with the version used to save the task. Consult the documentation for the version of NI-DAQ used to create the task for more details."
        }
    },
    {
        -200472,
        {
            Category::Other, ActionRequired::Warning,
            "Write cannot be performed when the auto start input to DAQmx Write is false, and timing for the task is not configured or Timing Type is set to On Demand. Set auto start to true, or configure timing and specify Timing Type other than On Demand."
        }
    },
    {
        -200473,
        {
            Category::Configuration, ActionRequired::Warning,
            "Read cannot be performed when the Auto Start property is false and the task is not running or committed. Start the task before reading, or set Auto Start to true."
        }
    },
    {
        -200474,
        {
            Category::Other, ActionRequired::Warning,
            "Specified operation did not complete, because the specified timeout expired."
        }
    },
    {
        -200475,
        {
            Category::Other, ActionRequired::Warning,
            "Specified operation can be performed only when the task is running. Start the task before requesting this operation."
        }
    },
    {
        -200477,
        {
            Category::Hardware, ActionRequired::Warning,
            "Specified operation cannot be performed when there are no devices in the task."
        }
    },
    {
        -200478,
        {
            Category::Other, ActionRequired::Warning,
            "Specified operation cannot be performed when there are no channels in the task."
        }
    },
    {
        -200479,
        {
            Category::Other, ActionRequired::Warning,
            "Specified operation cannot be performed while the task is running."
        }
    },
    {
        -200481,
        {
            Category::Hardware, ActionRequired::Warning,
            "Specified device cannot be added to the task, because it is already in the task."
        }
    },
    {
        -200482,
        {
            Category::Hardware, ActionRequired::Warning,
            "Specified device is not in the task."
        }
    },
    {
        -200483,
        {
            Category::Other, ActionRequired::Warning,
            "Specified virtual channel cannot be saved, because a virtual channel with that name already exists in Data Neighborhood in MAX. Save the virtual channel under a different name, or specify that the existing virtual channel should be replaced."
        }
    },
    {
        -200484,
        {
            Category::Other, ActionRequired::Warning,
            "Specified task cannot be saved, because a task with that name already exists in Data Neighborhood in MAX. Save the task under a different name, or specify that the existing task should be replaced."
        }
    },
    {
        -200485,
        {
            Category::Other, ActionRequired::Warning,
            "The specified task cannot be loaded, because it is not in Data Neighborhood. Check Data Neighborhood in MAX. Look for similar characters, such as the capital letter \"O\" and the number zero."
        }
    },
    {
        -200486,
        {
            Category::Other, ActionRequired::Warning,
            "Specified channel is not in the task."
        }
    },
    {
        -200488,
        {
            Category::Other, ActionRequired::Warning,
            "Specified virtual channel cannot be added to the task, because it does not exist. You cannot specify a physical channel. Instead, create a virtual channel using the DAQ Assistant or DAQmx Create Virtual Channel, and then add the virtual channel to the task."
        }
    },
    {
        -200489,
        {
            Category::Other, ActionRequired::Warning,
            "Specified channel cannot be added to the task, because a channel with the same name is already in the task."
        }
    },
    {
        -200490,
        {
            Category::Other, ActionRequired::Warning,
            "Sample value detected outside of the specified range."
        }
    },
    {
        -200491,
        {Category::Other, ActionRequired::Warning, "Reserved parameter must be zero."}
    },
    {
        -200492,
        {Category::Other, ActionRequired::Warning, "Reserved parameter must be NULL."}
    },
    {
        -200493,
        {
            Category::Other, ActionRequired::Warning,
            "Reserved character string parameter must be NULL or an empty string."
        }
    },
    {
        -200494,
        {
            Category::Hardware, ActionRequired::Warning,
            "Specified task cannot be loaded, because it requires a device that supports timing, and the associated device does not support timing. Create a new task without timing, or associate this task with a device that supports timing."
        }
    },
    {
        -200495,
        {
            Category::Software, ActionRequired::Warning,
            "An intermediate acquisition buffer has overflowed. The driver was unable to read samples from the intermediate buffer fast enough to prevent the buffer from overflowing."
        }
    },
    {
        -200496,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Pattern contains an invalid character."
        }
    },
    {
        -200497,
        {
            Category::Other, ActionRequired::Warning,
            "Attempted to enable output data lines that were not previously disabled. Make sure that you are enabling data lines for output only after explicitly disabling them."
        }
    },
    {
        -200498,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Syntax for a range of objects in the input string is invalid. For ranges of objects, specify a number immediately before and after every colon (\":\") in the input string. Or, if a name is specified after the colon, it must be identical to the name specified immediately before the colon. Colons are not allowed within the names of the individual objects."
        }
    },
    {
        -200499,
        {
            Category::Other, ActionRequired::Warning,
            "Value passed is not between 0 and 4,294,967,295 (unsigned 32-bit integer)."
        }
    },
    {
        -200500,
        {
            Category::Software, ActionRequired::ImmediateAction,
            "Error code could not be found. Reinstalling the driver might fix the issue. Otherwise, contact National Instruments technical support."
        }
    },
    {
        -200501,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "EEPROM contains an invalid calibration date and/or time. The calibration data in the EEPROM might have been corrupted. Perform an external calibration, perform a self-calibration, or contact National Instruments Technical Support."
        }
    },
    {
        -200502,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "A measurement taken during adjustment of main path pre-amplifier offset produced an invalid calibration constant. Make sure that the measured values passed to the calibration VI or function are correct. Verify the calibration procedure, and repeat the calibration. If the problem persists, there might be a hardware malfunction with your device. Contact National Instruments Technical Support."
        }
    },
    {
        -200503,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "A measurement taken during adjustment of main path pre-amplifier gain produced an invalid calibration constant. Make sure that the measured values passed to the calibration VI or function are correct. Verify the calibration procedure, and repeat the calibration. If the problem persists, there might be a hardware malfunction with your device. Contact National Instruments Technical Support."
        }
    },
    {
        -200504,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "A measurement taken during adjustment of main path post-amplifier gain and offset produced an invalid calibration constant. Make sure that the measured values passed to the calibration VI or function are correct. Verify the calibration procedure, and repeat the calibration. If the problem persists, there might be a hardware malfunction with your device. Contact National Instruments Technical Support."
        }
    },
    {
        -200505,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "A measurement taken during adjustment of direct path gain produced an invalid calibration constant. Make sure that the measured values passed to the calibration VI or function are correct. Verify the calibration procedure, and repeat the calibration. If the problem persists, there might be a hardware malfunction with your device. Contact National Instruments Technical Support."
        }
    },
    {
        -200506,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "A measurement taken during adjustment of main path output impedance produced an invalid calibration constant. Make sure that the measured values passed to the calibration VI or function are correct. Verify the calibration procedure, and repeat the calibration. If the problem persists, there might be a hardware malfunction with your device. Contact National Instruments Technical Support."
        }
    },
    {
        -200507,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "A measurement taken during adjustment of direct path output impedance produced an invalid calibration constant. Make sure that the measured values passed to the calibration VI or function are correct. Verify the calibration procedure, and repeat the calibration. If the problem persists, there might be a hardware malfunction with your device. Contact National Instruments Technical Support."
        }
    },
    {
        -200508,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "A measurement taken during adjustment of the oscillator frequency produced an invalid calibration constant. Make sure that the measured values passed to the calibration VI or function are correct. Verify the calibration procedure, and repeat the calibration. If the problem persists, there might be a hardware malfunction with your device. Contact National Instruments Technical Support."
        }
    },
    {
        -200509,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "A measurement taken during adjustment of calibration ADC produced an invalid calibration constant. Make sure that the measured values passed to the calibration VI or function are correct. Verify the calibration procedure, and repeat the calibration. If the problem persists, there might be a hardware malfunction with your device. Contact National Instruments Technical Support."
        }
    },
    {
        -200510,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested an invalid configuration value for adjusting the main path pre-amplifier offset. Refer to the documentation for a list of valid configuration values."
        }
    },
    {
        -200511,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested an invalid configuration value for adjusting the main path pre-amplifier gain. Refer to the documentation for a list of valid configuration values."
        }
    },
    {
        -200512,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested an invalid configuration value for adjusting the main path post-amplifier gain and offset. Refer to the documentation for a list of valid configuration values."
        }
    },
    {
        -200513,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested an invalid configuration value for adjusting the main path output impedance. Refer to the documentation for a list of valid configuration values."
        }
    },
    {
        -200514,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested an invalid configuration value for adjusting the direct path output impedance. Refer to the documentation for a list of valid configuration values."
        }
    },
    {
        -200515,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified number of reads to average from the calibration ADC is invalid. The number of reads to average must be greater than 0."
        }
    },
    {
        -200516,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Calibration constants stored in the EEPROM produced an invalid value for the gain DAC. An incorrect calibration might have been performed, or the calibration data in the EEPROM might have been corrupted. Perform an external calibration, perform a self-calibration, or contact National Instruments Technical Support."
        }
    },
    {
        -200517,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Calibration constants stored in the EEPROM produced an invalid value for the offset DAC. An incorrect calibration might have been performed, or the calibration data in the EEPROM might have been corrupted. Perform an external calibration, or a self-calibration, or contact National Instruments Technical Support."
        }
    },
    {
        -200518,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Calibration constants stored in the EEPROM produced an invalid value for the oscillator phase DAC. An incorrect calibration might have been performed, or the calibration data in the EEPROM might have been corrupted. Perform an external calibration, perform a self-calibration, or contact National Instruments Technical Support."
        }
    },
    {
        -200519,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Calibration constants stored in the EEPROM produced an invalid value for the oscillator frequency DAC. An incorrect calibration might have been performed, or the calibration data in the EEPROM might have been corrupted. Perform an external calibration, or contact National Instruments Technical Support."
        }
    },
    {
        -200520,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Calibration constants stored in the EEPROM and used to adjust reads from the cal ADC are invalid. An incorrect calibration might have been performed, or the calibration data in the EEPROM might have been corrupted. Perform an external calibration, or contact National Instruments Technical Support."
        }
    },
    {
        -200521,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "A measurement taken during adjustment of the oscillator phase DAC produced an invalid calibration constant. Make sure that the measured values passed to the calibration VI or function are correct. Verify the calibration procedure, and repeat the calibration. If the problem persists, there might be a hardware malfunction with your device. Contact National Instruments Technical Support."
        }
    },
    {
        -200522,
        {
            Category::Other, ActionRequired::Warning,
            "A self-calibration cannot be performed during external calibration."
        }
    },
    {
        -200523,
        {
            Category::Other, ActionRequired::Warning,
            "Read cannot be performed because this version of DAQmx Read only returns data from a single channel, and there are multiple channels in the task. Use the multichannel version of DAQmx Read."
        }
    },
    {
        -200524,
        {
            Category::Other, ActionRequired::Warning,
            "Write cannot be performed, because the number of channels in the data does not match the number of channels in the task. When writing, supply data for all channels in the task. Alternatively, modify the task to contain the same number of channels as the data written."
        }
    },
    {
        -200525,
        {
            Category::Other, ActionRequired::Warning,
            "Read cannot be performed because this version of DAQmx Read does not match the type of channels in the task. Use the version of DAQmx Read that corresponds to the channel type."
        }
    },
    {
        -200526,
        {
            Category::Other, ActionRequired::Warning,
            "Write cannot be performed because this version of DAQmx Write does not match the type of channels in the task. Use the version of DAQmx Write that corresponds to the channel type."
        }
    },
    {
        -200527,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Requested values of the Minimum and Maximum properties for the counter channel are not supported for the given type of device. The values that can be specified for Minimum and Maximum depend on the counter timebase rate."
        }
    },
    {
        -200528,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested line grouping is invalid. Either choose one channel for all lines or one channel for each line as the line grouping."
        }
    },
    {
        -200529,
        {
            Category::Communication, ActionRequired::Warning,
            "Unexpected identifier following the switch operation in the connection list. Switch operations must be separated by a comma inside the connection list."
        }
    },
    {
        -200530,
        {
            Category::Other, ActionRequired::Warning,
            "Unexpected identifier following the relay name in the relay list. Relay names must be separated by a comma inside the relay list."
        }
    },
    {
        -200531,
        {
            Category::Other, ActionRequired::Warning,
            "Relay name is not specified in the list entry."
        }
    },
    {
        -200532,
        {
            Category::Other, ActionRequired::Warning,
            "Unexpected identifier following switch channel name."
        }
    },
    {
        -200533,
        {
            Category::Other, ActionRequired::Warning,
            "Identifier found in the script is too long. Use identifiers with no more than 511 characters."
        }
    },
    {
        -200534,
        {
            Category::Other, ActionRequired::Warning,
            "Waveform name is too long. Use waveform names with no more than 511 characters."
        }
    },
    {
        -200535,
        {
            Category::Configuration, ActionRequired::Warning,
            "Specified value is larger than the maximum value supported for this property."
        }
    },
    {
        -200536,
        {
            Category::Configuration, ActionRequired::Warning,
            "Specified value is smaller than the minimum value supported for this property."
        }
    },
    {
        -200537,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Supplied forward and reverse coefficients yield inconsistent results when they are used for computations related to this property. In other words, using the result of the forward scale as input to the reverse scale does not yield the original data. Based on the forward coefficients, the value for the property is invalid. Supply forward and reverse coefficients that yield consistent results."
        }
    },
    {
        -200538,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Action requested is invalid."
        }
    },
    {
        -200539,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device does not support analog writes with multiple samples per channel. To output multiple samples, call DAQmx Analog Single Sample Write multiple times."
        }
    },
    {
        -200540,
        {
            Category::Configuration, ActionRequired::Warning,
            "DAC Reference Voltage Value is not set. When the DAC Reference Voltage Source property for a channel is set to External, the DAC Reference Voltage Value property must be set. Set the DAC Reference Voltage Value property so the value matches the reference voltage source connected to your device. Alternatively, consider using the internal DAC reference voltage source available on the device."
        }
    },
    {
        -200541,
        {
            Category::Hardware, ActionRequired::Warning,
            "Last Self Calibration Date/Time has not been stored on the device by NI-DAQmx. Self-calibrate the board using NI-DAQmx. Alternatively, externally calibrate the board using NI-DAQmx, and then call DAQmx Restore Last External Calibration Constants."
        }
    },
    {
        -200542,
        {
            Category::Hardware, ActionRequired::Warning,
            "Last Self Calibration Temperature has not been stored on the device by NI-DAQmx. Self-calibrate the board using NI-DAQmx. Alternatively, externally calibrate the board using NI-DAQmx, and then call DAQmx Restore Last External Calibration Constants."
        }
    },
    {
        -200543,
        {
            Category::Hardware, ActionRequired::Warning,
            "Last External Calibration Date/Time has not been stored on the device by NI-DAQmx. Externally calibrate the board using NI-DAQmx."
        }
    },
    {
        -200544,
        {
            Category::Hardware, ActionRequired::Warning,
            "Last External Calibration Temperature has not been stored on the device by NI-DAQmx. Externally calibrate the board using NI-DAQmx."
        }
    },
    {
        -200545,
        {
            Category::Hardware, ActionRequired::Warning,
            "Self-calibration failed. The self-calibration date has not changed. Disconnect the device from external signals, as they might introduce noise. Externally calibrate the device to recalibrate the onboard voltage reference that is used for self-calibration."
        }
    },
    {
        -200546,
        {
            Category::Hardware, ActionRequired::Warning,
            "External calibration failed, and the external calibration data has not been changed. Make sure that the reference signal used for the calibration is stable and that the voltage matches the value specified to the calibration software. Disconnect the device from any external signals that might be introducing noise."
        }
    },
    {
        -200547,
        {
            Category::Other, ActionRequired::Warning,
            "DAQmx Write failed, because a previous DAQmx Write automatically configured the output buffer size. The buffer size is equal to the original number of samples written per channel, so no more data can be written prior to starting the task. Start the generation before the second DAQmx Write, or set Auto Start to true in all occurences of DAQmx Write. To incrementally write into the buffer prior to starting the task, call DAQmx Configure Output Buffer before the first DAQmx Write."
        }
    },
    {
        -200548,
        {
            Category::Other, ActionRequired::Warning,
            "Requested coupling type is only valid when the trigger source is an external trigger channel."
        }
    },
    {
        -200549,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Self-calibration constants are invalid. Perform a self-calibration. Contact National Instruments Technical Support if you need additional information."
        }
    },
    {
        -200550,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external reference or sample clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that its rate matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200551,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external reference or sample clock, make sure it is connected and within the jitter and voltage level specifications at all times. Also, verify that its rate matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200552,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified string is not valid, because it contains an invalid character."
        }
    },
    {
        -200553,
        {
            Category::Other, ActionRequired::Warning,
            "Specified string is not valid, because its first character is a space character."
        }
    },
    {
        -200554,
        {
            Category::Other, ActionRequired::Warning,
            "Specified string is not valid, because its last character is a space character."
        }
    },
    {
        -200555,
        {
            Category::Other, ActionRequired::Warning,
            "Specified string is not valid, because its first character is an underscore."
        }
    },
    {
        -200556,
        {
            Category::Configuration, ActionRequired::Warning,
            "You only can get the specified property while the task is committed or while the task is running. Commit or start the task prior to getting the property."
        }
    },
    {
        -200557,
        {
            Category::Configuration, ActionRequired::Warning,
            "Specified property cannot be set while the task is running. Set the property prior to starting the task, or stop the task prior to setting the property."
        }
    },
    {
        -200558,
        {
            Category::Hardware, ActionRequired::Warning,
            "One task cannot contain multiple independent devices. Create one task for each independent device."
        }
    },
    {
        -200559,
        {
            Category::Other, ActionRequired::Warning,
            "Task cannot contain a channel with the specified channel type, because the task already contains channels with a different channel type. Create one task for each channel type."
        }
    },
    {
        -200560,
        {
            Category::Communication, ActionRequired::Warning,
            "Wait Until Done did not indicate that the task was done within the specified timeout. Increase the timeout, check the program, and make sure connections for external timing and triggering are in place."
        }
    },
    {
        -200561,
        {
            Category::Other, ActionRequired::Warning,
            "Attempted writing analog data that is too large or too small. Change Minimum Value and Maximum Value to reflect the range of the channel."
        }
    },
    {
        -200562,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Attempted writing digital data that is not supported."
        }
    },
    {
        -200563,
        {
            Category::Other, ActionRequired::Warning,
            "Specified digital channel contains more bits than supported by the 8-bit version of DAQmx Port Read. Use a version of DAQmx Port Read that supports wider digital ports."
        }
    },
    {
        -200564,
        {
            Category::Other, ActionRequired::Warning,
            "Specified digital channel contains more bits than supported by the 32-bit version of DAQmx Read. Use a version of DAQmx Read that returns an array of Boolean values or digital waveforms."
        }
    },
    {
        -200565,
        {
            Category::Other, ActionRequired::Warning,
            "Specified digital channel contains more bits than supported by the 8-bit version of DAQmx Port Write. Use the version of DAQmx Port Write that supports wider digital ports."
        }
    },
    {
        -200566,
        {
            Category::Other, ActionRequired::Warning,
            "Specified digital channel contains more bits than supported by the 32-bit version of DAQmx Port Write."
        }
    },
    {
        -200567,
        {
            Category::Other, ActionRequired::Warning,
            "Generation cannot be started, because the buffer size was changed since the last write, and this change caused data to be lost. Write data after changing the buffer size."
        }
    },
    {
        -200568,
        {
            Category::Configuration, ActionRequired::Warning,
            "Generation cannot be started, because the Regeneration Mode property was changed since the last write, and this change caused data to be lost. Write data after changing the Regeneration Mode property."
        }
    },
    {
        -200569,
        {
            Category::Other, ActionRequired::Warning,
            "Type of channel collection used to access the specified channel does not match the channel type. Access the channel through the channel collection that matches the channel type."
        }
    },
    {
        -200570,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested channel index is invalid. The value of the index must be between one and the number of channels in the task."
        }
    },
    {
        -200571,
        {
            Category::Configuration, ActionRequired::Warning,
            "Input Source Select property is set to an internal source for channels in different channel groups. On this device, only one channel group at a time can be configured to use an internal source. Refer to the documentation for details."
        }
    },
    {
        -200572,
        {
            Category::Configuration, ActionRequired::Warning,
            "Input Source Select property is set differently for channels in one channel group on a device that supports only identical settings within a channel group. Refer to the documentation for more details."
        }
    },
    {
        -200573,
        {
            Category::Configuration, ActionRequired::Warning,
            "Input Source Select property is set to different internal sources on some channels. On this device, Input Source Select must be set to the same value for all channels with internal sources. Refer to the documentation for more details."
        }
    },
    {
        -200574,
        {
            Category::Hardware, ActionRequired::Warning,
            "Driver cannot complete the route, because the only way to make the route requires a trigger bus line, and no trigger bus has been configured in MAX for this device. If you have a PXI chassis, make sure it has been properly identified in MAX. If you are using a PCI device, create a RTSI cable in MAX that includes your PCI device even if you are not using any RTSI cables."
        }
    },
    {
        -200575,
        {
            Category::Other, ActionRequired::Warning,
            "Requested value for Samples per Channel is too high when a reference trigger is used. In this case, Samples per Channel cannot exceed the sum of Pretrigger Samples per Channel and the maximum Post-trigger Samples per Channel. Reduce Samples per Channel. Alternatively, consider performing an acquisition with Continuous Sample Mode, or increase the Pretrigger Samples per Channel."
        }
    },
    {
        -200576,
        {
            Category::Hardware, ActionRequired::Warning,
            "CJC Source cannot be set to Built-In for the specified thermocouple channel. The physical channel does not support a built-in CJC temperature sensor. If your hardware contains a CJC temperature sensor on the physical channel corresponding to the built-in CJC source, make sure that the hardware configuration (including any accessories and/or terminal blocks) is correct. Alternatively, specify a different CJC Source, or use hardware with a built-in CJC temperature sensor."
        }
    },
    {
        -200577,
        {
            Category::Other, ActionRequired::Warning,
            "Requested AI Minimum value is too large."
        }
    },
    {
        -200578,
        {
            Category::Other, ActionRequired::Warning,
            "Requested AI Minimum value is too small."
        }
    },
    {
        -200579,
        {
            Category::Other, ActionRequired::Warning,
            "Requested AI Maximum value is too large."
        }
    },
    {
        -200580,
        {
            Category::Other, ActionRequired::Warning,
            "Requested AI Maximum value is too small."
        }
    },
    {
        -200581,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Specified property is not supported, because Reference Clock Source is \"None\"."
        }
    },
    {
        -200582,
        {
            Category::Hardware, ActionRequired::Warning,
            "Values for AO channel properties lead to an output voltage that exceeds the maximum for the device."
        }
    },
    {
        -200583,
        {
            Category::Hardware, ActionRequired::Warning,
            "Values of the AO channel properties lead to an output voltage that is less than the minimum for the device."
        }
    },
    {
        -200584,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Write failed, because the number of samples to write per channel is invalid. The number of samples to write per channel must be an integer multiple of the samples to write per channel increment."
        }
    },
    {
        -200585,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock, make sure it is connected and within specifications. If you are generating your sample clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200586,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock, make sure it is connected and within specifications. If you are generating your sample clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200587,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Requested operation could not be performed, because the specified digital lines are either reserved or the device is not present in NI-DAQmx. It is possible that these lines are reserved by another task, the device is being used through the Traditional NI-DAQ interface, or the device is being reset. You might also get the error if the specified resource is currently in use by LabVIEW network variables bound to the DAQ Channel, or if the DAQ Channel is being used in any OPC Client software. If you are using these lines with another task, wait for the task to complete. If you are using the device through the Traditional NI-DAQ interface, and you want to use it with NI-DAQmx, reset (initialize) the device using the Traditional NI-DAQ interface. If you are resetting the device, wait for the reset to finish."
        }
    },
    {
        -200588,
        {
            Category::Other, ActionRequired::Warning,
            "Specified event handler cannot be removed, because it is installed on a different NI-DAQmx object. Remove the event handler from the NI-DAQmx object on which it was installed."
        }
    },
    {
        -200589,
        {
            Category::Other, ActionRequired::Warning,
            "Specified event handler cannot be removed, because it has already been removed."
        }
    },
    {
        -200590,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified event handler cannot be removed, because it is invalid. It has never been installed on this or any other NI-DAQmx object."
        }
    },
    {
        -200591,
        {
            Category::Other, ActionRequired::Warning,
            "Negative buffer size was supplied. The buffer size must be zero or greater."
        }
    },
    {
        -200592,
        {
            Category::Other, ActionRequired::Warning,
            "Given range in the input string contains too many objects. Check the string. If necessary, split the input string into smaller ranges."
        }
    },
    {
        -200593,
        {
            Category::Configuration, ActionRequired::Warning,
            "Value of this property cannot be determined until the containing task is verified. Before attempting to get the value of this property, you must make sure the task has been verified. You can do this by starting the task, using the task control method to verify the task, reading from the task if the Read Auto Start property is true, or writing to the task and specifying true for the auto start parameter."
        }
    },
    {
        -200594,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Value passed for the direction parameter is invalid. Use one of the values of the corresponding enumeration."
        }
    },
    {
        -200595,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid identifier at the beginning of the switch operation in the list entry."
        }
    },
    {
        -200596,
        {
            Category::Hardware, ActionRequired::Warning,
            "Channels in the switch operation span different devices."
        }
    },
    {
        -200597,
        {
            Category::Other, ActionRequired::Warning,
            "Specified output operation cannot be satisfied, because it requires lines that are currently in use by another output operation."
        }
    },
    {
        -200598,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Repetition of a number in the Prescaled Values is invalid for input operations."
        }
    },
    {
        -200599,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Repetition of a number in Scaled Values is invalid for output operations."
        }
    },
    {
        -200600,
        {
            Category::Other, ActionRequired::Warning,
            "NI-DAQmx cannot generate virtual channel names for some of the physical channels specified, because the numeric suffix of the resulting channel names would be too large. Either explicitly specify a virtual channel name for each physical channel name, or decrease the numeric suffix of the last set of virtual channel names."
        }
    },
    {
        -200601,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Property not supported by this scale type."
        }
    },
    {
        -200602,
        {
            Category::Other, ActionRequired::Warning,
            "Prescaled Minimum cannot be equal to Prescaled Maximum for input operations."
        }
    },
    {
        -200603,
        {
            Category::Other, ActionRequired::Warning,
            "Scaled Minimum cannot be equal to Scaled Maximum for output operations."
        }
    },
    {
        -200604,
        {
            Category::Other, ActionRequired::Warning,
            "NULL pointer was passed for a required parameter."
        }
    },
    {
        -200605,
        {
            Category::Other, ActionRequired::Warning,
            "Given range in the input string contains a number that is too large. Check the string. Use smaller numbers in the range, or replace the range with a comma-separated list."
        }
    },
    {
        -200606,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested channel index is invalid. The value of the index must be between one and the number of channels in the task."
        }
    },
    {
        -200607,
        {
            Category::Other, ActionRequired::Warning,
            "Two signals cannot be simultaneously exported on the same terminal."
        }
    },
    {
        -200608,
        {
            Category::Other, ActionRequired::Warning,
            "Acquisition cannot be started, because the selected buffer size is too small. Increase the buffer size."
        }
    },
    {
        -200609,
        {
            Category::Other, ActionRequired::Warning,
            "Generation cannot be started, because the selected buffer size is too small. Increase the buffer size."
        }
    },
    {
        -200610,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested sample clock source is invalid for output. The specified sample clock source terminal is only supported for input."
        }
    },
    {
        -200611,
        {
            Category::Other, ActionRequired::Warning,
            "Operation cannot be performed, because there are no channels of the requested type in the task."
        }
    },
    {
        -200612,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested channel index is invalid. The value of the index must be between 0 and the number of channels in the task minus one."
        }
    },
    {
        -200613,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Acquisition has been stopped to prevent an input buffer overwrite. Your application was unable to read samples from the buffer fast enough to prevent new samples from overwriting unread data. To avoid this error, you can do any of the following: 1. Increase the size of the buffer. 2. Increase the number of samples you read each time you invoke a read operation. 3. Read samples more often. 4. Reduce the sample rate. 5. If your data transfer method is interrupts, try using DMA or USB Bulk. 6. Reduce the number of applications your computer is running concurrently. In addition, if you do not need to read every sample that is acquired, use the Relative To and Offset properties to read the desired samples."
        }
    },
    {
        -200614,
        {
            Category::Other, ActionRequired::Warning,
            "Combination of Samples to Read, Position, and Offset results in an attempt to read past the end of the record. You only can read samples within the record."
        }
    },
    {
        -200615,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200616,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified asynchronous operation handle is invalid."
        }
    },
    {
        -200617,
        {
            Category::Hardware, ActionRequired::Warning,
            "Output generation was aborted by the reverse power protection circuitry of the device. Either the output signal exceeded the output power limit, or power was being driven back into the output of the device by an external source. Correct the problem, then generate the signal again."
        }
    },
    {
        -200618,
        {
            Category::Hardware, ActionRequired::Warning,
            "Analog input virtual channels cannot be created out of order with respect to their physical channel numbers for the type of analog device you are using. For example, a virtual channel using physical channel ai0 must be created before a virtual channel with physical channel ai1."
        }
    },
    {
        -200619,
        {
            Category::Other, ActionRequired::Warning,
            "Chassis cannot be used for more than one scanning operation at the same time. Do only one scanning operation, or combine multiple scanning operations into a single operation."
        }
    },
    {
        -200620,
        {
            Category::Hardware, ActionRequired::Warning,
            "Requested multiple virtual channels that correspond to the same analog input physical channel within a single task. A task cannot contain multiple analog input physical channels for this type of device. Use different physical channels for each virtual channel."
        }
    },
    {
        -200621,
        {
            Category::Hardware, ActionRequired::Warning,
            "Onboard device memory underflow. Because of system and/or bus-bandwidth limitations, the driver could not write data to the device fast enough to keep up with the device output rate. Reduce your sample rate. If your data transfer method is interrupts, try using DMA or USB Bulk. You can also reduce the number of programs your computer is executing concurrently."
        }
    },
    {
        -200622,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested number of samples to write is invalid. Change the number of samples to be written to a number equal to or greater than zero."
        }
    },
    {
        -200623,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device has shut down because a sensor on the device detected a temperature above the device's maximum recommended operating temperature. To use the device again, either turn the chassis/computer off until the device has cooled, or ensure the device has cooled, and reset the device (either programmatically or through Measurements & Automation Explorer)."
        }
    },
    {
        -200624,
        {
            Category::Configuration, ActionRequired::Warning,
            "You have not specified an active channel when getting a property. Specify a single line as the active physical channel."
        }
    },
    {
        -200625,
        {
            Category::Configuration, ActionRequired::Warning,
            "You have specified more than one line as the active physical channel when getting a property. Specify a single line."
        }
    },
    {
        -200626,
        {
            Category::Other, ActionRequired::Warning,
            "An empty physical channel has been specified within the power up states array. Specify a correct physical channel."
        }
    },
    {
        -200627,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200628,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Pause and reference triggers are both configured, which is not supported in this task."
        }
    },
    {
        -200629,
        {
            Category::Hardware, ActionRequired::Warning,
            "Requested different values for properties that must have equal values on this device."
        }
    },
    {
        -200630,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested write offset is invalid, because it is not an integer multiple of the write increment."
        }
    },
    {
        -200631,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200632,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware clocking error occurred. If you are using an external sample clock, make sure it is connected and within the jitter and voltage level specifications. Also, verify that the rate of the external clock matches the specified clock rate. If you are generating your clock internally, please contact National Instruments Technical Support."
        }
    },
    {
        -200633,
        {
            Category::Other, ActionRequired::Warning,
            "Script name is the same as an existing waveform name. Make sure that the script name is different from the names of previously written or allocated waveforms."
        }
    },
    {
        -200634,
        {
            Category::Other, ActionRequired::Warning,
            "Waveform name is the same as an existing script name. Make sure that the waveform name is different from the names of previously written scripts."
        }
    },
    {
        -200635,
        {
            Category::Other, ActionRequired::Warning,
            "You have attempted to control a Watchdog Task, but the task supplied was not a Watchdog Task."
        }
    },
    {
        -200636,
        {
            Category::Communication, ActionRequired::Warning,
            "For this device, any connection specified in the scan list must wait for a trigger (;). If your scan list contains an action sequence similar to \"ch0->com0 &\" or \"ch0->com0 &&\", change the action sequence to \"ch0->com0;\"."
        }
    },
    {
        -200637,
        {
            Category::Communication, ActionRequired::Warning,
            "For this device, an action separator (& or &&) is required after breaking a connection in the scan list. If your scan list contains an action sequence similar to \"~ch0->com0;\", change the action sequence to \"~ch0->com0 &\" or \"~ch0->com0 &&\"."
        }
    },
    {
        -200638,
        {
            Category::Communication, ActionRequired::ImmediateAction,
            "For this device, two consecutive wait for triggers are not supported after any connection in the scan list. If your scan list contains an action sequence similar to \"ch0->com0;;\", change the action sequence to \"ch0->com0;\"."
        }
    },
    {
        -200639,
        {
            Category::Communication, ActionRequired::Warning,
            "For this device, connections specified in the scan list must be disconnected before making new connections. If your scanlist contains an action sequence similar to \"ch0->com0;;\", change the action sequence to \"ch0->com0; ~ch0->com0\"."
        }
    },
    {
        -200640,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "For this device, send advance complete (<sac>) is not supported after any wait for triggers (;) in the scan list. If your scan list contains an action sequence similar to \"ch0->com0; <sac>\", change the action sequence to \"ch0->com0;\"."
        }
    },
    {
        -200641,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property cannot be set separately for each line. When setting this property, specify a virtual channel as the active channel."
        }
    },
    {
        -200642,
        {
            Category::Configuration, ActionRequired::Warning,
            "You have not specified an active channel when getting a property. Specify a single line as the active channel."
        }
    },
    {
        -200643,
        {
            Category::Configuration, ActionRequired::Warning,
            "You have specified more than one line when getting a property. Specify a single line as the active channel."
        }
    },
    {
        -200644,
        {
            Category::Other, ActionRequired::Warning,
            "Attempt to reset watchdog timer failed, because the timer had already expired. Clear expiration of the watchdog timer, or configure a longer watchdog timer timeout."
        }
    },
    {
        -200645,
        {
            Category::Other, ActionRequired::Warning,
            "Attempt to set the digital filter interval failed, because another task has already configured a different digital filter interval. Use the same digital filter interval in the two tasks, or wait for the other task to finish before starting or committing this task."
        }
    },
    {
        -200646,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Some of the physical channels in the task are configured for different filter intervals, which is not supported by this type of device. Configure all lines in the task to use the same digital filter interval."
        }
    },
    {
        -200647,
        {
            Category::Hardware, ActionRequired::Warning,
            "Specified physical channel does not support digital input. Change the direction of the task, use another terminal, or use another device. To read from digital output lines, create a digital output task and use DAQmx Read."
        }
    },
    {
        -200648,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device identifier of the physical channel specified is not the same as the device used in the task. Use only the physical channels on the device used in the task."
        }
    },
    {
        -200649,
        {
            Category::Other, ActionRequired::Warning,
            "Specified a physical channel for change detection that is not contained by any channel in the task. Use only physical channels already contained by a channel, or create an additional channel containing the desired physical channel ."
        }
    },
    {
        -200650,
        {
            Category::Other, ActionRequired::Warning,
            "Attempt to set programmable powerup state failed, because the specified physical channel only supports digital input, and the programmable powerup state does not apply."
        }
    },
    {
        -200651,
        {
            Category::Other, ActionRequired::Warning,
            "Attempt to set watchdog timer expiration state failed, because the specified physical channel only supports digital input, and watchdog timer expiration state does not apply."
        }
    },
    {
        -200652,
        {
            Category::Hardware, ActionRequired::Warning,
            "Attempt to set programmable powerup state failed, because only some of the channels from the port were specified. For this type of device, you must specify programmable powerup state for entire ports."
        }
    },
    {
        -200653,
        {
            Category::Hardware, ActionRequired::Warning,
            "Attempt to set programmable powerup state failed, because some of the lines in a port were tristated and others were not. For this type of device, programmable powerup states of all lines in a port have to be either tristated or not tristated."
        }
    },
    {
        -200654,
        {
            Category::Hardware, ActionRequired::Warning,
            "Attempt to set watchdog timer expiration state failed, because some of the lines in a port were tristated, and others were not. For this type of device, watchdog timer expiration states of all lines in a port have to be either tristated or not tristated."
        }
    },
    {
        -200655,
        {
            Category::Configuration, ActionRequired::Warning,
            "Attempted to read more samples than what was configured in the acquisition. Restart the acquisition, increase the Samples Per Channel property, or set the Sample Mode property to Continuous Samples."
        }
    },
    {
        -200656,
        {
            Category::Configuration, ActionRequired::Warning,
            "Operation failed, because an attempt was made to use only the onboard memory for generation when regeneration of data was not allowed. Set the Regeneration Mode property to Allow Regeneration or set the Use Only Onboard Memory property to false."
        }
    },
    {
        -200657,
        {
            Category::Configuration, ActionRequired::Warning,
            "Attempt to get property failed, because you specified multiple channels, and the property has different values for different channels. Get this property one channel at a time."
        }
    },
    {
        -200658,
        {
            Category::Configuration, ActionRequired::Warning,
            "Attempt to get property failed, because your task contains multiple channels, and the property has different values for different channels. Get this property one channel at a time using Active Channel to specify each individual channel."
        }
    },
    {
        -200659,
        {
            Category::Configuration, ActionRequired::Warning,
            "Attempt to get property failed, because the single channel you specified corresponds to multiple physical channels, and the property has different values for those different physical channels. Get this property one physical channel at a time. For digital channels, you might have to specify a single digital line."
        }
    },
    {
        -200660,
        {
            Category::Other, ActionRequired::Warning,
            "Requested Sample Clock Rate is not available because this task shares the Sample Clock Source or the Sample Clock Timebase with another task. The other task has already programmed one of those properties in a manner inconsistent with the requested Sample Clock Rate. Specify a Sample Clock Rate consistent with the settings in the other task, or change the settings in the other task. Refer to documentation for more detailed information."
        }
    },
    {
        -200661,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested operation is not supported because the Reference Clock Source is \"None\"."
        }
    },
    {
        -200662,
        {
            Category::Hardware, ActionRequired::Warning,
            "Attempt to create a watchdog timer task failed because the device does not support the feature, or the wrong device type was specified. For NI CompactDAQ devices supporting this feature, specify the chassis name on which to create a watchdog timer task."
        }
    },
    {
        -200663,
        {
            Category::Hardware, ActionRequired::Warning,
            "Attempt to set programmable powerup states failed because the device does not support the feature."
        }
    },
    {
        -200664,
        {
            Category::Other, ActionRequired::Warning,
            "Specified operation cannot be performed because the task has not been started, committed, or reserved. Call DAQmx Start or DAQmx Control with action set to Commit or Reserve prior to requesting this operation."
        }
    },
    {
        -200665,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Expiration states requested are not supported by the lines in the port. For this device, the Watchdog Timer Expiration States must be either No Change for all lines in a port or a combination of values other than No Change for all lines in a port. For example, the combination of No Change for one line and High for another line is not supported, while a value of Low for one line and High for another line is supported."
        }
    },
    {
        -200666,
        {
            Category::Other, ActionRequired::Warning,
            "Attempt to configure a port or any of its lines for input failed, because this port is currently configured for digital output by the watchdog timer. Choose another port, or modify the watchdog timer task to set the expiration state of the port to Tristate or No Change."
        }
    },
    {
        -200667,
        {
            Category::Other, ActionRequired::Warning,
            "Attempt to configure a port or any of its lines to have an expiration state of Output failed, because the port or some of its lines are currently reserved for use by an input task. Set the expiration state of the port to Tristate or No Change, or choose a different port for digital input."
        }
    },
    {
        -200668,
        {
            Category::Other, ActionRequired::Warning,
            "Requested value is not a supported value for Watchdog Timer Timeout. Use special value -1.0 to indicate that the internal timer should be disabled, and the watchdog timer will expire based on the external expiration trigger, or specify another valid value."
        }
    },
    {
        -200669,
        {
            Category::Configuration, ActionRequired::Warning,
            "Attempt to set the Connect DAC Reference to Ground property failed, because the Allow Connecting DAC Reference to Ground property was not True. To connect DAQ reference to ground, you must set two properties to True: Connect DAC Reference to Ground and Allow Connecting DAC Reference to Ground."
        }
    },
    {
        -200670,
        {
            Category::Configuration, ActionRequired::Warning,
            "DAC Range Low is not equal in magnitude and opposite in sign from DAC Reference Value. If you do not set the DAC Range Low property, the driver sets it for you. Otherwise, ensure DAC Range Low and DAC Reference Voltage Value are equal in magnitude and opposite in sign."
        }
    },
    {
        -200671,
        {
            Category::Hardware, ActionRequired::Warning,
            "Switch device has been disabled to prevent it from exceeding its simultaneous relay drive limit. To recover, call DAQmx Disconnect All, or reset the device. The device can be reset either programmatically or by using Measurement & Automation Explorer."
        }
    },
    {
        -200672,
        {
            Category::Other, ActionRequired::Warning,
            "Input arrays are of different sizes. These arrays must have the same size."
        }
    },
    {
        -200673,
        {
            Category::Other, ActionRequired::Warning,
            "Multiple samples cannot be written using a single sample write. Ensure the waveform contains only a single sample."
        }
    },
    {
        -200674,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Switch operation failed due to a previous error. The device may have been powered off and back on. To use the device again, reset the device either programmatically or by using Measurement & Automation Explorer."
        }
    },
    {
        -200675,
        {
            Category::Hardware, ActionRequired::Warning,
            "Input Source Select is set to an internal source for channels in different channel groups. On this device, only one channel group at a time can be configured to use an internal source. Refer to documentation for details."
        }
    },
    {
        -200676,
        {
            Category::Configuration, ActionRequired::Warning,
            "Input Source Select property is set differently for channels in one channel group on a device that supports only identical settings within a channel group. Refer to documentation for details."
        }
    },
    {
        -200677,
        {
            Category::Configuration, ActionRequired::Warning,
            "Input Source Select property is set to different internal sources on some channels. On this device, Input Source Select must be set the same way for all channels with internal sources. Refer to documentation for details."
        }
    },
    {
        -200678,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Write failed because at least one of the lines in the task is also in a watchdog timer task whose watchdog timer has expired. Clear the expiration of the watchdog timer before writing by using DAQmx Control Watchdog Task, by restarting the watchdog timer task, or by resetting the device programmatically or through the Measurements & Automation Explorer. To prevent this error in the future reset the watchdog timer more frequently or increase the watchdog timer timeout."
        }
    },
    {
        -200679,
        {
            Category::Hardware, ActionRequired::Warning,
            "When Sample Mode is Hardware Timed Single Point, Memory Mapping for Programmed IO Enable cannot be true. Set Memory Mapping for Programmed IO Enable to false or change Sample Mode."
        }
    },
    {
        -200680,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device has shut down because a sensor on the device detected a temperature above the device's maximum recommended operating temperature. To use the device again, either turn the chassis/computer off until the device has cooled, or ensure the device has cooled, and reset the device (either programmatically or through Measurements & Automation Explorer)."
        }
    },
    {
        -200681,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Route failed because either the source or destination of the route is also a line in a watchdog timer task whose watchdog timer has expired. Clear the expiration of the watchdog timer before routing by using DAQmx Control Watchdog Task, by restarting the watchdog timer task, or by resetting the device programmatically or through Measurements & Automation Explorer. To prevent this error in the future reset the watchdog timer more frequently or increase the watchdog timer timeout."
        }
    },
    {
        -200682,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Data could not be read because the reference trigger master session is unavailable. To avoid this error, read data before closing the reference trigger master session."
        }
    },
    {
        -200683,
        {
            Category::Other, ActionRequired::Warning,
            "NI-DAQmx has detected a corrupt installation. Please re-install NI-DAQmx. If you continue to receive this message, please contact National Instruments for assistance."
        }
    },
    {
        -200684,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Pulse duty cycle specified is not supported for this device given the pulse frequency and Counter Timebase Rate."
        }
    },
    {
        -200685,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Pulse frequency specified is not supported for this device given the Counter Timebase Rate."
        }
    },
    {
        -200686,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Pulse high time specified is not supported for this device given the Counter Timebase Rate."
        }
    },
    {
        -200687,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Pulse low time specified is not supported for this device given the Counter Timebase Rate."
        }
    },
    {
        -200688,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Pulse high tick count specified is not supported for this device."
        }
    },
    {
        -200689,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Pulse low tick count specified is not supported for this device."
        }
    },
    {
        -200690,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Buffered operations are not supported if Sample Mode is Hardware Timed Single Point. Do not configure a buffer, or change the Sample Mode value."
        }
    },
    {
        -200691,
        {
            Category::Other, ActionRequired::Warning,
            "Buffered operations are not compatible with the requested Wait Mode. Do not configure a buffer, or set Wait Mode to Yield."
        }
    },
    {
        -200692,
        {
            Category::Hardware, ActionRequired::Warning,
            "Number of samples per channel to write multiplied by the number of channels in the task cannot be an odd number for this device. Adjust the number of samples per channel to write or the number of channels in the task so that their product is an integer multiple of two."
        }
    },
    {
        -200693,
        {
            Category::Hardware, ActionRequired::Warning,
            "Buffer size (in samples per channel) multiplied by the number of channels in the task cannot be an odd number for this device. Adjust the buffer size or the number of channels in the task so that their product is an integer multiple of two."
        }
    },
    {
        -200694,
        {
            Category::Other, ActionRequired::Warning,
            "AI Minimum was not specified for an operation that requires it."
        }
    },
    {
        -200695,
        {
            Category::Other, ActionRequired::Warning,
            "AI Maximum was not specified for an operation that requires it."
        }
    },
    {
        -200696,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Bridge offset nulling calibration is not supported by the specified channels. Specify only the analog input channels that are configured to measure sensors in a bridge configuration."
        }
    },
    {
        -200697,
        {
            Category::Hardware, ActionRequired::Warning,
            "Calibration data could not be acquired. Ensure that the device(s) are configured and functioning properly."
        }
    },
    {
        -200698,
        {
            Category::Other, ActionRequired::Warning,
            "Route failed because the PXI chassis is not identified. The existence of the source terminal depends on the chassis being identified. Use the Measurements & Automation Explorer (MAX) to identify your chassis."
        }
    },
    {
        -200699,
        {
            Category::Other, ActionRequired::Warning,
            "Route failed because the PXI chassis is not identified. The existence of the destination terminal depends on the chassis being identified. Use the Measurements & Automation Explorer (MAX) to identify your chassis."
        }
    },
    {
        -200700,
        {
            Category::Hardware, ActionRequired::Warning,
            "PXI_Star is not available as a source terminal for devices in PXI slot 2. PXI slot 2 has specific PXI_Star<n> lines, such as PXI_Star3. Move your device to one of slots 3 through 15, or select a different source terminal."
        }
    },
    {
        -200701,
        {
            Category::Hardware, ActionRequired::Warning,
            "PXI_Star is not available as a destination terminal for devices in PXI slot 2. PXI slot 2 has specific PXI_Star<n> lines, such as PXI_Star3. Move your device to one of slots 3 through 15, or select a different destination terminal."
        }
    },
    {
        -200702,
        {
            Category::Hardware, ActionRequired::Warning,
            "PXI_Star is not available as a destination terminal for devices in PXI slots 16 and above. Move your device to one of slots 3 through 15, or select a different destination terminal."
        }
    },
    {
        -200703,
        {
            Category::Hardware, ActionRequired::Warning,
            "PXI_Star is not available as a source terminal for devices in PXI slots 16 and above. Move your device to one of slots 3 through 15, or select a different source terminal."
        }
    },
    {
        -200704,
        {
            Category::Hardware, ActionRequired::Warning,
            "PXI_Star<n> is available as a source terminal only for devices in the star trigger controller slot (slot 2). To use PXI_Star (without any numbers), do not specify a star line number. To use PXI_Star<n>, move your device to slot 2."
        }
    },
    {
        -200705,
        {
            Category::Hardware, ActionRequired::Warning,
            "PXI_Star<n> is available as a destination terminal only for devices in the star trigger controller slot (slot 2). To use PXI_Star (without any numbers), do not specify a star line number. To use PXI_Star<n>, move your device to slot 2."
        }
    },
    {
        -200706,
        {
            Category::Hardware, ActionRequired::Warning,
            "PXI_Clk10In is available as a destination terminal only for devices in the star trigger controller slot (slot 2). Move your device to PXI slot 2."
        }
    },
    {
        -200708,
        {
            Category::Configuration, ActionRequired::Warning,
            "Getting a property that pertains to multiple items failed because the value was different for different items. Get the specified property for one item at a time. For example, if you are getting a property for two markers, such as \"marker0:1\" or \"marker0, marker1\", and the property values are different for the two markers, you must get them in two steps (one for marker0 and another for marker1)."
        }
    },
    {
        -200709,
        {
            Category::Hardware, ActionRequired::Warning,
            "No TEDS sensor was detected on the specified physical channel. Ensure that your sensor is properly connected. If the sensor is connected to a TEDS interface device with addresses, make sure the configured address matches the address set on the interface device."
        }
    },
    {
        -200710,
        {
            Category::Configuration, ActionRequired::Warning,
            "Input Source Select property is set to an internal source with more than one channel in the task. On this device, an internal input source is supported only when there is one channel in the task. Remove all of the channels currently in the task except the channel that will be used to acquire the internal input source."
        }
    },
    {
        -200711,
        {
            Category::Configuration, ActionRequired::Warning,
            "An attempt has been made to use an analog trigger with the Input Source Select property set to an internal source. Either change the Input Source Select property ot specify an external source, or use a different analog trigger source."
        }
    },
    {
        -200712,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Digital input configuration failed because at least one of the lines in the task was in an expired watchdog task, and the expiration state of the line was set to output. Clear the expiration of the watchdog task by using DAQmx Control Task, by restarting the watchdog task, or by resetting the device either programmatically or by using Measurement & Automation Explorer. To prevent this error in the future, reset the watchdog timer more frequently or increase the watchdog timer timeout."
        }
    },
    {
        -200713,
        {
            Category::Hardware, ActionRequired::Warning,
            "Channels specified cannot appear in the same task on this device. Create a separate task for each of the channels specified."
        }
    },
    {
        -200714,
        {
            Category::Hardware, ActionRequired::Warning,
            "Acquisition has stopped because the driver could not transfer the data from the device to the computer memory fast enough. This was caused by computer system limitations. Reduce your sample clock rate, the number of channels in the task, or the number of programs your computer is executing concurrently."
        }
    },
    {
        -200715,
        {
            Category::Other, ActionRequired::Warning,
            "Digital input detected a new sample clock before the previous sample was latched into onboard memory. If you are using an external sample clock, ensure that it is connected, within the jitter and voltage level specifications, and without glitches. If applicable, reduce your sample clock rate or use a product capable of higher sample clock rates."
        }
    },
    {
        -200716,
        {
            Category::Other, ActionRequired::Warning,
            "Digital output detected a new sample clock edge before the previous sample could be written from the onboard memory. If you are using an external sample clock, ensure that it is connected, within jitter and voltage level specifications, and without glitches. If applicable, reduce your sample clock rate or use a product capable of higher sample clock rates."
        }
    },
    {
        -200717,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Measurement taken during calibration produced an invalid AI offset calibration constant. If performing an external calibration, ensure that the reference voltage value passed to the calibration VI or function is correct. Repeat the calibration. If the error persists, contact National Instruments Technical Support."
        }
    },
    {
        -200718,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Measurement taken during calibration produced an invalid AI gain calibration constant. If performing an external calibration, ensure that the reference voltage value passed to the calibration VI or function is correct. Repeat the calibration. If the error persists, contact National Instruments Technical Support."
        }
    },
    {
        -200719,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Measurement taken during calibration produced an invalid AO offset calibration constant. If performing an external calibration, ensure that the reference voltage value passed to the calibration VI or function is correct. Repeat the calibration. If the error persists, contact National Instruments Technical Support."
        }
    },
    {
        -200720,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Measurement taken during calibration produced an invalid AO gain calibration constant. If performing an external calibration, ensure that the reference voltage passed to the calibration VI or function is correct. Repeat the calibration. If the error persists, contact National Instruments Technical Support."
        }
    },
    {
        -200721,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Computed frequency resolution of the VCXO CalDAC circuitry is not sufficient to perform timebase calibration. Ensure that the reference frequency is stable and that the frequency passed to the calibration VI or function is correct. Repeat the external timebase calibration. If the error persists, contact National Instruments Technical Support."
        }
    },
    {
        -200722,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Timebase calibration algorithm failed to converge within the required tolerance. Ensure that the reference frequency is stable and that the frequency passed to the calibration VI or function is correct. Repeat the external timebase calibration. If the error persists, contact National Instruments Technical Support."
        }
    },
    {
        -200723,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Variance of the measured external reference frequency is too large to perform timebase calibration. Ensure that the reference frequency is stable. Repeat the external timebase calibration. If the error persists, contact National Instruments Technical Support."
        }
    },
    {
        -200724,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Digital Input Tristate property has different values for different channels in the task, which is not supported for this type of device. Change the property to a single value for all channels in the task, or use more than one task."
        }
    },
    {
        -200725,
        {
            Category::Other, ActionRequired::Warning,
            "Some or all of the lines in the task are used by another task for handshaking input. These lines cannot be used in a static input task. Use a line that is not in a handshaking input task or stop using the line in the handshaking input task."
        }
    },
    {
        -200726,
        {
            Category::Other, ActionRequired::Warning,
            "Some or all of the lines in the task are used by another task for static input. These lines cannot be used in a handshaking input task. Use a port that is not already used by a static input task or stop using the line in the static input task."
        }
    },
    {
        -200727,
        {
            Category::Other, ActionRequired::Warning,
            "Some or all of the lines in the task are used by another task for handshaking control. These lines cannot be used in a static input task. Use a line that is not also a control line in a handshaking task, or stop using the line for handshaking control."
        }
    },
    {
        -200728,
        {
            Category::Other, ActionRequired::Warning,
            "Some or all of the handshaking control lines for this task are used by another task for static input. These lines cannot be used for handshaking control. Use a port whose handshaking control lines are not already used by a static input task or stop using the lines for handshaking control."
        }
    },
    {
        -200729,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Value of the Tristate property for some or all of the channels in the task is False, and Sample Timing Type is Handshake, which is not supported by this device. Set the Tristate property to True for all channels or change the Sample Timing Type."
        }
    },
    {
        -200730,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Value of Tristate property for some or all of the channels in the task is False, and Sample Timing Type is Change Detection, which is not supported by this device. Set the Tristate property to True for all channels or change the Sample Timing Type."
        }
    },
    {
        -200731,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Sample Mode is Hardware Timed, and Sample Timing Type is On Demand, which is not supported by this device. Change Sample Mode or Sample Timing Type."
        }
    },
    {
        -200732,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Some of all of the lines in the task have had their Digital Filter Enable property set, which is not supported when the value of the Tristate property is False. Change the value of the Tristate property, or do not set the Digital Filter Enable property."
        }
    },
    {
        -200733,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Some or all of the lines in the task have ahd their Digital Filter Minimum Pulse Width property set, which is not supported when the value of the Tristate property is False. Change the value of the Tristate property, or do not set the Digital Filter Minimum Pulse Width property."
        }
    },
    {
        -200734,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device does not support DMA for the Data Transfer Mechanism when performing non-buffered acquisitions. Set Data Transfer Mechanism to Programmed I/O."
        }
    },
    {
        -200735,
        {
            Category::Other, ActionRequired::Warning,
            "Given the specified Reference Clock Source, you must set the Reference Clock Rate to a value equal to the frequency of the supplied signal."
        }
    },
    {
        -200736,
        {
            Category::Other, ActionRequired::Warning,
            "FREQOUT counter cannot generate the desired frequency. The FREQOUT counter is a 4-bit counter that can divide either the 10 MHz Timebase or the 100 kHz Timebase by a number between one and sixteen. Chose a frequency within this range."
        }
    },
    {
        -200737,
        {
            Category::Configuration, ActionRequired::Warning,
            "For this type of device, the Input Buffer Size (in Samples per Channel) must be equal to the value of the Sample Quanity-Samples per Channel property when Sample Mode is Finite Samples."
        }
    },
    {
        -200738,
        {
            Category::Other, ActionRequired::Warning,
            "Specified timing source does not exist."
        }
    },
    {
        -200739,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Specified property is not supported for the given timing source."
        }
    },
    {
        -200740,
        {
            Category::Other, ActionRequired::Warning,
            "Task used as the timing source for a Timed Loop was started before the Timed Loop was executed. Let the Timed Loop start the task, or use the task without the Timed Loop."
        }
    },
    {
        -200741,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "A TEDS sensor not supported by DAQmx is connected to the specified physical channel. Consider using MAX to create a Task, a Global Channel, or a Scale to acquire data using this sensor."
        }
    },
    {
        -200742,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Memory of the TEDS sensor connected to the specified physical channel is corrupted, as indicated by an invalid check-sum. Replace the sensor or have the sensor repaired. If the memory is the only defective part of the sensor, consider using MAX to create a Task, a Global Channel, or a Scale to acquire data using this sensor."
        }
    },
    {
        -200743,
        {
            Category::Hardware, ActionRequired::Warning,
            "For this device, a TEDS terminal block must be connected to the device and configured in MAX in order to perform a TEDS operation."
        }
    },
    {
        -200744,
        {
            Category::Software, ActionRequired::Warning,
            "Specified reference clock rate does not match the specified reference clock source. Do not set the refererence clock rate when you are using an internal reference clock source. In this case, the driver sets the reference clock rate for you."
        }
    },
    {
        -200745,
        {
            Category::Configuration, ActionRequired::Warning,
            "Frequency and Initial Delay property values are inconsistent with one or more counter timebase properties. The conflicting properties must satisfy the following constraints: Counter Timebase Rate / Counter Maximum Count <= Frequency <= Counter Timebase Rate / 4 Counter Timebase Rate / Counter Maximum Count <= 1 / Initial Delay <= Counter Timebase Rate / 2 If the Counter Timebase Rate is not specified, it is inferred from the Counter Timebase Source selection."
        }
    },
    {
        -200746,
        {
            Category::Configuration, ActionRequired::Warning,
            "Initial Delay, High Time, and Low Time property values are inconsistent with one or more counter timebase properties. The conflicting properties must satisfy the following constraints: 2 / Counter Timebase Rate <= Initial Delay <= Counter Maximum Count / Counter Timebase Rate 2 / Counter Timebase Rate <= High Time <= Counter Maximum Count / Counter Timebase Rate 2 / Counter Timebase Rate <= Low Time <= Counter Maximum Count / Counter Timebase Rate If the Counter Timebase Rate is not specified, it is inferred from the Counter Timebase Source selection."
        }
    },
    {
        -200747,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Acquisition type specified is not supported by the FREQOUT channel. To use the FREQOUT channel, set the acquisition type to a value supported by FREQOUT. To use specified output type, use a different counter output channel."
        }
    },
    {
        -200748,
        {
            Category::Hardware, ActionRequired::Warning,
            "DAQmx Write was invoked more than once between two consecutive sample clocks. When Sample Mode is Hardware Timed Single Point, invoke DAQmx Write only once between two consecutive sample clocks."
        }
    },
    {
        -200749,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified gain is not supported."
        }
    },
    {
        -200750,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified voltage is invalid for the given gain."
        }
    },
    {
        -200751,
        {
            Category::Other, ActionRequired::Warning,
            "Sample Timing Type was set to Change Detection but no physical channels on which to detect changes were specified. Specify the Change Detection Digital Input Rising and/or Falling Edge Physical Channels, or specify a different Sample Timing Type."
        }
    },
    {
        -200752,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "You have specified more than one physical channel as the active channel which is not supported. Specify a single physical channel."
        }
    },
    {
        -200753,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "TEDS sensor data or the Virtual TEDS data file contains an error which was detected during parsing. Ensure your TEDS sensor or Virtual TEDS data file conforms to the specification. If this is not possible, use custom scales with the sensor."
        }
    },
    {
        -200754,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "TEDS template specifies parameters that are not supported by DAQmx. Use custom scales with this sensor."
        }
    },
    {
        -200755,
        {
            Category::Other, ActionRequired::Warning,
            "TEDS sensor data or Virtual TEDS data file contains multiple calibration templates. Only one calibration template is supported by DAQmx. Use custom scales with this sensor."
        }
    },
    {
        -200756,
        {
            Category::Other, ActionRequired::Warning,
            "Type of TEDS sensor associated with the channel is incompatible with the Measurement Type. Use the TEDS sensor for measurements compatible with the sensor."
        }
    },
    {
        -200757,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Sample Timing Type is set to On Demand which is not supported for analog input on this device. Set Sample Timing Type to Sample Clock. You can achieve this whlie setting related properties through DAQmx VIs or functions for configuring timing."
        }
    },
    {
        -200758,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Sample Timing Type is set to On Demand which is not supported for analog output on this device. Set Sample Timing Type to Sample Clock. You can achieve this while setting related properties through DAQmx VIs or functions for configuring timing."
        }
    },
    {
        -200759,
        {
            Category::Software, ActionRequired::Warning,
            "Range specified by the AO Maximum and Minimum Value, and AO Voltage Units properties does not lie within the range specified by the AO DAC Range High and Low properties. Change the values of these properties. If you do not specify AO DAC Range High and Low, the driver will set them based on other properties."
        }
    },
    {
        -200760,
        {
            Category::Configuration, ActionRequired::Warning,
            "Range specified by the AO Maximum and Minimum Value, and AO Voltage Units properties does not lie within the range specified by the AO Gain property. Change the values of these properties. If you do not specify AO Gain, the driver will set it based on other properties."
        }
    },
    {
        -200761,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Task cannot issue sync pulse because another task is currently running on this device. For this type of device, the task can issue a sync pulse if it is the only task running on the device. If your task is not being used for synchronization, set the Sync Pulse Source property to \"\" or \"None\" to avoid receiving an error."
        }
    },
    {
        -200762,
        {
            Category::Hardware, ActionRequired::Warning,
            "Task cannot issue sync pulse because the task has an external sample clock timebase. For this type of device, the task can issue a sync pulse if the Sample Clock Timebase Source is Onboard Clock."
        }
    },
    {
        -200763,
        {
            Category::Hardware, ActionRequired::Warning,
            "For analog input with the current Data Transfer Mechanism on this type of device, the input buffer size (in samples per channel) must be an integer multiple of the transfer size. Change the Data Transfer Mechanism or the input buffer size."
        }
    },
    {
        -200764,
        {
            Category::Other, ActionRequired::Warning,
            "TEDS sensor connected to the specified physical channel uses a linear mapping method and specifies the linear slope to be zero. Replace the sensor or have the sensor repaired. If the memory is the only defective part of the sensor, consider using MAX to create a Task, Global Channel, or a Scale to acquire data using this sensor."
        }
    },
    {
        -200765,
        {
            Category::Software, ActionRequired::ImmediateAction,
            "Mapping method of the TEDS sensor connected to the specified physical channel is invalid or is not supported by the driver. If the sensor is defective, replace it or have it repaired. Otherwise, consider using MAX to create a Task, Global Channel, or a Scale to acquire data using this sensor."
        }
    },
    {
        -200766,
        {
            Category::Software, ActionRequired::ImmediateAction,
            "Legacy template ID of the TEDS sensor connected to the specified physical channel is invalid or is not supported by the driver. If the sensor is defective, replace it or have it repaired. Otherwise, consider using MAX to create a Task, Global Channel, or a Scale to acquire data using this sensor."
        }
    },
    {
        -200767,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Data Transfer Mechanism is set to Programmed I/O which is not supported for buffered analog output. Change Data Transfer Mechanism or use non-buffered analog output."
        }
    },
    {
        -200768,
        {
            Category::Other, ActionRequired::Warning,
            "Data Transfer Mechanism is not set to Programmed I/O, the only value supported for non-buffered analog output. Set your Data Transfer Mechanism to Programmed I/O or use buffered analog output."
        }
    },
    {
        -200769,
        {
            Category::Hardware, ActionRequired::Warning,
            "Data Transfer Mechanism is not set to Programmed I/O, the only value supported when the Sample Mode is Hardware Timed Single Point. Set your Data Transfer Mechanism to Programmed I/O or change the Sample Mode."
        }
    },
    {
        -200770,
        {
            Category::Other, ActionRequired::Warning,
            "Digital Filter Enable and Digital Synchronization Enable properties cannot be true at the same time."
        }
    },
    {
        -200771,
        {
            Category::Configuration, ActionRequired::Warning,
            "Digital Filter Enable is set to true but the Minimum Pulse Width property is not configured. Configure the Minimum Pulse Width property or set Digital Filter Enable to false."
        }
    },
    {
        -200772,
        {
            Category::Other, ActionRequired::Warning,
            "Digital filtering is not available for the given terminal."
        }
    },
    {
        -200773,
        {
            Category::Other, ActionRequired::Warning,
            "Digital synchronization is not available for the given terminal."
        }
    },
    {
        -200774,
        {
            Category::Configuration, ActionRequired::Warning,
            "Terminal has already been configured with a different Minimum Pulse Width by another property in this task."
        }
    },
    {
        -200775,
        {
            Category::Other, ActionRequired::Warning,
            "Terminal has already been configured with a different Minimum Pulse Width by another task."
        }
    },
    {
        -200776,
        {
            Category::Configuration, ActionRequired::Warning,
            "Desired Minimum Pulse Width could not be produced. Minimum Pulse Width is affected by the Digital Filter Timebase Source and the Digital Filter Timebase Rate. To see how these two property settings can affect the Minimum Pulse Width, refer to product documentation for more details."
        }
    },
    {
        -200777,
        {
            Category::Hardware, ActionRequired::Warning,
            "Desired Minimum Pulse Width could not be produced by the device."
        }
    },
    {
        -200778,
        {
            Category::Hardware, ActionRequired::Warning,
            "Sample Mode is set to a value other than Hardware Timed Single Point. This is the only value supported for counter generations when Sample Timing Type is set to Sample Clock. Change the Sample Mode or the Sample Timing Type."
        }
    },
    {
        -200779,
        {
            Category::Other, ActionRequired::Warning,
            "Reverse Coefficients for a polynomial scale are not specified. This set of coefficients must contain at least one term. The polynomial scale class constructor has overloads that can calculate the Reverse Coefficients from the Forward Coefficients if only one set of coefficients is available."
        }
    },
    {
        -200780,
        {
            Category::Other, ActionRequired::Warning,
            "Forward Coefficients for a polynomial scale are not specified. This set of coefficients must contain at least one term. The polynomial scale class constructor has overloads that can calculate the Reverse Coefficients from the Forward Coefficients if only one set of coefficients is available."
        }
    },
    {
        -200781,
        {
            Category::Other, ActionRequired::Warning,
            "Forward and Reverse Coefficients for a polynomial scale are not specified. Each of these two sets of coefficients must contain at least one term. The polynomial scale class constructor has overloads that can calculate one set of coefficients from the other set if only one set is available."
        }
    },
    {
        -200782,
        {
            Category::Other, ActionRequired::Warning,
            "Reverse Coefficients for a polynomial scale are not specified. This set of coefficients must contain at least one term."
        }
    },
    {
        -200783,
        {
            Category::Other, ActionRequired::Warning,
            "Forward Coefficients for a polynomial scale are not specified. This set of coefficients must contain at least one term."
        }
    },
    {
        -200784,
        {
            Category::Other, ActionRequired::Warning,
            "Virtual TEDS file could not be found at the specified location. Specify correct location for the Virtual TEDS file."
        }
    },
    {
        -200785,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Timing source created is invalid because of the Sample Timing Type settings. To use this timing source with a Timed Loop, set the Sample Timing Type to Change Detection. You can configure the Sample Timing Type to Change Detection while setting related properties through the DAQmx Timing (Change Detection) VI or function."
        }
    },
    {
        -200786,
        {
            Category::Other, ActionRequired::Warning,
            "Requested Sample Clock Rate cannot be generated given the specified external Sample Clock Timebase Rate. To keep the specified Sample Clock Timebase Rate, use one of the Sample Clock Rates that can be generated."
        }
    },
    {
        -200787,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Specified Start Trigger Type is not supported for counter output tasks when the Sample Mode is Hardware Timed Single Point on this type of device. Set the Start Trigger Type to None to use this Sample Mode."
        }
    },
    {
        -200788,
        {
            Category::Hardware, ActionRequired::Warning,
            "Measured bridge offset is outside the limits allowed for offset nulling calibration for this device. Ensure your sensor is wired and functioning properly, and that its output offset is within device limits."
        }
    },
    {
        -200789,
        {
            Category::Other, ActionRequired::Warning,
            "Read cannot be performed because this version of DAQmx Read uses a data type that is too small for the channels in this task. Use a different version of DAQmx Read."
        }
    },
    {
        -200790,
        {
            Category::Other, ActionRequired::Warning,
            "Write cannot be performed because this version of DAQmx Write uses a data type that is too small for the channels in this task. Use a different version of DAQmx Write."
        }
    },
    {
        -200791,
        {
            Category::Other, ActionRequired::Warning,
            "TEDS cannot be configured for the specified channel. Ensure that your TEDS sensor is connected to the channel through a TEDS interface (for example BNC-2096, SC-2350, or SCXI-1314T), and that this interface is configured in MAX. Alternatively a virtual TEDS file can be used."
        }
    },
    {
        -200792,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "You have specified more than one physical channel which is not supported. Specify a single physical channel."
        }
    },
    {
        -200793,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "For a device of this type, setting the AO Idle Output Behavior to Maintain Existing Value is not supported when analog output is synchronized."
        }
    },
    {
        -200794,
        {
            Category::Other, ActionRequired::Warning,
            "You cannot use DAQmx Write for multiple counter channels within one task. If appropriate, create one task per counter output channel. To update multiple counter channels within one task use counter output properties."
        }
    },
    {
        -200795,
        {
            Category::Hardware, ActionRequired::Warning,
            "There was an overflow of the device onboard memory while performing a hardware timed non-buffered generation. Write only one sample per channel between two consecutive sample clocks to avoid this condition."
        }
    },
    {
        -200796,
        {
            Category::Hardware, ActionRequired::Warning,
            "Hardware timed non-buffered analog output could not be performed because Memory Mapping for Programmed I/O Enable was set to true. Disable memory mapping for hardware timed non-buffered analog output."
        }
    },
    {
        -200797,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "An empty string was specified as a terminal name which is not supported. Specify a valid terminal name."
        }
    },
    {
        -200798,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device does not support change detection for lines that do not allow digital input. Use lines that allow digital input for change detection."
        }
    },
    {
        -200799,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "DAQmx Create Timing Source created an invalid source because the specified Sample Mode is not supported when the signal is Event Counting. To use this timing source with a Timed Loop, change the Sample Mode."
        }
    },
    {
        -200800,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Event source signal specified is not supported with the Measurement Type and/or Sample Timing Type of the task."
        }
    },
    {
        -200801,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "DAQmx Create Timing Source created an invalid source because the requested signal is not supported for counter output. To use this task as the timing source with a Timed Loop, specify the Counter Output Event as the signal."
        }
    },
    {
        -200802,
        {
            Category::Other, ActionRequired::Warning,
            "Write cannot be performed when the task is not started, the sample timing type is something other than On Demand, and the output buffer size is zero. Call DAQmx Start before DAQmx Write, set auto start to true on DAQmx Write, modify the sample timing type, or change the output buffer size."
        }
    },
    {
        -200803,
        {
            Category::Hardware, ActionRequired::Warning,
            "Write cannot be performed before you start the task for on demand or hardware-timed single-point operations. Start the task before you write samples, set the autostart input on DAQmx Write to true, or use hardware timing with a sample mode of finite or continuous."
        }
    },
    {
        -200804,
        {
            Category::Hardware, ActionRequired::Warning,
            "Last External Calibration Date/Time is not available, because the last external calibration if the device was not performed using the NI-DAQmx API. Last External Calibration Date/Time will become available after you perform external calibration of the device using the NI-DAQmx API."
        }
    },
    {
        -200806,
        {
            Category::Hardware, ActionRequired::Warning,
            "Requested Minimum Pulse Width cannot be applied because the programmable filter clock has already been configured with a different Minimum Pulse Width by another task. For this type of device, there is only one programmable filter clock per device."
        }
    },
    {
        -200807,
        {
            Category::Hardware, ActionRequired::Warning,
            "Requested Minimum Pulse Width cannot be applied because the programmable filter clock has already been configured with a different Minimum Pulse Width when a different terminal was configured by the same task. For this type of device, there is only one programmable filter clock per device, and the device can use only one external timebase filter at a time."
        }
    },
    {
        -200808,
        {
            Category::Other, ActionRequired::Warning,
            "TEDS sensors cannot be configured on real-time (RT) systems. Use MAX to configure the TEDS sensor instead."
        }
    },
    {
        -200809,
        {
            Category::Other, ActionRequired::Warning,
            "TEDS sensors cannot be cleared on real-time (RT) systems."
        }
    },
    {
        -200810,
        {
            Category::Other, ActionRequired::Warning,
            "Requested string contains characters that cannot be interpreted by DAQmx due to installed language support and system locale settings. Ensure that the appropriate language support is installed on the system, and that the system locale is set correctly. For most Windows operating systems, this is done through the Regional Settings option in Control Panel. For a LabVIEW RT target, you should install \"Language Support for LabVIEW RT\" and change the locale setting for the remote system under the \"System Settings\" tab in MAX."
        }
    },
    {
        -200811,
        {
            Category::Other, ActionRequired::Warning,
            "Specified string contains characters that cannot be interpreted by DAQmx due to installed language support and system locale settings. If possible, do not use this character. Otherwise, ensure that the appropriate language support is installed on the system, and that the system locale is set correctly. For most Windows operating systems, this is done through the Regional Settings option in the Control Panel. For a LabVIEW RT target, you should install \"Language Support for LabVIEW RT\" and change the locale setting for the remote system under the \"System Setting\" tab in MAX."
        }
    },
    {
        -200812,
        {
            Category::Hardware, ActionRequired::Warning,
            "An attempt has been made to configure a trigger when analog output Sample Mode was set to Hardware Timed Single Point. Configure the analog output sample mode to something other than Hardware Timed Single Point to use a trigger."
        }
    },
    {
        -200813,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Selected Sample Mode is not supported with counter input position measurements."
        }
    },
    {
        -200814,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Onboard Clock is not supported as an Input Terminal for counter measurements. Refer to user documentation for a list of supported input terminals."
        }
    },
    {
        -200815,
        {
            Category::Other, ActionRequired::Warning,
            "TEDS sensor specifies a value for the Minimum Physical Value that is greater than or equal to the Maximum Physical Value. Replace the sensor or have the sensor repaired. If the memory is the only defective part of the sensor, consider using MAX to create a Task, a Global Channel, or a Scale to acquire data using this sensor."
        }
    },
    {
        -200816,
        {
            Category::Other, ActionRequired::Warning,
            "TEDS sensor specifies a value for the Minimum Electrical Value that is greater than or equal to the Maximum Electrical Value. Replace the sensor or have the sensor repaired. If the memory is the only defective part of the sensor, consider using MAX to create a Task, a Global Channel, or a Scale to acquire data using this sensor."
        }
    },
    {
        -200817,
        {
            Category::Hardware, ActionRequired::Warning,
            "Excitation Value can only be zero when the Input Terminal Configuration is set to Differential on this device. Change the Input Terminal Configuration or set the Excitation Value to zero."
        }
    },
    {
        -200818,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device has shut down because a sensor on the device detected a temperature in excess of the maximum recommended operating temperature. Possible causes incude excessive current on the device channels and inadequate chassis cooling. To use the device again, reduce the current and/or improve the chassis cooling. Ensure that the device has cooled and reset the device (either programmatically or through Measurements & Automation Explorer)."
        }
    },
    {
        -200819,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Programmed I/O is not supported as the Data Transfer Mechanism when the Use Only On Board Memory property is set to true. Change the Data Transfer Mechanism or set Use Only On Board Memory to false."
        }
    },
    {
        -200820,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Sample Mode of Hardware Timed Single Point is not supported for analog input channels on this type of device when the number of channels in the task is odd (not divisible by 2). Add a channel to the task, remove a channel from the task, or use a different Sample Mode."
        }
    },
    {
        -200821,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Sample Mode of Hardware Timed Single Point is not supported for analog input channels on this type of device. Use a different Sample Mode, or select a device which supports Hardware Timed Single Point."
        }
    },
    {
        -200822,
        {
            Category::Other, ActionRequired::Warning,
            "Attempt to write to the PROM on the TEDS failed because the TEDS sensor does not contain a PROM. Write the Basic TEDS data to the EEPROM of the sensor or replace the sensor."
        }
    },
    {
        -200823,
        {
            Category::Other, ActionRequired::Warning,
            "Attempt to write to the PROM on the TEDS sensor failed because the PROM has already been written and it cannot be rewritten. Do not write the Basic TEDS data to the TEDS sensor or replace the sensor."
        }
    },
    {
        -200824,
        {
            Category::Other, ActionRequired::Warning,
            "Attempt to write the Basic TEDS data to the EEPROM failed because the PROM on the TEDS sensor already contains Basic TEDS data. A TEDS sensor can contain the Basic TEDS data in either the PROM or the EEPROM, but not in both. Do not write the Basic TEDS data to the TEDS sensor or replace the sensor."
        }
    },
    {
        -200825,
        {
            Category::Other, ActionRequired::Warning,
            "Write failed because the data size is greater than the size of the EEPROM on the TEDS sensor. Make sure the data size does not exceed the EEPROM size."
        }
    },
    {
        -200826,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "TEDS sensor data being written to the TEDS sensor contains an error. Ensure your TEDS sensor data conforms to the specification."
        }
    },
    {
        -200827,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Virtual TEDS data file being written to the TEDS sensor contains an error. Ensure your Virtual TEDS data file conforms to the specification."
        }
    },
    {
        -200828,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Writing to TEDS sensors is not supported on real-time (RT) systems."
        }
    },
    {
        -200829,
        {
            Category::Other, ActionRequired::Warning,
            "DAQmx Write failed because the counter channels have different Output Types. Writes to multiple counter output channels are supported only when all of the counters have identical Output Types. Use identical Output Types for all channels. Alternatively, create multiple tasks (one for each Output Type)."
        }
    },
    {
        -200830,
        {
            Category::Hardware, ActionRequired::Warning,
            "On Demand Simultaneous Analog Output Enable and Memory Mapping for Programmed IO Enable cannot both be set to true for this device."
        }
    },
    {
        -200832,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "DAQmx Read is not supported if the Sample Timing Type is On Demand, the Auto Start property is false, and the task is not running. Start the task before reading samples by calling DAQmx Start Task, set the Read.Auto Start property to true, or change the Sample Timing Type."
        }
    },
    {
        -200833,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "DAQmx Read is not supported if the Sample Mode is Hardware Timed Single Point, the Auto Start property is false, and the task is not running. Start the task before reading samples by calling DAQmx Start Task, set the Read.Auto Start property to true, or change the Sample Mode."
        }
    },
    {
        -200834,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "DAQmx Read is not supported for non-buffered acquisitions if the Auto Start property is false and the task is not running. Start the task before reading samples by calling DAQmx Start Task, set the Read.Auto Start property to true, or call DAQmx Configure Input Buffer with a buffer size greater than zero."
        }
    },
    {
        -200835,
        {
            Category::Hardware, ActionRequired::Warning,
            "SCXI device cannot be used in this task because the power to the device was turned off after the task had been created. Call DAQmx Clear Task and then create a new task to use this SCXI device."
        }
    },
    {
        -200836,
        {
            Category::Other, ActionRequired::Warning,
            "Attempt to write to the TEDS sensor failed, possibly because the sensor is not connected properly or because the sensor is defective. Make sure the TEDS sensor is properly connected. Write to the TEDS sensor again. If the write fails again, try using another TEDS sensor. You may need to have the original TEDS sensor repaired."
        }
    },
    {
        -200837,
        {
            Category::Hardware, ActionRequired::Warning,
            "Samples per Channel must be an integer multiple of the transfer size for this device with the current Data Transfer Mechanism. Change Samples per Channel or the Data Transfer Mechanism."
        }
    },
    {
        -200838,
        {
            Category::Hardware, ActionRequired::Warning,
            "Output buffer size (in samples per channel) must be an integer multiple of the transfer size for this device with the current Data Transfer Mechanism. Change the output buffer size or the Data Transfer Mechanism."
        }
    },
    {
        -200839,
        {
            Category::Configuration, ActionRequired::Warning,
            "Tristate property cannot be set to False for any channel in the task when Sample Timing Type is Sample Clock on this device. Set the Tristate property to True for all channels or change the Sample Timing Type."
        }
    },
    {
        -200840,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Prescaler value requested is not supported by this device, given the requested Timebase Source. Set Prescaler to 1, or change the Timebase Source."
        }
    },
    {
        -200841,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Prescaler value requested is not supported by this device, given the requested Input Terminal. Set Prescaler to 1, or change the Input Terminal."
        }
    },
    {
        -200842,
        {
            Category::Hardware, ActionRequired::Warning,
            "Data Transfer Mechanism is not set to Programmed I/O, the only value supported when the Sample Mode is Hardware Timed Single Point. Set your Data Transfer Mechanism to Programmed I/O or change the Sample Mode."
        }
    },
    {
        -200843,
        {
            Category::Hardware, ActionRequired::Warning,
            "DAQmx Read did not complete before the arrival of three sample clocks which indicates that your program is not keeping up with the hardware clock. Slow down the hardware clock or else change your application so that it can keep up with the hardware clock."
        }
    },
    {
        -200844,
        {
            Category::Other, ActionRequired::Warning,
            "Task contains a 'freqout' counter channel, which cannot be updated while the task is running. Create separate tasks for the 'freqout' channel and the other counter channels if you wish to write to the other counter channels. Alternatively, stop the task, reprogram the counters, and restart the task."
        }
    },
    {
        -200845,
        {
            Category::Hardware, ActionRequired::Warning,
            "Data Transfer Mechanism is not set to Programmed I/O or DMA, the only values supported for non-buffered operations for this device and Channel Type. Set your Data Transfer Mechanism to Programmed I/O or DMA, or use buffering."
        }
    },
    {
        -200846,
        {
            Category::Other, ActionRequired::Warning,
            "Write cannot be performed when the auto start input to DAQmx Write is false, task is not running, and timing for the task is not configured or Timing Type is set to On Demand. Set auto start to true, start the task, or configure timing and specify Timing Type other than On Demand."
        }
    },
    {
        -200847,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Data Transfer Mechanism is set to Programmed I/O, which is not supported for buffered operations for this device and Channel Type. Change Data Transfer Mechanism or do not use buffering."
        }
    },
    {
        -200848,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "DAQmx Every N Samples Event is not supported within non-buffered tasks. To receive Every N Samples Event notifications, configure your task to use buffering."
        }
    },
    {
        -200849,
        {
            Category::Other, ActionRequired::Warning,
            "Number of samples to wait in a finite wait instruction must be a multiple of the alignment quantum."
        }
    },
    {
        -200850,
        {
            Category::Other, ActionRequired::Warning,
            "Number of samples to wait in a finite wait instruction must be greater than 0."
        }
    },
    {
        -200851,
        {
            Category::Hardware, ActionRequired::Warning,
            "Physical channel specified is not available through the cabled device connector used for the SCC carrier. To use the specified channel with an SCC, connect the SCC carrier to the appropriate connector on the cabled device and specify the new configuration through MAX."
        }
    },
    {
        -200852,
        {
            Category::Hardware, ActionRequired::Warning,
            "Given devices cannot be synchronized in a multiple-device task. Ensure that one of the devices in the task is in PXI slot 2, or specify the Synchronization Pulse Source and the Sample Clock Timebase Source to be from a device in PXI slot 2, even if that device is not in the task."
        }
    },
    {
        -200853,
        {
            Category::Hardware, ActionRequired::Warning,
            "Terminal specified must include the device name for the given multiple-device task. Include the device name in the terminal name. Example syntax is myDevice3/PFI4."
        }
    },
    {
        -200854,
        {
            Category::Hardware, ActionRequired::Warning,
            "Given devices cannot be synchronized in a multiple-device task because the Sample Clock Timebase Source specifies a different device from the Synchronization Pulse Source. Modify the Synchronization Pulse Source and/or the Sample Clock Timebase Source to be from the same device or leave one or both unspecified."
        }
    },
    {
        -200855,
        {
            Category::Hardware, ActionRequired::Warning,
            "Devices cannot be added to a task after configuring timing, triggering, buffers, and/or exported signals. Add all devices to the task before configuring other aspects of the task."
        }
    },
    {
        -200856,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Simulation disabling is not supported for this device, because it was created as a simulated device."
        }
    },
    {
        -200857,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified low pass cutoff frequency is not supported."
        }
    },
    {
        -200858,
        {
            Category::Hardware, ActionRequired::Warning,
            "To use Sample Clock as the Sample Timing Type for analog output on this device, call DAQmx Write before DAQmx Start."
        }
    },
    {
        -200859,
        {
            Category::Hardware, ActionRequired::Warning,
            "To use Sample Clock as the Sample Timing Type for analog output on this device, specify buffer size greater than 0 in DAQmx Configure Output Buffer."
        }
    },
    {
        -200860,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Combination of specified AI Maximum Sound Pressure Level and AI Microphone Sensitivity settings is not supported by the device. Consider using a microphone with lower sensitivity. If clipping signals at high levels is acceptable, you can use the microphone with specified sensitivity as long as you reduce the AI Maximum Sound Pressure."
        }
    },
    {
        -200861,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Combination of specified AI Maximum Sound Pressure Level, AI Microphone Sensitivity, and other related AI property settings is not supported by the device. Change the values of the related AI properties or do not set them at all. If you do not set the related AI properties, NI-DAQmx sets them for you. Alternatively, consider using a microphone with lower sensitivity."
        }
    },
    {
        -200862,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "One or more devices from a multiple-device task are in an unidentified PXI chassis, which is not supported. Identify the PXI chassis in MAX."
        }
    },
    {
        -200863,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "DAQmx Wait for Next Sample Clock is not supported by the given device for tasks containing channels of the given type or timing type. DAQmx Wait for Next Sample Clock is only supported for the hardware-timed single-point timing type."
        }
    },
    {
        -200864,
        {
            Category::Other, ActionRequired::Warning,
            "Data Transfer Request Condition being set to When Acquisition Complete is only supported when the Reference Trigger Type is other than None. Change the Data Transfer Request Condition or configure a reference trigger for the task."
        }
    },
    {
        -200865,
        {
            Category::Other, ActionRequired::Warning,
            "Data Transfer Request Condition is set to When Acquisition Complete, but the Number of Samples per Channel is greater than the On Board Buffer Size. Decrease the Number of Samples per Channel, remove some channels from the task, or change the Data Transfer Request Condition."
        }
    },
    {
        -200866,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified AO Maximum and Minimum Values are not supported given the specified AO DAC Reference and Offset Values. To keep the specified AO DAC Reference and Offset Values, change the AO Minimum and Maximum Values. To keep the specified AO Minimum and Maximum Values, supply higher reference voltage and specify the corresponding AO DAC Reference Value. Alternatively, supply an appropriate external DAC offset and specify the corresponding AO DAC Offset Value. When supplying an external DAC offset, to get the optimum accuracy, you should manually calibrate the offset. Refer to user documentation for details."
        }
    },
    {
        -200867,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified AO Maximum and Minimum Values are not supported given the specified AO DAC Reference Value. To keep the specified AO Minimum and Maximum Values, supply higher reference voltage and specify the corresponding AO DAC Reference Value. To keep the specified AO DAC Reference Value, change the AO Minimum and Maximum Values. Alternatively, supply an appropriate external DAC offset and specify the corresponding AO DAC Offset Value. When supplying an external DAC offset, to get the optimum accuracy, you should manually calibrate the offset. Refer to user documentation for details."
        }
    },
    {
        -200868,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified AO Maximum and Minimum Values are not supported given the specified AO DAC Reference Value. To keep the specified AO Minimum and Maximum Values, supply higher reference voltage and specify the corresponding AO DAC Reference Value. To keep the specified AO DAC Reference Value, change the AO Minimum and Maximum Values."
        }
    },
    {
        -200869,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified AO Maximum and Minimum Values are not supported given the specified AO DAC Offset Value. To keep the specified AO Minimum and Maximum Values, supply an appropriate offset and specify the corresponding AO DAC Offset Value. To keep the specified AO DAC Offset Value, change the AO Minimum and Maximum Values. Alternatively, supply an appropriate external DAC reference and specify the corresponding AO DAC Reference Value."
        }
    },
    {
        -200870,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified AO Maximum and Minimum Values are not supported given the specified AO DAC Offset Value. To keep the specified AO Minimum and Maximum Values, supply an appropriate offset and specify the corresponding AO DAC Offset Value. To keep the specified AO DAC Offset Value, change the AO Minimum and Maximum Values."
        }
    },
    {
        -200871,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified AO Maximum and Minimum Values are not supported given the specified AO DAC Range Low and High, and AO DAC Offset Value. To keep the specified AO DAC Range and Offset Values, change the AO Minimum and Maximum Values. To keep the specified AO Minimum and Maximum Values, supply higher reference voltage and specify the corresponding AO DAC Range. Alternatively, supply an appropriate external DAC offset and specify the corresponding AO DAC Offset Value. When supplying an external DAC offset, to get the optimum accuracy, you should manually calibrate the offset. Refer to user documentation for details."
        }
    },
    {
        -200872,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified AO Maximum and Minimum Values are not supported given the specified AO DAC Range Low and High. To keep the specified AO Minimum and Maximum Values, supply higher reference voltage and specify the corresponding AO DAC Range Low and High. To keep the specified AO DAC Range Low and High, change the AO Minimum and Maximum Values. Alternatively, supply an appropriate external DAC offset and specify the corresponding AO DAC Offset Value. When supplying an external DAC offset, to get the optimum accuracy, you should manually calibrate the offset. Refer to user documentation for details."
        }
    },
    {
        -200873,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified AO Maximum and Minimum Values are not supported given the specified AO DAC Range Low and High. To keep the specified AO Minimum and Maximum Values, supply higher reference voltage and specify the corresponding AO DAC Range Low and High. To keep the specified AO DAC Range Low and High, change the AO Minimum and Maximum Values."
        }
    },
    {
        -200874,
        {
            Category::Other, ActionRequired::Warning,
            "DAQmx Write is supported for counter output channels only while the task is running. To use DAQmx Write with the given task, invoke DAQmx Start prior to DAQmx Write. To specify the low and/or high time while the task is not running, set DAQmx properties instead of invoking DAQmx Write."
        }
    },
    {
        -200875,
        {
            Category::Other, ActionRequired::Warning,
            "DAQmx Write is supported for counter output channels only while the task is running. To use DAQmx Write with the given task, invoke DAQmx Start prior to DAQmx Write. To specify the frequency and/or the duty cycle while the task is not running, set DAQmx properties instead of invoking DAQmx Write."
        }
    },
    {
        -200876,
        {
            Category::Other, ActionRequired::Warning,
            "DAQmx Write is supported for counter output channels only while the task is running. To use DAQmx Write with the given task, invoke DAQmx Start prior to DAQmx Write. To specify the low and/or high ticks while the task is not running, set DAQmx properties instead of invoking DAQmx Write."
        }
    },
    {
        -200877,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested Every N Samples Event Interval is not supported for the given Data Transfer Mechanism and buffer size. To keep DMA or USB Bulk as the Data Transfer Mechanism, modify the buffer size and/or the Every N Samples Event Interval so the buffer size is an even multiple of the Every N Samples Event Interval. To keep the same Every N Samples Event Interval and buffer size, change the Data Transfer Mechanism to Interrupts if supported."
        }
    },
    {
        -200878,
        {
            Category::Other, ActionRequired::Warning,
            "Specified digital channel contains more bits than supported by the 16-bit version of DAQmx Port Read. Use a version of DAQmx Port Read that supports wider digital ports."
        }
    },
    {
        -200879,
        {
            Category::Other, ActionRequired::Warning,
            "Specified digital channel contains more bits than supported by the 16-bit version of DAQmx Port Write. Use the version of DAQmx Port Write that supports wider digital ports."
        }
    },
    {
        -200880,
        {
            Category::Other, ActionRequired::Warning,
            "Zero is not a supported value for the Every N Samples Event Interval. Specify an event interval greater than zero."
        }
    },
    {
        -200881,
        {
            Category::Other, ActionRequired::Warning,
            "Every N Samples Event registration has failed because the event is already registered within the task. Unregister the event before registering it again."
        }
    },
    {
        -200882,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified channel is not a valid global channel. Ensure that the Channel Name matches a channel in the Data Neighborhood in MAX. Check for typing errors."
        }
    },
    {
        -200883,
        {
            Category::Other, ActionRequired::Warning,
            "Task specified cannot be saved with interactive editing allowed, because the DAQ Assistant does not support at least one of the specified properties. Save the task with 'allow interactive editing' set to false, or specify only properties supported by the DAQ Assistant."
        }
    },
    {
        -200884,
        {
            Category::Other, ActionRequired::Warning,
            "Channel specified cannot be saved with interactive editing allowed, because the DAQ Assistant does not support at least one of the specified properties. Save the channel with 'allow interactive editing' set to false, or specify only properties supported by the DAQ Assistant."
        }
    },
    {
        -200885,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Combination of Reference Clock Source and Sample Clock Timebase Source specified is not supported by this device. To use the Reference Clock Source specified, do not set the Sample Clock Timebase Source. NI-DAQmx will set it to its default value: OnboardClock. To use the Sample Clock Timebase Source specified, do not set the Reference Clock Source. NI-DAQmx will set it to its default value: none."
        }
    },
    {
        -200886,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Start Trigger Digital Pattern Source requested is not supported because at least one of the corresponding channels is not tristated. Ensure all the corresponding channels are tristated or choose different channels."
        }
    },
    {
        -200887,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Start Trigger Digital Pattern Source requested is not supported because at least one of the corresponding channels is not in the task. Ensure all the corresponding channels are in the task or choose different channels."
        }
    },
    {
        -200888,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Reference Trigger Digital Pattern Source requested is not supported because at least one of the corresponding channels is not tristated. Ensure all the corresponding channels are tristated or choose different channels."
        }
    },
    {
        -200889,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Reference Trigger Digital Pattern Source requested is not supported because at least one of the corresponding channels is not in the task. Ensure all the corresponding channels are in the task or choose different channels."
        }
    },
    {
        -200890,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Active Edge requested is not supported because the Sample Clock Source is OnboardClock. To use the selected Sample Clock Source, set Sample Clock Active Edge to Rising Edge."
        }
    },
    {
        -200891,
        {
            Category::Hardware, ActionRequired::Warning,
            "Export of sample clock is supported by this device only when the Sample Clock Source is OnboardClock. Consider alternative methods for gaining access to the clock signal."
        }
    },
    {
        -200892,
        {
            Category::Hardware, ActionRequired::Warning,
            "When the Sample Timing Type is Change Detection and the Trigger Type is Digital Pattern Match, the relevant physical channels must be consistent for this device. Specifically, the Change Detection Rising Edge Physical Channels must match the physical channels from the Trigger Digital Pattern Source for which the Trigger Digital Pattern string is 0 or 1."
        }
    },
    {
        -200893,
        {
            Category::Hardware, ActionRequired::Warning,
            "Change Detection Rising and Falling Edge Physical Channels must be set identically on this device."
        }
    },
    {
        -200894,
        {
            Category::Other, ActionRequired::Warning,
            "Number of values specified via the Start Trigger Digital Pattern does not match the number of physical lines requested via the Start Trigger Digital Pattern Source. Change one or both of the properties so these two numbers are equal."
        }
    },
    {
        -200895,
        {
            Category::Other, ActionRequired::Warning,
            "Number of values specified in Reference Trigger Digital Pattern does not match the number of physical lines requested in Reference Trigger Digital Pattern Source. Change one or both of the properties so these two numbers are equal."
        }
    },
    {
        -200896,
        {
            Category::Hardware, ActionRequired::Warning,
            "Export of the requested trigger is supported on this device only when the corresponding Trigger Type is Digital Edge."
        }
    },
    {
        -200897,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Task contains a physical channel not supported by this device, given the requested Sample Timing Type. To keep the Sample Timing Type, use physical lines from port0/line0 through port3/line7. To access the requested channel, change the Sample Timing Type."
        }
    },
    {
        -200898,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Partial use of physical lines within a physical port is not supported by this device, given the requested Sample Timing Type. Consider specifying the entire port and tristating the lines you do not want driven."
        }
    },
    {
        -200899,
        {
            Category::Hardware, ActionRequired::Warning,
            "Port 1 cannot be used without port 0 on this device given the Sample Timing Type. You can use ports 0 and 2 by themselves. To use port 1, you also need to use port 0."
        }
    },
    {
        -200900,
        {
            Category::Hardware, ActionRequired::Warning,
            "Port 3 cannot be used without port 2 on this device given the Sample Timing Type. You can use ports 0 and 2 by themselves. To use port 3, you also need to use port 2."
        }
    },
    {
        -200901,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Digital Pattern string specified contains an invalid character."
        }
    },
    {
        -200902,
        {
            Category::Other, ActionRequired::Warning,
            "Data Voltage Low Level and Data Voltage High Level must be within a common voltage range."
        }
    },
    {
        -200903,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Reference clock is not supported by this device. Do not set the Reference Clock property."
        }
    },
    {
        -200904,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Start Trigger Type requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the Start Trigger Type property. NI-DAQmx automatically selects a compatible Start Trigger Type setting. To use the requested Start Trigger Type, select a different Timing Type."
        }
    },
    {
        -200905,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Handshake Trigger Type requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the Handshake Trigger Type property. NI-DAQmx automatically selects a compatible Handshake Trigger Type setting. To use the requested Handshake Trigger Type, select a different Timing Type."
        }
    },
    {
        -200906,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Pause Trigger Type requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the Pause Trigger Type property. NI-DAQmx automatically selects a compatible Pause Trigger Type setting. To use the requested Pause Trigger Type, select a different Timing Type."
        }
    },
    {
        -200907,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Reference Trigger Type requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the Reference Trigger Type property. NI-DAQmx automatically selects a compatible Reference Trigger Type setting. To use the requested Reference Trigger Type, select a different Timing Type."
        }
    },
    {
        -200908,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Sample Clock Source requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the Sample Clock Source property. NI-DAQmx automatically selects a compatible Sample Clock Source setting. To use the requested Sample Clock Source, select a different Timing Type."
        }
    },
    {
        -200909,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "20 Mhz Timebase Output Terminal requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the 20 Mhz Timebase Output Terminal property. NI-DAQmx automatically selects a compatible20 Mhz Timebase Output Terminal. To use the requested 20 Mhz Timebase Output Terminal, select a different Timing Type."
        }
    },
    {
        -200910,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Sample Clock Output Terminal requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the Sample Clock Output Terminal property. NI-DAQmx automatically selects a compatible Sample Clock Output Terminal. To use the requested Sample Clock Output Terminal, select a different Timing Type."
        }
    },
    {
        -200911,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Start Trigger Output Terminal requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the Start Trigger Output Terminal property. NI-DAQmx automatically selects a compatible Start Trigger Output Terminal. To use the requested Start Trigger Output Terminal, select a different Timing Type."
        }
    },
    {
        -200912,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Reference Trigger Output Terminal requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the Reference Trigger Output Terminal property. NI-DAQmx automatically selects a compatible Reference Trigger Output Terminal. To use the requested Reference Trigger Output Terminal, select a different Timing Type."
        }
    },
    {
        -200913,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Ready For Transfer Event Output Terminal requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the Ready For Transfer Event Output Terminal property. NI-DAQmx automatically selects a compatible Reference Trigger Event Output Terminal. To use the requested Ready For Transfer Trigger Event Output Terminal, select a different Timing Type."
        }
    },
    {
        -200914,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Change Detection Event Output Terminal requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the Change Detection Event Output Terminal property. NI-DAQmx automatically selects a compatible Change Detection Output Terminal. To use the requested Change Detection Event Output Terminal, select a different Timing Type."
        }
    },
    {
        -200915,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Handshake Event Output Terminal requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the Handshake Event Output Terminal property. NI-DAQmx automatically selects a compatible Handshake Event Output Terminal. To use the requested Handshake Event Output Terminal, select a different Timing Type."
        }
    },
    {
        -200916,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Sample Timebase Divisor requested is not supported given the requested Timing Type. To use the requested Timing Type, do not set the Sample Timebase Divisor property. NI-DAQmx automatically selects a compatible Sample Timebase Divisor. To use the requested Sample Timebase Divisor, select a different Timing Type."
        }
    },
    {
        -200917,
        {
            Category::Hardware, ActionRequired::Warning,
            "Every N Samples Event Interval requested must be an integer multiple of two for analog output tasks on this device."
        }
    },
    {
        -200918,
        {
            Category::Other, ActionRequired::Warning,
            "Global channel name specified is already used for a task in the Data Neighborhood. NI-DAQmx does not support overlapping task and global channel names. Select a different name."
        }
    },
    {
        -200919,
        {
            Category::Other, ActionRequired::Warning,
            "Task name specified is already used for a global channel in the Data Neighborhood. NI-DAQmx does not support overlapping task and global channel names. Select a different name."
        }
    },
    {
        -200920,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Requested Every N Samples Event Interval is not supported for the given buffer size. Modify the buffer size and/or the Every N Samples Event Interval so the buffer size is an even multiple of the Every N Samples Event Interval."
        }
    },
    {
        -200921,
        {
            Category::Other, ActionRequired::Warning,
            "Channel specified cannot be saved with interactive editing allowed, because the only digital channels with multiple lines supported by the DAQ Assistant are entire ports. Save the channel with 'allow interactive editing' set to false. Alternatively, change the channel so it contains an entire port, or break it up into channels with individual lines."
        }
    },
    {
        -200922,
        {
            Category::Other, ActionRequired::Warning,
            "Channel specified cannot be saved with interactive editing allowed, because the DAQ Assistant does not support digital channels with different settings for different lines. Save the channel with 'allow interactive editing' set to false. Alternatively, set all the lines in the channel identically, or break the channel up into channels with individual lines."
        }
    },
    {
        -200923,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Wait Mode property is not supported for the given non-buffered task. Do not use this property inside the task, or change the task to be buffered."
        }
    },
    {
        -200924,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Wait Mode specified is not supported for the given non-buffered task. Specify a different Wait Mode, or change the task to be buffered."
        }
    },
    {
        -200925,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Rising and Falling Edge Physical Channels for Change Detection requested are not supported because at least one of the channels is not tristated. Ensure all the corresponding channels are tristated or choose different channels."
        }
    },
    {
        -200926,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Rising and Falling Edge Physical Channels for Change Detection requested are not supported because at least one of the corresponding channels is not in the task. Ensure all the corresponding channels are in the task or choose different channels."
        }
    },
    {
        -200927,
        {
            Category::Configuration, ActionRequired::Warning,
            "Trigger Type was set to Digital Pattern Match but no physical channels were specified as the Digital Pattern Source. Specify physical channels for property."
        }
    },
    {
        -200928,
        {
            Category::Hardware, ActionRequired::Warning,
            "Pattern match hardware for this device can only be used for one trigger."
        }
    },
    {
        -200929,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Two specified ports alone are not supported given the Sample Timing Type on this device. To use only two ports, specify ports 0 and 1, or ports 2 and 3. To use the two ports specified, use four ports (0, 1, 2, and 3) on the device, and disregard data from the unwanted ports."
        }
    },
    {
        -200930,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Two specified ports alone are not supported given the Sample Timing Type on this device. To use only two ports, specify ports 0 and 1, or ports 2 and 3. To use the two ports specified, use four ports (0, 1, 2, and 3) on the device. Ensure the lines from the unwanted ports are unwired, tristated, or are connected so no equipment can be damaged."
        }
    },
    {
        -200931,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Three ports alone are not supported given the Sample Timing Type on this device. Specify four ports (0, 1, 2, and 3), and disregard data from the unwanted port."
        }
    },
    {
        -200932,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Three ports alone are not supported given the Sample Timing Type on this device. Specify four ports (0, 1, 2, and 3). Ensure the lines from the unwanted port are unwired, tristated, or are connected so no equipment can be damaged."
        }
    },
    {
        -200933,
        {
            Category::Configuration, ActionRequired::Warning,
            "Operation cannot be performed, because the Channel Calibration Expiration Date is not specified, and Channel Calibration Enable property is set to True. To use channel calibration, specify the Expiration Date; otherwise, set Channel Calibration Enable to false."
        }
    },
    {
        -200934,
        {
            Category::Configuration, ActionRequired::Warning,
            "Operation cannot be performed, because the Channel Calibration Expiration Date has passed, and the Channel Calibration Apply Calibration if Expired property is False. Update the channel calibration, including the Expiration Date, or set Apply Calibration If Expired to True."
        }
    },
    {
        -200935,
        {
            Category::Configuration, ActionRequired::Warning,
            "Operation cannot be performed, because the Channel Calibration Scale Type is not specified and Channel Calibration Enable property is set to True. To use channel calibration, specify the Scale Type; otherwise, set Channel Calibration Enable to false."
        }
    },
    {
        -200936,
        {
            Category::Configuration, ActionRequired::Warning,
            "Operation cannot be performed, because the Channel Calibration Table Pre-Scaled Values property is not specified when the Channel Calibration Scale Type is Table."
        }
    },
    {
        -200937,
        {
            Category::Configuration, ActionRequired::Warning,
            "Operation cannot be performed, because the Channel Calibration Table Scaled Values property is not specified when the Channel Calibration Scale Type is Table."
        }
    },
    {
        -200938,
        {
            Category::Configuration, ActionRequired::Warning,
            "Operation cannot be performed, because the number of elements in the array specified for the Channel Calibration Table Pre-Scaled Values property is not equal to the number of elements in the array specified for Channel Calibration Table Scaled Values."
        }
    },
    {
        -200939,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "The Channel Calibration Table Scale Pre-Scaled Values specified is not supported, because one of the numbers appears more than once in the specified array. Ensure unique numbers are specified in the array."
        }
    },
    {
        -200940,
        {
            Category::Configuration, ActionRequired::Warning,
            "Operation cannot be performed, because the Channel Calibration Polynomial Forward Coefficients property is not specified when the Channel Calibration Scale Type is Polynomial."
        }
    },
    {
        -200941,
        {
            Category::Configuration, ActionRequired::Warning,
            "Operation cannot be performed, because the Channel Calibration Polynomial Reverse Coefficients property is not specified when the Channel Calibration Scale Type is Polynomial."
        }
    },
    {
        -200942,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Call mechanism set to synchronous event callbacks is not supported for DAQmx events on this platform. To use DAQmx events on this platform, set the call mechanism to asynchronous event callbacks."
        }
    },
    {
        -200943,
        {
            Category::Configuration, ActionRequired::Warning,
            "Operation failed because the Data Transfer Custom Threshold property is not set, and the Data Transfer Request Condition is set to Onboard Memory Custom Threshold. Specify a value for the Data Transfer Custom Threshold, or change the Data Transfer Request Condition."
        }
    },
    {
        -200944,
        {
            Category::Configuration, ActionRequired::Warning,
            "Operation failed because the Data Transfer Custom Threshold property is set, and no value is specified for the Data Transfer Request Condition. To use the specified Data Transfer Custom Threshold, set Data Transfer Request Condition to Onboard Memory Custom Threshold. If you set Data Transfer Request Condition to any value other than Onboard Memory Custom Threshold, the Data Transfer Custom Threshold property is ignored."
        }
    },
    {
        -200945,
        {
            Category::Hardware, ActionRequired::Warning,
            "Operation failed because the Data Transfer Request Condition is set to Onboard Memory Custom Threshold. This device supports this Data Transfer Request Condition only when the Data Transfer Mechanism is set to DMA. To use the specified Data Transfer Request Condition, set Data Transfer Mechanism to DMA. Otherwise, specify a different Data Transfer Request Condition."
        }
    },
    {
        -200946,
        {
            Category::Hardware, ActionRequired::Warning,
            "Task could not be started, because the driver could not write enough data to the device. This was due to system and/or bus-bandwidth limitations. Reduce the number of programs your computer is executing concurrently. If possible, perform operations with heavy bus usage sequentially instead of in parallel. If you can't eliminate the problem, contact National Instruments support at ni.com/support."
        }
    },
    {
        -200947,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "DAQmx Events are not supported in this version of LabVIEW. To use DAQmx Events, install LabVIEW 7.1 and the LabVIEW 7.1.1 patch. The patch is available at ni.com/downloads."
        }
    },
    {
        -200948,
        {
            Category::Other, ActionRequired::Warning,
            "DAQmx tasks cannot provide a source for a Timed Loop and contain a DAQmx Signal Event at the same time. DAQmx Signal events include the Counter Output event, the Sample Complete Event, the Sample Clock Event, and the Digital Change Detection event. Tasks that contain a Timed Loop can contain DAQmx Events as long as the events are not a type of DAQmx Signal Event. See documentation for more details."
        }
    },
    {
        -200949,
        {
            Category::Other, ActionRequired::Warning,
            "You can only register one DAQmx Signal Event at a time on a task. DAQmx Signal events include the Counter Output event, the Sample Complete Event, the Sample Clock Event, and the Digital Change Detection event. Unregister the event before registering it again."
        }
    },
    {
        -200950,
        {
            Category::Other, ActionRequired::Warning,
            "Done Event registration has failed because the event is already registered within the task. Unregister the event before registering it again."
        }
    },
    {
        -200951,
        {
            Category::Other, ActionRequired::Warning,
            "DAQmx Write for counter output detected that no sample clock has occurred since the last call to write which means that the writes are happening at a rate that exceeds the sample clock rate. To avoid this problem use the Wait for Next Sample Clock in your application."
        }
    },
    {
        -200952,
        {
            Category::Hardware, ActionRequired::Warning,
            "Event Output Terminal cannot include the Trigger Source terminal in the same task for this device."
        }
    },
    {
        -200953,
        {
            Category::Hardware, ActionRequired::Warning,
            "Start Trigger Source cannot be the same as the Sample Clock Source in the same task for this device."
        }
    },
    {
        -200954,
        {
            Category::Hardware, ActionRequired::Warning,
            "Sample Clock Output Terminal cannot include the Start Trigger Source terminal in the same task for this device."
        }
    },
    {
        -200955,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Two channels in the task have different raw data format property values, which is not supported. All channels in the task must have identical raw data format property values when raw data compression is configured. Only include channels with identical raw data format property values in the task when compression is configured."
        }
    },
    {
        -200956,
        {
            Category::Other, ActionRequired::Warning,
            "Raw data compression has been configured for a channel that does not support raw data compression. Remove the channel from the task or set the Raw Data Compression Type to None."
        }
    },
    {
        -200957,
        {
            Category::Other, ActionRequired::Warning,
            "Compressed Sample Size exceeds the Resolution of the channel. Configure the Compressed Sample Size to be less than or equal to the channel Resolution."
        }
    },
    {
        -200958,
        {
            Category::Hardware, ActionRequired::Warning,
            "TEDS interface device configured in MAX was not detected. Make sure that the type of TEDS interface device configured in MAX is correct and that the device is properly connected."
        }
    },
    {
        -200959,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Two channels in the task have different raw data compression property values, which is not supported. All channels in the task must have the same raw data compression property values."
        }
    },
    {
        -200960,
        {
            Category::Software, ActionRequired::Warning,
            "DAQmx software event cannot be registered because the task is running. Register all your DAQmx software events prior to starting the task."
        }
    },
    {
        -200961,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Firmware for the device could not be downloaded, and the device cannot be used. This failure is due to a missing or damaged firmware image file. Reinstall the driver to eliminate this error."
        }
    },
    {
        -200962,
        {
            Category::Hardware, ActionRequired::Warning,
            "Firmware for this device could not be downloaded, and the device cannot be used. The failure may be due to damaged hardware. Contact National Instruments support at ni.com/support"
        }
    },
    {
        -200963,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Requested Sample Timing Type is not allowed, because there is already another task with analog output channels from the same device configured for a different Sample Timing Type. This is not supported on this device. Change your application so that all the channels from this device are used in one task, set Sample Timing Type to On Demand for all tasks, or consider using two devices for the two tasks."
        }
    },
    {
        -200964,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Every N Samples Acquired Into Buffer Event cannot be registered, because it is not supported for output tasks. Use the Every N Samples Transferred From Buffer Event."
        }
    },
    {
        -200965,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Every N Samples Transferred from Buffer Event cannot be registered, because it is not supported for input tasks. Use the Every N Samples Acquired Into Buffer Event."
        }
    },
    {
        -200966,
        {
            Category::Other, ActionRequired::Warning,
            "Every N Samples Acquired Into Buffer Event registration has failed because the event is already registered within the task. Unregister the event before registering it again."
        }
    },
    {
        -200967,
        {
            Category::Other, ActionRequired::Warning,
            "Every N Samples Transferred From Buffer Event registration has failed because the event is already registered within the task. Unregister the event before registering it again."
        }
    },
    {
        -200968,
        {
            Category::Other, ActionRequired::Warning,
            "Requested operation cannot be performed inside the aysnchronous DAQmx Event callback thread. Use synchronous callback mechanism or perform the operation in a different thread."
        }
    },
    {
        -200969,
        {
            Category::Other, ActionRequired::Warning,
            "TEDS operation failed because the corresponding physical channel is not an analog input channel."
        }
    },
    {
        -200970,
        {
            Category::Hardware, ActionRequired::Warning,
            "Every N Samples Event Interval requested must be an integer multiple of two for analog input tasks on this device."
        }
    },
    {
        -200971,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property cannot be set because the task is not running or committed. Start or commit the task prior to setting the specified property."
        }
    },
    {
        -200972,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property cannot be set because the task is not running. Start the task prior to setting the specified property."
        }
    },
    {
        -200973,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property cannot be queried because the task is not running. Start the task prior to getting the specified property."
        }
    },
    {
        -200974,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Allow Connecting DAC Reference to Ground at Runtime set to True is not supported by this device when DAC Reference Voltage Source is set to External."
        }
    },
    {
        -200975,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Querying the Counter Output Ready for New Value property is not supported by this device when the Sample Mode is set to Hardware Timed Single Point. Use DAQmx Wait for Next Sample Clock before DAQmx Write to make sure the counter is ready to accept the new value."
        }
    },
    {
        -200976,
        {
            Category::Other, ActionRequired::Warning,
            "Thermocouple CJC (cold junction compensation) Channel specified cannot be used for CJC because the corresponding physical channel does not support temperature measurement. Select a different CJC Channel, set CJC Source to Internal, or set CJC Source to Constant Value and use CJC Value to specify the temperature of the cold junction."
        }
    },
    {
        -200977,
        {
            Category::Other, ActionRequired::Warning,
            "Channel specified cannot be saved with Allow Interactive Editing set to True, because the DAQ Assistant does not support polynomial calibration scales. Save the channel with Allow Interactive Editing set to False, or use a table calibration scale."
        }
    },
    {
        -200978,
        {
            Category::Software, ActionRequired::Warning,
            "DAQmx Software Events cannot be registered with different call mechanisms on the same task. The software events for a task must all be registered with synchronous callbacks or they must be all registered with asynchronous callbacks."
        }
    },
    {
        -200979,
        {
            Category::Other, ActionRequired::Warning,
            "When you use synchronous events, you can clear, stop, abort, unreserve, or start a task only from the thread in which you registered synchronous events."
        }
    },
    {
        -200980,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "DAQmx Every N Samples Transferred from Buffer Event is not supported by the channel types or devices in your task."
        }
    },
    {
        -200981,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "DAQmx Every N Samples Acquired into Buffer Event is not supported by the channel types or devices in your task."
        }
    },
    {
        -200982,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "DAQmx Signal Events are not supported by your device. DAQmx Signal events include the Counter Output event, the Sample Complete Event, the Sample Clock Event, and the Digital Change Detection event."
        }
    },
    {
        -200983,
        {
            Category::Configuration, ActionRequired::Warning,
            "You only can get the specified property while the task is reserved, committed or while the task is running. Reserve, commit or start the task prior to getting the property."
        }
    },
    {
        -200984,
        {
            Category::Other, ActionRequired::Warning,
            "Auto Start cannot be set to True when one or more DAQmx events are registered for the task. Set Auto Start to False and start the task manually."
        }
    },
    {
        -200985,
        {
            Category::Other, ActionRequired::Warning,
            "DAQmx Write parameter Auto Start cannot be set to True when one or more DAQmx events are registered for the task. Set Auto Start to false and start the task manually."
        }
    },
    {
        -200986,
        {
            Category::Software, ActionRequired::Warning,
            "DAQmx software event cannot be unregistered because the task is running. Unregister all your DAQmx software events prior to starting the task."
        }
    },
    {
        -200987,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "DAQmx Signal Event type requested is not supported by the channel types or the devices in your task. DAQmx Signal events include the Counter Output event, the Sample Complete Event, the Sample Clock Event, and the Digital Change Detection event. Refer to product documentation for more details on which DAQmx Signal Events are supported by the channel types and devices in your task."
        }
    },
    {
        -200988,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Combination of requested values for Read Wait Mode and Wait ffor Next Sample Clock Wait Mode properties is not supported for the given task on this device. Set both properties to Wait for Interrupt or do not set either of the properties to Wait for Interrupt."
        }
    },
    {
        -200989,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Combination of requested values for Read Wait Mode and Wait for Next Sample Clock Wait Mode properties is not supported for the given task on this device. Set both properties to Wait for Interrupt or set Read Wait Mode to a value other than Wait for Interrupt."
        }
    },
    {
        -200990,
        {
            Category::Other, ActionRequired::Warning,
            "All synchronous events for the task must be registered from the same thread."
        }
    },
    {
        -200991,
        {
            Category::Other, ActionRequired::Warning,
            "Task cannot be stopped, because at least one installed event handler has not been removed. Remove all installed event handlers by calling CNiDAQmxEvent::RemoveEventHandler or CNiDAQmxEvent::RemoveAllEventHandlers. See the documentation for more information."
        }
    },
    {
        -200992,
        {
            Category::Software, ActionRequired::Warning,
            "DAQmx Software Events are generated too quickly for the driver to keep up, and some of them have been lost. Reduce the rate at which your application is generating the events. Consider reconfiguring the events you are using, or using different events."
        }
    },
    {
        -200993,
        {
            Category::Other, ActionRequired::Warning,
            "DAQmx Write failed, because it was called before the previously written value was output. This is likely a result of the sample clock period being shorter than the period of the generated pulse train. To correct this issue, increase your sample clock period and/or reduce the period of the generated pulse train."
        }
    },
    {
        -200994,
        {
            Category::Configuration, ActionRequired::Warning,
            "Requested property cannot be set while the task is running and the Sample Mode is set to Hardware Timing Single Point. Use DAQmx Write instead of setting the property."
        }
    },
    {
        -200995,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Requested Memory Mapping for Programmed IO Enable value, True, is not supported when Sample Mode is set to Hardware Timed Single Point. Change one or both of the properties."
        }
    },
    {
        -200996,
        {
            Category::Hardware, ActionRequired::Warning,
            "Data Transfer Mechanism is not set to Programmed I/O, which is the only value supported when the Sample Mode is Hardware Timed Single Point. Set your Data Transfer Mechanism to Programmed I/O or change the Sample Mode."
        }
    },
    {
        -200997,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Requested Sample Timing Type value, On Demand, is not supported when Sample Mode is Hardware Timed Single Point. Change Sample Timing Type and/or Sample Mode."
        }
    },
    {
        -200998,
        {
            Category::Hardware, ActionRequired::Warning,
            "Hardware Timed Single Point is not a supported Sample Mode for the specified Measurement Type. Change Sample Mode and/or Measurement Type."
        }
    },
    {
        -200999,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Requested property, Ready For New Value, is not supported when the Sample Timing Type is On Demand. To use the Ready For New Value property, change the Sample Timing Type."
        }
    },
    {
        -201000,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "DAQmx Events are not supported in this version of LabVIEW. LabVIEW 8.0 or later is required to use DAQmx Events."
        }
    },
    {
        -201001,
        {
            Category::Other, ActionRequired::Warning,
            "Sample Clock Rate requested is supported only if Enhanced Alias Rejection Enable is True. Set Enhanced Alias Rejection Enable to True or increase the Sample Clock Rate."
        }
    },
    {
        -201002,
        {
            Category::Other, ActionRequired::Warning,
            "Sample Clock Rate must match the frequency of the internal timebase specified as the Sample Clock Source. To use the specified Sample Clock Rate, set the Sample Clock Source to OnboardClock. To use the specified timebase as the Sample Clock, set the Sample Clock Rate to the frequency of that timebase."
        }
    },
    {
        -201003,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device cannot be accessed. Possible causes: Device is no longer present in the system. Device is not powered. Device is powered, but was temporarily without power. Device is damaged. Ensure the device is properly connected and powered. Turn the computer off and on again. If you suspect that the device is damaged, contact National Instruments at ni.com/support."
        }
    },
    {
        -201004,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device does not support simultaneous calibration of multiple channels. Calibrate channels one channel at a time, passing individual channels to different invocations of DAQmx Setup Calibration."
        }
    },
    {
        -201005,
        {
            Category::Other, ActionRequired::Warning,
            "Invoke DAQmx Setup Calibration before invoking the corresponding DAQmx Adjust Calibration."
        }
    },
    {
        -201006,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device does not support an external calibration password."
        }
    },
    {
        -201007,
        {
            Category::Other, ActionRequired::Warning,
            "Generate or finite \"Wait\" instruction expected before \"If-Else\" block. Insert a \"Generate\" or finite \"Wait\" instruction before the If-Else block."
        }
    },
    {
        -201008,
        {
            Category::Other, ActionRequired::Warning,
            "Waveform length is too small for the \"Generate\" instruction before the \"If-Else\" block."
        }
    },
    {
        -201009,
        {
            Category::Other, ActionRequired::Warning,
            "Length of waveform subset is too small for the \"Generate\" instruction before \"If-Else\" block."
        }
    },
    {
        -201010,
        {
            Category::Other, ActionRequired::Warning,
            "Marker position specified is too close to the end of the waveform in the \"Generate\" statement before the \"If-Else\" block."
        }
    },
    {
        -201011,
        {
            Category::Other, ActionRequired::Warning,
            "Wait duration is too small for the \"Wait\" instruction before the \"If-Else\" block."
        }
    },
    {
        -201012,
        {
            Category::Other, ActionRequired::Warning,
            "Clear Trigger instruction cannot be the last instruction in an \"If-Else\" block."
        }
    },
    {
        -201013,
        {
            Category::Other, ActionRequired::Warning,
            "If-Else blocks are not allowed in \"Repeat Until\" loops."
        }
    },
    {
        -201014,
        {
            Category::Other, ActionRequired::Warning,
            "If-Else blocks are not allowed in \"Finite Repeat\" loops. If possible, remove the \"Repeat\" and \"End Repeat\" instructions and explicitly duplicate the instructions originally in the loop the desired number of times."
        }
    },
    {
        -201015,
        {
            Category::Other, ActionRequired::Warning,
            "PLL lock operation failed or timed out. Ensure the module is fully inserted into the carrier."
        }
    },
    {
        -201016,
        {
            Category::Other, ActionRequired::Warning,
            "Too many compiled instructions in loop. \"Generate\" and \"Wait\" instructions each result in at least one compiled instruction. Each marker adds an additional compiled instruction. Clear instruction does not result in a compiled instruction. If possible, reduce the number of generate instructions by concatenating the waveforms on two or more consecutive generate instructions."
        }
    },
    {
        -201017,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Byte order marker of the specified file is not supported by NI-DAQmx. For tab-delimeted files, NI-DAQmx supports UTF-8, UTF-16 / UCS-2 little endian, and UTF-32 / UCS-4 little endian. For ini files, NI-DAQmx only supports UTF-8. Save the file in one of the supported formats with the appropriate byte order marker."
        }
    },
    {
        -201018,
        {
            Category::Hardware, ActionRequired::Warning,
            "You have selected an external clock source for the task, but the device importing the clock does not have the longest pipeline of all the devices in the task. This leads to an incomplete acquisition on that device because the device will not receive enough Sample Clock pulses. Route the external clock signal into the device with the longest pipeline. Refer to device documentation for information on pipeline depth."
        }
    },
    {
        -201019,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Pause triggering is not supported in a multi-device task. To configure pause triggering in a multi-device configuration, you must use no more than one device per task and manually route the clock signals in the application."
        }
    },
    {
        -201020,
        {
            Category::Other, ActionRequired::Warning,
            "Lines specified do not support change detection. Select lines that support change detection."
        }
    },
    {
        -201021,
        {
            Category::Other, ActionRequired::Warning,
            "Write recovery could not complete before detecting another Sample Clock pulse. Reduce the Sample Clock rate, increase the frequency of the generated pulse train, or set Write Recovery Mode to Poll."
        }
    },
    {
        -201022,
        {
            Category::Other, ActionRequired::Warning,
            "Write recovery could not complete before detecting another Sample Clock pulse. Reduce the Sample Clock rate or increase the frequency of the generated pulse train."
        }
    },
    {
        -201023,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "DAQmx Write failed because the previously written value has not been generated. This error can occur if the Sample Clock period is shorter than the period of the generated pulse train. Reduce the Sample Clock rate, increase the frequency of the generated pulse train, or set Write Recovery Mode to Poll."
        }
    },
    {
        -201024,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "DAQmx Write failed because the previously written value has not been generated. This error can occur if the Sample Clock period is shorter than the period of the generated pulse train. Reduce the Sample Clock rate or increase the frequency of the generated pulse train."
        }
    },
    {
        -201025,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Non-buffered hardware-timed operations are not supported for this device and Channel Type. Set the Buffer Size to greater than 0, do not configure Sample Clock timing, or set Sample Timing Type to On Demand."
        }
    },
    {
        -201026,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Data Transfer Mechanism is set to Programmed I/O which is not supported for hardware-timed operations for this device and Channel Type. Change Data Transfer Mechanism, do not configure Sample Clock timing, or set Sample Timing Type to On Demand."
        }
    },
    {
        -201027,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Counter task detected three or more missed Sample Clock pulses. Samples were lost before the application could read them. Decrease the Sample Clock rate or restructure the application so that DAQmx Read runs more frequently. Setting the Convert Error to Warning property to True does not eliminate the error, because samples were lost."
        }
    },
    {
        -201028,
        {
            Category::Other, ActionRequired::Warning,
            "Name specified is already in use. Specify a name that is not currently in use."
        }
    },
    {
        -201029,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device specified is already connected to a RTSI cable. To connect the device to another RTSI cable, remove it from the RTSI cable to which it is currently connected."
        }
    },
    {
        -201030,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device specified cannot be connected to a RTSI cable. If the device does not have a RTSI connector, you cannot connect it to a RTSI cable. If the device is a PXI device then it is automatically connected to the PXI backplane, and therefore does not need to be manually configured as connected to a RTSI cable."
        }
    },
    {
        -201031,
        {
            Category::Other, ActionRequired::Warning,
            "Address specified is already in use. Specify an address that is not in use."
        }
    },
    {
        -201032,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "SCXI slot number specified is invalid. Specify a slot number that is valid for the specified chassis."
        }
    },
    {
        -201033,
        {
            Category::Other, ActionRequired::Warning,
            "Slot specified is already occupied. Either specify a slot that is unoccupied or remove the module occupying the desired slot."
        }
    },
    {
        -201034,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Cascade digitization mode is not supported for SCXI. Select a different digitization mode."
        }
    },
    {
        -201035,
        {
            Category::Configuration, ActionRequired::Warning,
            "Digitizing Device Channels property is specified, but the Digitization Mode property is not set to parallel. Either remove the Digitizing Device Channels property or set the Digitizing property to parallel."
        }
    },
    {
        -201036,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Format of the time value specified is invalid. Enter the time value in the format: YYYY-MM-DDTHH:mm:ssZ UTC Where YYYY is the four digit year, MM is the two digit month, DD is the two digit day of the month, HH is the two digit hour of the day (24 hour clock), mm is the two digit minutes into the hour, and ss is the two digit seconds into the minute. T is a literal separator between date and time. For example, the string: 2004-10-19T16:30:45Z UTC Represents October 19th, 2004 at 4:30:45 PM GMT."
        }
    },
    {
        -201037,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Time value specified is invalid. Ensure that the time entered has only valid values for each of the fields in the time format. For example: the month section must be between 01 and 12."
        }
    },
    {
        -201038,
        {
            Category::Configuration, ActionRequired::Warning,
            "Author property cannot be set on a local channel. Remove the author property from the local channel."
        }
    },
    {
        -201039,
        {
            Category::Configuration, ActionRequired::Warning,
            "Object lacks a required property. Add the required property to the object."
        }
    },
    {
        -201040,
        {
            Category::Configuration, ActionRequired::Warning,
            "Object specified contains an extra property. Remove the extra property from the object."
        }
    },
    {
        -201041,
        {
            Category::Other, ActionRequired::Warning,
            "Product type and product number specified do not refer to the same product. Remove either the product type or the product number from the object."
        }
    },
    {
        -201042,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device specification provided does not match any hardware in the system. Change the device specification to match a device present in your system. You can also change your device specification to be less specific."
        }
    },
    {
        -201043,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device specification provided matches more than one device in the system. Change the device specification to be more specific."
        }
    },
    {
        -201044,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Specified accessory name is invalid. The name of the accessory should be in this format: accessoryProductType / connectedDeviceIdentifier / connectorNumber Connector numbers start at zero. A device with only one connector only has a connector zero."
        }
    },
    {
        -201046,
        {
            Category::Hardware, ActionRequired::Warning,
            "Accessory type specified cannot be connected to the specified device. Enter an accessory type that can be connected to the device specified."
        }
    },
    {
        -201047,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device and connector specified by the accessory can not be configured because there is already an accessory configured for that connector and device. Enter a device and connector that is not currently occupied, or remove the configuration of the existing accessory."
        }
    },
    {
        -201048,
        {
            Category::Other, ActionRequired::Warning,
            "Accessory setting specified does not apply to the accessory type. Remove the non-applicable accessory setting from the accessory specification."
        }
    },
    {
        -201049,
        {
            Category::Hardware, ActionRequired::Warning,
            "Digitizing device specified for the SCC carrier is not capable of digitizing for SCC carriers. Specify a device that is capable of digitizing for SCC carriers."
        }
    },
    {
        -201050,
        {
            Category::Other, ActionRequired::Warning,
            "Carrier specified for the SCC module could not be found. Ensure that the SCC carrier specified for the module is also defined in the configuration file."
        }
    },
    {
        -201051,
        {
            Category::Hardware, ActionRequired::Warning,
            "Controlling device specified for the TEDS interface is not capable of controlling a TEDS interface."
        }
    },
    {
        -201052,
        {
            Category::Other, ActionRequired::Warning,
            "Physical channels specified for the TEDS interface are too great in number for the specified type of TEDS interface. Reduce the number of physical channels specified."
        }
    },
    {
        -201053,
        {
            Category::Other, ActionRequired::Warning,
            "Physical channel specified for the TEDS interface is already connected to a TEDS interface. Enter a physical channel that is not currently occupied, or remove the existing physical channel configuration."
        }
    },
    {
        -201054,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Enumerated value specified is not a valid value for that enumeration. The enumerated value may have been exported from a later version of NI-DAQmx and is not supported by the version of NI-DAQmx installed on your system. Check the version specified in the file against the installed version of NI-DAQmx. You can upgrade the version of NI-DAQmx installed on your system, or change the value to one supported by the version of NI-DAQmx you have installed."
        }
    },
    {
        -201055,
        {
            Category::Configuration, ActionRequired::Warning,
            "Object contains two references to the same property. Remove one of the duplicate properties."
        }
    },
    {
        -201056,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Numeric value specified is in an invalid format. Remove any non-numeric characters from the specified numeric value."
        }
    },
    {
        -201057,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware product type specified is invalid. Enter a valid product type. If the product number you entered is an actual product type, ensure the product type is appropriate for the object you are configuring. For example, do not use the product type of a PXI device where an SCXI module is expected."
        }
    },
    {
        -201058,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware product number specified is invalid. Enter a valid product number. If the product number you entered is an actual product number, ensure the product type is appropriate for the object you are configuring. For example, do not use the product number of a PXI device where an SCXI module is expected."
        }
    },
    {
        -201059,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device information retrieval failed because PXI chassis is not identified. Use MAX or nipxiconfig to identify your chassis."
        }
    },
    {
        -201060,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Syntax error encountered in INI file. Valid INI syntax allows for the following 3 types of lines: section headers, items, and comments. A section header begins with an open bracket and ends with a closed bracket. Example: [mySection] An item has an equals sign in between two strings. Example: myItem = 46 A comment begins with a semicolon. Example: ; This is my comment."
        }
    },
    {
        -201061,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property specified cannot return its value because the custom scale for the channel does not include the value in the range or table of pre-scaled values. Ensure that the custom scale includes all potential values for this property in the range or table of pre-scaled values, or use a linear or polynomial scale."
        }
    },
    {
        -201062,
        {
            Category::Other, ActionRequired::Warning,
            "Selected lines do not support buffered operations. Ensure only lines that support buffered operations are being used in the task. If using change detection timing, the task must be changed to non-buffered to support these lines."
        }
    },
    {
        -201063,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device ID value in the driver does not match the device ID value from the device. Ensure the correct driver is being used for this device."
        }
    },
    {
        -201064,
        {
            Category::Other, ActionRequired::Warning,
            "Configuration file is missing the required header fields. Add required header information at the top of the text file prior to any data."
        }
    },
    {
        -201065,
        {
            Category::Configuration, ActionRequired::Warning,
            "Configuration file contains property names or values that are not contained within a valid table. Add the appropriate start of table string prior to property names."
        }
    },
    {
        -201066,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property setting found in a column with no property name heading. Remove the property setting from the column or add the property name to the table definition."
        }
    },
    {
        -201067,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Configuration file contains an invalid start of table identifier. The table identifier may have been exported from a later version of NI-DAQmx and is not supported by the version of NI-DAQmx installed on your system. Check the version specified in the file against the installed version of NI-DAQmx. You can upgrade the version of NI-DAQmx installed on your system, or remove the table from the file."
        }
    },
    {
        -201068,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Configuration file string contains invalid character escape sequence."
        }
    },
    {
        -201069,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Local channel name specified is invalid. Local channel names are of the form <task name>/<channel name>. Example: task1/chan1"
        }
    },
    {
        -201070,
        {
            Category::Configuration, ActionRequired::Warning,
            "Task does not include the Channels property. Specify the Channels property for this task."
        }
    },
    {
        -201071,
        {
            Category::Other, ActionRequired::Warning,
            "Task references a local channel from another task. Reference only global channels and local channels that belong to this task."
        }
    },
    {
        -201072,
        {
            Category::Other, ActionRequired::Warning,
            "Task references a local channel that does not exist in this task. Remove the reference to the missing local channel or create the local channel."
        }
    },
    {
        -201073,
        {
            Category::Other, ActionRequired::Warning,
            "Local channel specified is from a task that does not exist. Specify the task in question, move the local channel to an existing task, or change the local channel to a global channel."
        }
    },
    {
        -201074,
        {
            Category::Hardware, ActionRequired::Warning,
            "Import operation supports tasks, channels, and scales only, but a hardware object was found in the file. Remove the hardware objects from the input file."
        }
    },
    {
        -201075,
        {
            Category::Other, ActionRequired::Warning,
            "Import operation does not support tasks, channels, and scales, but such an object was found. Remove all tasks, channels, and scales from the input file."
        }
    },
    {
        -201076,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "NI-DAQmx version specified in the input file is newer than the installed NI-DAQmx version. Change the version in the file to match the installed version. The import might still fail if the file contains properties that are not supported by the installed version of NI-DAQmx."
        }
    },
    {
        -201077,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Section name specified is invalid. The format of the section name is [<objectType> <objectName>]. Example: [DAQmxDevice Dev1]"
        }
    },
    {
        -201078,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Section identifier specified is invalid. Refer to the NI-DAQmx configuration file documentation for a list of valid section identifiers. The section identifier may have been exported from a later version of NI-DAQmx and is not supported by the version of NI-DAQmx installed on your system. Check the version specified in the file against the installed version of NI-DAQmx. You can upgrade the version of NI-DAQmx installed on your system, or remove the section from the file."
        }
    },
    {
        -201079,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "SCC slot number specified is invalid. Specify a slot number that is valid for the specified carrier."
        }
    },
    {
        -201080,
        {
            Category::Other, ActionRequired::Warning,
            "SCC specified cannot be placed in the slot specified. Specify a supported SCC for the given slot or specify a slot that supports the given SCC."
        }
    },
    {
        -201081,
        {
            Category::Other, ActionRequired::Warning,
            "Channel was listed more than once in the task. A task cannot contain a channel with the same name twice. Remove the duplicate entries from the configuration file."
        }
    },
    {
        -201082,
        {
            Category::Other, ActionRequired::Warning,
            "Hidden channel was listed more than once in the task. A task cannot contain a hidden channel with the same name twice. Remove the duplicate entries from the configuration file."
        }
    },
    {
        -201083,
        {
            Category::Other, ActionRequired::Warning,
            "Task must contain at least one channel. Add a channel to the task in the configuration file."
        }
    },
    {
        -201084,
        {
            Category::Other, ActionRequired::Warning,
            "Hidden channels listed for the task were not in the channels list for the task. Add the missing hidden channels to the channel list in the configuration file."
        }
    },
    {
        -201085,
        {
            Category::Configuration, ActionRequired::Warning,
            "Thermocouple CJC channel name property must be set when the thermocouple CJC source property is Channel. Set the thermocouple CJC channel name property or set the thermocouple CJC source property to a value other than Channel."
        }
    },
    {
        -201086,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Attribute name specified is invalid. Validate the attribute name using your ADE. The attribute may have been exported from a later version of NI-DAQmx and is not supported by the version of NI-DAQmx installed on your system. Check the version specified in the file against the installed version of NI-DAQmx. You can upgrade the version of NI-DAQmx installed on your system, or remove the attribute from the file."
        }
    },
    {
        -201087,
        {
            Category::Hardware, ActionRequired::Warning,
            "Task contains physical channels on one or more devices that require you to specify the Sample Clock rate. Use the Sample Clock Timing function/VI to specify a Sample Clock rate. You cannot specify a Sample Clock rate if Mode is set to On Demand."
        }
    },
    {
        -201088,
        {
            Category::Hardware, ActionRequired::Warning,
            "Task contains physical channels on one or more devices that require a different Sample Clock Timebase Source than the one specified. Do not specify the Sample Clock Timebase Source. DAQmx will set the Sample Clock Timebase Source appropriately."
        }
    },
    {
        -201089,
        {
            Category::Hardware, ActionRequired::Warning,
            "Task contains physical channels on one or more devices that require a different Sample Clock Timebase Rate than the one specified. Do not specify the Sample Clock Timebase Rate. DAQmx will set the Sample Clock Timebase Rate appropriately."
        }
    },
    {
        -201090,
        {
            Category::Hardware, ActionRequired::Warning,
            "Task contains physical channels on one or more devices that require the driver to select the Sync Pulse Source. Do not specify the Sync Pulse Source. DAQmx will set the Sync Pulse Source appropriately."
        }
    },
    {
        -201091,
        {
            Category::Other, ActionRequired::Warning,
            "Minimum delay time between the sync pulse and start must be specified when using an external Sync Pulse Source. Specify SyncPulse.MinDelayToStart."
        }
    },
    {
        -201092,
        {
            Category::Other, ActionRequired::Warning,
            "Sync pulse cannot be exported when using an external sync pulse source. Do not export the sync pulse, or use the internal chassis sync pulse source."
        }
    },
    {
        -201093,
        {
            Category::Hardware, ActionRequired::Warning,
            "Task contains physical channels that have incompatible hardware restrictions for their Sample Clock Rates. Remove incompatible physical channels from the task."
        }
    },
    {
        -201094,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "An active device was specified for the attribute but it is not supported for channel expansion tasks. Do not specify an active device when setting the attribute or do not use channel expansion."
        }
    },
    {
        -201095,
        {
            Category::Software, ActionRequired::Warning,
            "Driver was unloaded and then reloaded at a different base address after the session was created. Session is unusable. Close and reopen the session."
        }
    },
    {
        -201096,
        {
            Category::Other, ActionRequired::Warning,
            "Sample clock timebase rate must be specified when using an external sample clock timebase."
        }
    },
    {
        -201097,
        {
            Category::Other, ActionRequired::Warning,
            "Sample Clock Rate requested is supported only if Enhanced Alias Rejection Enable is False. Set Enhanced Alias Rejection Enable to False or decrease the Sample Clock Rate."
        }
    },
    {
        -201098,
        {
            Category::Hardware, ActionRequired::Warning,
            "When enabling auto zero on this device, all channels that are using auto zero must have the same auto zero mode. Channels with auto zero disabled may be present in the same task as channels with auto zero enabled. Select the same auto zero mode for all channels that are using auto zero."
        }
    },
    {
        -201099,
        {
            Category::Configuration, ActionRequired::Warning,
            "Task specified cannot be saved because the DAQmx Timing properties were specified on a per device basis using the More: AI Convert: ActiveDevs property."
        }
    },
    {
        -201100,
        {
            Category::Other, ActionRequired::Warning,
            "Sample clock timebase divisor may not be specified when an external sample clock source is specified. Change the sample clock source to onboard clock or do not configure the sample clock timebase divisor."
        }
    },
    {
        -201101,
        {
            Category::Hardware, ActionRequired::Warning,
            "Number of channels in task exceeds the device maximum given the requested Timing Type. Reduce the number of channels or select a different Timing Type."
        }
    },
    {
        -201102,
        {
            Category::Hardware, ActionRequired::Warning,
            "Analog trigger source must be the first channel of the device in the acquisition or a valid analog trigger terminal. Create your channels in a different order so that this channel is first, select a different channel from this device, or select the first channel from another device in the task. If you explicitly named the virtual channel in DAQmx Create Channel you must use the name assigned to that channel."
        }
    },
    {
        -201103,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device support an analog channel as the source of an analog pause trigger only when it is the only channel from this device in the task. Remove all of this device's channels currently in the task except the channel that will be used as the analog trigger source, change the analog trigger source to a terminal, or select a channel from another device that only has one channel in the task."
        }
    },
    {
        -201104,
        {
            Category::Hardware, ActionRequired::Warning,
            "An attempt has been made to use an analog trigger in multiple situations with differing properties. Change the analog trigger properties so they are the same, select an analog trigger source from another device for one of the triggers, or do not use an analog trigger for all situations."
        }
    },
    {
        -201105,
        {
            Category::Hardware, ActionRequired::Warning,
            "Resource requested by this task has already been reserved by a different task with conflicting settings. Unreserve any other tasks using this device, or change their settings to be compatible with this task."
        }
    },
    {
        -201106,
        {
            Category::Other, ActionRequired::Warning,
            "Physical channel specified may only be used if the C Series module is installed in a slot that supports this physical channel. Move your cDAQ module to a slot that supports this physical channel."
        }
    },
    {
        -201107,
        {
            Category::Other, ActionRequired::Warning,
            "Selected lines do not support buffered operations if the C Series module is installed in the specified slot. Ensure only lines that support buffered operations are being used in the task. If using change detection timing, the task must be changed to nonbuffered to support these lines. Move your cDAQ module to a slot that supports buffered operations."
        }
    },
    {
        -201108,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device does not support both analog modulation and digital modulation simultaneously."
        }
    },
    {
        -201109,
        {
            Category::Other, ActionRequired::Warning,
            "You cannot use the Ref In/Out connector as both an input and an output at the same time."
        }
    },
    {
        -201110,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Digital Modulation User Defined Waveform is invalid."
        }
    },
    {
        -201111,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device does not support User Defined Waveform with OOK modulation."
        }
    },
    {
        -201112,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device component test failed. If problem persists, contact National Instruments technical support at www.ni.com/support."
        }
    },
    {
        -201113,
        {
            Category::Other, ActionRequired::Warning,
            "Power level is too low. OOK modulation requires the bypass path to be used, and power levels this low must use the main path."
        }
    },
    {
        -201114,
        {
            Category::Hardware, ActionRequired::Warning,
            "Multi-device tasks with channels from both 446x and 447x devices require a 446x device to be in PXI slot 2."
        }
    },
    {
        -201115,
        {
            Category::Other, ActionRequired::Warning,
            "Waveform length must be a multiple of the waveform quantum."
        }
    },
    {
        -201117,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property setting must be identical for all channels in the task."
        }
    },
    {
        -201118,
        {
            Category::Configuration, ActionRequired::Warning,
            "Operation must be performed on the entire task. It cannot be performed only on specific devices in the task. Do not use the indexer, Item property in Visual Basic, or index operator in C++ to specify device names when performing this operation."
        }
    },
    {
        -201119,
        {
            Category::Configuration, ActionRequired::Warning,
            "Next Write is Last property not settable if Regeneration Mode is set to Allow Regeneration."
        }
    },
    {
        -201120,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property requested is incompatible with the given Timing Response Mode. NI-DAQmx can automatically select a compatible property value for you. To use the requested Timing Response Mode, do not set the specified property and allow NI-DAQmx to set it for you. To use the requested property value, choose a different value for the Timing Response Mode."
        }
    },
    {
        -201121,
        {
            Category::Other, ActionRequired::Warning,
            "Task cannot be reserved because the CPU does not support the Streaming SIMD Extensions (SSE)."
        }
    },
    {
        -201122,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property requested is incompatible with the given Timing Type. NI-DAQmx can automatically select a compatible property value for you. To use the requested Timing Type, do not set the specified property and allow NI-DAQmx to set it for you. To use the requested property value, chose a different value for the Timing Type."
        }
    },
    {
        -201123,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device supports an analog channel as the source of an analog reference trigger only when it is the only channel from this device in the task. Remove all of the channels from this device in the task except the channel that you want to use as the analog trigger source, change the analog trigger source to a terminal, or select a channel from another device that only has one channel in the task."
        }
    },
    {
        -201124,
        {
            Category::Configuration, ActionRequired::Warning,
            "Only one task is permitted to have the Digital Output Memory Mapping for Programmed I/O Enable set to true at a time. If the value is unset, NI-DAQmx will choose a value that is compatible with the system while reserving the task. Do not set the property to true explicitly, set the property to false explicitly, or set the value to the default value to allow NI-DAQmx to choose a value that is compatible with the system."
        }
    },
    {
        -201125,
        {
            Category::Other, ActionRequired::Warning,
            "Channel properties conflict. If Analog Input Source is _aignd_vs_aignd, then Analog Input Coupling must be set to GND. If Analog Input Source is _external_channel, then Analog Input Coupling must be DC."
        }
    },
    {
        -201126,
        {
            Category::Configuration, ActionRequired::Warning,
            "DAQmx Timing property specified requires per device configuration. Explicitly specify the device(s) to which this property should apply."
        }
    },
    {
        -201127,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Your ratiometric device must use excitation for scaling. The Use Excitation for Scaling property cannot be set to false on this device. Use excitation for scaling by setting the Use Excitation for Scaling property to true. This will cause NI-DAQmx to return ratiometric data instead of voltage data which is not supported by ratiometric devices."
        }
    },
    {
        -201128,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Device index requested is invalid. The value of the index must be between one and the number of devices in the task."
        }
    },
    {
        -201129,
        {
            Category::Other, ActionRequired::Warning,
            "Memory Mapping for Programmed IO Enable cannot be set to true when Output Drive Type is Open Collector. Change Output Drive Type to Active Drive or change Memory Mapping for Programmed IO Enable to false."
        }
    },
    {
        -201130,
        {
            Category::Other, ActionRequired::Warning,
            "Memory Mapping for Programmed IO Enable setting is not compatible with some of the physical channels in the task. Change Memory Mapping for Programmed IO Enable to false or do not create the task with the incompatible physical channels."
        }
    },
    {
        -201131,
        {
            Category::Other, ActionRequired::Warning,
            "Reference and start trigger sources cannot be the same. Make the reference and start trigger sources different from one another."
        }
    },
    {
        -201132,
        {
            Category::Other, ActionRequired::Warning,
            "Attempted to write a sample beyond the final finite sample. The sample specified by the combination of position and offset will never be writable. Specify a position and offset which selects a sample up to but not beyond the final sample to generate."
        }
    },
    {
        -201133,
        {
            Category::Communication, ActionRequired::Warning,
            "Device cannot be configured for input or output because lines and/or terminals on this device are in use by another task or route. This operation requires temporarily reserving all lines and terminals for communication, which interferes with the other task or route. If possible, use DAQmx Control Task to reserve all tasks that use this device before committing any tasks that use this device. Otherwise, uncommit or unreserve the other task or disconnect the other route before attempting to configure the device for input or output."
        }
    },
    {
        -201134,
        {
            Category::Configuration, ActionRequired::Warning,
            "Number of values specified with Pause Trigger Digital Pattern property does not match the number of physical lines requested with the Pause Trigger Digital Pattern Source property. Change one or both of the properties so the two numbers are equal."
        }
    },
    {
        -201135,
        {
            Category::Configuration, ActionRequired::Warning,
            "Reference Trigger Digital Pattern Source property can be used only with the data lines of the devices. Do not specify a PFI or a RTSI line in the pattern match source."
        }
    },
    {
        -201136,
        {
            Category::Configuration, ActionRequired::Warning,
            "Pause Trigger Digital Pattern Source property can only be used with the data lines of the devices. Do not specify a PFI or a RTSI line in the pattern match source."
        }
    },
    {
        -201137,
        {
            Category::Configuration, ActionRequired::Warning,
            "Start Trigger Digital Pattern Source property can only be used with the data lines of the devices. Do not specify a PFI or a RTSI line in the pattern match source."
        }
    },
    {
        -201138,
        {
            Category::Other, ActionRequired::Warning,
            "Task cannot be restarted because the first sample is not available to generate."
        }
    },
    {
        -201139,
        {
            Category::Hardware, ActionRequired::Warning,
            "Hardware revision is newer that the latest revision supported by the currently installed driver. Please upgrade your driver to the version supplied with the device. Driver updates can also be downloaded from ni.com."
        }
    },
    {
        -201140,
        {
            Category::Hardware, ActionRequired::Warning,
            "Currently installed driver no longer supports this revision of the hardware. Please downgrade your driver to the version supplied with the device. Older driver versions can also be downloaded from ni.com."
        }
    },
    {
        -201141,
        {
            Category::Hardware, ActionRequired::Warning,
            "EEPROM format is newer than the latest revision supported by the currently installed driver. Either self-calibrate your device (this may modify the EEPROM format) or upgrade your driver to the version supplied with the device. Driver updates can also be downloaded from ni.com. If uncertain, contact National Instruments technical support."
        }
    },
    {
        -201142,
        {
            Category::Hardware, ActionRequired::Warning,
            "Currently installed driver no longer supports the EEPROM format. Either self-calibrate your device (this may modify the EEPROM format) or downgrade your driver to the version supplied with the device. Older driver versions can also be downloaded from ni.com. If uncertain, contact National Instruments technical support."
        }
    },
    {
        -201143,
        {
            Category::Hardware, ActionRequired::Warning,
            "Hardware external calibration data format is newer than the latest revision supported by the currently installed driver. Either externally calibrate your device (this may modify the calibration data) or upgrade your driver to the version supplied with the device. Driver updates can also be downloaded from ni.com. If uncertain, contact National Instruments technical support."
        }
    },
    {
        -201144,
        {
            Category::Hardware, ActionRequired::Warning,
            "Hardware self-calibration data format is newer than the latest revision supported by the currently installed driver. Either self-calibrate your device (this may modify the calibration data) or upgrade your driver to the version supplied with the device. Driver updates can also be downloaded form ni.com. If uncertain, contact National Instruments technical support."
        }
    },
    {
        -201145,
        {
            Category::Hardware, ActionRequired::Warning,
            "Currently installed driver no longer supports the hardware external calibration data format. Either externally calibrate your device (this may modify the calibration data) or downgrade your driver to the version supplied with the device. Older driver versions can also be downloaded from ni.com. If uncertain, contact National Instruments technical support."
        }
    },
    {
        -201146,
        {
            Category::Hardware, ActionRequired::Warning,
            "Currently installed driver no longer supports the hardware self-calibration data format. Either self-calibrate your device (this may modify the calibration data) or downgrade your driver to the version supplied with the device. Older driver versions can also be downloaded from ni.com. If uncertain, contact National Instruments technical support."
        }
    },
    {
        -201147,
        {
            Category::Hardware, ActionRequired::Warning,
            "Calibration procedure has failed to resolve the calibration data format conflict. Perform a complete external calibration on your device. This may modify the calibration data."
        }
    },
    {
        -201148,
        {
            Category::Other, ActionRequired::Warning,
            "Channel you are triggering on is not enabled. Enable the trigger source channel."
        }
    },
    {
        -201149,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Number of input/output points entered for the specified channel is insufficient for calibration. At least two points are needed because this device has only gain error calibration constants and not offset calibration constants. Enter more points to eliminate this error."
        }
    },
    {
        -201150,
        {
            Category::Hardware, ActionRequired::Warning,
            "Simulated device cannot be imported to replace a non-simulated device of the same name. Change the device name in the import file and try importing again."
        }
    },
    {
        -201151,
        {
            Category::Hardware, ActionRequired::Warning,
            "PXI slot and PXI chassis numbers are required when creating a new simulated PXI device. Add PXI slot and chassis number values for the device in the import file."
        }
    },
    {
        -201152,
        {
            Category::Other, ActionRequired::Warning,
            "IsSimulated flags for SCXI chassis and SCXI modules must have the same value. Either make both simulated or make both non-simulated in the import file."
        }
    },
    {
        -201153,
        {
            Category::Other, ActionRequired::Warning,
            "Non-simulated SCXI module cannot be connected to a simulated digitizer. Change the SCXI Module IsSimulated flag or change the SCXI Module so that it connects to a digitizer with the same simulation setting."
        }
    },
    {
        -201154,
        {
            Category::Hardware, ActionRequired::Warning,
            "Non-simulated SCXI module cannot be connected to a simulated cabled device. Change the SCXI module IsSimulated flag or change the SCXI module so that it connects to a cabled device with the same simulation flag setting."
        }
    },
    {
        -201155,
        {
            Category::Other, ActionRequired::Warning,
            "SCXI module type specified does not support simulation. Remove the module from the import file or change the module type."
        }
    },
    {
        -201156,
        {
            Category::Other, ActionRequired::Warning,
            "IsSimulated flags for an SCC carrier and all of the contained modules must be set to the same value. Change the IsSimulated flags to match."
        }
    },
    {
        -201157,
        {
            Category::Hardware, ActionRequired::Warning,
            "Non-simulated SCC carriers cannot be connected to simulated devices. Change the cabled device of the SCC carrier or change the IsSimulated flags on the SCC carrier to true."
        }
    },
    {
        -201158,
        {
            Category::Other, ActionRequired::Warning,
            "IsSimulated flag for cDAQ chassis and C Series modules must match. Change the IsSimulated flags in the import file so that they match."
        }
    },
    {
        -201159,
        {
            Category::Other, ActionRequired::Warning,
            "cDAQ chassis does not have a slot that matches the specified slot number. The slot number specified is probably too large. Change the slot number to be a valid slot number."
        }
    },
    {
        -201160,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device type specified does not support simulation. Remove the IsSimulated flag from the import file for this device or change the device type to one that supports simulation."
        }
    },
    {
        -201161,
        {
            Category::Other, ActionRequired::Warning,
            "Specified cDAQ chassis slot is already occupied by a C Series module. Change the slot numbers of the C Series modules in your import file so that they are unique."
        }
    },
    {
        -201162,
        {
            Category::Configuration, ActionRequired::Warning,
            "Physical channel cannot be used by the task because an output task has reserved this line and the Digital Input Tristate property is set to true. Set the Digital Input Tristate property to false."
        }
    },
    {
        -201163,
        {
            Category::Other, ActionRequired::Warning,
            "Physical channel cannot be used by the task because an input task has reserved this line to be tristated."
        }
    },
    {
        -201164,
        {
            Category::Other, ActionRequired::Warning,
            "Sample clock cannot be exported in this mode when the sample clock comes from an external source or an external timebase source."
        }
    },
    {
        -201165,
        {
            Category::Hardware, ActionRequired::Warning,
            "Firmware for this device is too new. Downgrade the firmware for this device. If you need help downgrading, visit ni.com/support."
        }
    },
    {
        -201166,
        {
            Category::Hardware, ActionRequired::Warning,
            "Firmware for this device is corrupt. Contact National Instruments for help with this device."
        }
    },
    {
        -201167,
        {
            Category::Hardware, ActionRequired::Warning,
            "Firmware for this device could not be updated. Contact National Instruments for help with this device."
        }
    },
    {
        -201168,
        {
            Category::Hardware, ActionRequired::Warning,
            "Firmware for this device is too old. Upgrade the firmware for this device."
        }
    },
    {
        -201169,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Device import failed because the device is not supported by the installed version and/or platform of NI-DAQmx. Change the device type or do not import this device."
        }
    },
    {
        -201170,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device import failed because the device does not support simulation and a device to overwrite could not be found. Change the device type or do not import this device."
        }
    },
    {
        -201171,
        {
            Category::Configuration, ActionRequired::Warning,
            "Minimum temperature specified for the thermocouple measurement falls outside of the accuracy limit when using polynomial scaling. Specify a value greater than the minumum temperature for polynomial scaling with this thermocouple type, or set the Thermocouple Scale Type property to Table."
        }
    },
    {
        -201172,
        {
            Category::Hardware, ActionRequired::Warning,
            "AI channels on this device do not support using DC coupling while using IEPE excitation. Set excitation source to none or the coupling mode to AC."
        }
    },
    {
        -201173,
        {
            Category::Hardware, ActionRequired::Warning,
            "Power supply configuration failed. Reboot or cycle the power on the device."
        }
    },
    {
        -201174,
        {
            Category::Other, ActionRequired::Warning,
            "SCXI-1600 does not support import through MAX. Deselect the SCXI-1600 in the import dialog or remove it from the import file."
        }
    },
    {
        -201175,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Dev.AssociatedResourceIDs property is not supported by the device. Remove the Dev.AssociatedResourceIDs property from the import file."
        }
    },
    {
        -201176,
        {
            Category::Hardware, ActionRequired::Warning,
            "Sample clock rate specified is too fast for the burst handshaking timing type. Change the sample clock rate to be equal to or less than the maximum value or consider using the pipelined sample clock timing type. If you use the pipelined sample clock timing type, refer to the device documentation for the differences between the burst handshake timing type and the pipelined sample clock timing type."
        }
    },
    {
        -201177,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device is not usable. The firmware was recently upgraded, and the system was not powered down and restarted. Power down the computer and restart."
        }
    },
    {
        -201178,
        {
            Category::Hardware, ActionRequired::Warning,
            "Sample clock rate specified is too fast for the sample clock timing type. Change the sample clock rate to be equal to or less than the maximum value or consider using the pipelined sample clock timing type. If you use the pipelined sample clock timing type, refer to device documentation for the differences between the sample clock timing type and the pipelined sample clock timing type."
        }
    },
    {
        -201179,
        {
            Category::Other, ActionRequired::Warning,
            "db reference value must be greater than zero."
        }
    },
    {
        -201180,
        {
            Category::Other, ActionRequired::Warning,
            "Input cal data point must be an AC Voltage for this module."
        }
    },
    {
        -201181,
        {
            Category::Other, ActionRequired::Warning,
            "Input source in not valid. Ensure that AI Input Source and AI Coupling are not both set for the same task."
        }
    },
    {
        -201182,
        {
            Category::Other, ActionRequired::Warning,
            "Internal excitation voltage selected for calibration is not valid."
        }
    },
    {
        -201183,
        {
            Category::Other, ActionRequired::Warning,
            "Internal excitation frequency selected for calibration is not valid."
        }
    },
    {
        -201184,
        {
            Category::Communication, ActionRequired::Warning,
            "Device specified for PXI backplane communication is not a PXI device."
        }
    },
    {
        -201185,
        {
            Category::Communication, ActionRequired::Warning,
            "Device specified for PXI backplane communication is not in the PXI chassis."
        }
    },
    {
        -201186,
        {
            Category::Communication, ActionRequired::Warning,
            "Device specified for PXI backplane communication is not in the rightmost slot of the PXI chassis."
        }
    },
    {
        -201187,
        {
            Category::Other, ActionRequired::Warning,
            "Cold junction compensation channel cannot be used unless the corresponding analog input channel is configured to measure temperature using a thermocouple. Use the thermocouple version of the DAQmx Create Channel VI/function to configure the channel."
        }
    },
    {
        -201188,
        {
            Category::Hardware, ActionRequired::Warning,
            "Onboard device memory overflow. Because of system and/or bus bandwidth limitations, the driver could not read data from the device fast enough to keep up with the device throughput. Reduce the number of programs your computer is executing concurrently or use a different computer to calibrate your device."
        }
    },
    {
        -201189,
        {
            Category::Hardware, ActionRequired::Warning,
            "Onboard device memory overflow. Because of system and/or bus bandwidth limitations, the driver could not read data from the device fast enough to keep up with the device throughput. This device supports high-speed (480Mb/s) USB but it is connected to a full-speed (12 Mb/s) USB port. Connect this device to a high-speed (480 Mb/s) USB port, reduce the number of programs your computer is executing concurrently, or use a different computer to calibrate your device. If you are using a USB hub, ensure that it supports high-speed operation."
        }
    },
    {
        -201190,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device is currently not usable and must be placed into firmware loader mode. Unplug the device USB cable and plug it back in. If the device is plugged into a USB hub, ensure that you unplug the device from the hub."
        }
    },
    {
        -201191,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Property specified is not supported unless excitation is enabled. Enable excitation before attempting to access the property."
        }
    },
    {
        -201192,
        {
            Category::Communication, ActionRequired::Warning,
            "Connection to target failed for the requested configuration operation. Confirm that NI-DAQmx is installed on the target."
        }
    },
    {
        -201193,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "SCXI chassis address specified is invalid. Specify an SCXI chassis address between 0 and 31."
        }
    },
    {
        -201194,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property cannot be read before reading the corresponding Channels Exist property. NI-DAQmx retrieves the channel state from the hardware when the application reads the corresponding Channels Exist property. After reading the corresponding Channels Exist property, you can retrieve other information about these channels."
        }
    },
    {
        -201195,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "PCI Express interface layer error detected. Contact National Instruments for support."
        }
    },
    {
        -201196,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "PCI Express interface layer error detected. Contact National Instruments for support."
        }
    },
    {
        -201197,
        {
            Category::Hardware, ActionRequired::Warning,
            "Circuit connected to the prototyping board causes channels to use too much power. Those channels were disabled to prevent the device from using too much power. Turn the prototyping board power switch to the off position; correct the circuit connected to the prototyping board; then turn the prototyping board power switch back on."
        }
    },
    {
        -201198,
        {
            Category::Hardware, ActionRequired::Warning,
            "Circuit connected to the prototyping board causes a short between the specified physical channel and the voltage source. The output of the function generator has been suspended to prevent the device from using too much power. Turn the prototyping board power switch to the off position; correct the circuit connected to the prototyping board; then turn the prototyping board power switch back on."
        }
    },
    {
        -201199,
        {
            Category::Hardware, ActionRequired::Warning,
            "Circuit connected to the prototyping board causes too much power to be drawn from the specified source. The output of the source has been suspended to prevent the device from using too much power. Turn the prototyping board power switch to the off position; correct the circuit connected to the prototyping board; then turn the prototyping board power switch back on."
        }
    },
    {
        -201200,
        {
            Category::Communication, ActionRequired::Warning,
            "Requested operation could not be completed because the prototyping board has been removed or disabled. The prototyping board can be disabled by either switching the prototyping board switch to the off position, or by an incorrect connection on the prototyping board causing too much power to be drawn from the device. Ensure that all connections on the prototyping board are correct and that the prototyping board is properly inserted and powered on before attempting the operation."
        }
    },
    {
        -201201,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Offset error measured for this calibration task is out of range for the device. Ensure that the reference voltage is accurate, specified correctly, and connected to the correct channel. Also ensure that the measured output voltage is specified correctly and that the device is functioning properly."
        }
    },
    {
        -201202,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Gain error measured for this calibration task is out of range for the device. Ensure that the reference voltage is accurate, specified correctly, and connected to the correct channel. Also ensure that the measured output voltage is specified correctly and that the device is functioning properly."
        }
    },
    {
        -201203,
        {
            Category::Other, ActionRequired::Warning,
            "Virtual channel specified does not support the strain gage shunt calibration procedure."
        }
    },
    {
        -201204,
        {
            Category::Other, ActionRequired::Warning,
            "Virtual channel specified does not support the Wheatstone bridge shunt calibration procedure."
        }
    },
    {
        -201205,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device simulation flag does not match the simulation flag of the RTSI cable."
        }
    },
    {
        -201206,
        {
            Category::Hardware, ActionRequired::Warning,
            "Task cannot contain a mixture of simulated devices and physical devices. Ensure the physical channels added to the task refer to all physical devices or all simulated devices."
        }
    },
    {
        -201207,
        {
            Category::Hardware, ActionRequired::Warning,
            "Active Device cannot be specified when reading or writing timing properties in this multidevice task, due to synchronization requirements."
        }
    },
    {
        -201208,
        {
            Category::Hardware, ActionRequired::Warning,
            "Sample rate specified is too fast for the ADC Timing Mode selected for this device. Decrease the sample rate or use a faster ADC Timing Mode."
        }
    },
    {
        -201209,
        {
            Category::Other, ActionRequired::Warning,
            "Waveform length is too small for the \"generate\" instruction before \"break\" block."
        }
    },
    {
        -201210,
        {
            Category::Other, ActionRequired::Warning,
            "Waveform subset length is too small for the \"generate\" instruction before \"break\" block."
        }
    },
    {
        -201211,
        {
            Category::Other, ActionRequired::Warning,
            "Wait duration is too small for the \"wait\" instruction before \"break\" block."
        }
    },
    {
        -201212,
        {
            Category::Other, ActionRequired::Warning,
            "Waveform length is too small for the \"generate\" instruction in \"break\" block."
        }
    },
    {
        -201213,
        {
            Category::Other, ActionRequired::Warning,
            "Waveform subset length is too small for the \"generate\" instruction in \"break\" block."
        }
    },
    {
        -201214,
        {
            Category::Other, ActionRequired::Warning,
            "Wait duration is too small for the \"wait\" instruction in \"break\" block."
        }
    },
    {
        -201215,
        {
            Category::Other, ActionRequired::Warning,
            "Marker position specified is either too close to the end or the beginning of the waveform, or too close to another marker in the \"generate\" statement in \"break\" block."
        }
    },
    {
        -201216,
        {
            Category::Other, ActionRequired::Warning,
            "Wait until trigger instruction not allowed in a \"break\" block."
        }
    },
    {
        -201217,
        {
            Category::Other, ActionRequired::Warning,
            "Repeat until trigger instruction not allowed in a \"break\" block."
        }
    },
    {
        -201218,
        {
            Category::Other, ActionRequired::Warning,
            "If-Else block not allowed in a \"break\" block."
        }
    },
    {
        -201219,
        {
            Category::Other, ActionRequired::Warning,
            "Break block cannot be nested in other \"break\" blocks."
        }
    },
    {
        -201220,
        {
            Category::Other, ActionRequired::Warning,
            "Clear trigger instruction not allowed in a \"break\" block."
        }
    },
    {
        -201221,
        {
            Category::Other, ActionRequired::Warning,
            "Break blocks are not allowed in finite or conditional loops."
        }
    },
    {
        -201222,
        {
            Category::Other, ActionRequired::Warning,
            "Generate or finite \"wait\" instruction expected before a \"break\" block."
        }
    },
    {
        -201223,
        {
            Category::Other, ActionRequired::Warning,
            "Tristate setting must be identical for all lines in the port."
        }
    },
    {
        -201224,
        {
            Category::Other, ActionRequired::Warning,
            "Tristate setting must be applied to all lines in the port. Include all lines in the port in the Active Channel list."
        }
    },
    {
        -201225,
        {
            Category::Hardware, ActionRequired::Warning,
            "Change detection timing cannot be used on this device while Memory Mapping for Programmed IO is enabled. Set MemMapEnable to false or use a different timing type."
        }
    },
    {
        -201226,
        {
            Category::Other, ActionRequired::Warning,
            "Memory Mapping for Programmed IO Enable setting must be the same for all virtual channels in the task."
        }
    },
    {
        -201227,
        {
            Category::Hardware, ActionRequired::Warning,
            "Timing engine requested can only be used with lines that span two contiguous ports. Use the default timing engine for the specified physical channels, use some lines from two contiguous ports, or use all of the physical data channels on the device."
        }
    },
    {
        -201228,
        {
            Category::Other, ActionRequired::Warning,
            "Watchdog timer task could not be created because one of the digital output lines in the task uses memory mapping for programmed I/O. Set DO.MemMapEnable to false for all lines in the task, or do not use a watchdog timer task."
        }
    },
    {
        -201229,
        {
            Category::Other, ActionRequired::Warning,
            "Memory mapping for programmed I/O cannot be enabled for the specified lines because a watchdog timer task uses these lines. Set DO.MemMapEnable to false for all lines in the task, or do not use a watchdog timer task."
        }
    },
    {
        -201230,
        {
            Category::Hardware, ActionRequired::Warning,
            "Simulation flag of the referenced device does not match the simulation flag of the target."
        }
    },
    {
        -201231,
        {
            Category::Other, ActionRequired::Warning,
            "Local channel is not referenced by the task specified in the local channel name."
        }
    },
    {
        -201232,
        {
            Category::Other, ActionRequired::Warning,
            "Target storage was altered by another process before the changes could be saved."
        }
    },
    {
        -201233,
        {
            Category::Other, ActionRequired::Warning,
            "Object specified could not be found in storage."
        }
    },
    {
        -201234,
        {
            Category::Other, ActionRequired::Warning,
            "Storage specified is not valid or could not be found."
        }
    },
    {
        -201235,
        {
            Category::Other, ActionRequired::Warning,
            "Required object dependency was not found in storage."
        }
    },
    {
        -201236,
        {
            Category::Communication, ActionRequired::Warning,
            "Communication mode specified is not valid for the SCXI Chassis."
        }
    },
    {
        -201237,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Physical channel name specified is invalid. Physical channel names are of the form <device name>/<physical channel name>, for example, dev1/ai0."
        }
    },
    {
        -201238,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "SCXI digitization is not supported by the device or physical channel."
        }
    },
    {
        -201239,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "SCXI multiplexed digitization is not supported by the device."
        }
    },
    {
        -201240,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Digitization mode specified is not supported by the SCXI module."
        }
    },
    {
        -201241,
        {
            Category::Communication, ActionRequired::Warning,
            "Connector 0 on the SCXI chassis communication device must be cabled to the SCXI module."
        }
    },
    {
        -201242,
        {
            Category::Communication, ActionRequired::ImmediateAction,
            "SCXI chassis communication is not supported by the device."
        }
    },
    {
        -201243,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Address specified is invalid."
        }
    },
    {
        -201244,
        {
            Category::Other, ActionRequired::Warning,
            "Module type in the source storage does not match the module type in the destination."
        }
    },
    {
        -201245,
        {
            Category::Other, ActionRequired::Warning,
            "User defined information string entered exceeds the maximum allowable string length."
        }
    },
    {
        -201246,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device cannot be created in MAX because the carrier contains no cartridge. Plug in a cartridge and attempt to create the device again."
        }
    },
    {
        -201247,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device cannot be created in MAX because a driver could not be found for the device. You may need to upgrade NI-DAQmx."
        }
    },
    {
        -201248,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device configuration may not be changed at this time because the device is locked."
        }
    },
    {
        -201249,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Wireless security setting is invalid. Ensure that all necessary settings are specified."
        }
    },
    {
        -201250,
        {
            Category::Communication, ActionRequired::Warning,
            "Connection to the network device has timed out. The network device did not respond properly for a period of time. If timeouts persist, contact your system administrator."
        }
    },
    {
        -201251,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device could not be found on the network. This usually indicates an incorrect hostname or a DNS failure."
        }
    },
    {
        -201252,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device cannot be reached because no known route to the device exists on the network."
        }
    },
    {
        -201253,
        {
            Category::Other, ActionRequired::Warning,
            "Network is currently unavailable. This usually indicates an unplugged network cable, a failing network component, or an improperly configured network."
        }
    },
    {
        -201254,
        {
            Category::Hardware, ActionRequired::Warning,
            "Serial numbers of the network device do not match the serial numbers NI-DAQmx expected. Replace the original device or module and try again, or use the reconnect button in MAX to locate the original device."
        }
    },
    {
        -201255,
        {
            Category::Hardware, ActionRequired::Warning,
            "EEPROM of this device has changed since this task began. Restart the task or reset the device to refresh the EEPROM contents."
        }
    },
    {
        -201256,
        {
            Category::Other, ActionRequired::Warning,
            "Wireless channel specified is not available for this country code configuration."
        }
    },
    {
        -201257,
        {
            Category::Other, ActionRequired::Warning,
            "Country code is not configured. This setting is required to determine available wireless channels."
        }
    },
    {
        -201258,
        {
            Category::Hardware, ActionRequired::Warning,
            "Wireless configuration has been rejected by the device."
        }
    },
    {
        -201259,
        {
            Category::Hardware, ActionRequired::Warning,
            "Network configuration has been rejected by the device."
        }
    },
    {
        -201260,
        {
            Category::Other, ActionRequired::Warning,
            "Manual control attribute cannot be read when manual control is disabled. Enable manual control before reading this attribute."
        }
    },
    {
        -201261,
        {
            Category::Other, ActionRequired::Warning,
            "Sample clock rate cannot be changed at this time. When changing the Sample clock rate for a running task, one full period of the Sample clock must complete at the previous rate before NI-DAQmx can safely update the timing circuitry."
        }
    },
    {
        -201262,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Property specified is not supported for the bus type of the device."
        }
    },
    {
        -201263,
        {
            Category::Hardware, ActionRequired::Warning,
            "Firmware for this device could not be downloaded. To retry downloading the firmware, unplug the device and plug it back in. If this problem continues, contact National Instruments for assistance."
        }
    },
    {
        -201264,
        {
            Category::Software, ActionRequired::Warning,
            "Scaled waveform is too large. After multiplication by the software scaling factor, the magnitude of each sample must be less than 1.0."
        }
    },
    {
        -201265,
        {
            Category::Hardware, ActionRequired::Warning,
            "Input voltage limits exceeded. Protection circuity disabled the inputs. Ensure proper voltage levels on device inputs."
        }
    },
    {
        -201266,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Property requested was not found. The property is either not supported by the object or has not been set."
        }
    },
    {
        -201267,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "User defined information string entered is of an invalid length."
        }
    },
    {
        -201268,
        {
            Category::Hardware, ActionRequired::Warning,
            "Current limit specified cannot be applied to the channel because all current limit resources on the device have been reserved. Use a current limit setting that is already in use for another channel, or free a current limit resource by disabling current limiting on all channels that use a common current limit."
        }
    },
    {
        -201269,
        {
            Category::Other, ActionRequired::Warning,
            "Calibration operation cannot be completed unless the prototyping board is powered on."
        }
    },
    {
        -201270,
        {
            Category::Other, ActionRequired::Warning,
            "Certificate provided is not in PEM (Privacy Enhanced Mail) format. Only PEM certificates are supported."
        }
    },
    {
        -201271,
        {
            Category::Hardware, ActionRequired::Warning,
            "Product at the address provided was not the expected type. This may be due to a module being replaced or IP addresses on the network being reassigned. Reconnect to the device in MAX or delete it from your system and rediscover it."
        }
    },
    {
        -201272,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "IP address provided is invalid. IP addresses must be of the form x.x.x.x where x is a number from 0 to 255."
        }
    },
    {
        -201273,
        {
            Category::Hardware, ActionRequired::Warning,
            "Network device is already in use by another host."
        }
    },
    {
        -201274,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Device specified is not supported in 64-bit applications. To use this device, configure your development environment to create a 32-bit application, or use a 32-bit development environment. Refer to the documentation for your development environment for more information."
        }
    },
    {
        -201275,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device cannot be calibrated using the coupling specified. Calibrate using a different coupling mode."
        }
    },
    {
        -201276,
        {
            Category::Other, ActionRequired::Warning,
            "Certificate provided is not in PEM (Privacy-enhanced Electronic Mail) or DER (Distinguished Encoding Rules) format. Only PEM or DER certificates are accepted."
        }
    },
    {
        -201277,
        {
            Category::Hardware, ActionRequired::Warning,
            "Certificate file is too large to transfer to the device."
        }
    },
    {
        -201278,
        {
            Category::Hardware, ActionRequired::Warning,
            "Consecutive writes to a digital line occurred more frequently than the device can safely allow."
        }
    },
    {
        -201279,
        {
            Category::Other, ActionRequired::Warning,
            "Coupling specified conflicts with the Measurement Type of the channel. Configure the channel to use a coupling appropriate for the measurement and sensor. For example, use DC coupling for DC sensors."
        }
    },
    {
        -201280,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Vertical Offset is not supported by this device."
        }
    },
    {
        -201281,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device does not support using more than one trigger at a time. Configure the device to use only one trigger, or use a device that supports using multiple triggers."
        }
    },
    {
        -201282,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Device power up failed. Try resetting the device. If the error persists contact National Instruments."
        }
    },
    {
        -201283,
        {
            Category::Communication, ActionRequired::ImmediateAction,
            "Internal serial communication bus failed. Try resetting the device. If the error persists, contact National Instruments."
        }
    },
    {
        -201284,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Improper chassis power levels detected. The +3.3 V fuse on the device might have blown, or there might be a problem with the +3.3 V rail on the chassis. Try resetting the device. If the error persists, contact National Instruments."
        }
    },
    {
        -201285,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Improper chassis power levels detected. The +5 V fuse on the device might have blown, or there might be a problem with the +5 V rail on the chassis. Try resetting the device. If the error persists, contact National Instruments."
        }
    },
    {
        -201286,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Improper chassis power levels detected. The +12 V fuse on the device might have blown, or there might be a problem with the +12 V rail on the chassis. Try resetting the device. If the error persists, contact National Instruments."
        }
    },
    {
        -201287,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Improper chassis power levels detected. The -12 V fuse on the device might have blown, or there might be a problem with the -12 V rail on the chassis. Try resetting the device. If the error persists, contact National Instruments."
        }
    },
    {
        -201288,
        {
            Category::Configuration, ActionRequired::Warning,
            "Property specified cannot be set while the task is reserved. Set the property prior to reserving the task or unreserve the task prior to setting the property."
        }
    },
    {
        -201289,
        {
            Category::Hardware, ActionRequired::Warning,
            "Calibration procedure for your device does not support shorted inputs. Refer to the calibration procedure for your device for more information."
        }
    },
    {
        -201290,
        {
            Category::Other, ActionRequired::Warning,
            "WEP key must be either 10 or 26 characters long."
        }
    },
    {
        -201291,
        {
            Category::Hardware, ActionRequired::Warning,
            "Pulse specifications cannot be written to a finite counter output task on this device."
        }
    },
    {
        -201292,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Sample Clock Overrun and Underflow Behaviors must be set to consistent values. Either set Overrun Behavior to Stop Task and Error and Underflow Behavior to Halt Output and Error, or set Overrun Behavior to Ignore Overruns and Underflow Behavior to Pause Until Data Available."
        }
    },
    {
        -201293,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device could not complete the calibration operation. Calibration could fail for the following reasons: 1. The actual reference signal applied for calibration was different from the value you specified. Ensure that the reference signal applied is the same as the values that were input. 2. The reference signal was not stable over the period of time that the hardware was being calibrated. Ensure that the reference signal specified is free of noise and does not drift over the duration of the calibration. 3. The device is not functioning properly."
        }
    },
    {
        -201294,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device could not complete the calibration operation. Calibration could fail for the following reasons: 1. The actual reference signal applied for calibration was different from the value you specified. Ensure that the reference signal applied is the same as the values that were input. 2. The reference signal was not stable over the period of time that the hardware was being calibrated. Ensure that the reference signal specified is free of noise and does not drift over the duration of the calibration. 3. The device is not functioning properly."
        }
    },
    {
        -201295,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device could not complete the calibration operation. Calibration could fail for the following reasons: 1. The actual reference signal applied for calibration was different from the value you specified. Ensure that the reference signal applied is the same as the values that were input. 2. The reference signal was not stable over the period of time that the hardware was being calibrated. Ensure that the reference signal specified is free of noise and does not drift over the duration of the calibration. 3. The device is not functioning properly."
        }
    },
    {
        -201296,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Sample Timing Type specified is not supported for counter output tasks on this device. Change the Sample Timing Type to Implicit."
        }
    },
    {
        -201297,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Sample and hold is not supported for SCXI modules in parallel mode using digitizer channels other than 0 through 7. Disable sample and hold, use only digitizer channels 0 through 7, or use multiplexed mode."
        }
    },
    {
        -201298,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Property specified is not supported in conjunction with a conflicting property."
        }
    },
    {
        -201299,
        {
            Category::Other, ActionRequired::Warning,
            "Log Only mode is only supported for buffered tasks. Either use Log and Read mode or configure the task as Finite or Continuous."
        }
    },
    {
        -201300,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Property specified is not supported when logging data."
        }
    },
    {
        -201301,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Reading samples is not supported in the specified logging mode. To access the samples while logging, either open the file being written to or use a different logging mode."
        }
    },
    {
        -201302,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "File permission error. You do not have the correct permissions for the file."
        }
    },
    {
        -201303,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Logging to this file format is not supported by NI-DAQmx. Convert the file to TDMS version 2.0 or later, or specify a different file."
        }
    },
    {
        -201304,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "File path specified is invalid, or the file is not a valid TDMS file. Specify the location of a valid TDMS file."
        }
    },
    {
        -201305,
        {
            Category::Other, ActionRequired::Warning,
            "Disk is full. Free up disk space, or specify a different file path."
        }
    },
    {
        -201306,
        {
            Category::Other, ActionRequired::Warning,
            "File specified is already opened for output by another task. Specify a different file path."
        }
    },
    {
        -201307,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Logging is not supported for this measurement type. Change the measurement type in order to use logging."
        }
    },
    {
        -201308,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Logging is not supported for finite counter tasks on this device. Change the Sample Mode of the task to continuous."
        }
    },
    {
        -201309,
        {
            Category::Other, ActionRequired::Warning,
            "Unable to write to disk. Ensure that the file is accessible. If the problem persists, try logging to a different file."
        }
    },
    {
        -201310,
        {
            Category::Other, ActionRequired::Warning,
            "TDMS support is not installed or is too old. Use an NI-DAQmx runtime that includes TDMS support, or install a supported version of TDMS from a stand-alone installer."
        }
    },
    {
        -201311,
        {
            Category::Other, ActionRequired::Warning,
            "File specified is already opened for output. NI-DAQmx requires exclusive write access."
        }
    },
    {
        -201312,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Logging is not supported for this channel and/or measurement type. Use a different measurement type or channel."
        }
    },
    {
        -201313,
        {
            Category::Other, ActionRequired::Warning,
            "Non-responsive counter detected. NI-DAQmx reset the counter. Counter timebases at the specified speed must remain periodic, otherwise the counter can become non-responsive. Use an internal timebase or an external timebase that remains periodic."
        }
    },
    {
        -201314,
        {
            Category::Other, ActionRequired::Warning,
            "Multiple Sample Clock pulses were detected within one period of the input signal. Use a Sample Clock rate that is slower than the input signal. If you are using an external Sample Clock, ensure that clock signal is within the jitter and voltage level specifications and without glitches."
        }
    },
    {
        -201315,
        {
            Category::Other, ActionRequired::Warning,
            "Events cannot be configured after writing samples to the task."
        }
    },
    {
        -201316,
        {
            Category::Hardware, ActionRequired::Warning,
            "Timebase specified is too fast for a hardware-timed single-point counter output task."
        }
    },
    {
        -201317,
        {
            Category::Other, ActionRequired::Warning,
            "Multiple counters are not allowed in a buffered counter output task."
        }
    },
    {
        -201318,
        {
            Category::Other, ActionRequired::Warning,
            "Sample Clock pulse occurred before a pulse could be generated using the previous pulse specification. Use a Sample Clock that is slower than the pulse train you want to generate."
        }
    },
    {
        -201319,
        {
            Category::Hardware, ActionRequired::Warning,
            "Sample Clock pulse occurred before the previous sample was acquired from all channels in the task. Use a Sample Clock rate that allows time for the device to acquire samples from all channels. If you are using an external Sample Clock, ensure that clock signal is within the jitter and voltage level specifications and without glitches."
        }
    },
    {
        -201320,
        {
            Category::Other, ActionRequired::Warning,
            "Retriggering can only be enabled for finite task with a Start trigger configured."
        }
    },
    {
        -201321,
        {
            Category::Other, ActionRequired::Warning,
            "Memory-mapped task detected data corruption because the memory was accessed by another program, such as a debugger or virus scanner. Disable other programs that might access this memory, or disable memory mapping for the task."
        }
    },
    {
        -201322,
        {
            Category::Other, ActionRequired::Warning,
            "Memory mapping must be the same setting for all simultaneous tasks that use channels from a single subsystem."
        }
    },
    {
        -201323,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Sample Clock timing is not supported when using a two-counter measurement method. Use a one-counter measurement method, or use a different timing type."
        }
    },
    {
        -201324,
        {
            Category::Other, ActionRequired::Warning,
            "Multiple counters are not allowed in a single counter output task when using Sample Clock timing."
        }
    },
    {
        -201325,
        {
            Category::Other, ActionRequired::Warning,
            "Module specified cannot be used as the first stage of a dual -stage analog input channel."
        }
    },
    {
        -201326,
        {
            Category::Other, ActionRequired::Warning,
            "Module specified cannot be used as the second stage of a dual-stage analog input channel."
        }
    },
    {
        -201327,
        {
            Category::Other, ActionRequired::Warning,
            "Modules specified are not valid for dual-stage analog input channels."
        }
    },
    {
        -201328,
        {
            Category::Other, ActionRequired::Warning,
            "Second stage of a dual-stage analog input channel cannot be empty."
        }
    },
    {
        -201329,
        {
            Category::Hardware, ActionRequired::Warning,
            "Data Transfer Mechanism must be DMA when Sample Mode is Hardware Timed Single Point on this device."
        }
    },
    {
        -201330,
        {
            Category::Other, ActionRequired::Warning,
            "Pulse train specifications cannot be modified while the task is running if Auto Increment Count is greater than 0"
        }
    },
    {
        -201331,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Memory mapping is not supported on this device for non-buffered tasks using Sample Clock timing. Disable memory mapping or change the buffer size."
        }
    },
    {
        -201332,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Logging is not supported for output tasks."
        }
    },
    {
        -201333,
        {
            Category::Other, ActionRequired::Warning,
            "Required scaling parameter has not been specified."
        }
    },
    {
        -201334,
        {
            Category::Other, ActionRequired::Warning,
            "Linear scaling requires unique electrical and physical values."
        }
    },
    {
        -201335,
        {
            Category::Other, ActionRequired::Warning,
            "Table scaling requires the same number of electrical values as physical values."
        }
    },
    {
        -201336,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Bridge scales are not supported for this measurement type. Use a custom scale for additional scaling."
        }
    },
    {
        -201337,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device names must be 254 characters or shorter."
        }
    },
    {
        -201338,
        {
            Category::Other, ActionRequired::Warning,
            "Shunt resistor location specified is not valid for this calibration procedure."
        }
    },
    {
        -201339,
        {
            Category::Hardware, ActionRequired::Warning,
            "External timebase rate specified is too fast for this device. Reduce the timebase rate to less than 1/4 the device oscillator frequency."
        }
    },
    {
        -201340,
        {
            Category::Hardware, ActionRequired::Warning,
            "Firmware on the device is out of date. Use Measurement & Automation Explorer to update the device firmware."
        }
    },
    {
        -201341,
        {
            Category::Other, ActionRequired::Warning,
            "Accessory cannot be connected while the task runs. The accessory uses different scaling or is unsupported. Ensure the accessory is seated properly and restart the task."
        }
    },
    {
        -201342,
        {
            Category::Other, ActionRequired::Warning,
            "Unsupported accessory is connected to the module. Insert a supported accessory and restart the task."
        }
    },
    {
        -201343,
        {
            Category::Hardware, ActionRequired::Warning,
            "Overcurrent detected in the power supply for the accessory connected to the module. Check the external wiring and ensure the accessory, if present, is properly seated. Then, reset the module using DAQmx Reset Device or Measurement & Automation Explorer."
        }
    },
    {
        -201344,
        {
            Category::Configuration, ActionRequired::Warning,
            "ADC Timing Mode property was set to Custom, but the Custom Timing Mode property was not set. Specify a value for Custom Timing Mode or change the ADC Timing Mode."
        }
    },
    {
        -201345,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Custom Timing Mode property is not supported unless the ADC Timing Mode property is set to Custom. Set ADC Timing Mode to Custom before setting Custom Timing Mode."
        }
    },
    {
        -201346,
        {
            Category::Hardware, ActionRequired::Warning,
            "Synchronization Type cannot be Slave without configuring the device to use an external Sync Pulse. Set the Synchronization Type to Master or configure the device to use an external Sync Pulse."
        }
    },
    {
        -201347,
        {
            Category::Hardware, ActionRequired::Warning,
            "Sync Pulse was not detected before attempting to start the task. Ensure you connected the source of the Sync Pulse to the device."
        }
    },
    {
        -201348,
        {
            Category::Other, ActionRequired::Warning,
            "Retriggered counter tasks do not support trigger skew correction. Set the Synchronization Type to Default or disable retriggering."
        }
    },
    {
        -201349,
        {
            Category::Hardware, ActionRequired::Warning,
            "Multidevice tasks using the specified devices do not support a Start Trigger and Reference Trigger from different devices. Using triggers from different devices can cause unwanted latency or incorrect behavior. Use triggers from a single device or manually synchronize the devices."
        }
    },
    {
        -201350,
        {
            Category::Hardware, ActionRequired::Warning,
            "Synchronization Type cannot be Slave unless you configure the device to use a trigger with an external source. Configure an external trigger or set Synchronization Type to None or Master."
        }
    },
    {
        -201351,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Memory mapping is not supported by this device for on-demand acquisitions."
        }
    },
    {
        -201352,
        {
            Category::Configuration, ActionRequired::Warning,
            "Analog bus line(s) in use by another device. Verify other devices are not currently using the analog bus line(s). If you intend to share the line(s) between devices, ensure the Analog Bus Sharing Enabled property is True for all shared channels."
        }
    },
    {
        -201353,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Analog bus needed by the current operation is invalid. Verify that all carriers connected using expansion bridge(s) are functioning properly."
        }
    },
    {
        -201354,
        {
            Category::Hardware, ActionRequired::Warning,
            "Cards specified cannot be used to create a SwitchBlock device. Use the current set of cards as single-card devices, or use a different set of cards to create a valid combination."
        }
    },
    {
        -201355,
        {
            Category::Hardware, ActionRequired::Warning,
            "Switch device has been disabled to prevent it from exceeding the power limit for the carrier. To recover, call DAQmx Disconnect All, or reset the device. The device can be reset either programmatically or by using Measurement & Automation Explorer. Refer to your device documentation for more information."
        }
    },
    {
        -201356,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Topology specified is not supported by this card."
        }
    },
    {
        -201357,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device must consist of at least one card."
        }
    },
    {
        -201358,
        {
            Category::Hardware, ActionRequired::Warning,
            "Simulation flag must be the same for the cards, devices, and carrier."
        }
    },
    {
        -201359,
        {
            Category::Hardware, ActionRequired::Warning,
            "Multicard devices must consist of cards contained in the same carrier."
        }
    },
    {
        -201360,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Carrier slot number specified for the card is invalid."
        }
    },
    {
        -201361,
        {
            Category::Hardware, ActionRequired::Warning,
            "Sync pulse cannot originate from the specified device for this combination of devices."
        }
    },
    {
        -201362,
        {
            Category::Hardware, ActionRequired::Warning,
            "Analog reference trigger and sync pulse must come from the same device."
        }
    },
    {
        -201363,
        {
            Category::Other, ActionRequired::Warning,
            "Minimum and maximum values specified are outside the bounds of the specified physical values for the table. Change the table or the minimum and maximum value appropriately."
        }
    },
    {
        -201364,
        {
            Category::Hardware, ActionRequired::Warning,
            "Trigger detected that could not be acted upon by the device. Slow down your trigger source."
        }
    },
    {
        -201365,
        {
            Category::Other, ActionRequired::Warning,
            "Onboard regeneration cannot be used when there are more than 16 channels in the task. Reduce the number of channels in the task or set Use Only Onboard Memory to false."
        }
    },
    {
        -201366,
        {
            Category::Other, ActionRequired::Warning,
            "Reference Clock signal was not found at the specified source."
        }
    },
    {
        -201367,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Filtering or digital synchronization of an internal signal is not supported by the device."
        }
    },
    {
        -201368,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Invalid number of calibration adjustment points provided. Use DAQmx Adjust Calibration to provide calibration points for the device."
        }
    },
    {
        -201369,
        {
            Category::Hardware, ActionRequired::Warning,
            "Hostname specified is in use by another device."
        }
    },
    {
        -201370,
        {
            Category::Hardware, ActionRequired::Warning,
            "IP address specified is in use by another device."
        }
    },
    {
        -201371,
        {
            Category::Hardware, ActionRequired::Warning,
            "Multicast DNS service instance specified is in use by another device."
        }
    },
    {
        -201372,
        {
            Category::Other, ActionRequired::Warning,
            "Open thermocouple condition detected on thermocouple channel(s) you are attempting to calibrate. Ensure thermocouples are properly connected and functioning before performing lead offset nulling calibration."
        }
    },
    {
        -201373,
        {
            Category::Other, ActionRequired::Warning,
            "Output peer-to-peer streaming with the Stream statement is not allowed in a script with If Else, Repeat Until, or Break statements."
        }
    },
    {
        -201374,
        {
            Category::Other, ActionRequired::Warning,
            "Self-calibration failed. Performing an external calibration may fix the problem."
        }
    },
    {
        -201375,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Thermocouple lead offset nulling calibration is not supported by the specified channels. Make sure the device supports thermocouple lead offset nulling calibration. Make sure all channels are thermocouple channels. Make sure open thermocouple detection is enabled. Set skip unsupported channels to true."
        }
    },
    {
        -201376,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Data read from the EEPROM on the accessory attached to the device is invalid. Verify that any accessories configured with this device are connected. If the problem continues, contact National Instruments technical support. The device might need to be recalibrated or repaired by NI."
        }
    },
    {
        -201377,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device is unusable while firmware update is in progress."
        }
    },
    {
        -201378,
        {
            Category::Other, ActionRequired::Warning,
            "Firmware version requested was not found on the system."
        }
    },
    {
        -201379,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Network devices are not supported on this platform."
        }
    },
    {
        -201380,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "The requested operation is not supported by this device."
        }
    },
    {
        -201381,
        {
            Category::Communication, ActionRequired::Warning,
            "One or more connections to external power rails are drawing too much power. The operation has been aborted to prevent the device from using too much power. Remove the connection(s) to the external power rails and restart your task."
        }
    },
    {
        -201382,
        {
            Category::Other, ActionRequired::Warning,
            "The file write size specified is not evenly divisible by the volume sector size."
        }
    },
    {
        -201383,
        {
            Category::Other, ActionRequired::Warning,
            "The file write size specified is too large. Performance can suffer if the file write size is larger than one-fourth the size of the buffer length."
        }
    },
    {
        -201384,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "The simulated C Series module is not supported on this simulated chassis."
        }
    },
    {
        -201385,
        {
            Category::Other, ActionRequired::Warning,
            "Self-calibration failed to converge. Performing an external calibration may fix the problem."
        }
    },
    {
        -201386,
        {
            Category::Other, ActionRequired::Warning,
            "Self-calibration failed. Contact National Instruments technical support at ni.com/support."
        }
    },
    {
        -201387,
        {
            Category::Other, ActionRequired::Warning,
            "The DAQmx Adjust DSA AI Calibration with Gain and Coupling function/VI was executed more than once for the same combination of gain and coupling settings. Call the DAQmx Adjust DSA AI Calibration with Gain and Coupling function/VI only once for the following combination of gain and coupling settings:"
        }
    },
    {
        -201388,
        {
            Category::Hardware, ActionRequired::Warning,
            "Network device is not reserved for this host."
        }
    },
    {
        -201389,
        {
            Category::Communication, ActionRequired::Warning,
            "Modules were inserted or removed while the connection to the network device was lost. Reset the chassis using DAQmx Reset Device or Measurement & Automation Explorer and wait for the modules to be redetected before proceeding."
        }
    },
    {
        -201390,
        {
            Category::Communication, ActionRequired::Warning,
            "Connection to the network device was lost. This can indicate an unplugged network cable, a failing network component, or a network device that is reserved by another host."
        }
    },
    {
        -201391,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device could not complete calibration because calibration was not performed for all gain and coupling settings. Use DAQmx Adjust DSA AI Calibration function/VI to calibrate for the following gain and coupling settings:"
        }
    },
    {
        -201392,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device does not support configuring tristate logic level in software."
        }
    },
    {
        -201393,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Tristate logic level is not supported on output only lines."
        }
    },
    {
        -201394,
        {
            Category::Hardware, ActionRequired::Warning,
            "Tristate logic level is only port configurable for this device."
        }
    },
    {
        -201395,
        {
            Category::Other, ActionRequired::Warning,
            "The task is not buffered or has no channels. If the task is not buffered, use the scalar version of this function. If the task has no channels, add one to the task."
        }
    },
    {
        -201396,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Filter Delay Removal is not supported when an analog start trigger is in use. Change Filter Delay Removal to false when using an analog start trigger. Refer to Filter Delay Removal in your DSA documentation for more details."
        }
    },
    {
        -201397,
        {
            Category::Hardware, ActionRequired::Warning,
            "Change detection has detected interrupts occurring at a higher rate than can be handled. The change detection task has been stopped to prevent the device from being reset because of this condition. If this is the result of unwanted noise on a digital signal, use a digital filter to eliminate unwanted digital transitions."
        }
    },
    {
        -201398,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Requested value is not supported for this property. If you did not directly set this property to the unsupported value, check other properties that you have set, as they can influence the scaled value of this property."
        }
    },
    {
        -201399,
        {
            Category::Hardware, ActionRequired::Warning,
            "Multi-device timed DIO tasks require all modules to be the same type. You can select either all your serial digital modules or all your parallel digital modules in this task."
        }
    },
    {
        -201400,
        {
            Category::Other, ActionRequired::Warning,
            "Failed to reserve file size. File size pre-allocation might require you to run the application with administrator privileges. If the operating system uses User Account Control, configure this control properly."
        }
    },
    {
        -201401,
        {
            Category::Hardware, ActionRequired::Warning,
            "Retrieving properties from the network device failed. Make sure the device is connected."
        }
    },
    {
        -201402,
        {
            Category::Other, ActionRequired::Warning,
            "The samples per file specified is not evenly divisible by the file write size. Either change the samples per file or modify the file write size. If not explicitly set, the file write size can be inferred from the buffer size, which is based on the sample rate."
        }
    },
    {
        -201403,
        {
            Category::Other, ActionRequired::Warning,
            "You have specified a new file path but did not call DAQmx Start New File. To change the file path while logging, configure Logging.SampsPerFile or call DAQmx Start New File."
        }
    },
    {
        -201404,
        {
            Category::Hardware, ActionRequired::Warning,
            "One or more cards for your NI SwitchBlock device have been inserted and/or removed while your system was powered on. This can lead to unexpected behavior. Restart your system."
        }
    },
    {
        -201405,
        {
            Category::Other, ActionRequired::Warning,
            "An expansion bridge has been inserted or removed while your system is powered on. This can lead to unexpected behavior. Restart your system."
        }
    },
    {
        -201406,
        {
            Category::Other, ActionRequired::Warning,
            "The 5 V fuse on your NI SwitchBlock carrier is blown. Refer to your documentation for help with replacing the fuse."
        }
    },
    {
        -201407,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Specified property is not supported unless Sample Mode is set to Hardware Timed Single Point."
        }
    },
    {
        -201408,
        {
            Category::Hardware, ActionRequired::Warning,
            "Averaging of data is only supported when the Sample Mode is set to Hardware Timed Single Point and the Sample Clock source is external to the device."
        }
    },
    {
        -201409,
        {
            Category::Hardware, ActionRequired::Warning,
            "The requested delay from Sample Clock is out of range for a hardware-timed single-point acquisition."
        }
    },
    {
        -201410,
        {
            Category::Hardware, ActionRequired::Warning,
            "The requested Sample Clock rate is too fast for hardware-timed single point. Consider decreasing the Sample Clock rate, increasing the convert rate, or decreasing the delay from the Sample Clock. If using an external Sample Clock source, you might also decrease the number of samples to average."
        }
    },
    {
        -201411,
        {
            Category::Communication, ActionRequired::Warning,
            "The accessory attached to the device does not support connections. Attach an accessory that supports connections."
        }
    },
    {
        -201412,
        {
            Category::Hardware, ActionRequired::Warning,
            "The specified accessory channels are not present on this device. The accessory channels should be specified for the device in the calibration session."
        }
    },
    {
        -201413,
        {
            Category::Hardware, ActionRequired::Warning,
            "There is no accessory attached to the device."
        }
    },
    {
        -201414,
        {
            Category::Communication, ActionRequired::ImmediateAction,
            "The specified connection is not supported on the attached accessory. Refer to your accessory documentation for supported connections."
        }
    },
    {
        -201415,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "The module is not supported by the NI 9163."
        }
    },
    {
        -201416,
        {
            Category::Hardware, ActionRequired::Warning,
            "Switch device has been disabled to prevent it from exceeding its simultaneous relay drive limit. To recover, disconnect a relay or channel."
        }
    },
    {
        -201417,
        {
            Category::Other, ActionRequired::Warning,
            "Timing is configured without supplying a clock signal. Either supply an external clock or use an internal timebase."
        }
    },
    {
        -201418,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device firmware has not been updated because the firmware file uploaded is corrupt or is not a valid firmware image file. Please verify that the file specified is a valid National Instruments firmware image."
        }
    },
    {
        -201419,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device firmware has not been updated because the firmware file uploaded is for a different type of device or an older revision of this device. Please verify that the firmware file is correct for this device."
        }
    },
    {
        -201420,
        {
            Category::Hardware, ActionRequired::Warning,
            "The network device is currently reserved by another host. Specify whether you want to override the other host's reservation."
        }
    },
    {
        -201421,
        {
            Category::Configuration, ActionRequired::Warning,
            "The accessory attached to the device does not support this property."
        }
    },
    {
        -201422,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Your SwitchBlock carrier contains one or more cards with power characteristics unknown to the driver. To protect the hardware from overheating, all devices within your carrier are disallowed from drawing power. Upgrade your software driver to the latest version or shutdown your system and remove any unknown or invalid card(s)."
        }
    },
    {
        -201423,
        {
            Category::Communication, ActionRequired::Warning,
            "Communication with the chassis has been interrupted. Check the cabling and/or the wireless signal to the chassis. Then reset the chassis using DAQmx Reset Device or Device Reset in MAX to re-establish communication."
        }
    },
    {
        -201424,
        {
            Category::Hardware, ActionRequired::Warning,
            "The multidevice task cannot be synchronized in its current configuration."
        }
    },
    {
        -201425,
        {
            Category::Hardware, ActionRequired::Warning,
            "The multidevice task does not have a method for synchronizing timing that is compatible with all of the included devices."
        }
    },
    {
        -201426,
        {
            Category::Hardware, ActionRequired::Warning,
            "One or more devices do not support multidevice tasks."
        }
    },
    {
        -201427,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "The specified device is not supported within the NI-DAQmx API."
        }
    },
    {
        -201428,
        {
            Category::Other, ActionRequired::Warning,
            "More than one sync pulse was detected. For proper operation, only a single sync pulse signal can be provided to all DSA modules in a task."
        }
    },
    {
        -201429,
        {
            Category::Other, ActionRequired::Warning,
            "Start trigger delay is not available when a C Series Delta-Sigma module or a Reference Clock module is in the task."
        }
    },
    {
        -201430,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "You have requested an invalid number of reference voltages to write. Ensure the number of values to write is the same as the number of entries in the array."
        }
    },
    {
        -201431,
        {
            Category::Other, ActionRequired::Warning,
            "Calibration offset adjustment has failed because the wrong channel was selected. Check the module calibration procedure to decide which channel to use for the calibration offset adjustment."
        }
    },
    {
        -201432,
        {
            Category::Hardware, ActionRequired::Warning,
            "Calibration adjustment cannot be completed on a device performing different types of measurements (for instance, voltage and current measurements). Make sure only one measurement type is being calibrated in each calibration session."
        }
    },
    {
        -201433,
        {
            Category::Other, ActionRequired::Warning,
            "Write failed because a watchdog timer task expired and changed the direction of the lines to tristate after the digital output task was committed. To avoid this, unreserve and recommit the digital output task after the watchdog timer expiration has been cleared to reconfigure the lines to output."
        }
    },
    {
        -201434,
        {
            Category::Other, ActionRequired::Warning,
            "Configuration failed because the task tried to change the direction of a line while the watchdog timer is expired. Clear the expiration of the watchdog timer task before trying to change the direction of any line, even if the line is not watched by the watchdog timer task."
        }
    },
    {
        -201435,
        {
            Category::Other, ActionRequired::Warning,
            "No samples provided to DAQmx Write to initialize buffered generation."
        }
    },
    {
        -201436,
        {
            Category::Hardware, ActionRequired::Warning,
            "Unable to route signals through the analog bus that are composed of different wire modes. Please disconnect any devices of different wire modes from the analog bus before routing this device through the analog bus."
        }
    },
    {
        -201437,
        {
            Category::Hardware, ActionRequired::Warning,
            "The timeout value specified exceeds the maximum timeout value supported by this device."
        }
    },
    {
        -201438,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "The device was rebooted after a watchdog timer expired due to unresponsive firmware or hardware components. A watchdog timer that times out can cause digital output lines to change state. To clear this error, reset or power cycle the device. Please also contact National Instruments technical support."
        }
    },
    {
        -201439,
        {
            Category::Other, ActionRequired::Warning,
            "Retriggering is not allowed for finite Sample Clock-timed counter output tasks. Reconfigure the task to use a different sample timing type or disable retriggering."
        }
    },
    {
        -201440,
        {
            Category::Hardware, ActionRequired::Warning,
            "Neither an external reference clock nor a sample clock timebase has been specified. For multi-device synchronization, you must specify the sync pulse source and either an external reference clock or sample clock timebase. Refer to the device documentation for details on multi-device synchronization."
        }
    },
    {
        -201441,
        {
            Category::Other, ActionRequired::Warning,
            "On an NI 449x, specify either the sync pulse source or the sample clock timebase source but not both."
        }
    },
    {
        -201442,
        {
            Category::Other, ActionRequired::Warning,
            "Cannot measure two-edge separation with both the first and second terminal set to the same signal and both the first and second edge set to the same edge. To measure the period of a signal, use a counter input period task. Otherwise, change one of the terminals to a different signal, or change one of the edges to be different from the other."
        }
    },
    {
        -201443,
        {
            Category::Other, ActionRequired::Warning,
            "FREQOUT counter cannot generate the desired frequency. The 4-bit FREQOUT counter can divide the 20 MHz, 10 MHz (20 MHz / 2), or 100 kHz (20 MHz / 200) timebase by a number between 1 and 16. Choose a frequency within this range."
        }
    },
    {
        -201444,
        {
            Category::Hardware, ActionRequired::Warning,
            "Multidevice tasks cannot use the on-demand sample timing type. Configure timing to synchronize and acquire samples from multiple devices."
        }
    },
    {
        -201445,
        {
            Category::Other, ActionRequired::Warning,
            "You must configure strain gage channels for each arm of the rosette. Specify the strain gage channel names."
        }
    },
    {
        -201446,
        {
            Category::Other, ActionRequired::Warning,
            "Each tee rosette requires two physical channels. Make sure the physical channel list contains two physical channels for each tee rosette."
        }
    },
    {
        -201447,
        {
            Category::Other, ActionRequired::Warning,
            "The selected rosette requires three physical channels. Make sure the physical channel list contains three physical channels for each rosette."
        }
    },
    {
        -201448,
        {
            Category::Other, ActionRequired::Warning,
            "No rosette measurements specified. Please specify one or more rosette measurements."
        }
    },
    {
        -201449,
        {
            Category::Communication, ActionRequired::Warning,
            "The requested port connection string is not in a valid format. The valid format is <device name>/port<port number> (e.g. Dev1/port2)."
        }
    },
    {
        -201450,
        {
            Category::Communication, ActionRequired::Warning,
            "The requested device does not support cDAQ Sync connections."
        }
    },
    {
        -201451,
        {
            Category::Other, ActionRequired::Warning,
            "The operation cannot be completed on only one port. Two ports are required."
        }
    },
    {
        -201452,
        {Category::Other, ActionRequired::Warning, "The operation has been aborted."}
    },
    {
        -201453,
        {
            Category::Communication, ActionRequired::Warning,
            "Cannot find disconnected connections between devices in different states (present and simulated). Please check requested set."
        }
    },
    {
        -201454,
        {
            Category::Other, ActionRequired::Warning,
            "Carrier physically unable to contain the declared cards."
        }
    },
    {
        -201455,
        {
            Category::Communication, ActionRequired::Warning,
            "cDAQ Sync connections are not allowed between physical and NI-DAQmx simulated devices. Use cDAQ Sync for either only physical devices or only NI-DAQmx simulated devices."
        }
    },
    {
        -201456,
        {
            Category::Communication, ActionRequired::Warning,
            "The devices attempting to be configured for cDAQ Sync do not have a common sync connection strategy."
        }
    },
    {
        -201457,
        {
            Category::Communication, ActionRequired::Warning,
            "cDAQ Sync cannot add a connection from a port to the same port. Add a connection to a different port instead."
        }
    },
    {
        -201458,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Specified timeout value is not supported. Set the timeout to to a value > 0 or -1 (wait infinitely)."
        }
    },
    {
        -201459,
        {
            Category::Communication, ActionRequired::Warning,
            "Specified devices do not support cDAQ Sync connections. Please select a different set of devices."
        }
    },
    {
        -201460,
        {
            Category::Communication, ActionRequired::Warning,
            "No devices scanned support cDAQ Sync connections. Requesting an empty string only scans physical devices (present and offline) and ignores NI-DAQmx simulated devices. Please check your hardware configuration."
        }
    },
    {
        -201461,
        {
            Category::Hardware, ActionRequired::Warning,
            "Reference clock is not present. This task requires a reference clock to be present before configuring hardware."
        }
    },
    {
        -201462,
        {
            Category::Communication, ActionRequired::Warning,
            "The attempted connection is not between an output port and an input port. Port 0 is input only. All other ports are output only. Please check your connection and try again."
        }
    },
    {
        -201463,
        {
            Category::Other, ActionRequired::Warning,
            "The requested ports are not reciprocal. Make sure that the ports point at each other."
        }
    },
    {
        -201464,
        {
            Category::Hardware, ActionRequired::Warning,
            "A hardware fault has occurred. Please contact National Instruments technical support. To clear the fault, power cycle the device."
        }
    },
    {
        -201465,
        {
            Category::Hardware, ActionRequired::Warning,
            "Cannot perform auto-configure on offline devices. Please remove offline devices from the requested set."
        }
    },
    {
        -201466,
        {
            Category::Communication, ActionRequired::Warning,
            "Cannot auto-configure connections between devices in different states (present, offline, and simulated). Please check requested set."
        }
    },
    {
        -201467,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "Associated channels have conflicting properties. Make the conflicting property values consistent across channels to fix the error."
        }
    },
    {
        -201468,
        {
            Category::Hardware, ActionRequired::Warning,
            "The requested cDAQ Sync device cannot be configured because the master timebase is not present. Tasks containing cDAQ Sync devices that export or import a timebase must be committed in cascading order from the source to the destination. Use DAQmx Control Task to commit the master timebase source task prior to committing or starting a slave task."
        }
    },
    {
        -201469,
        {
            Category::Other, ActionRequired::Warning,
            "You selected a different digitizer and communicator for an SCXI module that is in multiplexed mode. Configure the SCXI module with the digitizer set to the same mode as the chassis communicator in MAX."
        }
    },
    {
        -201470,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid calibration adjustment point(s) provided. Use DAQmx Get Calibration Adjustment Points to retrieve valid adjustment values."
        }
    },
    {
        -201471,
        {
            Category::Hardware, ActionRequired::Warning,
            "The physical channel string contains multiple devices. You can include multiple physical channels on a single device (for example, \"Dev1/ao0:3\") but not multiple physical channels on multiple devices (for example, \"Dev1/ao0:3, Dev2/ao2\")."
        }
    },
    {
        -201472,
        {
            Category::Other, ActionRequired::Warning,
            "You must specify exactly one timestamp channel for a Navigation With Timestamp read call."
        }
    },
    {
        -201474,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "The device is not supported on the local system, is incompatible, or the installation is corrupt. Install and/or repair the appropriate driver."
        }
    },
    {
        -201475,
        {
            Category::Hardware, ActionRequired::Warning,
            "Auxiliary power not detected. Verify that the auxiliary power source is properly connected to the device and that the auxiliary input fuse has not blown. Refer to the NI-DCPower documentation for information about replacing the input fuse."
        }
    },
    {
        -201476,
        {
            Category::Other, ActionRequired::Warning,
            "Task reservation failed because a watchdog timer task is reserved. Unreserve or commit the watchdog timer task to reserve a new task."
        }
    },
    {
        -201477,
        {
            Category::Other, ActionRequired::Warning,
            "Reservation of watchdog timer task failed because another task is reserved or running. Stop and unreserve all other tasks before reserving a watchdog timer task."
        }
    },
    {
        -201478,
        {
            Category::Other, ActionRequired::Warning,
            "Watchdog timer task has expired. Reset the chassis to resume normal operation."
        }
    },
    {
        -201479,
        {
            Category::Configuration, ActionRequired::Warning,
            "The attribute/property ArmStart.Term cannot be queried when a software arm start trigger is configured because it cannot be exported to any terminal. Configure an external arm start trigger and query the desired attribute/property."
        }
    },
    {
        -201480,
        {
            Category::Hardware, ActionRequired::Warning,
            "There is already a session open to the device from another process, or a calibration session is open. You must close the open session, exit the application holding the device, or release the device in the Soft Front Panel."
        }
    },
    {
        -201481,
        {
            Category::Other, ActionRequired::Warning,
            "If you specify an expiration state for any line on an NI 9401 in the range of line 0:3 or line 4:7, you must specify an expiration state for every line in that range."
        }
    },
    {
        -201482,
        {
            Category::Other, ActionRequired::Warning,
            "The current number of channels chosen for calibration is incorrect. You must specify all channels for calibration."
        }
    },
    {
        -201483,
        {
            Category::Hardware, ActionRequired::Warning,
            "The selected master for the configured multi-device task is not able to export signals. Make sure that the first channel in the task is from a device on a chassis that has an NI 9469 capable of exporting signals to the slave devices. If you have any delta-sigma modules in your task, at least one must be in the master chassis. For time-based synchronization, ensure all the chassis in the task are in the same synchronization network."
        }
    },
    {
        -201484,
        {
            Category::Hardware, ActionRequired::Warning,
            "The current task contains channels from both delta-sigma and non delta-sigma devices. Make sure the first channel in the task is from a delta-sigma device and that it is located in a chassis that is able to share signals with its slave chassis through an NI 9469, or the task is running on devices that support time-based network synchronization."
        }
    },
    {
        -201485,
        {
            Category::Other, ActionRequired::Warning,
            "Measured data size does not match reference data size. The module acquires data and receives reference data in different functions. Please make sure these calibration functions are executed the same number of times during calibration."
        }
    },
    {
        -201486,
        {
            Category::Other, ActionRequired::Warning,
            "You must call DAQmx Setup Calibration first in order to calibrate this module."
        }
    },
    {
        -201487,
        {
            Category::Hardware, ActionRequired::Warning,
            "You must disable automatic firmware updates before attempting to manually update the firmware of your device."
        }
    },
    {
        -201488,
        {
            Category::Hardware, ActionRequired::Warning,
            "Firmware update failed. System attempted to update a different device than what you specified. Try updating again."
        }
    },
    {
        -201489,
        {
            Category::Other, ActionRequired::ImmediateAction,
            "Invalid state for firmware update. You must call the firmware update action after calling beginFirmwareAction and before calling disposeFirmwareAction."
        }
    },
    {
        -201490,
        {
            Category::Other, ActionRequired::Warning,
            "Firmware version being installed is older than the currently installed firmware."
        }
    },
    {
        -201491,
        {
            Category::Other, ActionRequired::Warning,
            "Firmware version being installed matches the currently installed version."
        }
    },
    {
        -201492,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Attempted operation is not supported on simulated devices."
        }
    },
    {
        -201493,
        {
            Category::Hardware, ActionRequired::Warning,
            "Shunt calibration failed. The calculated gain adjust is out of range. Ensure that the shunt calibration terminals are connected properly and that the shunt resistance and shunt element location settings match how the hardware is wired."
        }
    },
    {
        -201494,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device does not support simultaneous calibration of multiple channels. Calibrate channels one channel at a time passing individual channels to different invocations of DAQmx Adjust Calibration."
        }
    },
    {
        -201495,
        {
            Category::Hardware, ActionRequired::Warning,
            "Device calibration requires all ranges to be calibrated for a single channel. Calibrate the ranges specified in the procedure."
        }
    },
    {
        -201496,
        {
            Category::Other, ActionRequired::Warning,
            "Voltage settings not calibrated. Ensure all voltage settings are calibrated before trying to calibrate bridge settings."
        }
    },
    {
        -201497,
        {
            Category::Other, ActionRequired::Warning,
            "You must run setupCalibration before running getCalDataPoints."
        }
    },
    {
        -201498,
        {
            Category::Hardware, ActionRequired::Warning,
            "AI channels on this device do not support using AC coupling while not using IEPE excitation. Enable IEPE excitation or set the coupling mode to DC."
        }
    },
    {
        -201499,
        {
            Category::Other, ActionRequired::Warning,
            "The selected calibration mode cannot query calibration data points. Make sure you follow the calibration procedure."
        }
    },
    {
        -201500,
        {
            Category::Other, ActionRequired::Warning,
            "The channel in calibration adjustment did not call calibration setup. Make sure you call calibration setup before calibration adjustment for this channel."
        }
    },
    {
        -201501,
        {
            Category::Other, ActionRequired::Warning,
            "The reference value input of calibration adjustment is out of range."
        }
    },
    {
        -201502,
        {
            Category::Configuration, ActionRequired::Warning,
            "DAC Offset Voltage Value is not set. When the DAC Offset Voltage Source property for a channel is set to External, the DAC Offset Voltage Value property must be set. Set the DAC Offset Voltage Value property so the value matches the offset voltage source connected to your device. Alternatively, use the internal DAC offset voltage source available on the device."
        }
    },
    {
        -209800,
        {
            Category::Hardware, ActionRequired::Warning,
            "DAQmx Read did not complete before the arrival of the next sample clock or change detection event which indicates that your program is not keeping up with the hardware clock or the external change event. For tasks using sample clock timing, slow down the hardware clock or else change your application so that it can keep up with the hardware clock. For tasks using change detection timing, decrease the frequency of your event or else change your application so that it can keep up with the change event."
        }
    },
    {
        200079,
        {
            Category::Other, ActionRequired::Warning,
            "When specifying \"No Change\" as a watchdog state on a DSA channel, the actual behavior is determined by the idle output behavior configured by the last task run on the module."
        }
    },
    {
        -209801,
        {
            Category::Hardware, ActionRequired::Warning,
            "DAQmx Write did not complete before the arrival of the next sample clock which indicates that your program is not keeping up with the hardware clock. Slow down the hardware clock or else change your application so that it can keep up with the hardware clock."
        }
    },
    {
        -209802,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "DAQmx Wait for Next Sample Clock detected one or more missed sample clocks since the last call to Wait for Next Sample Clock which indicates that your program is not keeping up with the sample clock. To remove this error, slow down the sample clock, or else change your application so that it can keep up with the sample clock. Alternatively, consider setting the Convert Errors to Warnings property to true and then handling the warning case appropriately."
        }
    },
    {
        -209803,
        {
            Category::Configuration, ActionRequired::ImmediateAction,
            "DAQmx Wait for Next Sample Clock detected 3 or more missed sample clocks since the last call to Wait for Next Sample Clock which indicates your program is not keeping up with the sample clock, and data was subsequently lost before it could be read by the application. To remove this error, slow down the sample clock. Consider restructuring the application so you can call DAQmx Read more often. Setting the Convert Error to Warning property to True will not eliminate the error because the data is lost."
        }
    },
    {
        -209805,
        {
            Category::Other, ActionRequired::Warning,
            "Task with counter output detected a sample clock before the corresponding DAQmx Write was completed. This may have occurred because the frequency of the counter output is too low for the given sample clock rate. A full output period must complete before new data can be written. To avoid this problem make sure that the counter output frequency is significantly higher than the sample clock rate."
        }
    },
    {
        -200831,
        {
            Category::Other, ActionRequired::Warning,
            "On Demand Simultaneous Analog Output Enable cannot be set to true unless Sample Timing Type is On Demand."
        }
    },
    {
        -201045,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Device connector number specified is invalid. Enter a valid connector number."
        }
    },
    {
        -201116,
        {
            Category::Hardware, ActionRequired::ImmediateAction,
            "Hardware timed non-buffered analog output is not supported on this device."
        }
    },

};
}
