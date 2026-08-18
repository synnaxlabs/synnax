// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Code generated from api.h by generate_mock.py. Regenerate when api.h changes.

#pragma once

#include <string>
#include <vector>

#include "nlohmann/json.hpp"

#include "driver/ni/daqmx/api.h"

namespace driver::ni::daqmx {
/// @brief a mock DAQmx API that records every call and its scalar/string
/// arguments as JSON for assertion in tests. All calls succeed.
class MockAPI final : public API {
public:
    /// @brief one entry per API call, in call order: {"fn": name, "args": {...}}.
    std::vector<nlohmann::json> calls;

    /// @brief returns the recorded calls to the given function, in order.
    [[nodiscard]] std::vector<nlohmann::json> calls_to(const std::string &fn) const {
        std::vector<nlohmann::json> out;
        for (const auto &c: calls)
            if (c["fn"] == fn) out.push_back(c);
        return out;
    }

    int32 AddCDAQSyncConnection(const char portList[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (portList != nullptr) args["portList"] = std::string(portList);
        this->calls.push_back({{"fn", "AddCDAQSyncConnection"}, {"args", args}});
        return 0;
    }

    int32 AddGlobalChansToTask(TaskHandle task, const char channelNames[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (channelNames != nullptr) args["channelNames"] = std::string(channelNames);
        this->calls.push_back({{"fn", "AddGlobalChansToTask"}, {"args", args}});
        return 0;
    }

    int32 AddNetworkDevice(
        const char ipAddress[],
        const char deviceName[],
        bool32 attemptReservation,
        float64 timeout,
        char deviceNameOut[],
        uInt32 deviceNameOutBufferSize
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (ipAddress != nullptr) args["ipAddress"] = std::string(ipAddress);
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attemptReservation"] = attemptReservation;
        args["timeout"] = timeout;
        if (deviceNameOut != nullptr)
            args["deviceNameOut"] = std::string(deviceNameOut);
        args["deviceNameOutBufferSize"] = deviceNameOutBufferSize;
        this->calls.push_back({{"fn", "AddNetworkDevice"}, {"args", args}});
        return 0;
    }

    int32 AreConfiguredCDAQSyncPortsDisconnected(
        const char chassisDevicesPorts[],
        float64 timeout,
        bool32 *disconnectedPortsExist
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (chassisDevicesPorts != nullptr)
            args["chassisDevicesPorts"] = std::string(chassisDevicesPorts);
        args["timeout"] = timeout;
        args["disconnectedPortsExist"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "AreConfiguredCDAQSyncPortsDisconnected"}, {"args", args}}
        );
        return 0;
    }

    int32 AutoConfigureCDAQSyncConnections(
        const char chassisDevicesPorts[],
        float64 timeout
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (chassisDevicesPorts != nullptr)
            args["chassisDevicesPorts"] = std::string(chassisDevicesPorts);
        args["timeout"] = timeout;
        this->calls.push_back(
            {{"fn", "AutoConfigureCDAQSyncConnections"}, {"args", args}}
        );
        return 0;
    }

    int32 CalculateReversePolyCoeff(
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffsIn,
        float64 minValX,
        float64 maxValX,
        int32 numPointsToCompute,
        int32 reversePolyOrder,
        float64 reverseCoeffs[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (forwardCoeffs != nullptr) args["forwardCoeffs"] = "<array>";
        args["numForwardCoeffsIn"] = numForwardCoeffsIn;
        args["minValX"] = minValX;
        args["maxValX"] = maxValX;
        args["numPointsToCompute"] = numPointsToCompute;
        args["reversePolyOrder"] = reversePolyOrder;
        if (reverseCoeffs != nullptr) args["reverseCoeffs"] = "<array>";
        this->calls.push_back({{"fn", "CalculateReversePolyCoeff"}, {"args", args}});
        return 0;
    }

    int32 CfgAnlgEdgeRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerSlope,
        float64 triggerLevel,
        uInt32 pretriggerSamples
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (triggerSource != nullptr)
            args["triggerSource"] = std::string(triggerSource);
        args["triggerSlope"] = triggerSlope;
        args["triggerLevel"] = triggerLevel;
        args["pretriggerSamples"] = pretriggerSamples;
        this->calls.push_back({{"fn", "CfgAnlgEdgeRefTrig"}, {"args", args}});
        return 0;
    }

    int32 CfgAnlgEdgeStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerSlope,
        float64 triggerLevel
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (triggerSource != nullptr)
            args["triggerSource"] = std::string(triggerSource);
        args["triggerSlope"] = triggerSlope;
        args["triggerLevel"] = triggerLevel;
        this->calls.push_back({{"fn", "CfgAnlgEdgeStartTrig"}, {"args", args}});
        return 0;
    }

    int32 CfgAnlgMultiEdgeRefTrig(
        TaskHandle task,
        const char triggerSources[],
        const int32 triggerSlopeArray[],
        const float64 triggerLevelArray[],
        uInt32 pretriggerSamples,
        uInt32 arraySize
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (triggerSources != nullptr)
            args["triggerSources"] = std::string(triggerSources);
        if (triggerSlopeArray != nullptr) args["triggerSlopeArray"] = "<array>";
        if (triggerLevelArray != nullptr) args["triggerLevelArray"] = "<array>";
        args["pretriggerSamples"] = pretriggerSamples;
        args["arraySize"] = arraySize;
        this->calls.push_back({{"fn", "CfgAnlgMultiEdgeRefTrig"}, {"args", args}});
        return 0;
    }

    int32 CfgAnlgMultiEdgeStartTrig(
        TaskHandle task,
        const char triggerSources[],
        const int32 triggerSlopeArray[],
        const float64 triggerLevelArray[],
        uInt32 arraySize
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (triggerSources != nullptr)
            args["triggerSources"] = std::string(triggerSources);
        if (triggerSlopeArray != nullptr) args["triggerSlopeArray"] = "<array>";
        if (triggerLevelArray != nullptr) args["triggerLevelArray"] = "<array>";
        args["arraySize"] = arraySize;
        this->calls.push_back({{"fn", "CfgAnlgMultiEdgeStartTrig"}, {"args", args}});
        return 0;
    }

    int32 CfgAnlgWindowRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerWhen,
        float64 windowTop,
        float64 windowBottom,
        uInt32 pretriggerSamples
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (triggerSource != nullptr)
            args["triggerSource"] = std::string(triggerSource);
        args["triggerWhen"] = triggerWhen;
        args["windowTop"] = windowTop;
        args["windowBottom"] = windowBottom;
        args["pretriggerSamples"] = pretriggerSamples;
        this->calls.push_back({{"fn", "CfgAnlgWindowRefTrig"}, {"args", args}});
        return 0;
    }

    int32 CfgAnlgWindowStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerWhen,
        float64 windowTop,
        float64 windowBottom
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (triggerSource != nullptr)
            args["triggerSource"] = std::string(triggerSource);
        args["triggerWhen"] = triggerWhen;
        args["windowTop"] = windowTop;
        args["windowBottom"] = windowBottom;
        this->calls.push_back({{"fn", "CfgAnlgWindowStartTrig"}, {"args", args}});
        return 0;
    }

    int32 CfgBurstHandshakingTimingExportClock(
        TaskHandle task,
        int32 sampleMode,
        uInt64 sampsPerChan,
        float64 sampleClkRate,
        const char sampleClkOutpTerm[],
        int32 sampleClkPulsePolarity,
        int32 pauseWhen,
        int32 readyEventActiveLevel
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["sampleMode"] = sampleMode;
        args["sampsPerChan"] = sampsPerChan;
        args["sampleClkRate"] = sampleClkRate;
        if (sampleClkOutpTerm != nullptr)
            args["sampleClkOutpTerm"] = std::string(sampleClkOutpTerm);
        args["sampleClkPulsePolarity"] = sampleClkPulsePolarity;
        args["pauseWhen"] = pauseWhen;
        args["readyEventActiveLevel"] = readyEventActiveLevel;
        this->calls.push_back(
            {{"fn", "CfgBurstHandshakingTimingExportClock"}, {"args", args}}
        );
        return 0;
    }

    int32 CfgBurstHandshakingTimingImportClock(
        TaskHandle task,
        int32 sampleMode,
        uInt64 sampsPerChan,
        float64 sampleClkRate,
        const char sampleClkSrc[],
        int32 sampleClkActiveEdge,
        int32 pauseWhen,
        int32 readyEventActiveLevel
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["sampleMode"] = sampleMode;
        args["sampsPerChan"] = sampsPerChan;
        args["sampleClkRate"] = sampleClkRate;
        if (sampleClkSrc != nullptr) args["sampleClkSrc"] = std::string(sampleClkSrc);
        args["sampleClkActiveEdge"] = sampleClkActiveEdge;
        args["pauseWhen"] = pauseWhen;
        args["readyEventActiveLevel"] = readyEventActiveLevel;
        this->calls.push_back(
            {{"fn", "CfgBurstHandshakingTimingImportClock"}, {"args", args}}
        );
        return 0;
    }

    int32 CfgChangeDetectionTiming(
        TaskHandle task,
        const char risingEdgeChan[],
        const char fallingEdgeChan[],
        int32 sampleMode,
        uInt64 sampsPerChan
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (risingEdgeChan != nullptr)
            args["risingEdgeChan"] = std::string(risingEdgeChan);
        if (fallingEdgeChan != nullptr)
            args["fallingEdgeChan"] = std::string(fallingEdgeChan);
        args["sampleMode"] = sampleMode;
        args["sampsPerChan"] = sampsPerChan;
        this->calls.push_back({{"fn", "CfgChangeDetectionTiming"}, {"args", args}});
        return 0;
    }

    int32 CfgDigEdgeRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerEdge,
        uInt32 pretriggerSamples
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (triggerSource != nullptr)
            args["triggerSource"] = std::string(triggerSource);
        args["triggerEdge"] = triggerEdge;
        args["pretriggerSamples"] = pretriggerSamples;
        this->calls.push_back({{"fn", "CfgDigEdgeRefTrig"}, {"args", args}});
        return 0;
    }

    int32 CfgDigEdgeStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerEdge
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (triggerSource != nullptr)
            args["triggerSource"] = std::string(triggerSource);
        args["triggerEdge"] = triggerEdge;
        this->calls.push_back({{"fn", "CfgDigEdgeStartTrig"}, {"args", args}});
        return 0;
    }

    int32 CfgDigPatternRefTrig(
        TaskHandle task,
        const char triggerSource[],
        const char triggerPattern[],
        int32 triggerWhen,
        uInt32 pretriggerSamples
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (triggerSource != nullptr)
            args["triggerSource"] = std::string(triggerSource);
        if (triggerPattern != nullptr)
            args["triggerPattern"] = std::string(triggerPattern);
        args["triggerWhen"] = triggerWhen;
        args["pretriggerSamples"] = pretriggerSamples;
        this->calls.push_back({{"fn", "CfgDigPatternRefTrig"}, {"args", args}});
        return 0;
    }

    int32 CfgDigPatternStartTrig(
        TaskHandle task,
        const char triggerSource[],
        const char triggerPattern[],
        int32 triggerWhen
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (triggerSource != nullptr)
            args["triggerSource"] = std::string(triggerSource);
        if (triggerPattern != nullptr)
            args["triggerPattern"] = std::string(triggerPattern);
        args["triggerWhen"] = triggerWhen;
        this->calls.push_back({{"fn", "CfgDigPatternStartTrig"}, {"args", args}});
        return 0;
    }

    int32 CfgHandshakingTiming(
        TaskHandle task,
        int32 sampleMode,
        uInt64 sampsPerChan
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["sampleMode"] = sampleMode;
        args["sampsPerChan"] = sampsPerChan;
        this->calls.push_back({{"fn", "CfgHandshakingTiming"}, {"args", args}});
        return 0;
    }

    int32
    CfgImplicitTiming(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan) override {
        nlohmann::json args = nlohmann::json::object();
        args["sampleMode"] = sampleMode;
        args["sampsPerChan"] = sampsPerChan;
        this->calls.push_back({{"fn", "CfgImplicitTiming"}, {"args", args}});
        return 0;
    }

    int32 CfgInputBuffer(TaskHandle task, uInt32 numSampsPerChan) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        this->calls.push_back({{"fn", "CfgInputBuffer"}, {"args", args}});
        return 0;
    }

    int32 CfgOutputBuffer(TaskHandle task, uInt32 numSampsPerChan) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        this->calls.push_back({{"fn", "CfgOutputBuffer"}, {"args", args}});
        return 0;
    }

    int32 CfgPipelinedSampClkTiming(
        TaskHandle task,
        const char source[],
        float64 rate,
        int32 activeEdge,
        int32 sampleMode,
        uInt64 sampsPerChan
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (source != nullptr) args["source"] = std::string(source);
        args["rate"] = rate;
        args["activeEdge"] = activeEdge;
        args["sampleMode"] = sampleMode;
        args["sampsPerChan"] = sampsPerChan;
        this->calls.push_back({{"fn", "CfgPipelinedSampClkTiming"}, {"args", args}});
        return 0;
    }

    int32 CfgSampClkTiming(
        TaskHandle task,
        const char source[],
        float64 rate,
        int32 activeEdge,
        int32 sampleMode,
        uInt64 sampsPerChan
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (source != nullptr) args["source"] = std::string(source);
        args["rate"] = rate;
        args["activeEdge"] = activeEdge;
        args["sampleMode"] = sampleMode;
        args["sampsPerChan"] = sampsPerChan;
        this->calls.push_back({{"fn", "CfgSampClkTiming"}, {"args", args}});
        return 0;
    }

    int32
    CfgTimeStartTrig(TaskHandle task, CVIAbsoluteTime when, int32 timescale) override {
        nlohmann::json args = nlohmann::json::object();
        args["when"] = "<ptr>";
        args["timescale"] = timescale;
        this->calls.push_back({{"fn", "CfgTimeStartTrig"}, {"args", args}});
        return 0;
    }

    int32 CfgWatchdogAOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const float64 expirStateArray[],
        const int32 outputTypeArray[],
        uInt32 arraySize
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channelNames != nullptr) args["channelNames"] = std::string(channelNames);
        if (expirStateArray != nullptr) args["expirStateArray"] = "<array>";
        if (outputTypeArray != nullptr) args["outputTypeArray"] = "<array>";
        args["arraySize"] = arraySize;
        this->calls.push_back({{"fn", "CfgWatchdogAOExpirStates"}, {"args", args}});
        return 0;
    }

    int32 CfgWatchdogCOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const int32 expirStateArray[],
        uInt32 arraySize
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channelNames != nullptr) args["channelNames"] = std::string(channelNames);
        if (expirStateArray != nullptr) args["expirStateArray"] = "<array>";
        args["arraySize"] = arraySize;
        this->calls.push_back({{"fn", "CfgWatchdogCOExpirStates"}, {"args", args}});
        return 0;
    }

    int32 CfgWatchdogDOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const int32 expirStateArray[],
        uInt32 arraySize
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channelNames != nullptr) args["channelNames"] = std::string(channelNames);
        if (expirStateArray != nullptr) args["expirStateArray"] = "<array>";
        args["arraySize"] = arraySize;
        this->calls.push_back({{"fn", "CfgWatchdogDOExpirStates"}, {"args", args}});
        return 0;
    }

    int32 ClearTEDS(const char physicalChannel[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        this->calls.push_back({{"fn", "ClearTEDS"}, {"args", args}});
        return 0;
    }

    int32 ClearTask(TaskHandle task) override {
        nlohmann::json args = nlohmann::json::object();

        this->calls.push_back({{"fn", "ClearTask"}, {"args", args}});
        return 0;
    }

    int32 ConfigureLogging(
        TaskHandle task,
        const char filePath[],
        int32 loggingMode,
        const char groupName[],
        int32 operation
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (filePath != nullptr) args["filePath"] = std::string(filePath);
        args["loggingMode"] = loggingMode;
        if (groupName != nullptr) args["groupName"] = std::string(groupName);
        args["operation"] = operation;
        this->calls.push_back({{"fn", "ConfigureLogging"}, {"args", args}});
        return 0;
    }

    int32 ConfigureTEDS(const char physicalChannel[], const char filePath[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (filePath != nullptr) args["filePath"] = std::string(filePath);
        this->calls.push_back({{"fn", "ConfigureTEDS"}, {"args", args}});
        return 0;
    }

    int32 ConnectTerms(
        const char sourceTerminal[],
        const char destinationTerminal[],
        int32 signalModifiers
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (sourceTerminal != nullptr)
            args["sourceTerminal"] = std::string(sourceTerminal);
        if (destinationTerminal != nullptr)
            args["destinationTerminal"] = std::string(destinationTerminal);
        args["signalModifiers"] = signalModifiers;
        this->calls.push_back({{"fn", "ConnectTerms"}, {"args", args}});
        return 0;
    }

    int32 ControlWatchdogTask(TaskHandle task, int32 action) override {
        nlohmann::json args = nlohmann::json::object();
        args["action"] = action;
        this->calls.push_back({{"fn", "ControlWatchdogTask"}, {"args", args}});
        return 0;
    }

    int32 CreateAIAccel4WireDCVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 sensitivity,
        int32 sensitivityUnits,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        bool32 useExcitForScaling,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["sensitivity"] = sensitivity;
        args["sensitivityUnits"] = sensitivityUnits;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["useExcitForScaling"] = useExcitForScaling;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back(
            {{"fn", "CreateAIAccel4WireDCVoltageChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateAIAccelChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 sensitivity,
        int32 sensitivityUnits,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["sensitivity"] = sensitivity;
        args["sensitivityUnits"] = sensitivityUnits;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIAccelChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIAccelChargeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 sensitivity,
        int32 sensitivityUnits,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["sensitivity"] = sensitivity;
        args["sensitivityUnits"] = sensitivityUnits;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIAccelChargeChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 bridgeConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 nominalBridgeResistance,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["bridgeConfig"] = bridgeConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["nominalBridgeResistance"] = nominalBridgeResistance;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIBridgeChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIChargeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIChargeChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAICurrentChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 shuntResistorLoc,
        float64 extShuntResistorVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["shuntResistorLoc"] = shuntResistorLoc;
        args["extShuntResistorVal"] = extShuntResistorVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAICurrentChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAICurrentRMSChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 shuntResistorLoc,
        float64 extShuntResistorVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["shuntResistorLoc"] = shuntResistorLoc;
        args["extShuntResistorVal"] = extShuntResistorVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAICurrentRMSChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIForceBridgePolynomialChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 bridgeConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 nominalBridgeResistance,
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffs,
        const float64 reverseCoeffs[],
        uInt32 numReverseCoeffs,
        int32 electricalUnits,
        int32 physicalUnits,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["bridgeConfig"] = bridgeConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["nominalBridgeResistance"] = nominalBridgeResistance;
        if (forwardCoeffs != nullptr) args["forwardCoeffs"] = "<array>";
        args["numForwardCoeffs"] = numForwardCoeffs;
        if (reverseCoeffs != nullptr) args["reverseCoeffs"] = "<array>";
        args["numReverseCoeffs"] = numReverseCoeffs;
        args["electricalUnits"] = electricalUnits;
        args["physicalUnits"] = physicalUnits;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back(
            {{"fn", "CreateAIForceBridgePolynomialChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateAIForceBridgeTableChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 bridgeConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 nominalBridgeResistance,
        const float64 electricalVals[],
        uInt32 numElectricalVals,
        int32 electricalUnits,
        const float64 physicalVals[],
        uInt32 numPhysicalVals,
        int32 physicalUnits,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["bridgeConfig"] = bridgeConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["nominalBridgeResistance"] = nominalBridgeResistance;
        if (electricalVals != nullptr) args["electricalVals"] = "<array>";
        args["numElectricalVals"] = numElectricalVals;
        args["electricalUnits"] = electricalUnits;
        if (physicalVals != nullptr) args["physicalVals"] = "<array>";
        args["numPhysicalVals"] = numPhysicalVals;
        args["physicalUnits"] = physicalUnits;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIForceBridgeTableChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIForceBridgeTwoPointLinChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 bridgeConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 nominalBridgeResistance,
        float64 firstElectricalVal,
        float64 secondElectricalVal,
        int32 electricalUnits,
        float64 firstPhysicalVal,
        float64 secondPhysicalVal,
        int32 physicalUnits,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["bridgeConfig"] = bridgeConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["nominalBridgeResistance"] = nominalBridgeResistance;
        args["firstElectricalVal"] = firstElectricalVal;
        args["secondElectricalVal"] = secondElectricalVal;
        args["electricalUnits"] = electricalUnits;
        args["firstPhysicalVal"] = firstPhysicalVal;
        args["secondPhysicalVal"] = secondPhysicalVal;
        args["physicalUnits"] = physicalUnits;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back(
            {{"fn", "CreateAIForceBridgeTwoPointLinChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateAIForceIEPEChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 sensitivity,
        int32 sensitivityUnits,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["sensitivity"] = sensitivity;
        args["sensitivityUnits"] = sensitivityUnits;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIForceIEPEChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIFreqVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 thresholdLevel,
        float64 hysteresis,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["thresholdLevel"] = thresholdLevel;
        args["hysteresis"] = hysteresis;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIFreqVoltageChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIMicrophoneChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        int32 units,
        float64 micSensitivity,
        float64 maxSndPressLevel,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["units"] = units;
        args["micSensitivity"] = micSensitivity;
        args["maxSndPressLevel"] = maxSndPressLevel;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIMicrophoneChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIPosEddyCurrProxProbeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 sensitivity,
        int32 sensitivityUnits,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["sensitivity"] = sensitivity;
        args["sensitivityUnits"] = sensitivityUnits;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back(
            {{"fn", "CreateAIPosEddyCurrProxProbeChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateAIPosLVDTChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 sensitivity,
        int32 sensitivityUnits,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 voltageExcitFreq,
        int32 acExcitWireMode,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["sensitivity"] = sensitivity;
        args["sensitivityUnits"] = sensitivityUnits;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["voltageExcitFreq"] = voltageExcitFreq;
        args["acExcitWireMode"] = acExcitWireMode;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIPosLVDTChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIPosRVDTChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 sensitivity,
        int32 sensitivityUnits,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 voltageExcitFreq,
        int32 acExcitWireMode,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["sensitivity"] = sensitivity;
        args["sensitivityUnits"] = sensitivityUnits;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["voltageExcitFreq"] = voltageExcitFreq;
        args["acExcitWireMode"] = acExcitWireMode;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIPosRVDTChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIPowerChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 voltageSetpoint,
        float64 currentSetpoint,
        bool32 outputEnable
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["voltageSetpoint"] = voltageSetpoint;
        args["currentSetpoint"] = currentSetpoint;
        args["outputEnable"] = outputEnable;
        this->calls.push_back({{"fn", "CreateAIPowerChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIPressureBridgePolynomialChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 bridgeConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 nominalBridgeResistance,
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffs,
        const float64 reverseCoeffs[],
        uInt32 numReverseCoeffs,
        int32 electricalUnits,
        int32 physicalUnits,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["bridgeConfig"] = bridgeConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["nominalBridgeResistance"] = nominalBridgeResistance;
        if (forwardCoeffs != nullptr) args["forwardCoeffs"] = "<array>";
        args["numForwardCoeffs"] = numForwardCoeffs;
        if (reverseCoeffs != nullptr) args["reverseCoeffs"] = "<array>";
        args["numReverseCoeffs"] = numReverseCoeffs;
        args["electricalUnits"] = electricalUnits;
        args["physicalUnits"] = physicalUnits;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back(
            {{"fn", "CreateAIPressureBridgePolynomialChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateAIPressureBridgeTableChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 bridgeConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 nominalBridgeResistance,
        const float64 electricalVals[],
        uInt32 numElectricalVals,
        int32 electricalUnits,
        const float64 physicalVals[],
        uInt32 numPhysicalVals,
        int32 physicalUnits,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["bridgeConfig"] = bridgeConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["nominalBridgeResistance"] = nominalBridgeResistance;
        if (electricalVals != nullptr) args["electricalVals"] = "<array>";
        args["numElectricalVals"] = numElectricalVals;
        args["electricalUnits"] = electricalUnits;
        if (physicalVals != nullptr) args["physicalVals"] = "<array>";
        args["numPhysicalVals"] = numPhysicalVals;
        args["physicalUnits"] = physicalUnits;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back(
            {{"fn", "CreateAIPressureBridgeTableChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateAIPressureBridgeTwoPointLinChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 bridgeConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 nominalBridgeResistance,
        float64 firstElectricalVal,
        float64 secondElectricalVal,
        int32 electricalUnits,
        float64 firstPhysicalVal,
        float64 secondPhysicalVal,
        int32 physicalUnits,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["bridgeConfig"] = bridgeConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["nominalBridgeResistance"] = nominalBridgeResistance;
        args["firstElectricalVal"] = firstElectricalVal;
        args["secondElectricalVal"] = secondElectricalVal;
        args["electricalUnits"] = electricalUnits;
        args["firstPhysicalVal"] = firstPhysicalVal;
        args["secondPhysicalVal"] = secondPhysicalVal;
        args["physicalUnits"] = physicalUnits;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back(
            {{"fn", "CreateAIPressureBridgeTwoPointLinChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateAIRTDChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 rtdType,
        int32 resistanceConfig,
        int32 currentExcitSource,
        float64 currentExcitVal,
        float64 r0
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["rtdType"] = rtdType;
        args["resistanceConfig"] = resistanceConfig;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        args["r0"] = r0;
        this->calls.push_back({{"fn", "CreateAIRTDChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIResistanceChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["resistanceConfig"] = resistanceConfig;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIResistanceChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIRosetteStrainGageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 rosetteType,
        float64 gageOrientation,
        const int32 rosetteMeasTypes[],
        uInt32 numRosetteMeasTypes,
        int32 strainConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 gageFactor,
        float64 nominalGageResistance,
        float64 poissonRatio,
        float64 leadWireResistance
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["rosetteType"] = rosetteType;
        args["gageOrientation"] = gageOrientation;
        if (rosetteMeasTypes != nullptr) args["rosetteMeasTypes"] = "<array>";
        args["numRosetteMeasTypes"] = numRosetteMeasTypes;
        args["strainConfig"] = strainConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["gageFactor"] = gageFactor;
        args["nominalGageResistance"] = nominalGageResistance;
        args["poissonRatio"] = poissonRatio;
        args["leadWireResistance"] = leadWireResistance;
        this->calls.push_back(
            {{"fn", "CreateAIRosetteStrainGageChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateAIStrainGageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 strainConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 gageFactor,
        float64 initialBridgeVoltage,
        float64 nominalGageResistance,
        float64 poissonRatio,
        float64 leadWireResistance,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["strainConfig"] = strainConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["gageFactor"] = gageFactor;
        args["initialBridgeVoltage"] = initialBridgeVoltage;
        args["nominalGageResistance"] = nominalGageResistance;
        args["poissonRatio"] = poissonRatio;
        args["leadWireResistance"] = leadWireResistance;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIStrainGageChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAITempBuiltInSensorChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 units
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["units"] = units;
        this->calls.push_back(
            {{"fn", "CreateAITempBuiltInSensorChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateAIThrmcplChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 thermocoupleType,
        int32 cjcSource,
        float64 cjcVal,
        const char cjcChannel[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["thermocoupleType"] = thermocoupleType;
        args["cjcSource"] = cjcSource;
        args["cjcVal"] = cjcVal;
        if (cjcChannel != nullptr) args["cjcChannel"] = std::string(cjcChannel);
        this->calls.push_back({{"fn", "CreateAIThrmcplChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIThrmstrChanIex(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 currentExcitSource,
        float64 currentExcitVal,
        float64 a,
        float64 b,
        float64 c
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["resistanceConfig"] = resistanceConfig;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        args["a"] = a;
        args["b"] = b;
        args["c"] = c;
        this->calls.push_back({{"fn", "CreateAIThrmstrChanIex"}, {"args", args}});
        return 0;
    }

    int32 CreateAIThrmstrChanVex(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 a,
        float64 b,
        float64 c,
        float64 r1
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["resistanceConfig"] = resistanceConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["a"] = a;
        args["b"] = b;
        args["c"] = c;
        args["r1"] = r1;
        this->calls.push_back({{"fn", "CreateAIThrmstrChanVex"}, {"args", args}});
        return 0;
    }

    int32 CreateAITorqueBridgePolynomialChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 bridgeConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 nominalBridgeResistance,
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffs,
        const float64 reverseCoeffs[],
        uInt32 numReverseCoeffs,
        int32 electricalUnits,
        int32 physicalUnits,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["bridgeConfig"] = bridgeConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["nominalBridgeResistance"] = nominalBridgeResistance;
        if (forwardCoeffs != nullptr) args["forwardCoeffs"] = "<array>";
        args["numForwardCoeffs"] = numForwardCoeffs;
        if (reverseCoeffs != nullptr) args["reverseCoeffs"] = "<array>";
        args["numReverseCoeffs"] = numReverseCoeffs;
        args["electricalUnits"] = electricalUnits;
        args["physicalUnits"] = physicalUnits;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back(
            {{"fn", "CreateAITorqueBridgePolynomialChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateAITorqueBridgeTableChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 bridgeConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 nominalBridgeResistance,
        const float64 electricalVals[],
        uInt32 numElectricalVals,
        int32 electricalUnits,
        const float64 physicalVals[],
        uInt32 numPhysicalVals,
        int32 physicalUnits,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["bridgeConfig"] = bridgeConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["nominalBridgeResistance"] = nominalBridgeResistance;
        if (electricalVals != nullptr) args["electricalVals"] = "<array>";
        args["numElectricalVals"] = numElectricalVals;
        args["electricalUnits"] = electricalUnits;
        if (physicalVals != nullptr) args["physicalVals"] = "<array>";
        args["numPhysicalVals"] = numPhysicalVals;
        args["physicalUnits"] = physicalUnits;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back(
            {{"fn", "CreateAITorqueBridgeTableChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateAITorqueBridgeTwoPointLinChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 bridgeConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 nominalBridgeResistance,
        float64 firstElectricalVal,
        float64 secondElectricalVal,
        int32 electricalUnits,
        float64 firstPhysicalVal,
        float64 secondPhysicalVal,
        int32 physicalUnits,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["bridgeConfig"] = bridgeConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["nominalBridgeResistance"] = nominalBridgeResistance;
        args["firstElectricalVal"] = firstElectricalVal;
        args["secondElectricalVal"] = secondElectricalVal;
        args["electricalUnits"] = electricalUnits;
        args["firstPhysicalVal"] = firstPhysicalVal;
        args["secondPhysicalVal"] = secondPhysicalVal;
        args["physicalUnits"] = physicalUnits;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back(
            {{"fn", "CreateAITorqueBridgeTwoPointLinChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateAIVelocityIEPEChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 sensitivity,
        int32 sensitivityUnits,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["sensitivity"] = sensitivity;
        args["sensitivityUnits"] = sensitivityUnits;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIVelocityIEPEChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIVoltageChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAIVoltageChanWithExcit(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 bridgeConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        bool32 useExcitForScaling,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["bridgeConfig"] = bridgeConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["useExcitForScaling"] = useExcitForScaling;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIVoltageChanWithExcit"}, {"args", args}});
        return 0;
    }

    int32 CreateAIVoltageRMSChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAIVoltageRMSChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAOCurrentChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAOCurrentChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAOFuncGenChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 type,
        float64 freq,
        float64 amplitude,
        float64 offset
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["type"] = type;
        args["freq"] = freq;
        args["amplitude"] = amplitude;
        args["offset"] = offset;
        this->calls.push_back({{"fn", "CreateAOFuncGenChan"}, {"args", args}});
        return 0;
    }

    int32 CreateAOVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateAOVoltageChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCIAngEncoderChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 decodingType,
        bool32 zidxEnable,
        float64 zidxVal,
        int32 zidxPhase,
        int32 units,
        uInt32 pulsesPerRev,
        float64 initialAngle,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["decodingType"] = decodingType;
        args["zidxEnable"] = zidxEnable;
        args["zidxVal"] = zidxVal;
        args["zidxPhase"] = zidxPhase;
        args["units"] = units;
        args["pulsesPerRev"] = pulsesPerRev;
        args["initialAngle"] = initialAngle;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateCIAngEncoderChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCIAngVelocityChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 decodingType,
        int32 units,
        uInt32 pulsesPerRev,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["decodingType"] = decodingType;
        args["units"] = units;
        args["pulsesPerRev"] = pulsesPerRev;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateCIAngVelocityChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCICountEdgesChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 edge,
        uInt32 initialCount,
        int32 countDirection
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["edge"] = edge;
        args["initialCount"] = initialCount;
        args["countDirection"] = countDirection;
        this->calls.push_back({{"fn", "CreateCICountEdgesChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCIDutyCycleChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minFreq,
        float64 maxFreq,
        int32 edge,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minFreq"] = minFreq;
        args["maxFreq"] = maxFreq;
        args["edge"] = edge;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateCIDutyCycleChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCIFreqChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 edge,
        int32 measMethod,
        float64 measTime,
        uInt32 divisor,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["edge"] = edge;
        args["measMethod"] = measMethod;
        args["measTime"] = measTime;
        args["divisor"] = divisor;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateCIFreqChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCIGPSTimestampChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 syncMethod,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["units"] = units;
        args["syncMethod"] = syncMethod;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateCIGPSTimestampChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCILinEncoderChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 decodingType,
        bool32 zidxEnable,
        float64 zidxVal,
        int32 zidxPhase,
        int32 units,
        float64 distPerPulse,
        float64 initialPos,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["decodingType"] = decodingType;
        args["zidxEnable"] = zidxEnable;
        args["zidxVal"] = zidxVal;
        args["zidxPhase"] = zidxPhase;
        args["units"] = units;
        args["distPerPulse"] = distPerPulse;
        args["initialPos"] = initialPos;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateCILinEncoderChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCILinVelocityChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 decodingType,
        int32 units,
        float64 distPerPulse,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["decodingType"] = decodingType;
        args["units"] = units;
        args["distPerPulse"] = distPerPulse;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateCILinVelocityChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCIPeriodChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 edge,
        int32 measMethod,
        float64 measTime,
        uInt32 divisor,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["edge"] = edge;
        args["measMethod"] = measMethod;
        args["measTime"] = measTime;
        args["divisor"] = divisor;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateCIPeriodChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCIPulseChanFreq(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        this->calls.push_back({{"fn", "CreateCIPulseChanFreq"}, {"args", args}});
        return 0;
    }

    int32 CreateCIPulseChanTicks(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        const char sourceTerminal[],
        float64 minVal,
        float64 maxVal
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        if (sourceTerminal != nullptr)
            args["sourceTerminal"] = std::string(sourceTerminal);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        this->calls.push_back({{"fn", "CreateCIPulseChanTicks"}, {"args", args}});
        return 0;
    }

    int32 CreateCIPulseChanTime(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        this->calls.push_back({{"fn", "CreateCIPulseChanTime"}, {"args", args}});
        return 0;
    }

    int32 CreateCIPulseWidthChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 startingEdge,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["startingEdge"] = startingEdge;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateCIPulseWidthChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCISemiPeriodChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateCISemiPeriodChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCITwoEdgeSepChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 firstEdge,
        int32 secondEdge,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["firstEdge"] = firstEdge;
        args["secondEdge"] = secondEdge;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateCITwoEdgeSepChan"}, {"args", args}});
        return 0;
    }

    int32 CreateCOPulseChanFreq(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 idleState,
        float64 initialDelay,
        float64 freq,
        float64 dutyCycle
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["units"] = units;
        args["idleState"] = idleState;
        args["initialDelay"] = initialDelay;
        args["freq"] = freq;
        args["dutyCycle"] = dutyCycle;
        this->calls.push_back({{"fn", "CreateCOPulseChanFreq"}, {"args", args}});
        return 0;
    }

    int32 CreateCOPulseChanTicks(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        const char sourceTerminal[],
        int32 idleState,
        int32 initialDelay,
        int32 lowTicks,
        int32 highTicks
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        if (sourceTerminal != nullptr)
            args["sourceTerminal"] = std::string(sourceTerminal);
        args["idleState"] = idleState;
        args["initialDelay"] = initialDelay;
        args["lowTicks"] = lowTicks;
        args["highTicks"] = highTicks;
        this->calls.push_back({{"fn", "CreateCOPulseChanTicks"}, {"args", args}});
        return 0;
    }

    int32 CreateCOPulseChanTime(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 idleState,
        float64 initialDelay,
        float64 lowTime,
        float64 highTime
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (counter != nullptr) args["counter"] = std::string(counter);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["units"] = units;
        args["idleState"] = idleState;
        args["initialDelay"] = initialDelay;
        args["lowTime"] = lowTime;
        args["highTime"] = highTime;
        this->calls.push_back({{"fn", "CreateCOPulseChanTime"}, {"args", args}});
        return 0;
    }

    int32 CreateDIChan(
        TaskHandle task,
        const char lines[],
        const char nameToAssignToLines[],
        int32 lineGrouping
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (lines != nullptr) args["lines"] = std::string(lines);
        if (nameToAssignToLines != nullptr)
            args["nameToAssignToLines"] = std::string(nameToAssignToLines);
        args["lineGrouping"] = lineGrouping;
        this->calls.push_back({{"fn", "CreateDIChan"}, {"args", args}});
        return 0;
    }

    int32 CreateDOChan(
        TaskHandle task,
        const char lines[],
        const char nameToAssignToLines[],
        int32 lineGrouping
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (lines != nullptr) args["lines"] = std::string(lines);
        if (nameToAssignToLines != nullptr)
            args["nameToAssignToLines"] = std::string(nameToAssignToLines);
        args["lineGrouping"] = lineGrouping;
        this->calls.push_back({{"fn", "CreateDOChan"}, {"args", args}});
        return 0;
    }

    int32 CreateLinScale(
        const char name[],
        float64 slope,
        float64 yIntercept,
        int32 preScaledUnits,
        const char scaledUnits[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (name != nullptr) args["name"] = std::string(name);
        args["slope"] = slope;
        args["yIntercept"] = yIntercept;
        args["preScaledUnits"] = preScaledUnits;
        if (scaledUnits != nullptr) args["scaledUnits"] = std::string(scaledUnits);
        this->calls.push_back({{"fn", "CreateLinScale"}, {"args", args}});
        return 0;
    }

    int32 CreateMapScale(
        const char name[],
        float64 prescaledMin,
        float64 prescaledMax,
        float64 scaledMin,
        float64 scaledMax,
        int32 preScaledUnits,
        const char scaledUnits[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (name != nullptr) args["name"] = std::string(name);
        args["prescaledMin"] = prescaledMin;
        args["prescaledMax"] = prescaledMax;
        args["scaledMin"] = scaledMin;
        args["scaledMax"] = scaledMax;
        args["preScaledUnits"] = preScaledUnits;
        if (scaledUnits != nullptr) args["scaledUnits"] = std::string(scaledUnits);
        this->calls.push_back({{"fn", "CreateMapScale"}, {"args", args}});
        return 0;
    }

    int32 CreatePolynomialScale(
        const char name[],
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffsIn,
        const float64 reverseCoeffs[],
        uInt32 numReverseCoeffsIn,
        int32 preScaledUnits,
        const char scaledUnits[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (name != nullptr) args["name"] = std::string(name);
        if (forwardCoeffs != nullptr) args["forwardCoeffs"] = "<array>";
        args["numForwardCoeffsIn"] = numForwardCoeffsIn;
        if (reverseCoeffs != nullptr) args["reverseCoeffs"] = "<array>";
        args["numReverseCoeffsIn"] = numReverseCoeffsIn;
        args["preScaledUnits"] = preScaledUnits;
        if (scaledUnits != nullptr) args["scaledUnits"] = std::string(scaledUnits);
        this->calls.push_back({{"fn", "CreatePolynomialScale"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIAccelChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateTEDSAIAccelChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateTEDSAIBridgeChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAICurrentChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 shuntResistorLoc,
        float64 extShuntResistorVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["shuntResistorLoc"] = shuntResistorLoc;
        args["extShuntResistorVal"] = extShuntResistorVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateTEDSAICurrentChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIForceBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateTEDSAIForceBridgeChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIForceIEPEChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateTEDSAIForceIEPEChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIMicrophoneChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        int32 units,
        float64 maxSndPressLevel,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["units"] = units;
        args["maxSndPressLevel"] = maxSndPressLevel;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateTEDSAIMicrophoneChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIPosLVDTChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 voltageExcitFreq,
        int32 acExcitWireMode,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["voltageExcitFreq"] = voltageExcitFreq;
        args["acExcitWireMode"] = acExcitWireMode;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateTEDSAIPosLVDTChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIPosRVDTChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 voltageExcitFreq,
        int32 acExcitWireMode,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["voltageExcitFreq"] = voltageExcitFreq;
        args["acExcitWireMode"] = acExcitWireMode;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateTEDSAIPosRVDTChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIPressureBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back(
            {{"fn", "CreateTEDSAIPressureBridgeChan"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateTEDSAIRTDChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 currentExcitSource,
        float64 currentExcitVal
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["resistanceConfig"] = resistanceConfig;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        this->calls.push_back({{"fn", "CreateTEDSAIRTDChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIResistanceChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["resistanceConfig"] = resistanceConfig;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateTEDSAIResistanceChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIStrainGageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 initialBridgeVoltage,
        float64 leadWireResistance,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["initialBridgeVoltage"] = initialBridgeVoltage;
        args["leadWireResistance"] = leadWireResistance;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateTEDSAIStrainGageChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIThrmcplChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 cjcSource,
        float64 cjcVal,
        const char cjcChannel[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["cjcSource"] = cjcSource;
        args["cjcVal"] = cjcVal;
        if (cjcChannel != nullptr) args["cjcChannel"] = std::string(cjcChannel);
        this->calls.push_back({{"fn", "CreateTEDSAIThrmcplChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIThrmstrChanIex(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 currentExcitSource,
        float64 currentExcitVal
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["resistanceConfig"] = resistanceConfig;
        args["currentExcitSource"] = currentExcitSource;
        args["currentExcitVal"] = currentExcitVal;
        this->calls.push_back({{"fn", "CreateTEDSAIThrmstrChanIex"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIThrmstrChanVex(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 r1
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["resistanceConfig"] = resistanceConfig;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        args["r1"] = r1;
        this->calls.push_back({{"fn", "CreateTEDSAIThrmstrChanVex"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAITorqueBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateTEDSAITorqueBridgeChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back({{"fn", "CreateTEDSAIVoltageChan"}, {"args", args}});
        return 0;
    }

    int32 CreateTEDSAIVoltageChanWithExcit(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (nameToAssignToChannel != nullptr)
            args["nameToAssignToChannel"] = std::string(nameToAssignToChannel);
        args["terminalConfig"] = terminalConfig;
        args["minVal"] = minVal;
        args["maxVal"] = maxVal;
        args["units"] = units;
        args["voltageExcitSource"] = voltageExcitSource;
        args["voltageExcitVal"] = voltageExcitVal;
        if (customScaleName != nullptr)
            args["customScaleName"] = std::string(customScaleName);
        this->calls.push_back(
            {{"fn", "CreateTEDSAIVoltageChanWithExcit"}, {"args", args}}
        );
        return 0;
    }

    int32 CreateTableScale(
        const char name[],
        const float64 prescaledVals[],
        uInt32 numPrescaledValsIn,
        const float64 scaledVals[],
        uInt32 numScaledValsIn,
        int32 preScaledUnits,
        const char scaledUnits[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (name != nullptr) args["name"] = std::string(name);
        if (prescaledVals != nullptr) args["prescaledVals"] = "<array>";
        args["numPrescaledValsIn"] = numPrescaledValsIn;
        if (scaledVals != nullptr) args["scaledVals"] = "<array>";
        args["numScaledValsIn"] = numScaledValsIn;
        args["preScaledUnits"] = preScaledUnits;
        if (scaledUnits != nullptr) args["scaledUnits"] = std::string(scaledUnits);
        this->calls.push_back({{"fn", "CreateTableScale"}, {"args", args}});
        return 0;
    }

    int32 CreateTask(const char sessionName[], TaskHandle *task) override {
        nlohmann::json args = nlohmann::json::object();
        if (sessionName != nullptr) args["sessionName"] = std::string(sessionName);
        this->calls.push_back({{"fn", "CreateTask"}, {"args", args}});
        return 0;
    }

    int32 CreateWatchdogTimerTaskEx(
        const char deviceName[],
        const char sessionName[],
        TaskHandle *task,
        float64 timeout
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        if (sessionName != nullptr) args["sessionName"] = std::string(sessionName);
        args["timeout"] = timeout;
        this->calls.push_back({{"fn", "CreateWatchdogTimerTaskEx"}, {"args", args}});
        return 0;
    }

    int32 DeleteNetworkDevice(const char deviceName[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        this->calls.push_back({{"fn", "DeleteNetworkDevice"}, {"args", args}});
        return 0;
    }

    int32 DeleteSavedGlobalChan(const char channelName[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (channelName != nullptr) args["channelName"] = std::string(channelName);
        this->calls.push_back({{"fn", "DeleteSavedGlobalChan"}, {"args", args}});
        return 0;
    }

    int32 DeleteSavedScale(const char scaleName[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (scaleName != nullptr) args["scaleName"] = std::string(scaleName);
        this->calls.push_back({{"fn", "DeleteSavedScale"}, {"args", args}});
        return 0;
    }

    int32 DeleteSavedTask(const char taskName[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (taskName != nullptr) args["taskName"] = std::string(taskName);
        this->calls.push_back({{"fn", "DeleteSavedTask"}, {"args", args}});
        return 0;
    }

    int32 DeviceSupportsCal(const char deviceName[], bool32 *calSupported) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["calSupported"] = "<ptr>";
        this->calls.push_back({{"fn", "DeviceSupportsCal"}, {"args", args}});
        return 0;
    }

    int32 DisableRefTrig(TaskHandle task) override {
        nlohmann::json args = nlohmann::json::object();

        this->calls.push_back({{"fn", "DisableRefTrig"}, {"args", args}});
        return 0;
    }

    int32 DisableStartTrig(TaskHandle task) override {
        nlohmann::json args = nlohmann::json::object();

        this->calls.push_back({{"fn", "DisableStartTrig"}, {"args", args}});
        return 0;
    }

    int32 DisconnectTerms(
        const char sourceTerminal[],
        const char destinationTerminal[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (sourceTerminal != nullptr)
            args["sourceTerminal"] = std::string(sourceTerminal);
        if (destinationTerminal != nullptr)
            args["destinationTerminal"] = std::string(destinationTerminal);
        this->calls.push_back({{"fn", "DisconnectTerms"}, {"args", args}});
        return 0;
    }

    int32 ExportSignal(
        TaskHandle task,
        int32 signalID,
        const char outputTerminal[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["signalID"] = signalID;
        if (outputTerminal != nullptr)
            args["outputTerminal"] = std::string(outputTerminal);
        this->calls.push_back({{"fn", "ExportSignal"}, {"args", args}});
        return 0;
    }

    int32 GetAIChanCalCalDate(
        TaskHandle task,
        const char channelName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channelName != nullptr) args["channelName"] = std::string(channelName);
        args["year"] = "<ptr>";
        args["month"] = "<ptr>";
        args["day"] = "<ptr>";
        args["hour"] = "<ptr>";
        args["minute"] = "<ptr>";
        this->calls.push_back({{"fn", "GetAIChanCalCalDate"}, {"args", args}});
        return 0;
    }

    int32 GetAIChanCalExpDate(
        TaskHandle task,
        const char channelName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channelName != nullptr) args["channelName"] = std::string(channelName);
        args["year"] = "<ptr>";
        args["month"] = "<ptr>";
        args["day"] = "<ptr>";
        args["hour"] = "<ptr>";
        args["minute"] = "<ptr>";
        this->calls.push_back({{"fn", "GetAIChanCalExpDate"}, {"args", args}});
        return 0;
    }

    int32 GetAnalogPowerUpStatesWithOutputType(
        const char channelNames[],
        float64 stateArray[],
        int32 channelTypeArray[],
        uInt32 *arraySize
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channelNames != nullptr) args["channelNames"] = std::string(channelNames);
        if (stateArray != nullptr) args["stateArray"] = "<array>";
        if (channelTypeArray != nullptr) args["channelTypeArray"] = "<array>";
        args["arraySize"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetAnalogPowerUpStatesWithOutputType"}, {"args", args}}
        );
        return 0;
    }

    int32 GetArmStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back({{"fn", "GetArmStartTrigTimestampVal"}, {"args", args}});
        return 0;
    }

    int32 GetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back({{"fn", "GetArmStartTrigTrigWhen"}, {"args", args}});
        return 0;
    }

    int32 GetAutoConfiguredCDAQSyncConnections(
        char portList[],
        uInt32 portListSize
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (portList != nullptr) args["portList"] = std::string(portList);
        args["portListSize"] = portListSize;
        this->calls.push_back(
            {{"fn", "GetAutoConfiguredCDAQSyncConnections"}, {"args", args}}
        );
        return 0;
    }

    int32
    GetBufferAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetBufferAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32 GetCalInfoAttributeBool(
        const char deviceName[],
        int32 attribute,
        bool32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetCalInfoAttributeBool"}, {"args", args}});
        return 0;
    }

    int32 GetCalInfoAttributeDouble(
        const char deviceName[],
        int32 attribute,
        float64 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetCalInfoAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 GetCalInfoAttributeString(
        const char deviceName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back({{"fn", "GetCalInfoAttributeString"}, {"args", args}});
        return 0;
    }

    int32 GetCalInfoAttributeUInt32(
        const char deviceName[],
        int32 attribute,
        uInt32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetCalInfoAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32 GetChanAttributeBool(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        bool32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetChanAttributeBool"}, {"args", args}});
        return 0;
    }

    int32 GetChanAttributeDouble(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetChanAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 GetChanAttributeDoubleArray(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back({{"fn", "GetChanAttributeDoubleArray"}, {"args", args}});
        return 0;
    }

    int32 GetChanAttributeInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        int32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetChanAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 GetChanAttributeString(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back({{"fn", "GetChanAttributeString"}, {"args", args}});
        return 0;
    }

    int32 GetChanAttributeUInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        uInt32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetChanAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32 GetDeviceAttributeBool(
        const char deviceName[],
        int32 attribute,
        bool32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetDeviceAttributeBool"}, {"args", args}});
        return 0;
    }

    int32 GetDeviceAttributeDouble(
        const char deviceName[],
        int32 attribute,
        float64 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetDeviceAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 GetDeviceAttributeDoubleArray(
        const char deviceName[],
        int32 attribute,
        float64 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back(
            {{"fn", "GetDeviceAttributeDoubleArray"}, {"args", args}}
        );
        return 0;
    }

    int32 GetDeviceAttributeInt32(
        const char deviceName[],
        int32 attribute,
        int32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetDeviceAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 GetDeviceAttributeInt32Array(
        const char deviceName[],
        int32 attribute,
        int32 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back({{"fn", "GetDeviceAttributeInt32Array"}, {"args", args}});
        return 0;
    }

    int32 GetDeviceAttributeString(
        const char deviceName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back({{"fn", "GetDeviceAttributeString"}, {"args", args}});
        return 0;
    }

    int32 GetDeviceAttributeUInt32(
        const char deviceName[],
        int32 attribute,
        uInt32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetDeviceAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32 GetDeviceAttributeUInt32Array(
        const char deviceName[],
        int32 attribute,
        uInt32 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back(
            {{"fn", "GetDeviceAttributeUInt32Array"}, {"args", args}}
        );
        return 0;
    }

    int32 GetDigitalLogicFamilyPowerUpState(
        const char deviceName[],
        int32 *logicFamily
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["logicFamily"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetDigitalLogicFamilyPowerUpState"}, {"args", args}}
        );
        return 0;
    }

    int32 GetDisconnectedCDAQSyncPorts(char portList[], uInt32 portListSize) override {
        nlohmann::json args = nlohmann::json::object();
        if (portList != nullptr) args["portList"] = std::string(portList);
        args["portListSize"] = portListSize;
        this->calls.push_back({{"fn", "GetDisconnectedCDAQSyncPorts"}, {"args", args}});
        return 0;
    }

    int32
    GetErrorString(int32 errorCode, char errorString[], uInt32 bufferSize) override {
        nlohmann::json args = nlohmann::json::object();
        args["errorCode"] = errorCode;
        if (errorString != nullptr) args["errorString"] = std::string(errorString);
        args["bufferSize"] = bufferSize;
        this->calls.push_back({{"fn", "GetErrorString"}, {"args", args}});
        return 0;
    }

    int32 GetExportedSignalAttributeBool(
        TaskHandle task,
        int32 attribute,
        bool32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetExportedSignalAttributeBool"}, {"args", args}}
        );
        return 0;
    }

    int32 GetExportedSignalAttributeDouble(
        TaskHandle task,
        int32 attribute,
        float64 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetExportedSignalAttributeDouble"}, {"args", args}}
        );
        return 0;
    }

    int32 GetExportedSignalAttributeInt32(
        TaskHandle task,
        int32 attribute,
        int32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetExportedSignalAttributeInt32"}, {"args", args}}
        );
        return 0;
    }

    int32 GetExportedSignalAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back(
            {{"fn", "GetExportedSignalAttributeString"}, {"args", args}}
        );
        return 0;
    }

    int32 GetExportedSignalAttributeUInt32(
        TaskHandle task,
        int32 attribute,
        uInt32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetExportedSignalAttributeUInt32"}, {"args", args}}
        );
        return 0;
    }

    int32 GetExtCalLastDateAndTime(
        const char deviceName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["year"] = "<ptr>";
        args["month"] = "<ptr>";
        args["day"] = "<ptr>";
        args["hour"] = "<ptr>";
        args["minute"] = "<ptr>";
        this->calls.push_back({{"fn", "GetExtCalLastDateAndTime"}, {"args", args}});
        return 0;
    }

    int32 GetExtendedErrorInfo(char errorString[], uInt32 bufferSize) override {
        nlohmann::json args = nlohmann::json::object();
        if (errorString != nullptr) args["errorString"] = std::string(errorString);
        args["bufferSize"] = bufferSize;
        this->calls.push_back({{"fn", "GetExtendedErrorInfo"}, {"args", args}});
        return 0;
    }

    int32 GetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime *data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back({{"fn", "GetFirstSampClkWhen"}, {"args", args}});
        return 0;
    }

    int32 GetFirstSampTimestampVal(TaskHandle task, CVIAbsoluteTime *data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back({{"fn", "GetFirstSampTimestampVal"}, {"args", args}});
        return 0;
    }

    int32 GetNthTaskChannel(
        TaskHandle task,
        uInt32 index,
        char buffer[],
        int32 bufferSize
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["index"] = index;
        if (buffer != nullptr) args["buffer"] = std::string(buffer);
        args["bufferSize"] = bufferSize;
        this->calls.push_back({{"fn", "GetNthTaskChannel"}, {"args", args}});
        return 0;
    }

    int32 GetNthTaskDevice(
        TaskHandle task,
        uInt32 index,
        char buffer[],
        int32 bufferSize
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["index"] = index;
        if (buffer != nullptr) args["buffer"] = std::string(buffer);
        args["bufferSize"] = bufferSize;
        this->calls.push_back({{"fn", "GetNthTaskDevice"}, {"args", args}});
        return 0;
    }

    int32 GetNthTaskReadChannel(
        TaskHandle task,
        uInt32 index,
        char buffer[],
        int32 bufferSize
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["index"] = index;
        if (buffer != nullptr) args["buffer"] = std::string(buffer);
        args["bufferSize"] = bufferSize;
        this->calls.push_back({{"fn", "GetNthTaskReadChannel"}, {"args", args}});
        return 0;
    }

    int32 GetPersistedChanAttributeBool(
        const char channel[],
        int32 attribute,
        bool32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetPersistedChanAttributeBool"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPersistedChanAttributeString(
        const char channel[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back(
            {{"fn", "GetPersistedChanAttributeString"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPersistedScaleAttributeBool(
        const char scaleName[],
        int32 attribute,
        bool32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (scaleName != nullptr) args["scaleName"] = std::string(scaleName);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetPersistedScaleAttributeBool"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPersistedScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (scaleName != nullptr) args["scaleName"] = std::string(scaleName);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back(
            {{"fn", "GetPersistedScaleAttributeString"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPersistedTaskAttributeBool(
        const char taskName[],
        int32 attribute,
        bool32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (taskName != nullptr) args["taskName"] = std::string(taskName);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetPersistedTaskAttributeBool"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPersistedTaskAttributeString(
        const char taskName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (taskName != nullptr) args["taskName"] = std::string(taskName);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back(
            {{"fn", "GetPersistedTaskAttributeString"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPhysicalChanAttributeBool(
        const char physicalChannel[],
        int32 attribute,
        bool32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetPhysicalChanAttributeBool"}, {"args", args}});
        return 0;
    }

    int32 GetPhysicalChanAttributeBytes(
        const char physicalChannel[],
        int32 attribute,
        uInt8 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back(
            {{"fn", "GetPhysicalChanAttributeBytes"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPhysicalChanAttributeDouble(
        const char physicalChannel[],
        int32 attribute,
        float64 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetPhysicalChanAttributeDouble"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPhysicalChanAttributeDoubleArray(
        const char physicalChannel[],
        int32 attribute,
        float64 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back(
            {{"fn", "GetPhysicalChanAttributeDoubleArray"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPhysicalChanAttributeInt32(
        const char physicalChannel[],
        int32 attribute,
        int32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetPhysicalChanAttributeInt32"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPhysicalChanAttributeInt32Array(
        const char physicalChannel[],
        int32 attribute,
        int32 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back(
            {{"fn", "GetPhysicalChanAttributeInt32Array"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPhysicalChanAttributeString(
        const char physicalChannel[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back(
            {{"fn", "GetPhysicalChanAttributeString"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPhysicalChanAttributeUInt32(
        const char physicalChannel[],
        int32 attribute,
        uInt32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetPhysicalChanAttributeUInt32"}, {"args", args}}
        );
        return 0;
    }

    int32 GetPhysicalChanAttributeUInt32Array(
        const char physicalChannel[],
        int32 attribute,
        uInt32 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back(
            {{"fn", "GetPhysicalChanAttributeUInt32Array"}, {"args", args}}
        );
        return 0;
    }

    int32
    GetReadAttributeBool(TaskHandle task, int32 attribute, bool32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetReadAttributeBool"}, {"args", args}});
        return 0;
    }

    int32
    GetReadAttributeDouble(TaskHandle task, int32 attribute, float64 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetReadAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32
    GetReadAttributeInt32(TaskHandle task, int32 attribute, int32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetReadAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 GetReadAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back({{"fn", "GetReadAttributeString"}, {"args", args}});
        return 0;
    }

    int32
    GetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetReadAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32
    GetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetReadAttributeUInt64"}, {"args", args}});
        return 0;
    }

    int32
    GetRealTimeAttributeBool(TaskHandle task, int32 attribute, bool32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetRealTimeAttributeBool"}, {"args", args}});
        return 0;
    }

    int32
    GetRealTimeAttributeInt32(TaskHandle task, int32 attribute, int32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetRealTimeAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 GetRealTimeAttributeUInt32(
        TaskHandle task,
        int32 attribute,
        uInt32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetRealTimeAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32 GetRefTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back({{"fn", "GetRefTrigTimestampVal"}, {"args", args}});
        return 0;
    }

    int32 GetScaleAttributeDouble(
        const char scaleName[],
        int32 attribute,
        float64 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (scaleName != nullptr) args["scaleName"] = std::string(scaleName);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetScaleAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 GetScaleAttributeDoubleArray(
        const char scaleName[],
        int32 attribute,
        float64 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (scaleName != nullptr) args["scaleName"] = std::string(scaleName);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back({{"fn", "GetScaleAttributeDoubleArray"}, {"args", args}});
        return 0;
    }

    int32 GetScaleAttributeInt32(
        const char scaleName[],
        int32 attribute,
        int32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (scaleName != nullptr) args["scaleName"] = std::string(scaleName);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetScaleAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 GetScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (scaleName != nullptr) args["scaleName"] = std::string(scaleName);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back({{"fn", "GetScaleAttributeString"}, {"args", args}});
        return 0;
    }

    int32 GetSelfCalLastDateAndTime(
        const char deviceName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["year"] = "<ptr>";
        args["month"] = "<ptr>";
        args["day"] = "<ptr>";
        args["hour"] = "<ptr>";
        args["minute"] = "<ptr>";
        this->calls.push_back({{"fn", "GetSelfCalLastDateAndTime"}, {"args", args}});
        return 0;
    }

    int32 GetStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back({{"fn", "GetStartTrigTimestampVal"}, {"args", args}});
        return 0;
    }

    int32 GetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back({{"fn", "GetStartTrigTrigWhen"}, {"args", args}});
        return 0;
    }

    int32 GetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime *data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back({{"fn", "GetSyncPulseTimeWhen"}, {"args", args}});
        return 0;
    }

    int32
    GetSystemInfoAttributeString(int32 attribute, char value[], uInt32 size) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back({{"fn", "GetSystemInfoAttributeString"}, {"args", args}});
        return 0;
    }

    int32 GetSystemInfoAttributeUInt32(int32 attribute, uInt32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetSystemInfoAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32
    GetTaskAttributeBool(TaskHandle task, int32 attribute, bool32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTaskAttributeBool"}, {"args", args}});
        return 0;
    }

    int32 GetTaskAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back({{"fn", "GetTaskAttributeString"}, {"args", args}});
        return 0;
    }

    int32
    GetTaskAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTaskAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32
    GetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTimingAttributeBool"}, {"args", args}});
        return 0;
    }

    int32 GetTimingAttributeDouble(
        TaskHandle task,
        int32 attribute,
        float64 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTimingAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 GetTimingAttributeExBool(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        bool32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTimingAttributeExBool"}, {"args", args}});
        return 0;
    }

    int32 GetTimingAttributeExDouble(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        float64 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTimingAttributeExDouble"}, {"args", args}});
        return 0;
    }

    int32 GetTimingAttributeExInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        int32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTimingAttributeExInt32"}, {"args", args}});
        return 0;
    }

    int32 GetTimingAttributeExString(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back({{"fn", "GetTimingAttributeExString"}, {"args", args}});
        return 0;
    }

    int32 GetTimingAttributeExTimestamp(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        CVIAbsoluteTime *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetTimingAttributeExTimestamp"}, {"args", args}}
        );
        return 0;
    }

    int32 GetTimingAttributeExUInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTimingAttributeExUInt32"}, {"args", args}});
        return 0;
    }

    int32 GetTimingAttributeExUInt64(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt64 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTimingAttributeExUInt64"}, {"args", args}});
        return 0;
    }

    int32
    GetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTimingAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 GetTimingAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back({{"fn", "GetTimingAttributeString"}, {"args", args}});
        return 0;
    }

    int32 GetTimingAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTimingAttributeTimestamp"}, {"args", args}});
        return 0;
    }

    int32
    GetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTimingAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32
    GetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTimingAttributeUInt64"}, {"args", args}});
        return 0;
    }

    int32
    GetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTrigAttributeBool"}, {"args", args}});
        return 0;
    }

    int32
    GetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTrigAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 GetTrigAttributeDoubleArray(
        TaskHandle task,
        int32 attribute,
        float64 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back({{"fn", "GetTrigAttributeDoubleArray"}, {"args", args}});
        return 0;
    }

    int32
    GetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTrigAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 GetTrigAttributeInt32Array(
        TaskHandle task,
        int32 attribute,
        int32 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back({{"fn", "GetTrigAttributeInt32Array"}, {"args", args}});
        return 0;
    }

    int32 GetTrigAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back({{"fn", "GetTrigAttributeString"}, {"args", args}});
        return 0;
    }

    int32 GetTrigAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTrigAttributeTimestamp"}, {"args", args}});
        return 0;
    }

    int32
    GetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetTrigAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32 GetWatchdogAttributeBool(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        bool32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (lines != nullptr) args["lines"] = std::string(lines);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetWatchdogAttributeBool"}, {"args", args}});
        return 0;
    }

    int32 GetWatchdogAttributeDouble(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        float64 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (lines != nullptr) args["lines"] = std::string(lines);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetWatchdogAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 GetWatchdogAttributeInt32(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        int32 *value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (lines != nullptr) args["lines"] = std::string(lines);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetWatchdogAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 GetWatchdogAttributeString(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (lines != nullptr) args["lines"] = std::string(lines);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back({{"fn", "GetWatchdogAttributeString"}, {"args", args}});
        return 0;
    }

    int32
    GetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetWriteAttributeBool"}, {"args", args}});
        return 0;
    }

    int32
    GetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetWriteAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32
    GetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetWriteAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 GetWriteAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        args["size"] = size;
        this->calls.push_back({{"fn", "GetWriteAttributeString"}, {"args", args}});
        return 0;
    }

    int32
    GetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetWriteAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32
    GetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "GetWriteAttributeUInt64"}, {"args", args}});
        return 0;
    }

    int32 IsTaskDone(TaskHandle task, bool32 *isTaskDone) override {
        nlohmann::json args = nlohmann::json::object();
        args["isTaskDone"] = "<ptr>";
        this->calls.push_back({{"fn", "IsTaskDone"}, {"args", args}});
        return 0;
    }

    int32 LoadTask(const char sessionName[], TaskHandle *task) override {
        nlohmann::json args = nlohmann::json::object();
        if (sessionName != nullptr) args["sessionName"] = std::string(sessionName);
        this->calls.push_back({{"fn", "LoadTask"}, {"args", args}});
        return 0;
    }

    int32 PerformBridgeOffsetNullingCalEx(
        TaskHandle task,
        const char channel[],
        bool32 skipUnsupportedChannels
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["skipUnsupportedChannels"] = skipUnsupportedChannels;
        this->calls.push_back(
            {{"fn", "PerformBridgeOffsetNullingCalEx"}, {"args", args}}
        );
        return 0;
    }

    int32 PerformBridgeShuntCalEx(
        TaskHandle task,
        const char channel[],
        float64 shuntResistorValue,
        int32 shuntResistorLocation,
        int32 shuntResistorSelect,
        int32 shuntResistorSource,
        float64 bridgeResistance,
        bool32 skipUnsupportedChannels
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["shuntResistorValue"] = shuntResistorValue;
        args["shuntResistorLocation"] = shuntResistorLocation;
        args["shuntResistorSelect"] = shuntResistorSelect;
        args["shuntResistorSource"] = shuntResistorSource;
        args["bridgeResistance"] = bridgeResistance;
        args["skipUnsupportedChannels"] = skipUnsupportedChannels;
        this->calls.push_back({{"fn", "PerformBridgeShuntCalEx"}, {"args", args}});
        return 0;
    }

    int32 PerformStrainShuntCalEx(
        TaskHandle task,
        const char channel[],
        float64 shuntResistorValue,
        int32 shuntResistorLocation,
        int32 shuntResistorSelect,
        int32 shuntResistorSource,
        bool32 skipUnsupportedChannels
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["shuntResistorValue"] = shuntResistorValue;
        args["shuntResistorLocation"] = shuntResistorLocation;
        args["shuntResistorSelect"] = shuntResistorSelect;
        args["shuntResistorSource"] = shuntResistorSource;
        args["skipUnsupportedChannels"] = skipUnsupportedChannels;
        this->calls.push_back({{"fn", "PerformStrainShuntCalEx"}, {"args", args}});
        return 0;
    }

    int32 PerformThrmcplLeadOffsetNullingCal(
        TaskHandle task,
        const char channel[],
        bool32 skipUnsupportedChannels
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["skipUnsupportedChannels"] = skipUnsupportedChannels;
        this->calls.push_back(
            {{"fn", "PerformThrmcplLeadOffsetNullingCal"}, {"args", args}}
        );
        return 0;
    }

    int32 ReadAnalogF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadAnalogF64"}, {"args", args}});
        return 0;
    }

    int32 ReadAnalogScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *value,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["timeout"] = timeout;
        args["value"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadAnalogScalarF64"}, {"args", args}});
        return 0;
    }

    int32 ReadBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadBinaryI16"}, {"args", args}});
        return 0;
    }

    int32 ReadBinaryI32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadBinaryI32"}, {"args", args}});
        return 0;
    }

    int32 ReadBinaryU16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadBinaryU16"}, {"args", args}});
        return 0;
    }

    int32 ReadBinaryU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadBinaryU32"}, {"args", args}});
        return 0;
    }

    int32 ReadCounterF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadCounterF64"}, {"args", args}});
        return 0;
    }

    int32 ReadCounterF64Ex(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadCounterF64Ex"}, {"args", args}});
        return 0;
    }

    int32 ReadCounterScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *value,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["timeout"] = timeout;
        args["value"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadCounterScalarF64"}, {"args", args}});
        return 0;
    }

    int32 ReadCounterScalarU32(
        TaskHandle task,
        float64 timeout,
        uInt32 *value,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["timeout"] = timeout;
        args["value"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadCounterScalarU32"}, {"args", args}});
        return 0;
    }

    int32 ReadCounterU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadCounterU32"}, {"args", args}});
        return 0;
    }

    int32 ReadCounterU32Ex(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadCounterU32Ex"}, {"args", args}});
        return 0;
    }

    int32 ReadCtrFreq(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 interleaved,
        float64 readArrayFrequency[],
        float64 readArrayDutyCycle[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["interleaved"] = interleaved;
        if (readArrayFrequency != nullptr) args["readArrayFrequency"] = "<array>";
        if (readArrayDutyCycle != nullptr) args["readArrayDutyCycle"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadCtrFreq"}, {"args", args}});
        return 0;
    }

    int32 ReadCtrFreqScalar(
        TaskHandle task,
        float64 timeout,
        float64 *frequency,
        float64 *dutyCycle,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["timeout"] = timeout;
        args["frequency"] = "<ptr>";
        args["dutyCycle"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadCtrFreqScalar"}, {"args", args}});
        return 0;
    }

    int32 ReadCtrTicks(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 interleaved,
        uInt32 readArrayHighTicks[],
        uInt32 readArrayLowTicks[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["interleaved"] = interleaved;
        if (readArrayHighTicks != nullptr) args["readArrayHighTicks"] = "<array>";
        if (readArrayLowTicks != nullptr) args["readArrayLowTicks"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadCtrTicks"}, {"args", args}});
        return 0;
    }

    int32 ReadCtrTicksScalar(
        TaskHandle task,
        float64 timeout,
        uInt32 *highTicks,
        uInt32 *lowTicks,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["timeout"] = timeout;
        args["highTicks"] = "<ptr>";
        args["lowTicks"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadCtrTicksScalar"}, {"args", args}});
        return 0;
    }

    int32 ReadCtrTime(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 interleaved,
        float64 readArrayHighTime[],
        float64 readArrayLowTime[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["interleaved"] = interleaved;
        if (readArrayHighTime != nullptr) args["readArrayHighTime"] = "<array>";
        if (readArrayLowTime != nullptr) args["readArrayLowTime"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadCtrTime"}, {"args", args}});
        return 0;
    }

    int32 ReadCtrTimeScalar(
        TaskHandle task,
        float64 timeout,
        float64 *highTime,
        float64 *lowTime,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["timeout"] = timeout;
        args["highTime"] = "<ptr>";
        args["lowTime"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadCtrTimeScalar"}, {"args", args}});
        return 0;
    }

    int32 ReadDigitalLines(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt8 readArray[],
        uInt32 arraySizeInBytes,
        int32 *sampsPerChanRead,
        int32 *numBytesPerSamp,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInBytes"] = arraySizeInBytes;
        args["sampsPerChanRead"] = "<ptr>";
        args["numBytesPerSamp"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadDigitalLines"}, {"args", args}});
        return 0;
    }

    int32 ReadDigitalScalarU32(
        TaskHandle task,
        float64 timeout,
        uInt32 *value,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["timeout"] = timeout;
        args["value"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadDigitalScalarU32"}, {"args", args}});
        return 0;
    }

    int32 ReadDigitalU16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadDigitalU16"}, {"args", args}});
        return 0;
    }

    int32 ReadDigitalU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadDigitalU32"}, {"args", args}});
        return 0;
    }

    int32 ReadDigitalU8(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt8 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadDigitalU8"}, {"args", args}});
        return 0;
    }

    int32 ReadPowerBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int16 readArrayVoltage[],
        int16 readArrayCurrent[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArrayVoltage != nullptr) args["readArrayVoltage"] = "<array>";
        if (readArrayCurrent != nullptr) args["readArrayCurrent"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadPowerBinaryI16"}, {"args", args}});
        return 0;
    }

    int32 ReadPowerF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArrayVoltage[],
        float64 readArrayCurrent[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        args["fillMode"] = fillMode;
        if (readArrayVoltage != nullptr) args["readArrayVoltage"] = "<array>";
        if (readArrayCurrent != nullptr) args["readArrayCurrent"] = "<array>";
        args["arraySizeInSamps"] = arraySizeInSamps;
        args["sampsPerChanRead"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadPowerF64"}, {"args", args}});
        return 0;
    }

    int32 ReadPowerScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *voltage,
        float64 *current,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["timeout"] = timeout;
        args["voltage"] = "<ptr>";
        args["current"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadPowerScalarF64"}, {"args", args}});
        return 0;
    }

    int32 ReadRaw(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        uInt8 readArray[],
        uInt32 arraySizeInBytes,
        int32 *sampsRead,
        int32 *numBytesPerSamp,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["timeout"] = timeout;
        if (readArray != nullptr) args["readArray"] = "<array>";
        args["arraySizeInBytes"] = arraySizeInBytes;
        args["sampsRead"] = "<ptr>";
        args["numBytesPerSamp"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "ReadRaw"}, {"args", args}});
        return 0;
    }

    int32 RegisterDoneEvent(
        TaskHandle task,
        uInt32 options,
        DAQmxDoneEventCallbackPtr callbackFunction,
        void *callbackData
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["options"] = options;
        args["callbackFunction"] = "<ptr>";
        args["callbackData"] = "<ptr>";
        this->calls.push_back({{"fn", "RegisterDoneEvent"}, {"args", args}});
        return 0;
    }

    int32 RegisterEveryNSamplesEvent(
        TaskHandle task,
        int32 everyNSamplesEventType,
        uInt32 nSamples,
        uInt32 options,
        DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
        void *callbackData
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["everyNSamplesEventType"] = everyNSamplesEventType;
        args["nSamples"] = nSamples;
        args["options"] = options;
        args["callbackFunction"] = "<ptr>";
        args["callbackData"] = "<ptr>";
        this->calls.push_back({{"fn", "RegisterEveryNSamplesEvent"}, {"args", args}});
        return 0;
    }

    int32 RegisterSignalEvent(
        TaskHandle task,
        int32 signalID,
        uInt32 options,
        DAQmxSignalEventCallbackPtr callbackFunction,
        void *callbackData
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["signalID"] = signalID;
        args["options"] = options;
        args["callbackFunction"] = "<ptr>";
        args["callbackData"] = "<ptr>";
        this->calls.push_back({{"fn", "RegisterSignalEvent"}, {"args", args}});
        return 0;
    }

    int32 RemoveCDAQSyncConnection(const char portList[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (portList != nullptr) args["portList"] = std::string(portList);
        this->calls.push_back({{"fn", "RemoveCDAQSyncConnection"}, {"args", args}});
        return 0;
    }

    int32
    ReserveNetworkDevice(const char deviceName[], bool32 overrideReservation) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["overrideReservation"] = overrideReservation;
        this->calls.push_back({{"fn", "ReserveNetworkDevice"}, {"args", args}});
        return 0;
    }

    int32 ResetBufferAttribute(TaskHandle task, int32 attribute) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        this->calls.push_back({{"fn", "ResetBufferAttribute"}, {"args", args}});
        return 0;
    }

    int32 ResetChanAttribute(
        TaskHandle task,
        const char channel[],
        int32 attribute
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        this->calls.push_back({{"fn", "ResetChanAttribute"}, {"args", args}});
        return 0;
    }

    int32 ResetDevice(const char deviceName[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        this->calls.push_back({{"fn", "ResetDevice"}, {"args", args}});
        return 0;
    }

    int32 ResetExportedSignalAttribute(TaskHandle task, int32 attribute) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        this->calls.push_back({{"fn", "ResetExportedSignalAttribute"}, {"args", args}});
        return 0;
    }

    int32 ResetReadAttribute(TaskHandle task, int32 attribute) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        this->calls.push_back({{"fn", "ResetReadAttribute"}, {"args", args}});
        return 0;
    }

    int32 ResetRealTimeAttribute(TaskHandle task, int32 attribute) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        this->calls.push_back({{"fn", "ResetRealTimeAttribute"}, {"args", args}});
        return 0;
    }

    int32 ResetTimingAttribute(TaskHandle task, int32 attribute) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        this->calls.push_back({{"fn", "ResetTimingAttribute"}, {"args", args}});
        return 0;
    }

    int32 ResetTimingAttributeEx(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        this->calls.push_back({{"fn", "ResetTimingAttributeEx"}, {"args", args}});
        return 0;
    }

    int32 ResetTrigAttribute(TaskHandle task, int32 attribute) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        this->calls.push_back({{"fn", "ResetTrigAttribute"}, {"args", args}});
        return 0;
    }

    int32 ResetWatchdogAttribute(
        TaskHandle task,
        const char lines[],
        int32 attribute
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (lines != nullptr) args["lines"] = std::string(lines);
        args["attribute"] = attribute;
        this->calls.push_back({{"fn", "ResetWatchdogAttribute"}, {"args", args}});
        return 0;
    }

    int32 ResetWriteAttribute(TaskHandle task, int32 attribute) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        this->calls.push_back({{"fn", "ResetWriteAttribute"}, {"args", args}});
        return 0;
    }

    int32 RestoreLastExtCalConst(const char deviceName[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        this->calls.push_back({{"fn", "RestoreLastExtCalConst"}, {"args", args}});
        return 0;
    }

    int32 SaveGlobalChan(
        TaskHandle task,
        const char channelName[],
        const char saveAs[],
        const char author[],
        uInt32 options
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channelName != nullptr) args["channelName"] = std::string(channelName);
        if (saveAs != nullptr) args["saveAs"] = std::string(saveAs);
        if (author != nullptr) args["author"] = std::string(author);
        args["options"] = options;
        this->calls.push_back({{"fn", "SaveGlobalChan"}, {"args", args}});
        return 0;
    }

    int32 SaveScale(
        const char scaleName[],
        const char saveAs[],
        const char author[],
        uInt32 options
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (scaleName != nullptr) args["scaleName"] = std::string(scaleName);
        if (saveAs != nullptr) args["saveAs"] = std::string(saveAs);
        if (author != nullptr) args["author"] = std::string(author);
        args["options"] = options;
        this->calls.push_back({{"fn", "SaveScale"}, {"args", args}});
        return 0;
    }

    int32 SaveTask(
        TaskHandle task,
        const char saveAs[],
        const char author[],
        uInt32 options
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (saveAs != nullptr) args["saveAs"] = std::string(saveAs);
        if (author != nullptr) args["author"] = std::string(author);
        args["options"] = options;
        this->calls.push_back({{"fn", "SaveTask"}, {"args", args}});
        return 0;
    }

    int32 SelfCal(const char deviceName[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        this->calls.push_back({{"fn", "SelfCal"}, {"args", args}});
        return 0;
    }

    int32 SelfTestDevice(const char deviceName[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        this->calls.push_back({{"fn", "SelfTestDevice"}, {"args", args}});
        return 0;
    }

    int32 SetAIChanCalCalDate(
        TaskHandle task,
        const char channelName[],
        uInt32 year,
        uInt32 month,
        uInt32 day,
        uInt32 hour,
        uInt32 minute
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channelName != nullptr) args["channelName"] = std::string(channelName);
        args["year"] = year;
        args["month"] = month;
        args["day"] = day;
        args["hour"] = hour;
        args["minute"] = minute;
        this->calls.push_back({{"fn", "SetAIChanCalCalDate"}, {"args", args}});
        return 0;
    }

    int32 SetAIChanCalExpDate(
        TaskHandle task,
        const char channelName[],
        uInt32 year,
        uInt32 month,
        uInt32 day,
        uInt32 hour,
        uInt32 minute
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channelName != nullptr) args["channelName"] = std::string(channelName);
        args["year"] = year;
        args["month"] = month;
        args["day"] = day;
        args["hour"] = hour;
        args["minute"] = minute;
        this->calls.push_back({{"fn", "SetAIChanCalExpDate"}, {"args", args}});
        return 0;
    }

    int32 SetAnalogPowerUpStatesWithOutputType(
        const char channelNames[],
        const float64 stateArray[],
        const int32 channelTypeArray[],
        uInt32 arraySize
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channelNames != nullptr) args["channelNames"] = std::string(channelNames);
        if (stateArray != nullptr) args["stateArray"] = "<array>";
        if (channelTypeArray != nullptr) args["channelTypeArray"] = "<array>";
        args["arraySize"] = arraySize;
        this->calls.push_back(
            {{"fn", "SetAnalogPowerUpStatesWithOutputType"}, {"args", args}}
        );
        return 0;
    }

    int32 SetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back({{"fn", "SetArmStartTrigTrigWhen"}, {"args", args}});
        return 0;
    }

    int32
    SetBufferAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetBufferAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32 SetCalInfoAttributeBool(
        const char deviceName[],
        int32 attribute,
        bool32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetCalInfoAttributeBool"}, {"args", args}});
        return 0;
    }

    int32 SetCalInfoAttributeDouble(
        const char deviceName[],
        int32 attribute,
        float64 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetCalInfoAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 SetCalInfoAttributeString(
        const char deviceName[],
        int32 attribute,
        const char value[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        this->calls.push_back({{"fn", "SetCalInfoAttributeString"}, {"args", args}});
        return 0;
    }

    int32 SetCalInfoAttributeUInt32(
        const char deviceName[],
        int32 attribute,
        uInt32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetCalInfoAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32 SetChanAttributeBool(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        bool32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetChanAttributeBool"}, {"args", args}});
        return 0;
    }

    int32 SetChanAttributeDouble(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetChanAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 SetChanAttributeDoubleArray(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        const float64 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back({{"fn", "SetChanAttributeDoubleArray"}, {"args", args}});
        return 0;
    }

    int32 SetChanAttributeInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        int32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetChanAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 SetChanAttributeString(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        const char value[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        this->calls.push_back({{"fn", "SetChanAttributeString"}, {"args", args}});
        return 0;
    }

    int32 SetChanAttributeUInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        uInt32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (channel != nullptr) args["channel"] = std::string(channel);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetChanAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32 SetDigitalLogicFamilyPowerUpState(
        const char deviceName[],
        int32 logicFamily
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        args["logicFamily"] = logicFamily;
        this->calls.push_back(
            {{"fn", "SetDigitalLogicFamilyPowerUpState"}, {"args", args}}
        );
        return 0;
    }

    int32 SetExportedSignalAttributeBool(
        TaskHandle task,
        int32 attribute,
        bool32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back(
            {{"fn", "SetExportedSignalAttributeBool"}, {"args", args}}
        );
        return 0;
    }

    int32 SetExportedSignalAttributeDouble(
        TaskHandle task,
        int32 attribute,
        float64 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back(
            {{"fn", "SetExportedSignalAttributeDouble"}, {"args", args}}
        );
        return 0;
    }

    int32 SetExportedSignalAttributeInt32(
        TaskHandle task,
        int32 attribute,
        int32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back(
            {{"fn", "SetExportedSignalAttributeInt32"}, {"args", args}}
        );
        return 0;
    }

    int32 SetExportedSignalAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        this->calls.push_back(
            {{"fn", "SetExportedSignalAttributeString"}, {"args", args}}
        );
        return 0;
    }

    int32 SetExportedSignalAttributeUInt32(
        TaskHandle task,
        int32 attribute,
        uInt32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back(
            {{"fn", "SetExportedSignalAttributeUInt32"}, {"args", args}}
        );
        return 0;
    }

    int32 SetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back({{"fn", "SetFirstSampClkWhen"}, {"args", args}});
        return 0;
    }

    int32
    SetReadAttributeBool(TaskHandle task, int32 attribute, bool32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetReadAttributeBool"}, {"args", args}});
        return 0;
    }

    int32
    SetReadAttributeDouble(TaskHandle task, int32 attribute, float64 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetReadAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32
    SetReadAttributeInt32(TaskHandle task, int32 attribute, int32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetReadAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 SetReadAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        this->calls.push_back({{"fn", "SetReadAttributeString"}, {"args", args}});
        return 0;
    }

    int32
    SetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetReadAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32
    SetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetReadAttributeUInt64"}, {"args", args}});
        return 0;
    }

    int32
    SetRealTimeAttributeBool(TaskHandle task, int32 attribute, bool32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetRealTimeAttributeBool"}, {"args", args}});
        return 0;
    }

    int32
    SetRealTimeAttributeInt32(TaskHandle task, int32 attribute, int32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetRealTimeAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 SetRealTimeAttributeUInt32(
        TaskHandle task,
        int32 attribute,
        uInt32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetRealTimeAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32 SetRuntimeEnvironment(
        const char environment[],
        const char environmentVersion[],
        const char reserved1[],
        const char reserved2[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (environment != nullptr) args["environment"] = std::string(environment);
        if (environmentVersion != nullptr)
            args["environmentVersion"] = std::string(environmentVersion);
        if (reserved1 != nullptr) args["reserved1"] = std::string(reserved1);
        if (reserved2 != nullptr) args["reserved2"] = std::string(reserved2);
        this->calls.push_back({{"fn", "SetRuntimeEnvironment"}, {"args", args}});
        return 0;
    }

    int32 SetScaleAttributeDouble(
        const char scaleName[],
        int32 attribute,
        float64 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (scaleName != nullptr) args["scaleName"] = std::string(scaleName);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetScaleAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 SetScaleAttributeDoubleArray(
        const char scaleName[],
        int32 attribute,
        const float64 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (scaleName != nullptr) args["scaleName"] = std::string(scaleName);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back({{"fn", "SetScaleAttributeDoubleArray"}, {"args", args}});
        return 0;
    }

    int32 SetScaleAttributeInt32(
        const char scaleName[],
        int32 attribute,
        int32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (scaleName != nullptr) args["scaleName"] = std::string(scaleName);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetScaleAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 SetScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        const char value[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (scaleName != nullptr) args["scaleName"] = std::string(scaleName);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        this->calls.push_back({{"fn", "SetScaleAttributeString"}, {"args", args}});
        return 0;
    }

    int32 SetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back({{"fn", "SetStartTrigTrigWhen"}, {"args", args}});
        return 0;
    }

    int32 SetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back({{"fn", "SetSyncPulseTimeWhen"}, {"args", args}});
        return 0;
    }

    int32
    SetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTimingAttributeBool"}, {"args", args}});
        return 0;
    }

    int32
    SetTimingAttributeDouble(TaskHandle task, int32 attribute, float64 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTimingAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 SetTimingAttributeExBool(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        bool32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTimingAttributeExBool"}, {"args", args}});
        return 0;
    }

    int32 SetTimingAttributeExDouble(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        float64 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTimingAttributeExDouble"}, {"args", args}});
        return 0;
    }

    int32 SetTimingAttributeExInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        int32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTimingAttributeExInt32"}, {"args", args}});
        return 0;
    }

    int32 SetTimingAttributeExString(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        const char value[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        this->calls.push_back({{"fn", "SetTimingAttributeExString"}, {"args", args}});
        return 0;
    }

    int32 SetTimingAttributeExTimestamp(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        CVIAbsoluteTime value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "SetTimingAttributeExTimestamp"}, {"args", args}}
        );
        return 0;
    }

    int32 SetTimingAttributeExUInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTimingAttributeExUInt32"}, {"args", args}});
        return 0;
    }

    int32 SetTimingAttributeExUInt64(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt64 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceNames != nullptr) args["deviceNames"] = std::string(deviceNames);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTimingAttributeExUInt64"}, {"args", args}});
        return 0;
    }

    int32
    SetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTimingAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 SetTimingAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        this->calls.push_back({{"fn", "SetTimingAttributeString"}, {"args", args}});
        return 0;
    }

    int32 SetTimingAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "SetTimingAttributeTimestamp"}, {"args", args}});
        return 0;
    }

    int32
    SetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTimingAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32
    SetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTimingAttributeUInt64"}, {"args", args}});
        return 0;
    }

    int32
    SetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTrigAttributeBool"}, {"args", args}});
        return 0;
    }

    int32
    SetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTrigAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 SetTrigAttributeDoubleArray(
        TaskHandle task,
        int32 attribute,
        const float64 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back({{"fn", "SetTrigAttributeDoubleArray"}, {"args", args}});
        return 0;
    }

    int32
    SetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTrigAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 SetTrigAttributeInt32Array(
        TaskHandle task,
        int32 attribute,
        const int32 value[],
        uInt32 size
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = "<array>";
        args["size"] = size;
        this->calls.push_back({{"fn", "SetTrigAttributeInt32Array"}, {"args", args}});
        return 0;
    }

    int32 SetTrigAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        this->calls.push_back({{"fn", "SetTrigAttributeString"}, {"args", args}});
        return 0;
    }

    int32 SetTrigAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = "<ptr>";
        this->calls.push_back({{"fn", "SetTrigAttributeTimestamp"}, {"args", args}});
        return 0;
    }

    int32
    SetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetTrigAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32 SetWatchdogAttributeBool(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        bool32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (lines != nullptr) args["lines"] = std::string(lines);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetWatchdogAttributeBool"}, {"args", args}});
        return 0;
    }

    int32 SetWatchdogAttributeDouble(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        float64 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (lines != nullptr) args["lines"] = std::string(lines);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetWatchdogAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32 SetWatchdogAttributeInt32(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        int32 value
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (lines != nullptr) args["lines"] = std::string(lines);
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetWatchdogAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 SetWatchdogAttributeString(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        const char value[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (lines != nullptr) args["lines"] = std::string(lines);
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        this->calls.push_back({{"fn", "SetWatchdogAttributeString"}, {"args", args}});
        return 0;
    }

    int32
    SetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetWriteAttributeBool"}, {"args", args}});
        return 0;
    }

    int32
    SetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetWriteAttributeDouble"}, {"args", args}});
        return 0;
    }

    int32
    SetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetWriteAttributeInt32"}, {"args", args}});
        return 0;
    }

    int32 SetWriteAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        if (value != nullptr) args["value"] = std::string(value);
        this->calls.push_back({{"fn", "SetWriteAttributeString"}, {"args", args}});
        return 0;
    }

    int32
    SetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetWriteAttributeUInt32"}, {"args", args}});
        return 0;
    }

    int32
    SetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) override {
        nlohmann::json args = nlohmann::json::object();
        args["attribute"] = attribute;
        args["value"] = value;
        this->calls.push_back({{"fn", "SetWriteAttributeUInt64"}, {"args", args}});
        return 0;
    }

    int32 StartNewFile(TaskHandle task, const char filePath[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (filePath != nullptr) args["filePath"] = std::string(filePath);
        this->calls.push_back({{"fn", "StartNewFile"}, {"args", args}});
        return 0;
    }

    int32 StartTask(TaskHandle task) override {
        nlohmann::json args = nlohmann::json::object();

        this->calls.push_back({{"fn", "StartTask"}, {"args", args}});
        return 0;
    }

    int32 StopTask(TaskHandle task) override {
        nlohmann::json args = nlohmann::json::object();

        this->calls.push_back({{"fn", "StopTask"}, {"args", args}});
        return 0;
    }

    int32 TaskControl(TaskHandle task, int32 action) override {
        nlohmann::json args = nlohmann::json::object();
        args["action"] = action;
        this->calls.push_back({{"fn", "TaskControl"}, {"args", args}});
        return 0;
    }

    int32 TristateOutputTerm(const char outputTerminal[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (outputTerminal != nullptr)
            args["outputTerminal"] = std::string(outputTerminal);
        this->calls.push_back({{"fn", "TristateOutputTerm"}, {"args", args}});
        return 0;
    }

    int32 UnregisterDoneEvent(
        TaskHandle task,
        uInt32 options,
        DAQmxDoneEventCallbackPtr callbackFunction,
        void *callbackData
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["options"] = options;
        args["callbackFunction"] = "<ptr>";
        args["callbackData"] = "<ptr>";
        this->calls.push_back({{"fn", "UnregisterDoneEvent"}, {"args", args}});
        return 0;
    }

    int32 UnregisterEveryNSamplesEvent(
        TaskHandle task,
        int32 everyNSamplesEventType,
        uInt32 nSamples,
        uInt32 options,
        DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
        void *callbackData
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["everyNSamplesEventType"] = everyNSamplesEventType;
        args["nSamples"] = nSamples;
        args["options"] = options;
        args["callbackFunction"] = "<ptr>";
        args["callbackData"] = "<ptr>";
        this->calls.push_back({{"fn", "UnregisterEveryNSamplesEvent"}, {"args", args}});
        return 0;
    }

    int32 UnregisterSignalEvent(
        TaskHandle task,
        int32 signalID,
        uInt32 options,
        DAQmxSignalEventCallbackPtr callbackFunction,
        void *callbackData
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["signalID"] = signalID;
        args["options"] = options;
        args["callbackFunction"] = "<ptr>";
        args["callbackData"] = "<ptr>";
        this->calls.push_back({{"fn", "UnregisterSignalEvent"}, {"args", args}});
        return 0;
    }

    int32 UnreserveNetworkDevice(const char deviceName[]) override {
        nlohmann::json args = nlohmann::json::object();
        if (deviceName != nullptr) args["deviceName"] = std::string(deviceName);
        this->calls.push_back({{"fn", "UnreserveNetworkDevice"}, {"args", args}});
        return 0;
    }

    int32
    WaitForNextSampleClock(TaskHandle task, float64 timeout, bool32 *isLate) override {
        nlohmann::json args = nlohmann::json::object();
        args["timeout"] = timeout;
        args["isLate"] = "<ptr>";
        this->calls.push_back({{"fn", "WaitForNextSampleClock"}, {"args", args}});
        return 0;
    }

    int32 WaitForValidTimestamp(
        TaskHandle task,
        int32 timestampEvent,
        float64 timeout,
        CVIAbsoluteTime *timestamp
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["timestampEvent"] = timestampEvent;
        args["timeout"] = timeout;
        args["timestamp"] = "<ptr>";
        this->calls.push_back({{"fn", "WaitForValidTimestamp"}, {"args", args}});
        return 0;
    }

    int32 WaitUntilTaskDone(TaskHandle task, float64 timeToWait) override {
        nlohmann::json args = nlohmann::json::object();
        args["timeToWait"] = timeToWait;
        this->calls.push_back({{"fn", "WaitUntilTaskDone"}, {"args", args}});
        return 0;
    }

    int32 WriteAnalogF64(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["dataLayout"] = dataLayout;
        if (writeArray != nullptr) args["writeArray"] = "<array>";
        args["sampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteAnalogF64"}, {"args", args}});
        return 0;
    }

    int32 WriteAnalogScalarF64(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 value,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["value"] = value;
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteAnalogScalarF64"}, {"args", args}});
        return 0;
    }

    int32 WriteBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const int16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["dataLayout"] = dataLayout;
        if (writeArray != nullptr) args["writeArray"] = "<array>";
        args["sampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteBinaryI16"}, {"args", args}});
        return 0;
    }

    int32 WriteBinaryI32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const int32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["dataLayout"] = dataLayout;
        if (writeArray != nullptr) args["writeArray"] = "<array>";
        args["sampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteBinaryI32"}, {"args", args}});
        return 0;
    }

    int32 WriteBinaryU16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["dataLayout"] = dataLayout;
        if (writeArray != nullptr) args["writeArray"] = "<array>";
        args["sampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteBinaryU16"}, {"args", args}});
        return 0;
    }

    int32 WriteBinaryU32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["dataLayout"] = dataLayout;
        if (writeArray != nullptr) args["writeArray"] = "<array>";
        args["sampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteBinaryU32"}, {"args", args}});
        return 0;
    }

    int32 WriteCtrFreq(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 frequency[],
        const float64 dutyCycle[],
        int32 *numSampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["dataLayout"] = dataLayout;
        if (frequency != nullptr) args["frequency"] = "<array>";
        if (dutyCycle != nullptr) args["dutyCycle"] = "<array>";
        args["numSampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteCtrFreq"}, {"args", args}});
        return 0;
    }

    int32 WriteCtrFreqScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 frequency,
        float64 dutyCycle,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["frequency"] = frequency;
        args["dutyCycle"] = dutyCycle;
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteCtrFreqScalar"}, {"args", args}});
        return 0;
    }

    int32 WriteCtrTicks(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 highTicks[],
        const uInt32 lowTicks[],
        int32 *numSampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["dataLayout"] = dataLayout;
        if (highTicks != nullptr) args["highTicks"] = "<array>";
        if (lowTicks != nullptr) args["lowTicks"] = "<array>";
        args["numSampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteCtrTicks"}, {"args", args}});
        return 0;
    }

    int32 WriteCtrTicksScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        uInt32 highTicks,
        uInt32 lowTicks,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["highTicks"] = highTicks;
        args["lowTicks"] = lowTicks;
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteCtrTicksScalar"}, {"args", args}});
        return 0;
    }

    int32 WriteCtrTime(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 highTime[],
        const float64 lowTime[],
        int32 *numSampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["dataLayout"] = dataLayout;
        if (highTime != nullptr) args["highTime"] = "<array>";
        if (lowTime != nullptr) args["lowTime"] = "<array>";
        args["numSampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteCtrTime"}, {"args", args}});
        return 0;
    }

    int32 WriteCtrTimeScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 highTime,
        float64 lowTime,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["highTime"] = highTime;
        args["lowTime"] = lowTime;
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteCtrTimeScalar"}, {"args", args}});
        return 0;
    }

    int32 WriteDigitalLines(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["dataLayout"] = dataLayout;
        if (writeArray != nullptr) args["writeArray"] = "<array>";
        args["sampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteDigitalLines"}, {"args", args}});
        return 0;
    }

    int32 WriteDigitalScalarU32(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        uInt32 value,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["value"] = value;
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteDigitalScalarU32"}, {"args", args}});
        return 0;
    }

    int32 WriteDigitalU16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["dataLayout"] = dataLayout;
        if (writeArray != nullptr) args["writeArray"] = "<array>";
        args["sampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteDigitalU16"}, {"args", args}});
        return 0;
    }

    int32 WriteDigitalU32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["dataLayout"] = dataLayout;
        if (writeArray != nullptr) args["writeArray"] = "<array>";
        args["sampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteDigitalU32"}, {"args", args}});
        return 0;
    }

    int32 WriteDigitalU8(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSampsPerChan"] = numSampsPerChan;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        args["dataLayout"] = dataLayout;
        if (writeArray != nullptr) args["writeArray"] = "<array>";
        args["sampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteDigitalU8"}, {"args", args}});
        return 0;
    }

    int32 WriteRaw(
        TaskHandle task,
        int32 numSamps,
        bool32 autoStart,
        float64 timeout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override {
        nlohmann::json args = nlohmann::json::object();
        args["numSamps"] = numSamps;
        args["autoStart"] = autoStart;
        args["timeout"] = timeout;
        if (writeArray != nullptr) args["writeArray"] = "<array>";
        args["sampsPerChanWritten"] = "<ptr>";
        args["reserved"] = "<ptr>";
        this->calls.push_back({{"fn", "WriteRaw"}, {"args", args}});
        return 0;
    }

    int32 WriteToTEDSFromArray(
        const char physicalChannel[],
        const uInt8 bitStream[],
        uInt32 arraySize,
        int32 basicTEDSOptions
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (bitStream != nullptr) args["bitStream"] = "<array>";
        args["arraySize"] = arraySize;
        args["basicTEDSOptions"] = basicTEDSOptions;
        this->calls.push_back({{"fn", "WriteToTEDSFromArray"}, {"args", args}});
        return 0;
    }

    int32 WriteToTEDSFromFile(
        const char physicalChannel[],
        const char filePath[],
        int32 basicTEDSOptions
    ) override {
        nlohmann::json args = nlohmann::json::object();
        if (physicalChannel != nullptr)
            args["physicalChannel"] = std::string(physicalChannel);
        if (filePath != nullptr) args["filePath"] = std::string(filePath);
        args["basicTEDSOptions"] = basicTEDSOptions;
        this->calls.push_back({{"fn", "WriteToTEDSFromFile"}, {"args", args}});
        return 0;
    }

    int32 SetReadRelativeTo(TaskHandle taskHandle, int32 data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = data;
        this->calls.push_back({{"fn", "SetReadRelativeTo"}, {"args", args}});
        return 0;
    }

    int32 SetReadOffset(TaskHandle taskHandle, int32 data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = data;
        this->calls.push_back({{"fn", "SetReadOffset"}, {"args", args}});
        return 0;
    }

    int32 SetReadOverWrite(TaskHandle taskHandle, int32 data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = data;
        this->calls.push_back({{"fn", "SetReadOverWrite"}, {"args", args}});
        return 0;
    }

    int32
    GetReadTotalSampPerChanAcquired(TaskHandle taskHandle, uInt64 *data) override {
        nlohmann::json args = nlohmann::json::object();
        args["data"] = "<ptr>";
        this->calls.push_back(
            {{"fn", "GetReadTotalSampPerChanAcquired"}, {"args", args}}
        );
        return 0;
    }
};
}
