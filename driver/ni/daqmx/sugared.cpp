// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ni/daqmx/sugared.h"

xerrors::Error SugaredDAQmx::process_error(int32 status) {

}

xerrors::Error SugaredDAQmx::AddCDAQSyncConnection(const char portList[]) {
    return process_error(dmx->AddCDAQSyncConnection(portList));
}

xerrors::Error SugaredDAQmx::AddGlobalChansToTask(TaskHandle task, const char channelNames[]) {
    return process_error(dmx->AddGlobalChansToTask(task, channelNames));
}

xerrors::Error SugaredDAQmx::AddNetworkDevice(const char ipAddress[], const char deviceName[], bool32 attemptReservation, float64 timeout, char deviceNameOut[], uInt32 deviceNameOutBufferSize) {
    return process_error(dmx->AddNetworkDevice(ipAddress, deviceName, attemptReservation, timeout, deviceNameOut, deviceNameOutBufferSize));
}

xerrors::Error SugaredDAQmx::AreConfiguredCDAQSyncPortsDisconnected(const char chassisDevicesPorts[], float64 timeout, bool32* disconnectedPortsExist) {
    return process_error(dmx->AreConfiguredCDAQSyncPortsDisconnected(chassisDevicesPorts, timeout, disconnectedPortsExist));
}

xerrors::Error SugaredDAQmx::AutoConfigureCDAQSyncConnections(const char chassisDevicesPorts[], float64 timeout) {
    return process_error(dmx->AutoConfigureCDAQSyncConnections(chassisDevicesPorts, timeout));
}

xerrors::Error SugaredDAQmx::CalculateReversePolyCoeff(const float64 forwardCoeffs[], uInt32 numForwardCoeffsIn, float64 minValX, float64 maxValX, int32 numPointsToCompute, int32 reversePolyOrder, float64 reverseCoeffs[]) {
    return process_error(dmx->CalculateReversePolyCoeff(forwardCoeffs, numForwardCoeffsIn, minValX, maxValX, numPointsToCompute, reversePolyOrder, reverseCoeffs));
}

xerrors::Error SugaredDAQmx::CfgAnlgEdgeRefTrig(TaskHandle task, const char triggerSource[], int32 triggerSlope, float64 triggerLevel, uInt32 pretriggerSamples) {
    return process_error(dmx->CfgAnlgEdgeRefTrig(task, triggerSource, triggerSlope, triggerLevel, pretriggerSamples));
}

xerrors::Error SugaredDAQmx::CfgAnlgEdgeStartTrig(TaskHandle task, const char triggerSource[], int32 triggerSlope, float64 triggerLevel) {
    return process_error(dmx->CfgAnlgEdgeStartTrig(task, triggerSource, triggerSlope, triggerLevel));
}

xerrors::Error SugaredDAQmx::CfgAnlgMultiEdgeRefTrig(TaskHandle task, const char triggerSources[], const int32 triggerSlopeArray[], const float64 triggerLevelArray[], uInt32 pretriggerSamples, uInt32 arraySize) {
    return process_error(dmx->CfgAnlgMultiEdgeRefTrig(task, triggerSources, triggerSlopeArray, triggerLevelArray, pretriggerSamples, arraySize));
}

xerrors::Error SugaredDAQmx::CfgAnlgMultiEdgeStartTrig(TaskHandle task, const char triggerSources[], const int32 triggerSlopeArray[], const float64 triggerLevelArray[], uInt32 arraySize) {
    return process_error(dmx->CfgAnlgMultiEdgeStartTrig(task, triggerSources, triggerSlopeArray, triggerLevelArray, arraySize));
}

xerrors::Error SugaredDAQmx::CfgAnlgWindowRefTrig(TaskHandle task, const char triggerSource[], int32 triggerWhen, float64 windowTop, float64 windowBottom, uInt32 pretriggerSamples) {
    return process_error(dmx->CfgAnlgWindowRefTrig(task, triggerSource, triggerWhen, windowTop, windowBottom, pretriggerSamples));
}

xerrors::Error SugaredDAQmx::CfgAnlgWindowStartTrig(TaskHandle task, const char triggerSource[], int32 triggerWhen, float64 windowTop, float64 windowBottom) {
    return process_error(dmx->CfgAnlgWindowStartTrig(task, triggerSource, triggerWhen, windowTop, windowBottom));
}

xerrors::Error SugaredDAQmx::CfgBurstHandshakingTimingExportClock(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan, float64 sampleClkRate, const char sampleClkOutpTerm[], int32 sampleClkPulsePolarity, int32 pauseWhen, int32 readyEventActiveLevel) {
    return process_error(dmx->CfgBurstHandshakingTimingExportClock(task, sampleMode, sampsPerChan, sampleClkRate, sampleClkOutpTerm, sampleClkPulsePolarity, pauseWhen, readyEventActiveLevel));
}

xerrors::Error SugaredDAQmx::CfgBurstHandshakingTimingImportClock(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan, float64 sampleClkRate, const char sampleClkSrc[], int32 sampleClkActiveEdge, int32 pauseWhen, int32 readyEventActiveLevel) {
    return process_error(dmx->CfgBurstHandshakingTimingImportClock(task, sampleMode, sampsPerChan, sampleClkRate, sampleClkSrc, sampleClkActiveEdge, pauseWhen, readyEventActiveLevel));
}

xerrors::Error SugaredDAQmx::CfgChangeDetectionTiming(TaskHandle task, const char risingEdgeChan[], const char fallingEdgeChan[], int32 sampleMode, uInt64 sampsPerChan) {
    return process_error(dmx->CfgChangeDetectionTiming(task, risingEdgeChan, fallingEdgeChan, sampleMode, sampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgDigEdgeRefTrig(TaskHandle task, const char triggerSource[], int32 triggerEdge, uInt32 pretriggerSamples) {
    return process_error(dmx->CfgDigEdgeRefTrig(task, triggerSource, triggerEdge, pretriggerSamples));
}

xerrors::Error SugaredDAQmx::CfgDigEdgeStartTrig(TaskHandle task, const char triggerSource[], int32 triggerEdge) {
    return process_error(dmx->CfgDigEdgeStartTrig(task, triggerSource, triggerEdge));
}

xerrors::Error SugaredDAQmx::CfgDigPatternRefTrig(TaskHandle task, const char triggerSource[], const char triggerPattern[], int32 triggerWhen, uInt32 pretriggerSamples) {
    return process_error(dmx->CfgDigPatternRefTrig(task, triggerSource, triggerPattern, triggerWhen, pretriggerSamples));
}

xerrors::Error SugaredDAQmx::CfgDigPatternStartTrig(TaskHandle task, const char triggerSource[], const char triggerPattern[], int32 triggerWhen) {
    return process_error(dmx->CfgDigPatternStartTrig(task, triggerSource, triggerPattern, triggerWhen));
}

xerrors::Error SugaredDAQmx::CfgHandshakingTiming(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan) {
    return process_error(dmx->CfgHandshakingTiming(task, sampleMode, sampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgImplicitTiming(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan) {
    return process_error(dmx->CfgImplicitTiming(task, sampleMode, sampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgInputBuffer(TaskHandle task, uInt32 numSampsPerChan) {
    return process_error(dmx->CfgInputBuffer(task, numSampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgOutputBuffer(TaskHandle task, uInt32 numSampsPerChan) {
    return process_error(dmx->CfgOutputBuffer(task, numSampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgPipelinedSampClkTiming(TaskHandle task, const char source[], float64 rate, int32 activeEdge, int32 sampleMode, uInt64 sampsPerChan) {
    return process_error(dmx->CfgPipelinedSampClkTiming(task, source, rate, activeEdge, sampleMode, sampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgSampClkTiming(TaskHandle task, const char source[], float64 rate, int32 activeEdge, int32 sampleMode, uInt64 sampsPerChan) {
    return process_error(dmx->CfgSampClkTiming(task, source, rate, activeEdge, sampleMode, sampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgTimeStartTrig(TaskHandle task, CVIAbsoluteTime when, int32 timescale) {
    return process_error(dmx->CfgTimeStartTrig(task, when, timescale));
}

xerrors::Error SugaredDAQmx::CfgWatchdogAOExpirStates(TaskHandle task, const char channelNames[], const float64 expirStateArray[], const int32 outputTypeArray[], uInt32 arraySize) {
    return process_error(dmx->CfgWatchdogAOExpirStates(task, channelNames, expirStateArray, outputTypeArray, arraySize));
}

xerrors::Error SugaredDAQmx::CfgWatchdogCOExpirStates(TaskHandle task, const char channelNames[], const int32 expirStateArray[], uInt32 arraySize) {
    return process_error(dmx->CfgWatchdogCOExpirStates(task, channelNames, expirStateArray, arraySize));
}

xerrors::Error SugaredDAQmx::CfgWatchdogDOExpirStates(TaskHandle task, const char channelNames[], const int32 expirStateArray[], uInt32 arraySize) {
    return process_error(dmx->CfgWatchdogDOExpirStates(task, channelNames, expirStateArray, arraySize));
}

xerrors::Error SugaredDAQmx::ClearTEDS(const char physicalChannel[]) {
    return process_error(dmx->ClearTEDS(physicalChannel));
}

xerrors::Error SugaredDAQmx::ClearTask(TaskHandle task) {
    return process_error(dmx->ClearTask(task));
}

xerrors::Error SugaredDAQmx::ConfigureLogging(TaskHandle task, const char filePath[], int32 loggingMode, const char groupName[], int32 operation) {
    return process_error(dmx->ConfigureLogging(task, filePath, loggingMode, groupName, operation));
}

xerrors::Error SugaredDAQmx::ConfigureTEDS(const char physicalChannel[], const char filePath[]) {
    return process_error(dmx->ConfigureTEDS(physicalChannel, filePath));
}

xerrors::Error SugaredDAQmx::ConnectTerms(const char sourceTerminal[], const char destinationTerminal[], int32 signalModifiers) {
    return process_error(dmx->ConnectTerms(sourceTerminal, destinationTerminal, signalModifiers));
}

xerrors::Error SugaredDAQmx::ControlWatchdogTask(TaskHandle task, int32 action) {
    return process_error(dmx->ControlWatchdogTask(task, action));
}

xerrors::Error SugaredDAQmx::CreateAIAccel4WireDCVoltageChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 terminalConfig, float64 minVal, float64 maxVal, int32 units, float64 sensitivity, int32 sensitivityUnits, int32 voltageExcitSource, float64 voltageExcitVal, bool32 useExcitForScaling, const char customScaleName[]) {
    return process_error(dmx->CreateAIAccel4WireDCVoltageChan(task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal, units, sensitivity, sensitivityUnits, voltageExcitSource, voltageExcitVal, useExcitForScaling, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIAccelChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 terminalConfig, float64 minVal, float64 maxVal, int32 units, float64 sensitivity, int32 sensitivityUnits, int32 currentExcitSource, float64 currentExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateAIAccelChan(task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal, units, sensitivity, sensitivityUnits, currentExcitSource, currentExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIAccelChargeChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 terminalConfig, float64 minVal, float64 maxVal, int32 units, float64 sensitivity, int32 sensitivityUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAIAccelChargeChan(task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal, units, sensitivity, sensitivityUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIBridgeChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig, int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance, const char customScaleName[]) {
    return process_error(dmx->CreateAIBridgeChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIChargeChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 terminalConfig, float64 minVal, float64 maxVal, int32 units, const char customScaleName[]) {
    return process_error(dmx->CreateAIChargeChan(task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal, units, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAICurrentChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 terminalConfig, float64 minVal, float64 maxVal, int32 units, int32 shuntResistorLoc, float64 extShuntResistorVal, const char customScaleName[]) {
    return process_error(dmx->CreateAICurrentChan(task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal, units, shuntResistorLoc, extShuntResistorVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAICurrentRMSChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 terminalConfig, float64 minVal, float64 maxVal, int32 units, int32 shuntResistorLoc, float64 extShuntResistorVal, const char customScaleName[]) {
    return process_error(dmx->CreateAICurrentRMSChan(task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal, units, shuntResistorLoc, extShuntResistorVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIForceBridgePolynomialChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig, int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance, const float64 forwardCoeffs[], uInt32 numForwardCoeffs, const float64 reverseCoeffs[], uInt32 numReverseCoeffs, int32 electricalUnits, int32 physicalUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAIForceBridgePolynomialChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance, forwardCoeffs, numForwardCoeffs, reverseCoeffs, numReverseCoeffs, electricalUnits, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIForceBridgeTableChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig, int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance, const float64 electricalVals[], uInt32 numElectricalVals, int32 electricalUnits, const float64 physicalVals[], uInt32 numPhysicalVals, int32 physicalUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAIForceBridgeTableChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance, electricalVals, numElectricalVals, electricalUnits, physicalVals, numPhysicalVals, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIForceBridgeTwoPointLinChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig, int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance, float64 firstElectricalVal, float64 secondElectricalVal, int32 electricalUnits, float64 firstPhysicalVal, float64 secondPhysicalVal, int32 physicalUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAIForceBridgeTwoPointLinChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance, firstElectricalVal, secondElectricalVal, electricalUnits, firstPhysicalVal, secondPhysicalVal, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIForceIEPEChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 terminalConfig, float64 minVal, float64 maxVal, int32 units, float64 sensitivity, int32 sensitivityUnits, int32 currentExcitSource, float64 currentExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateAIForceIEPEChan(task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal, units, sensitivity, sensitivityUnits, currentExcitSource, currentExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIFreqVoltageChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, float64 thresholdLevel, float64 hysteresis, const char customScaleName[]) {
    return process_error(dmx->CreateAIFreqVoltageChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, thresholdLevel, hysteresis, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIMicrophoneChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 terminalConfig, int32 units, float64 micSensitivity, float64 maxSndPressLevel, int32 currentExcitSource, float64 currentExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateAIMicrophoneChan(task, physicalChannel, nameToAssignToChannel, terminalConfig, units, micSensitivity, maxSndPressLevel, currentExcitSource, currentExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIPosEddyCurrProxProbeChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, float64 sensitivity, int32 sensitivityUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAIPosEddyCurrProxProbeChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, sensitivity, sensitivityUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIPosLVDTChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, float64 sensitivity, int32 sensitivityUnits, int32 voltageExcitSource, float64 voltageExcitVal, float64 voltageExcitFreq, int32 acExcitWireMode, const char customScaleName[]) {
    return process_error(dmx->CreateAIPosLVDTChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, sensitivity, sensitivityUnits, voltageExcitSource, voltageExcitVal, voltageExcitFreq, acExcitWireMode, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIPosRVDTChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, float64 sensitivity, int32 sensitivityUnits, int32 voltageExcitSource, float64 voltageExcitVal, float64 voltageExcitFreq, int32 acExcitWireMode, const char customScaleName[]) {
    return process_error(dmx->CreateAIPosRVDTChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, sensitivity, sensitivityUnits, voltageExcitSource, voltageExcitVal, voltageExcitFreq, acExcitWireMode, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIPowerChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 voltageSetpoint, float64 currentSetpoint, bool32 outputEnable) {
    return process_error(dmx->CreateAIPowerChan(task, physicalChannel, nameToAssignToChannel, voltageSetpoint, currentSetpoint, outputEnable));
}

xerrors::Error SugaredDAQmx::CreateAIPressureBridgePolynomialChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig, int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance, const float64 forwardCoeffs[], uInt32 numForwardCoeffs, const float64 reverseCoeffs[], uInt32 numReverseCoeffs, int32 electricalUnits, int32 physicalUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAIPressureBridgePolynomialChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance, forwardCoeffs, numForwardCoeffs, reverseCoeffs, numReverseCoeffs, electricalUnits, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIPressureBridgeTableChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig, int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance, const float64 electricalVals[], uInt32 numElectricalVals, int32 electricalUnits, const float64 physicalVals[], uInt32 numPhysicalVals, int32 physicalUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAIPressureBridgeTableChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance, electricalVals, numElectricalVals, electricalUnits, physicalVals, numPhysicalVals, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIPressureBridgeTwoPointLinChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig, int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance, float64 firstElectricalVal, float64 secondElectricalVal, int32 electricalUnits, float64 firstPhysicalVal, float64 secondPhysicalVal, int32 physicalUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAIPressureBridgeTwoPointLinChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance, firstElectricalVal, secondElectricalVal, electricalUnits, firstPhysicalVal, secondPhysicalVal, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIRTDChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 rtdType, int32 resistanceConfig, int32 currentExcitSource, float64 currentExcitVal, float64 r0) {
    return process_error(dmx->CreateAIRTDChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, rtdType, resistanceConfig, currentExcitSource, currentExcitVal, r0));
}

xerrors::Error SugaredDAQmx::CreateAIResistanceChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 resistanceConfig, int32 currentExcitSource, float64 currentExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateAIResistanceChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, resistanceConfig, currentExcitSource, currentExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIRosetteStrainGageChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 rosetteType, float64 gageOrientation, const int32 rosetteMeasTypes[], uInt32 numRosetteMeasTypes, int32 strainConfig, int32 voltageExcitSource, float64 voltageExcitVal, float64 gageFactor, float64 nominalGageResistance, float64 poissonRatio, float64 leadWireResistance) {
    return process_error(dmx->CreateAIRosetteStrainGageChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, rosetteType, gageOrientation, rosetteMeasTypes, numRosetteMeasTypes, strainConfig, voltageExcitSource, voltageExcitVal, gageFactor, nominalGageResistance, poissonRatio, leadWireResistance));
}

xerrors::Error SugaredDAQmx::CreateAIStrainGageChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 strainConfig, int32 voltageExcitSource, float64 voltageExcitVal, float64 gageFactor, float64 initialBridgeVoltage, float64 nominalGageResistance, float64 poissonRatio, float64 leadWireResistance, const char customScaleName[]) {
    return process_error(dmx->CreateAIStrainGageChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, strainConfig, voltageExcitSource, voltageExcitVal, gageFactor, initialBridgeVoltage, nominalGageResistance, poissonRatio, leadWireResistance, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAITempBuiltInSensorChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 units) {
    return process_error(dmx->CreateAITempBuiltInSensorChan(task, physicalChannel, nameToAssignToChannel, units));
}

xerrors::Error SugaredDAQmx::CreateAIThrmcplChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 thermocoupleType, int32 cjcSource, float64 cjcVal, const char cjcChannel[]) {
    return process_error(dmx->CreateAIThrmcplChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, thermocoupleType, cjcSource, cjcVal, cjcChannel));
}

xerrors::Error SugaredDAQmx::CreateAIThrmstrChanIex(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 resistanceConfig, int32 currentExcitSource, float64 currentExcitVal, float64 a, float64 b, float64 c) {
    return process_error(dmx->CreateAIThrmstrChanIex(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, resistanceConfig, currentExcitSource, currentExcitVal, a, b, c));
}

xerrors::Error SugaredDAQmx::SetScaleAttributeDouble(const char scaleName[], int32 attribute, float64 value) {
    return process_error(dmx->SetScaleAttributeDouble(scaleName, attribute, value));
}

xerrors::Error SugaredDAQmx::SetScaleAttributeDoubleArray(const char scaleName[], int32 attribute, const float64 value[], uInt32 size) {
    return process_error(dmx->SetScaleAttributeDoubleArray(scaleName, attribute, value, size));
}

xerrors::Error SugaredDAQmx::SetScaleAttributeInt32(const char scaleName[], int32 attribute, int32 value) {
    return process_error(dmx->SetScaleAttributeInt32(scaleName, attribute, value));
}

xerrors::Error SugaredDAQmx::SetScaleAttributeString(const char scaleName[], int32 attribute, const char value[]) {
    return process_error(dmx->SetScaleAttributeString(scaleName, attribute, value));
}

xerrors::Error SugaredDAQmx::SetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data) {
    return process_error(dmx->SetStartTrigTrigWhen(task, data));
}

xerrors::Error SugaredDAQmx::SetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime data) {
    return process_error(dmx->SetSyncPulseTimeWhen(task, data));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 value) {
    return process_error(dmx->SetTimingAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeDouble(TaskHandle task, int32 attribute, float64 value) {
    return process_error(dmx->SetTimingAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExBool(TaskHandle task, const char deviceNames[], int32 attribute, bool32 value) {
    return process_error(dmx->SetTimingAttributeExBool(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExDouble(TaskHandle task, const char deviceNames[], int32 attribute, float64 value) {
    return process_error(dmx->SetTimingAttributeExDouble(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExInt32(TaskHandle task, const char deviceNames[], int32 attribute, int32 value) {
    return process_error(dmx->SetTimingAttributeExInt32(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExString(TaskHandle task, const char deviceNames[], int32 attribute, const char value[]) {
    return process_error(dmx->SetTimingAttributeExString(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExTimestamp(TaskHandle task, const char deviceNames[], int32 attribute, CVIAbsoluteTime value) {
    return process_error(dmx->SetTimingAttributeExTimestamp(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExUInt32(TaskHandle task, const char deviceNames[], int32 attribute, uInt32 value) {
    return process_error(dmx->SetTimingAttributeExUInt32(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExUInt64(TaskHandle task, const char deviceNames[], int32 attribute, uInt64 value) {
    return process_error(dmx->SetTimingAttributeExUInt64(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 value) {
    return process_error(dmx->SetTimingAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeString(TaskHandle task, int32 attribute, const char value[]) {
    return process_error(dmx->SetTimingAttributeString(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeTimestamp(TaskHandle task, int32 attribute, CVIAbsoluteTime value) {
    return process_error(dmx->SetTimingAttributeTimestamp(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetTimingAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) {
    return process_error(dmx->SetTimingAttributeUInt64(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 value) {
    return process_error(dmx->SetTrigAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 value) {
    return process_error(dmx->SetTrigAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeDoubleArray(TaskHandle task, int32 attribute, const float64 value[], uInt32 size) {
    return process_error(dmx->SetTrigAttributeDoubleArray(task, attribute, value, size));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 value) {
    return process_error(dmx->SetTrigAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeInt32Array(TaskHandle task, int32 attribute, const int32 value[], uInt32 size) {
    return process_error(dmx->SetTrigAttributeInt32Array(task, attribute, value, size));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeString(TaskHandle task, int32 attribute, const char value[]) {
    return process_error(dmx->SetTrigAttributeString(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeTimestamp(TaskHandle task, int32 attribute, CVIAbsoluteTime value) {
    return process_error(dmx->SetTrigAttributeTimestamp(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetTrigAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWatchdogAttributeBool(TaskHandle task, const char lines[], int32 attribute, bool32 value) {
    return process_error(dmx->SetWatchdogAttributeBool(task, lines, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWatchdogAttributeDouble(TaskHandle task, const char lines[], int32 attribute, float64 value) {
    return process_error(dmx->SetWatchdogAttributeDouble(task, lines, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWatchdogAttributeInt32(TaskHandle task, const char lines[], int32 attribute, int32 value) {
    return process_error(dmx->SetWatchdogAttributeInt32(task, lines, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWatchdogAttributeString(TaskHandle task, const char lines[], int32 attribute, const char value[]) {
    return process_error(dmx->SetWatchdogAttributeString(task, lines, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 value) {
    return process_error(dmx->SetWriteAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 value) {
    return process_error(dmx->SetWriteAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 value) {
    return process_error(dmx->SetWriteAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWriteAttributeString(TaskHandle task, int32 attribute, const char value[]) {
    return process_error(dmx->SetWriteAttributeString(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetWriteAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) {
    return process_error(dmx->SetWriteAttributeUInt64(task, attribute, value));
}

xerrors::Error SugaredDAQmx::StartNewFile(TaskHandle task, const char filePath[]) {
    return process_error(dmx->StartNewFile(task, filePath));
}

xerrors::Error SugaredDAQmx::StartTask(TaskHandle task) {
    return process_error(dmx->StartTask(task));
}

xerrors::Error SugaredDAQmx::StopTask(TaskHandle task) {
    return process_error(dmx->StopTask(task));
}

xerrors::Error SugaredDAQmx::TaskControl(TaskHandle task, int32 action) {
    return process_error(dmx->TaskControl(task, action));
}

xerrors::Error SugaredDAQmx::TristateOutputTerm(const char outputTerminal[]) {
    return process_error(dmx->TristateOutputTerm(outputTerminal));
}

xerrors::Error SugaredDAQmx::UnregisterDoneEvent(TaskHandle task, uInt32 options, DAQmxDoneEventCallbackPtr callbackFunction, void* callbackData) {
    return process_error(dmx->UnregisterDoneEvent(task, options, callbackFunction, callbackData));
}

xerrors::Error SugaredDAQmx::UnregisterEveryNSamplesEvent(TaskHandle task, int32 everyNSamplesEventType, uInt32 nSamples, uInt32 options, DAQmxEveryNSamplesEventCallbackPtr callbackFunction, void* callbackData) {
    return process_error(dmx->UnregisterEveryNSamplesEvent(task, everyNSamplesEventType, nSamples, options, callbackFunction, callbackData));
}

xerrors::Error SugaredDAQmx::UnregisterSignalEvent(TaskHandle task, int32 signalID, uInt32 options, DAQmxSignalEventCallbackPtr callbackFunction, void* callbackData) {
    return process_error(dmx->UnregisterSignalEvent(task, signalID, options, callbackFunction, callbackData));
}

xerrors::Error SugaredDAQmx::UnreserveNetworkDevice(const char deviceName[]) {
    return process_error(dmx->UnreserveNetworkDevice(deviceName));
}

xerrors::Error SugaredDAQmx::WaitForNextSampleClock(TaskHandle task, float64 timeout, bool32* isLate) {
    return process_error(dmx->WaitForNextSampleClock(task, timeout, isLate));
}

xerrors::Error SugaredDAQmx::WaitForValidTimestamp(TaskHandle task, int32 timestampEvent, float64 timeout, CVIAbsoluteTime* timestamp) {
    return process_error(dmx->WaitForValidTimestamp(task, timestampEvent, timeout, timestamp));
}

xerrors::Error SugaredDAQmx::WaitUntilTaskDone(TaskHandle task, float64 timeToWait) {
    return process_error(dmx->WaitUntilTaskDone(task, timeToWait));
}

xerrors::Error SugaredDAQmx::WriteAnalogF64(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const float64 writeArray[], int32* sampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteAnalogF64(task, numSampsPerChan, autoStart, timeout, dataLayout, writeArray, sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteAnalogScalarF64(TaskHandle task, bool32 autoStart, float64 timeout, float64 value, bool32* reserved) {
    return process_error(dmx->WriteAnalogScalarF64(task, autoStart, timeout, value, reserved));
}

xerrors::Error SugaredDAQmx::WriteBinaryI16(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const int16 writeArray[], int32* sampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteBinaryI16(task, numSampsPerChan, autoStart, timeout, dataLayout, writeArray, sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteBinaryI32(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const int32 writeArray[], int32* sampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteBinaryI32(task, numSampsPerChan, autoStart, timeout, dataLayout, writeArray, sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteBinaryU16(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const uInt16 writeArray[], int32* sampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteBinaryU16(task, numSampsPerChan, autoStart, timeout, dataLayout, writeArray, sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteBinaryU32(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const uInt32 writeArray[], int32* sampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteBinaryU32(task, numSampsPerChan, autoStart, timeout, dataLayout, writeArray, sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteCtrFreq(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const float64 frequency[], const float64 dutyCycle[], int32* numSampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteCtrFreq(task, numSampsPerChan, autoStart, timeout, dataLayout, frequency, dutyCycle, numSampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteCtrFreqScalar(TaskHandle task, bool32 autoStart, float64 timeout, float64 frequency, float64 dutyCycle, bool32* reserved) {
    return process_error(dmx->WriteCtrFreqScalar(task, autoStart, timeout, frequency, dutyCycle, reserved));
}

xerrors::Error SugaredDAQmx::WriteCtrTicks(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const uInt32 highTicks[], const uInt32 lowTicks[], int32* numSampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteCtrTicks(task, numSampsPerChan, autoStart, timeout, dataLayout, highTicks, lowTicks, numSampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteCtrTicksScalar(TaskHandle task, bool32 autoStart, float64 timeout, uInt32 highTicks, uInt32 lowTicks, bool32* reserved) {
    return process_error(dmx->WriteCtrTicksScalar(task, autoStart, timeout, highTicks, lowTicks, reserved));
}

xerrors::Error SugaredDAQmx::WriteCtrTime(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const float64 highTime[], const float64 lowTime[], int32* numSampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteCtrTime(task, numSampsPerChan, autoStart, timeout, dataLayout, highTime, lowTime, numSampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteCtrTimeScalar(TaskHandle task, bool32 autoStart, float64 timeout, float64 highTime, float64 lowTime, bool32* reserved) {
    return process_error(dmx->WriteCtrTimeScalar(task, autoStart, timeout, highTime, lowTime, reserved));
}

xerrors::Error SugaredDAQmx::WriteDigitalLines(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const uInt8 writeArray[], int32* sampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteDigitalLines(task, numSampsPerChan, autoStart, timeout, dataLayout, writeArray, sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteDigitalScalarU32(TaskHandle task, bool32 autoStart, float64 timeout, uInt32 value, bool32* reserved) {
    return process_error(dmx->WriteDigitalScalarU32(task, autoStart, timeout, value, reserved));
}

xerrors::Error SugaredDAQmx::WriteDigitalU16(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const uInt16 writeArray[], int32* sampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteDigitalU16(task, numSampsPerChan, autoStart, timeout, dataLayout, writeArray, sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteDigitalU32(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const uInt32 writeArray[], int32* sampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteDigitalU32(task, numSampsPerChan, autoStart, timeout, dataLayout, writeArray, sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteDigitalU8(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const uInt8 writeArray[], int32* sampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteDigitalU8(task, numSampsPerChan, autoStart, timeout, dataLayout, writeArray, sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteRaw(TaskHandle task, int32 numSamps, bool32 autoStart, float64 timeout, const uInt8 writeArray[], int32* sampsPerChanWritten, bool32* reserved) {
    return process_error(dmx->WriteRaw(task, numSamps, autoStart, timeout, writeArray, sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteToTEDSFromArray(const char physicalChannel[], const uInt8 bitStream[], uInt32 arraySize, int32 basicTEDSOptions) {
    return process_error(dmx->WriteToTEDSFromArray(physicalChannel, bitStream, arraySize, basicTEDSOptions));
}

xerrors::Error SugaredDAQmx::WriteToTEDSFromFile(const char physicalChannel[], const char filePath[], int32 basicTEDSOptions) {
    return process_error(dmx->WriteToTEDSFromFile(physicalChannel, filePath, basicTEDSOptions));
}

xerrors::Error SugaredDAQmx::CreateLinScale(const char scaleName[], float64 slope, float64 yIntercept, int32 preScaledUnits, const char customScaleName[]) {
    return process_error(dmx->CreateLinScale(scaleName, slope, yIntercept, preScaledUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateMapScale(const char scaleName[], float64 prescaledMin, float64 prescaledMax, float64 scaledMin, float64 scaledMax, int32 preScaledUnits, const char customScaleName[]) {
    return process_error(dmx->CreateMapScale(scaleName, prescaledMin, prescaledMax, scaledMin, scaledMax, preScaledUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTableScale(const char scaleName[], const float64 prescaledVals[], uInt32 numPrescaledVals, const float64 scaledVals[], uInt32 numScaledVals, int32 preScaledUnits, const char customScaleName[]) {
    return process_error(dmx->CreateTableScale(scaleName, prescaledVals, numPrescaledVals, scaledVals, numScaledVals, preScaledUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIVoltageChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 terminalConfig, float64 minVal, float64 maxVal, int32 units, const char customScaleName[]) {
    return process_error(dmx->CreateAIVoltageChan(task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal, units, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAOCurrentChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, const char customScaleName[]) {
    return process_error(dmx->CreateAOCurrentChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAOFuncGenChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 type, float64 freq, float64 amplitude, float64 offset) {
    return process_error(dmx->CreateAOFuncGenChan(task, physicalChannel, nameToAssignToChannel, type, freq, amplitude, offset));
}

xerrors::Error SugaredDAQmx::CreateAOVoltageChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, const char customScaleName[]) {
    return process_error(dmx->CreateAOVoltageChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, customScaleName));
}

xerrors::Error SugaredDAQmx::CreatePolynomialScale(const char scaleName[], const float64 forwardCoeffs[], uInt32 numForwardCoeffs, const float64 reverseCoeffs[], uInt32 numReverseCoeffs, int32 preScaledUnits, const char customScaleName[]) {
    return process_error(dmx->CreatePolynomialScale(scaleName, forwardCoeffs, numForwardCoeffs, reverseCoeffs, numReverseCoeffs, preScaledUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIVelocityIEPEChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 terminalConfig, float64 minVal, float64 maxVal, int32 units, float64 sensitivity, int32 sensitivityUnits, int32 currentExcitSource, float64 currentExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateAIVelocityIEPEChan(task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal, units, sensitivity, sensitivityUnits, currentExcitSource, currentExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAITorqueBridgeTableChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig, int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance, const float64 electricalVals[], uInt32 numElectricalVals, int32 electricalUnits, const float64 physicalVals[], uInt32 numPhysicalVals, int32 physicalUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAITorqueBridgeTableChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance, electricalVals, numElectricalVals, electricalUnits, physicalVals, numPhysicalVals, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAITorqueBridgePolynomialChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig, int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance, const float64 forwardCoeffs[], uInt32 numForwardCoeffs, const float64 reverseCoeffs[], uInt32 numReverseCoeffs, int32 electricalUnits, int32 physicalUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAITorqueBridgePolynomialChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance, forwardCoeffs, numForwardCoeffs, reverseCoeffs, numReverseCoeffs, electricalUnits, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAITorqueBridgeTwoPointLinChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig, int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance, float64 firstElectricalVal, float64 secondElectricalVal, int32 electricalUnits, float64 firstPhysicalVal, float64 secondPhysicalVal, int32 physicalUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAITorqueBridgeTwoPointLinChan(task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance, firstElectricalVal, secondElectricalVal, electricalUnits, firstPhysicalVal, secondPhysicalVal, physicalUnits, customScaleName));
}
