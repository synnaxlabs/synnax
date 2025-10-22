// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstring>
#include <memory>
#include <utility>

#include "x/cpp/xlib/xlib.h"
#include "x/cpp/xos/xos.h"

#include "driver/ni/daqmx/prod.h"

#ifdef _WIN32
static const std::string LIB_NAME = "nicaiu.dll";
#else
static const std::string LIB_NAME = "libnidaqmx.so.1";
#endif

namespace daqmx {
const auto LOAD_ERROR = xerrors::Error(
    xlib::LOAD_ERROR,
    "NI DAQmx shared libraries are not installed."
);

std::pair<std::shared_ptr<API>, xerrors::Error> ProdAPI::load() {
    const auto os = xos::get();
    if (os == xos::MACOS_NAME || os == xos::UNKNOWN_NAME)
        return {nullptr, xerrors::NIL};
    auto lib = std::make_unique<xlib::SharedLib>(LIB_NAME);
    if (!lib->load()) return {nullptr, LOAD_ERROR};
    return {std::make_shared<ProdAPI>(lib), xerrors::Error()};
}

ProdAPI::ProdAPI(std::unique_ptr<xlib::SharedLib> &lib_): lib(std::move(lib_)) {
    memset(&function_pointers_, 0, sizeof(function_pointers_));
    function_pointers_
        .AddCDAQSyncConnection = reinterpret_cast<AddCDAQSyncConnectionPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxAddCDAQSyncConnection"))
    );
    function_pointers_.AddGlobalChansToTask = reinterpret_cast<AddGlobalChansToTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxAddGlobalChansToTask"))
    );
    function_pointers_.AddNetworkDevice = reinterpret_cast<AddNetworkDevicePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxAddNetworkDevice"))
    );
    function_pointers_.AreConfiguredCDAQSyncPortsDisconnected = reinterpret_cast<
        AreConfiguredCDAQSyncPortsDisconnectedPtr>(const_cast<void *>(
        lib->get_func_ptr("DAQmxAreConfiguredCDAQSyncPortsDisconnected")
    ));
    function_pointers_.AutoConfigureCDAQSyncConnections = reinterpret_cast<
        AutoConfigureCDAQSyncConnectionsPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxAutoConfigureCDAQSyncConnections"))
    );
    function_pointers_
        .CalculateReversePolyCoeff = reinterpret_cast<CalculateReversePolyCoeffPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCalculateReversePolyCoeff"))
    );
    function_pointers_.CfgAnlgEdgeRefTrig = reinterpret_cast<CfgAnlgEdgeRefTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgAnlgEdgeRefTrig"))
    );
    function_pointers_.CfgAnlgEdgeStartTrig = reinterpret_cast<CfgAnlgEdgeStartTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgAnlgEdgeStartTrig"))
    );
    function_pointers_
        .CfgAnlgMultiEdgeRefTrig = reinterpret_cast<CfgAnlgMultiEdgeRefTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgAnlgMultiEdgeRefTrig"))
    );
    function_pointers_
        .CfgAnlgMultiEdgeStartTrig = reinterpret_cast<CfgAnlgMultiEdgeStartTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgAnlgMultiEdgeStartTrig"))
    );
    function_pointers_.CfgAnlgWindowRefTrig = reinterpret_cast<CfgAnlgWindowRefTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgAnlgWindowRefTrig"))
    );
    function_pointers_
        .CfgAnlgWindowStartTrig = reinterpret_cast<CfgAnlgWindowStartTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgAnlgWindowStartTrig"))
    );
    function_pointers_.CfgBurstHandshakingTimingExportClock = reinterpret_cast<
        CfgBurstHandshakingTimingExportClockPtr>(const_cast<void *>(
        lib->get_func_ptr("DAQmxCfgBurstHandshakingTimingExportClock")
    ));
    function_pointers_.CfgBurstHandshakingTimingImportClock = reinterpret_cast<
        CfgBurstHandshakingTimingImportClockPtr>(const_cast<void *>(
        lib->get_func_ptr("DAQmxCfgBurstHandshakingTimingImportClock")
    ));
    function_pointers_
        .CfgChangeDetectionTiming = reinterpret_cast<CfgChangeDetectionTimingPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgChangeDetectionTiming"))
    );
    function_pointers_.CfgDigEdgeRefTrig = reinterpret_cast<CfgDigEdgeRefTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgDigEdgeRefTrig"))
    );
    function_pointers_.CfgDigEdgeStartTrig = reinterpret_cast<CfgDigEdgeStartTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgDigEdgeStartTrig"))
    );
    function_pointers_.CfgDigPatternRefTrig = reinterpret_cast<CfgDigPatternRefTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgDigPatternRefTrig"))
    );
    function_pointers_
        .CfgDigPatternStartTrig = reinterpret_cast<CfgDigPatternStartTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgDigPatternStartTrig"))
    );
    function_pointers_.CfgHandshakingTiming = reinterpret_cast<CfgHandshakingTimingPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgHandshakingTiming"))
    );
    function_pointers_.CfgImplicitTiming = reinterpret_cast<CfgImplicitTimingPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgImplicitTiming"))
    );
    function_pointers_.CfgInputBuffer = reinterpret_cast<CfgInputBufferPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgInputBuffer"))
    );
    function_pointers_.CfgOutputBuffer = reinterpret_cast<CfgOutputBufferPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgOutputBuffer"))
    );
    function_pointers_
        .CfgPipelinedSampClkTiming = reinterpret_cast<CfgPipelinedSampClkTimingPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgPipelinedSampClkTiming"))
    );
    function_pointers_.CfgSampClkTiming = reinterpret_cast<CfgSampClkTimingPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgSampClkTiming"))
    );
    function_pointers_.CfgTimeStartTrig = reinterpret_cast<CfgTimeStartTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgTimeStartTrig"))
    );
    function_pointers_
        .CfgWatchdogAOExpirStates = reinterpret_cast<CfgWatchdogAOExpirStatesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgWatchdogAOExpirStates"))
    );
    function_pointers_
        .CfgWatchdogCOExpirStates = reinterpret_cast<CfgWatchdogCOExpirStatesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgWatchdogCOExpirStates"))
    );
    function_pointers_
        .CfgWatchdogDOExpirStates = reinterpret_cast<CfgWatchdogDOExpirStatesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCfgWatchdogDOExpirStates"))
    );
    function_pointers_.ClearTEDS = reinterpret_cast<ClearTEDSPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxClearTEDS"))
    );
    function_pointers_.ClearTask = reinterpret_cast<ClearTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxClearTask"))
    );
    function_pointers_.ConfigureLogging = reinterpret_cast<ConfigureLoggingPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxConfigureLogging"))
    );
    function_pointers_.ConfigureTEDS = reinterpret_cast<ConfigureTEDSPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxConfigureTEDS"))
    );
    function_pointers_.ConnectTerms = reinterpret_cast<ConnectTermsPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxConnectTerms"))
    );
    function_pointers_.ControlWatchdogTask = reinterpret_cast<ControlWatchdogTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxControlWatchdogTask"))
    );
    function_pointers_.CreateAIAccel4WireDCVoltageChan = reinterpret_cast<
        CreateAIAccel4WireDCVoltageChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIAccel4WireDCVoltageChan"))
    );
    function_pointers_.CreateAIAccelChan = reinterpret_cast<CreateAIAccelChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIAccelChan"))
    );
    function_pointers_
        .CreateAIAccelChargeChan = reinterpret_cast<CreateAIAccelChargeChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIAccelChargeChan"))
    );
    function_pointers_.CreateAIBridgeChan = reinterpret_cast<CreateAIBridgeChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIBridgeChan"))
    );
    function_pointers_.CreateAIChargeChan = reinterpret_cast<CreateAIChargeChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIChargeChan"))
    );
    function_pointers_.CreateAICurrentChan = reinterpret_cast<CreateAICurrentChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAICurrentChan"))
    );
    function_pointers_
        .CreateAICurrentRMSChan = reinterpret_cast<CreateAICurrentRMSChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAICurrentRMSChan"))
    );
    function_pointers_.CreateAIForceBridgePolynomialChan = reinterpret_cast<
        CreateAIForceBridgePolynomialChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIForceBridgePolynomialChan"))
    );
    function_pointers_.CreateAIForceBridgeTableChan = reinterpret_cast<
        CreateAIForceBridgeTableChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIForceBridgeTableChan"))
    );
    function_pointers_.CreateAIForceBridgeTwoPointLinChan = reinterpret_cast<
        CreateAIForceBridgeTwoPointLinChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIForceBridgeTwoPointLinChan"))
    );
    function_pointers_
        .CreateAIForceIEPEChan = reinterpret_cast<CreateAIForceIEPEChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIForceIEPEChan"))
    );
    function_pointers_
        .CreateAIFreqVoltageChan = reinterpret_cast<CreateAIFreqVoltageChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIFreqVoltageChan"))
    );
    function_pointers_
        .CreateAIMicrophoneChan = reinterpret_cast<CreateAIMicrophoneChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIMicrophoneChan"))
    );
    function_pointers_.CreateAIPosEddyCurrProxProbeChan = reinterpret_cast<
        CreateAIPosEddyCurrProxProbeChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIPosEddyCurrProxProbeChan"))
    );
    function_pointers_.CreateAIPosLVDTChan = reinterpret_cast<CreateAIPosLVDTChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIPosLVDTChan"))
    );
    function_pointers_.CreateAIPosRVDTChan = reinterpret_cast<CreateAIPosRVDTChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIPosRVDTChan"))
    );
    function_pointers_.CreateAIPressureBridgePolynomialChan = reinterpret_cast<
        CreateAIPressureBridgePolynomialChanPtr>(const_cast<void *>(
        lib->get_func_ptr("DAQmxCreateAIPressureBridgePolynomialChan")
    ));
    function_pointers_.CreateAIPressureBridgeTableChan = reinterpret_cast<
        CreateAIPressureBridgeTableChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIPressureBridgeTableChan"))
    );
    function_pointers_.CreateAIPressureBridgeTwoPointLinChan = reinterpret_cast<
        CreateAIPressureBridgeTwoPointLinChanPtr>(const_cast<void *>(
        lib->get_func_ptr("DAQmxCreateAIPressureBridgeTwoPointLinChan")
    ));
    function_pointers_.CreateAIRTDChan = reinterpret_cast<CreateAIRTDChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIRTDChan"))
    );
    function_pointers_
        .CreateAIResistanceChan = reinterpret_cast<CreateAIResistanceChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIResistanceChan"))
    );
    function_pointers_.CreateAIRosetteStrainGageChan = reinterpret_cast<
        CreateAIRosetteStrainGageChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIRosetteStrainGageChan"))
    );
    function_pointers_
        .CreateAIStrainGageChan = reinterpret_cast<CreateAIStrainGageChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIStrainGageChan"))
    );
    function_pointers_.CreateAITempBuiltInSensorChan = reinterpret_cast<
        CreateAITempBuiltInSensorChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAITempBuiltInSensorChan"))
    );
    function_pointers_.CreateAIThrmcplChan = reinterpret_cast<CreateAIThrmcplChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIThrmcplChan"))
    );
    function_pointers_
        .CreateAIThrmstrChanIex = reinterpret_cast<CreateAIThrmstrChanIexPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIThrmstrChanIex"))
    );
    function_pointers_
        .CreateAIThrmstrChanVex = reinterpret_cast<CreateAIThrmstrChanVexPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIThrmstrChanVex"))
    );
    function_pointers_.CreateAITorqueBridgePolynomialChan = reinterpret_cast<
        CreateAITorqueBridgePolynomialChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAITorqueBridgePolynomialChan"))
    );
    function_pointers_.CreateAITorqueBridgeTableChan = reinterpret_cast<
        CreateAITorqueBridgeTableChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAITorqueBridgeTableChan"))
    );
    function_pointers_.CreateAITorqueBridgeTwoPointLinChan = reinterpret_cast<
        CreateAITorqueBridgeTwoPointLinChanPtr>(const_cast<void *>(
        lib->get_func_ptr("DAQmxCreateAITorqueBridgeTwoPointLinChan")
    ));
    function_pointers_
        .CreateAIVelocityIEPEChan = reinterpret_cast<CreateAIVelocityIEPEChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIVelocityIEPEChan"))
    );
    function_pointers_.CreateAIVoltageChan = reinterpret_cast<CreateAIVoltageChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIVoltageChan"))
    );
    function_pointers_.CreateAIVoltageChanWithExcit = reinterpret_cast<
        CreateAIVoltageChanWithExcitPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIVoltageChanWithExcit"))
    );
    function_pointers_
        .CreateAIVoltageRMSChan = reinterpret_cast<CreateAIVoltageRMSChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAIVoltageRMSChan"))
    );
    function_pointers_.CreateAOCurrentChan = reinterpret_cast<CreateAOCurrentChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAOCurrentChan"))
    );
    function_pointers_.CreateAOFuncGenChan = reinterpret_cast<CreateAOFuncGenChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAOFuncGenChan"))
    );
    function_pointers_.CreateAOVoltageChan = reinterpret_cast<CreateAOVoltageChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateAOVoltageChan"))
    );
    function_pointers_
        .CreateCIAngEncoderChan = reinterpret_cast<CreateCIAngEncoderChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCIAngEncoderChan"))
    );
    function_pointers_
        .CreateCIAngVelocityChan = reinterpret_cast<CreateCIAngVelocityChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCIAngVelocityChan"))
    );
    function_pointers_
        .CreateCICountEdgesChan = reinterpret_cast<CreateCICountEdgesChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCICountEdgesChan"))
    );
    function_pointers_
        .CreateCIDutyCycleChan = reinterpret_cast<CreateCIDutyCycleChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCIDutyCycleChan"))
    );
    function_pointers_.CreateCIFreqChan = reinterpret_cast<CreateCIFreqChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCIFreqChan"))
    );
    function_pointers_
        .CreateCIGPSTimestampChan = reinterpret_cast<CreateCIGPSTimestampChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCIGPSTimestampChan"))
    );
    function_pointers_
        .CreateCILinEncoderChan = reinterpret_cast<CreateCILinEncoderChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCILinEncoderChan"))
    );
    function_pointers_
        .CreateCILinVelocityChan = reinterpret_cast<CreateCILinVelocityChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCILinVelocityChan"))
    );
    function_pointers_.CreateCIPeriodChan = reinterpret_cast<CreateCIPeriodChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCIPeriodChan"))
    );
    function_pointers_
        .CreateCIPulseChanFreq = reinterpret_cast<CreateCIPulseChanFreqPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCIPulseChanFreq"))
    );
    function_pointers_
        .CreateCIPulseChanTicks = reinterpret_cast<CreateCIPulseChanTicksPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCIPulseChanTicks"))
    );
    function_pointers_
        .CreateCIPulseChanTime = reinterpret_cast<CreateCIPulseChanTimePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCIPulseChanTime"))
    );
    function_pointers_
        .CreateCIPulseWidthChan = reinterpret_cast<CreateCIPulseWidthChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCIPulseWidthChan"))
    );
    function_pointers_
        .CreateCISemiPeriodChan = reinterpret_cast<CreateCISemiPeriodChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCISemiPeriodChan"))
    );
    function_pointers_
        .CreateCITwoEdgeSepChan = reinterpret_cast<CreateCITwoEdgeSepChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCITwoEdgeSepChan"))
    );
    function_pointers_
        .CreateCOPulseChanFreq = reinterpret_cast<CreateCOPulseChanFreqPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCOPulseChanFreq"))
    );
    function_pointers_
        .CreateCOPulseChanTicks = reinterpret_cast<CreateCOPulseChanTicksPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCOPulseChanTicks"))
    );
    function_pointers_
        .CreateCOPulseChanTime = reinterpret_cast<CreateCOPulseChanTimePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateCOPulseChanTime"))
    );
    function_pointers_.CreateDIChan = reinterpret_cast<CreateDIChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateDIChan"))
    );
    function_pointers_.CreateDOChan = reinterpret_cast<CreateDOChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateDOChan"))
    );
    function_pointers_.CreateLinScale = reinterpret_cast<CreateLinScalePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateLinScale"))
    );
    function_pointers_.CreateMapScale = reinterpret_cast<CreateMapScalePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateMapScale"))
    );
    function_pointers_
        .CreatePolynomialScale = reinterpret_cast<CreatePolynomialScalePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreatePolynomialScale"))
    );
    function_pointers_
        .CreateTEDSAIAccelChan = reinterpret_cast<CreateTEDSAIAccelChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIAccelChan"))
    );
    function_pointers_
        .CreateTEDSAIBridgeChan = reinterpret_cast<CreateTEDSAIBridgeChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIBridgeChan"))
    );
    function_pointers_
        .CreateTEDSAICurrentChan = reinterpret_cast<CreateTEDSAICurrentChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAICurrentChan"))
    );
    function_pointers_
        .CreateTEDSAIForceBridgeChan = reinterpret_cast<CreateTEDSAIForceBridgeChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIForceBridgeChan"))
    );
    function_pointers_
        .CreateTEDSAIForceIEPEChan = reinterpret_cast<CreateTEDSAIForceIEPEChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIForceIEPEChan"))
    );
    function_pointers_
        .CreateTEDSAIMicrophoneChan = reinterpret_cast<CreateTEDSAIMicrophoneChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIMicrophoneChan"))
    );
    function_pointers_
        .CreateTEDSAIPosLVDTChan = reinterpret_cast<CreateTEDSAIPosLVDTChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIPosLVDTChan"))
    );
    function_pointers_
        .CreateTEDSAIPosRVDTChan = reinterpret_cast<CreateTEDSAIPosRVDTChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIPosRVDTChan"))
    );
    function_pointers_.CreateTEDSAIPressureBridgeChan = reinterpret_cast<
        CreateTEDSAIPressureBridgeChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIPressureBridgeChan"))
    );
    function_pointers_.CreateTEDSAIRTDChan = reinterpret_cast<CreateTEDSAIRTDChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIRTDChan"))
    );
    function_pointers_
        .CreateTEDSAIResistanceChan = reinterpret_cast<CreateTEDSAIResistanceChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIResistanceChan"))
    );
    function_pointers_
        .CreateTEDSAIStrainGageChan = reinterpret_cast<CreateTEDSAIStrainGageChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIStrainGageChan"))
    );
    function_pointers_
        .CreateTEDSAIThrmcplChan = reinterpret_cast<CreateTEDSAIThrmcplChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIThrmcplChan"))
    );
    function_pointers_
        .CreateTEDSAIThrmstrChanIex = reinterpret_cast<CreateTEDSAIThrmstrChanIexPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIThrmstrChanIex"))
    );
    function_pointers_
        .CreateTEDSAIThrmstrChanVex = reinterpret_cast<CreateTEDSAIThrmstrChanVexPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIThrmstrChanVex"))
    );
    function_pointers_.CreateTEDSAITorqueBridgeChan = reinterpret_cast<
        CreateTEDSAITorqueBridgeChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAITorqueBridgeChan"))
    );
    function_pointers_
        .CreateTEDSAIVoltageChan = reinterpret_cast<CreateTEDSAIVoltageChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIVoltageChan"))
    );
    function_pointers_.CreateTEDSAIVoltageChanWithExcit = reinterpret_cast<
        CreateTEDSAIVoltageChanWithExcitPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTEDSAIVoltageChanWithExcit"))
    );
    function_pointers_.CreateTableScale = reinterpret_cast<CreateTableScalePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTableScale"))
    );
    function_pointers_.CreateTask = reinterpret_cast<CreateTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateTask"))
    );
    function_pointers_
        .CreateWatchdogTimerTask = reinterpret_cast<CreateWatchdogTimerTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateWatchdogTimerTask"))
    );
    function_pointers_
        .CreateWatchdogTimerTaskEx = reinterpret_cast<CreateWatchdogTimerTaskExPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxCreateWatchdogTimerTaskEx"))
    );
    function_pointers_.DeleteNetworkDevice = reinterpret_cast<DeleteNetworkDevicePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxDeleteNetworkDevice"))
    );
    function_pointers_
        .DeleteSavedGlobalChan = reinterpret_cast<DeleteSavedGlobalChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxDeleteSavedGlobalChan"))
    );
    function_pointers_.DeleteSavedScale = reinterpret_cast<DeleteSavedScalePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxDeleteSavedScale"))
    );
    function_pointers_.DeleteSavedTask = reinterpret_cast<DeleteSavedTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxDeleteSavedTask"))
    );
    function_pointers_.DeviceSupportsCal = reinterpret_cast<DeviceSupportsCalPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxDeviceSupportsCal"))
    );
    function_pointers_.DisableRefTrig = reinterpret_cast<DisableRefTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxDisableRefTrig"))
    );
    function_pointers_.DisableStartTrig = reinterpret_cast<DisableStartTrigPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxDisableStartTrig"))
    );
    function_pointers_.DisconnectTerms = reinterpret_cast<DisconnectTermsPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxDisconnectTerms"))
    );
    function_pointers_.ExportSignal = reinterpret_cast<ExportSignalPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxExportSignal"))
    );
    function_pointers_.GetAIChanCalCalDate = reinterpret_cast<GetAIChanCalCalDatePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetAIChanCalCalDate"))
    );
    function_pointers_.GetAIChanCalExpDate = reinterpret_cast<GetAIChanCalExpDatePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetAIChanCalExpDate"))
    );
    function_pointers_
        .GetAnalogPowerUpStates = reinterpret_cast<GetAnalogPowerUpStatesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetAnalogPowerUpStates"))
    );
    function_pointers_.GetAnalogPowerUpStatesWithOutputType = reinterpret_cast<
        GetAnalogPowerUpStatesWithOutputTypePtr>(const_cast<void *>(
        lib->get_func_ptr("DAQmxGetAnalogPowerUpStatesWithOutputType")
    ));
    function_pointers_
        .GetArmStartTrigTimestampVal = reinterpret_cast<GetArmStartTrigTimestampValPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetArmStartTrigTimestampVal"))
    );
    function_pointers_
        .GetArmStartTrigTrigWhen = reinterpret_cast<GetArmStartTrigTrigWhenPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetArmStartTrigTrigWhen"))
    );
    function_pointers_.GetAutoConfiguredCDAQSyncConnections = reinterpret_cast<
        GetAutoConfiguredCDAQSyncConnectionsPtr>(const_cast<void *>(
        lib->get_func_ptr("DAQmxGetAutoConfiguredCDAQSyncConnections")
    ));
    function_pointers_
        .GetBufferAttributeUInt32 = reinterpret_cast<GetBufferAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetBufferAttribute"))
    );
    function_pointers_
        .GetCalInfoAttributeBool = reinterpret_cast<GetCalInfoAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetCalInfoAttribute"))
    );
    function_pointers_
        .GetCalInfoAttributeDouble = reinterpret_cast<GetCalInfoAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetCalInfoAttribute"))
    );
    function_pointers_
        .GetCalInfoAttributeString = reinterpret_cast<GetCalInfoAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetCalInfoAttribute"))
    );
    function_pointers_
        .GetCalInfoAttributeUInt32 = reinterpret_cast<GetCalInfoAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetCalInfoAttribute"))
    );
    function_pointers_.GetChanAttributeBool = reinterpret_cast<GetChanAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetChanAttribute"))
    );
    function_pointers_
        .GetChanAttributeDouble = reinterpret_cast<GetChanAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetChanAttribute"))
    );
    function_pointers_
        .GetChanAttributeDoubleArray = reinterpret_cast<GetChanAttributeDoubleArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetChanAttribute"))
    );
    function_pointers_
        .GetChanAttributeInt32 = reinterpret_cast<GetChanAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetChanAttribute"))
    );
    function_pointers_
        .GetChanAttributeString = reinterpret_cast<GetChanAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetChanAttribute"))
    );
    function_pointers_
        .GetChanAttributeUInt32 = reinterpret_cast<GetChanAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetChanAttribute"))
    );
    function_pointers_
        .GetDeviceAttributeBool = reinterpret_cast<GetDeviceAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetDeviceAttribute"))
    );
    function_pointers_
        .GetDeviceAttributeDouble = reinterpret_cast<GetDeviceAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetDeviceAttribute"))
    );
    function_pointers_.GetDeviceAttributeDoubleArray = reinterpret_cast<
        GetDeviceAttributeDoubleArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetDeviceAttribute"))
    );
    function_pointers_
        .GetDeviceAttributeInt32 = reinterpret_cast<GetDeviceAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetDeviceAttribute"))
    );
    function_pointers_.GetDeviceAttributeInt32Array = reinterpret_cast<
        GetDeviceAttributeInt32ArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetDeviceAttribute"))
    );
    function_pointers_
        .GetDeviceAttributeString = reinterpret_cast<GetDeviceAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetDeviceAttribute"))
    );
    function_pointers_
        .GetDeviceAttributeUInt32 = reinterpret_cast<GetDeviceAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetDeviceAttribute"))
    );
    function_pointers_.GetDeviceAttributeUInt32Array = reinterpret_cast<
        GetDeviceAttributeUInt32ArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetDeviceAttribute"))
    );
    function_pointers_.GetDigitalLogicFamilyPowerUpState = reinterpret_cast<
        GetDigitalLogicFamilyPowerUpStatePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetDigitalLogicFamilyPowerUpState"))
    );
    function_pointers_
        .GetDigitalPowerUpStates = reinterpret_cast<GetDigitalPowerUpStatesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetDigitalPowerUpStates"))
    );
    function_pointers_.GetDigitalPullUpPullDownStates = reinterpret_cast<
        GetDigitalPullUpPullDownStatesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetDigitalPullUpPullDownStates"))
    );
    function_pointers_.GetDisconnectedCDAQSyncPorts = reinterpret_cast<
        GetDisconnectedCDAQSyncPortsPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetDisconnectedCDAQSyncPorts"))
    );
    function_pointers_.GetErrorString = reinterpret_cast<GetErrorStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetErrorString"))
    );
    function_pointers_.GetExportedSignalAttributeBool = reinterpret_cast<
        GetExportedSignalAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetExportedSignalAttribute"))
    );
    function_pointers_.GetExportedSignalAttributeDouble = reinterpret_cast<
        GetExportedSignalAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetExportedSignalAttribute"))
    );
    function_pointers_.GetExportedSignalAttributeInt32 = reinterpret_cast<
        GetExportedSignalAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetExportedSignalAttribute"))
    );
    function_pointers_.GetExportedSignalAttributeString = reinterpret_cast<
        GetExportedSignalAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetExportedSignalAttribute"))
    );
    function_pointers_.GetExportedSignalAttributeUInt32 = reinterpret_cast<
        GetExportedSignalAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetExportedSignalAttribute"))
    );
    function_pointers_
        .GetExtCalLastDateAndTime = reinterpret_cast<GetExtCalLastDateAndTimePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetExtCalLastDateAndTime"))
    );
    function_pointers_.GetExtendedErrorInfo = reinterpret_cast<GetExtendedErrorInfoPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetExtendedErrorInfo"))
    );
    function_pointers_.GetFirstSampClkWhen = reinterpret_cast<GetFirstSampClkWhenPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetFirstSampClkWhen"))
    );
    function_pointers_
        .GetFirstSampTimestampVal = reinterpret_cast<GetFirstSampTimestampValPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetFirstSampTimestampVal"))
    );
    function_pointers_.GetNthTaskChannel = reinterpret_cast<GetNthTaskChannelPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetNthTaskChannel"))
    );
    function_pointers_.GetNthTaskDevice = reinterpret_cast<GetNthTaskDevicePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetNthTaskDevice"))
    );
    function_pointers_
        .GetNthTaskReadChannel = reinterpret_cast<GetNthTaskReadChannelPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetNthTaskReadChannel"))
    );
    function_pointers_.GetPersistedChanAttributeBool = reinterpret_cast<
        GetPersistedChanAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPersistedChanAttribute"))
    );
    function_pointers_.GetPersistedChanAttributeString = reinterpret_cast<
        GetPersistedChanAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPersistedChanAttribute"))
    );
    function_pointers_.GetPersistedScaleAttributeBool = reinterpret_cast<
        GetPersistedScaleAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPersistedScaleAttribute"))
    );
    function_pointers_.GetPersistedScaleAttributeString = reinterpret_cast<
        GetPersistedScaleAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPersistedScaleAttribute"))
    );
    function_pointers_.GetPersistedTaskAttributeBool = reinterpret_cast<
        GetPersistedTaskAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPersistedTaskAttribute"))
    );
    function_pointers_.GetPersistedTaskAttributeString = reinterpret_cast<
        GetPersistedTaskAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPersistedTaskAttribute"))
    );
    function_pointers_.GetPhysicalChanAttributeBool = reinterpret_cast<
        GetPhysicalChanAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPhysicalChanAttribute"))
    );
    function_pointers_.GetPhysicalChanAttributeBytes = reinterpret_cast<
        GetPhysicalChanAttributeBytesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPhysicalChanAttribute"))
    );
    function_pointers_.GetPhysicalChanAttributeDouble = reinterpret_cast<
        GetPhysicalChanAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPhysicalChanAttribute"))
    );
    function_pointers_.GetPhysicalChanAttributeDoubleArray = reinterpret_cast<
        GetPhysicalChanAttributeDoubleArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPhysicalChanAttribute"))
    );
    function_pointers_.GetPhysicalChanAttributeInt32 = reinterpret_cast<
        GetPhysicalChanAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPhysicalChanAttribute"))
    );
    function_pointers_.GetPhysicalChanAttributeInt32Array = reinterpret_cast<
        GetPhysicalChanAttributeInt32ArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPhysicalChanAttribute"))
    );
    function_pointers_.GetPhysicalChanAttributeString = reinterpret_cast<
        GetPhysicalChanAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPhysicalChanAttribute"))
    );
    function_pointers_.GetPhysicalChanAttributeUInt32 = reinterpret_cast<
        GetPhysicalChanAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPhysicalChanAttribute"))
    );
    function_pointers_.GetPhysicalChanAttributeUInt32Array = reinterpret_cast<
        GetPhysicalChanAttributeUInt32ArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetPhysicalChanAttribute"))
    );
    function_pointers_.GetReadAttributeBool = reinterpret_cast<GetReadAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetReadAttribute"))
    );
    function_pointers_
        .GetReadAttributeDouble = reinterpret_cast<GetReadAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetReadAttribute"))
    );
    function_pointers_
        .GetReadAttributeInt32 = reinterpret_cast<GetReadAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetReadAttribute"))
    );
    function_pointers_
        .GetReadAttributeString = reinterpret_cast<GetReadAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetReadAttribute"))
    );
    function_pointers_
        .GetReadAttributeUInt32 = reinterpret_cast<GetReadAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetReadAttribute"))
    );
    function_pointers_
        .GetReadAttributeUInt64 = reinterpret_cast<GetReadAttributeUInt64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetReadAttribute"))
    );
    function_pointers_
        .GetRealTimeAttributeBool = reinterpret_cast<GetRealTimeAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetRealTimeAttribute"))
    );
    function_pointers_
        .GetRealTimeAttributeInt32 = reinterpret_cast<GetRealTimeAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetRealTimeAttribute"))
    );
    function_pointers_
        .GetRealTimeAttributeUInt32 = reinterpret_cast<GetRealTimeAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetRealTimeAttribute"))
    );
    function_pointers_
        .GetRefTrigTimestampVal = reinterpret_cast<GetRefTrigTimestampValPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetRefTrigTimestampVal"))
    );
    function_pointers_
        .GetScaleAttributeDouble = reinterpret_cast<GetScaleAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetScaleAttribute"))
    );
    function_pointers_.GetScaleAttributeDoubleArray = reinterpret_cast<
        GetScaleAttributeDoubleArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetScaleAttribute"))
    );
    function_pointers_
        .GetScaleAttributeInt32 = reinterpret_cast<GetScaleAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetScaleAttribute"))
    );
    function_pointers_
        .GetScaleAttributeString = reinterpret_cast<GetScaleAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetScaleAttribute"))
    );
    function_pointers_
        .GetSelfCalLastDateAndTime = reinterpret_cast<GetSelfCalLastDateAndTimePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetSelfCalLastDateAndTime"))
    );
    function_pointers_
        .GetStartTrigTimestampVal = reinterpret_cast<GetStartTrigTimestampValPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetStartTrigTimestampVal"))
    );
    function_pointers_.GetStartTrigTrigWhen = reinterpret_cast<GetStartTrigTrigWhenPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetStartTrigTrigWhen"))
    );
    function_pointers_.GetSyncPulseTimeWhen = reinterpret_cast<GetSyncPulseTimeWhenPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetSyncPulseTimeWhen"))
    );
    function_pointers_.GetSystemInfoAttributeString = reinterpret_cast<
        GetSystemInfoAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetSystemInfoAttribute"))
    );
    function_pointers_.GetSystemInfoAttributeUInt32 = reinterpret_cast<
        GetSystemInfoAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetSystemInfoAttribute"))
    );
    function_pointers_.GetTaskAttributeBool = reinterpret_cast<GetTaskAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTaskAttribute"))
    );
    function_pointers_
        .GetTaskAttributeString = reinterpret_cast<GetTaskAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTaskAttribute"))
    );
    function_pointers_
        .GetTaskAttributeUInt32 = reinterpret_cast<GetTaskAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTaskAttribute"))
    );
    function_pointers_
        .GetTimingAttributeBool = reinterpret_cast<GetTimingAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttribute"))
    );
    function_pointers_
        .GetTimingAttributeDouble = reinterpret_cast<GetTimingAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttribute"))
    );
    function_pointers_
        .GetTimingAttributeExBool = reinterpret_cast<GetTimingAttributeExBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttributeEx"))
    );
    function_pointers_
        .GetTimingAttributeExDouble = reinterpret_cast<GetTimingAttributeExDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttributeEx"))
    );
    function_pointers_
        .GetTimingAttributeExInt32 = reinterpret_cast<GetTimingAttributeExInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttributeEx"))
    );
    function_pointers_
        .GetTimingAttributeExString = reinterpret_cast<GetTimingAttributeExStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttributeEx"))
    );
    function_pointers_.GetTimingAttributeExTimestamp = reinterpret_cast<
        GetTimingAttributeExTimestampPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttributeEx"))
    );
    function_pointers_
        .GetTimingAttributeExUInt32 = reinterpret_cast<GetTimingAttributeExUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttributeEx"))
    );
    function_pointers_
        .GetTimingAttributeExUInt64 = reinterpret_cast<GetTimingAttributeExUInt64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttributeEx"))
    );
    function_pointers_
        .GetTimingAttributeInt32 = reinterpret_cast<GetTimingAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttribute"))
    );
    function_pointers_
        .GetTimingAttributeString = reinterpret_cast<GetTimingAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttribute"))
    );
    function_pointers_
        .GetTimingAttributeTimestamp = reinterpret_cast<GetTimingAttributeTimestampPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttribute"))
    );
    function_pointers_
        .GetTimingAttributeUInt32 = reinterpret_cast<GetTimingAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttribute"))
    );
    function_pointers_
        .GetTimingAttributeUInt64 = reinterpret_cast<GetTimingAttributeUInt64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTimingAttribute"))
    );
    function_pointers_.GetTrigAttributeBool = reinterpret_cast<GetTrigAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTrigAttribute"))
    );
    function_pointers_
        .GetTrigAttributeDouble = reinterpret_cast<GetTrigAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTrigAttribute"))
    );
    function_pointers_
        .GetTrigAttributeDoubleArray = reinterpret_cast<GetTrigAttributeDoubleArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTrigAttribute"))
    );
    function_pointers_
        .GetTrigAttributeInt32 = reinterpret_cast<GetTrigAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTrigAttribute"))
    );
    function_pointers_
        .GetTrigAttributeInt32Array = reinterpret_cast<GetTrigAttributeInt32ArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTrigAttribute"))
    );
    function_pointers_
        .GetTrigAttributeString = reinterpret_cast<GetTrigAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTrigAttribute"))
    );
    function_pointers_
        .GetTrigAttributeTimestamp = reinterpret_cast<GetTrigAttributeTimestampPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTrigAttribute"))
    );
    function_pointers_
        .GetTrigAttributeUInt32 = reinterpret_cast<GetTrigAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetTrigAttribute"))
    );
    function_pointers_
        .GetWatchdogAttributeBool = reinterpret_cast<GetWatchdogAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetWatchdogAttribute"))
    );
    function_pointers_
        .GetWatchdogAttributeDouble = reinterpret_cast<GetWatchdogAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetWatchdogAttribute"))
    );
    function_pointers_
        .GetWatchdogAttributeInt32 = reinterpret_cast<GetWatchdogAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetWatchdogAttribute"))
    );
    function_pointers_
        .GetWatchdogAttributeString = reinterpret_cast<GetWatchdogAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetWatchdogAttribute"))
    );
    function_pointers_
        .GetWriteAttributeBool = reinterpret_cast<GetWriteAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetWriteAttribute"))
    );
    function_pointers_
        .GetWriteAttributeDouble = reinterpret_cast<GetWriteAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetWriteAttribute"))
    );
    function_pointers_
        .GetWriteAttributeInt32 = reinterpret_cast<GetWriteAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetWriteAttribute"))
    );
    function_pointers_
        .GetWriteAttributeString = reinterpret_cast<GetWriteAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetWriteAttribute"))
    );
    function_pointers_
        .GetWriteAttributeUInt32 = reinterpret_cast<GetWriteAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetWriteAttribute"))
    );
    function_pointers_
        .GetWriteAttributeUInt64 = reinterpret_cast<GetWriteAttributeUInt64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetWriteAttribute"))
    );
    function_pointers_.IsTaskDone = reinterpret_cast<IsTaskDonePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxIsTaskDone"))
    );
    function_pointers_.LoadTask = reinterpret_cast<LoadTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxLoadTask"))
    );
    function_pointers_.PerformBridgeOffsetNullingCalEx = reinterpret_cast<
        PerformBridgeOffsetNullingCalExPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxPerformBridgeOffsetNullingCalEx"))
    );
    function_pointers_
        .PerformBridgeShuntCalEx = reinterpret_cast<PerformBridgeShuntCalExPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxPerformBridgeShuntCalEx"))
    );
    function_pointers_
        .PerformStrainShuntCalEx = reinterpret_cast<PerformStrainShuntCalExPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxPerformStrainShuntCalEx"))
    );
    function_pointers_.PerformThrmcplLeadOffsetNullingCal = reinterpret_cast<
        PerformThrmcplLeadOffsetNullingCalPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxPerformThrmcplLeadOffsetNullingCal"))
    );
    function_pointers_.ReadAnalogF64 = reinterpret_cast<ReadAnalogF64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadAnalogF64"))
    );
    function_pointers_.ReadAnalogScalarF64 = reinterpret_cast<ReadAnalogScalarF64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadAnalogScalarF64"))
    );
    function_pointers_.ReadBinaryI16 = reinterpret_cast<ReadBinaryI16Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadBinaryI16"))
    );
    function_pointers_.ReadBinaryI32 = reinterpret_cast<ReadBinaryI32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadBinaryI32"))
    );
    function_pointers_.ReadBinaryU16 = reinterpret_cast<ReadBinaryU16Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadBinaryU16"))
    );
    function_pointers_.ReadBinaryU32 = reinterpret_cast<ReadBinaryU32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadBinaryU32"))
    );
    function_pointers_.ReadCounterF64 = reinterpret_cast<ReadCounterF64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadCounterF64"))
    );
    function_pointers_.ReadCounterF64Ex = reinterpret_cast<ReadCounterF64ExPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadCounterF64Ex"))
    );
    function_pointers_.ReadCounterScalarF64 = reinterpret_cast<ReadCounterScalarF64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadCounterScalarF64"))
    );
    function_pointers_.ReadCounterScalarU32 = reinterpret_cast<ReadCounterScalarU32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadCounterScalarU32"))
    );
    function_pointers_.ReadCounterU32 = reinterpret_cast<ReadCounterU32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadCounterU32"))
    );
    function_pointers_.ReadCounterU32Ex = reinterpret_cast<ReadCounterU32ExPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadCounterU32Ex"))
    );
    function_pointers_.ReadCtrFreq = reinterpret_cast<ReadCtrFreqPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadCtrFreq"))
    );
    function_pointers_.ReadCtrFreqScalar = reinterpret_cast<ReadCtrFreqScalarPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadCtrFreqScalar"))
    );
    function_pointers_.ReadCtrTicks = reinterpret_cast<ReadCtrTicksPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadCtrTicks"))
    );
    function_pointers_.ReadCtrTicksScalar = reinterpret_cast<ReadCtrTicksScalarPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadCtrTicksScalar"))
    );
    function_pointers_.ReadCtrTime = reinterpret_cast<ReadCtrTimePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadCtrTime"))
    );
    function_pointers_.ReadCtrTimeScalar = reinterpret_cast<ReadCtrTimeScalarPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadCtrTimeScalar"))
    );
    function_pointers_.ReadDigitalLines = reinterpret_cast<ReadDigitalLinesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadDigitalLines"))
    );
    function_pointers_.ReadDigitalScalarU32 = reinterpret_cast<ReadDigitalScalarU32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadDigitalScalarU32"))
    );
    function_pointers_.ReadDigitalU16 = reinterpret_cast<ReadDigitalU16Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadDigitalU16"))
    );
    function_pointers_.ReadDigitalU32 = reinterpret_cast<ReadDigitalU32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadDigitalU32"))
    );
    function_pointers_.ReadDigitalU8 = reinterpret_cast<ReadDigitalU8Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadDigitalU8"))
    );
    function_pointers_.ReadRaw = reinterpret_cast<ReadRawPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReadRaw"))
    );
    function_pointers_.RegisterDoneEvent = reinterpret_cast<RegisterDoneEventPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRegisterDoneEvent"))
    );
    function_pointers_
        .RegisterEveryNSamplesEvent = reinterpret_cast<RegisterEveryNSamplesEventPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRegisterEveryNSamplesEvent"))
    );
    function_pointers_.RegisterSignalEvent = reinterpret_cast<RegisterSignalEventPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRegisterSignalEvent"))
    );
    function_pointers_
        .RemoveCDAQSyncConnection = reinterpret_cast<RemoveCDAQSyncConnectionPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRemoveCDAQSyncConnection"))
    );
    function_pointers_.ReserveNetworkDevice = reinterpret_cast<ReserveNetworkDevicePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReserveNetworkDevice"))
    );
    function_pointers_.ResetBufferAttribute = reinterpret_cast<ResetBufferAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetBufferAttribute"))
    );
    function_pointers_.ResetChanAttribute = reinterpret_cast<ResetChanAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetChanAttribute"))
    );
    function_pointers_.ResetDevice = reinterpret_cast<ResetDevicePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetDevice"))
    );
    function_pointers_.ResetExportedSignalAttribute = reinterpret_cast<
        ResetExportedSignalAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetExportedSignalAttribute"))
    );
    function_pointers_.ResetReadAttribute = reinterpret_cast<ResetReadAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetReadAttribute"))
    );
    function_pointers_
        .ResetRealTimeAttribute = reinterpret_cast<ResetRealTimeAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetRealTimeAttribute"))
    );
    function_pointers_.ResetTimingAttribute = reinterpret_cast<ResetTimingAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetTimingAttribute"))
    );
    function_pointers_
        .ResetTimingAttributeEx = reinterpret_cast<ResetTimingAttributeExPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetTimingAttributeEx"))
    );
    function_pointers_.ResetTrigAttribute = reinterpret_cast<ResetTrigAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetTrigAttribute"))
    );
    function_pointers_
        .ResetWatchdogAttribute = reinterpret_cast<ResetWatchdogAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetWatchdogAttribute"))
    );
    function_pointers_.ResetWriteAttribute = reinterpret_cast<ResetWriteAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetWriteAttribute"))
    );
    function_pointers_
        .RestoreLastExtCalConst = reinterpret_cast<RestoreLastExtCalConstPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRestoreLastExtCalConst"))
    );
    function_pointers_.SaveGlobalChan = reinterpret_cast<SaveGlobalChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSaveGlobalChan"))
    );
    function_pointers_.SaveScale = reinterpret_cast<SaveScalePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSaveScale"))
    );
    function_pointers_.SaveTask = reinterpret_cast<SaveTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSaveTask"))
    );
    function_pointers_.SelfCal = reinterpret_cast<SelfCalPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSelfCal"))
    );
    function_pointers_.SelfTestDevice = reinterpret_cast<SelfTestDevicePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSelfTestDevice"))
    );
    function_pointers_.SetAIChanCalCalDate = reinterpret_cast<SetAIChanCalCalDatePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetAIChanCalCalDate"))
    );
    function_pointers_.SetAIChanCalExpDate = reinterpret_cast<SetAIChanCalExpDatePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetAIChanCalExpDate"))
    );
    function_pointers_
        .SetAnalogPowerUpStates = reinterpret_cast<SetAnalogPowerUpStatesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetAnalogPowerUpStates"))
    );
    function_pointers_.SetAnalogPowerUpStatesWithOutputType = reinterpret_cast<
        SetAnalogPowerUpStatesWithOutputTypePtr>(const_cast<void *>(
        lib->get_func_ptr("DAQmxSetAnalogPowerUpStatesWithOutputType")
    ));
    function_pointers_
        .SetArmStartTrigTrigWhen = reinterpret_cast<SetArmStartTrigTrigWhenPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetArmStartTrigTrigWhen"))
    );
    function_pointers_
        .SetBufferAttributeUInt32 = reinterpret_cast<SetBufferAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetBufferAttribute"))
    );
    function_pointers_
        .SetCalInfoAttributeBool = reinterpret_cast<SetCalInfoAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetCalInfoAttribute"))
    );
    function_pointers_
        .SetCalInfoAttributeDouble = reinterpret_cast<SetCalInfoAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetCalInfoAttribute"))
    );
    function_pointers_
        .SetCalInfoAttributeString = reinterpret_cast<SetCalInfoAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetCalInfoAttribute"))
    );
    function_pointers_
        .SetCalInfoAttributeUInt32 = reinterpret_cast<SetCalInfoAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetCalInfoAttribute"))
    );
    function_pointers_.SetChanAttributeBool = reinterpret_cast<SetChanAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetChanAttribute"))
    );
    function_pointers_
        .SetChanAttributeDouble = reinterpret_cast<SetChanAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetChanAttribute"))
    );
    function_pointers_
        .SetChanAttributeDoubleArray = reinterpret_cast<SetChanAttributeDoubleArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetChanAttribute"))
    );
    function_pointers_
        .SetChanAttributeInt32 = reinterpret_cast<SetChanAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetChanAttribute"))
    );
    function_pointers_
        .SetChanAttributeString = reinterpret_cast<SetChanAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetChanAttribute"))
    );
    function_pointers_
        .SetChanAttributeUInt32 = reinterpret_cast<SetChanAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetChanAttribute"))
    );
    function_pointers_.SetDigitalLogicFamilyPowerUpState = reinterpret_cast<
        SetDigitalLogicFamilyPowerUpStatePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetDigitalLogicFamilyPowerUpState"))
    );
    function_pointers_
        .SetDigitalPowerUpStates = reinterpret_cast<SetDigitalPowerUpStatesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetDigitalPowerUpStates"))
    );
    function_pointers_.SetDigitalPullUpPullDownStates = reinterpret_cast<
        SetDigitalPullUpPullDownStatesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetDigitalPullUpPullDownStates"))
    );
    function_pointers_.SetExportedSignalAttributeBool = reinterpret_cast<
        SetExportedSignalAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetExportedSignalAttribute"))
    );
    function_pointers_.SetExportedSignalAttributeDouble = reinterpret_cast<
        SetExportedSignalAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetExportedSignalAttribute"))
    );
    function_pointers_.SetExportedSignalAttributeInt32 = reinterpret_cast<
        SetExportedSignalAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetExportedSignalAttribute"))
    );
    function_pointers_.SetExportedSignalAttributeString = reinterpret_cast<
        SetExportedSignalAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetExportedSignalAttribute"))
    );
    function_pointers_.SetExportedSignalAttributeUInt32 = reinterpret_cast<
        SetExportedSignalAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetExportedSignalAttribute"))
    );
    function_pointers_.SetFirstSampClkWhen = reinterpret_cast<SetFirstSampClkWhenPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetFirstSampClkWhen"))
    );
    function_pointers_.SetReadAttributeBool = reinterpret_cast<SetReadAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadAttribute"))
    );
    function_pointers_
        .SetReadAttributeDouble = reinterpret_cast<SetReadAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadAttribute"))
    );
    function_pointers_
        .SetReadAttributeInt32 = reinterpret_cast<SetReadAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadAttribute"))
    );
    function_pointers_
        .SetReadAttributeString = reinterpret_cast<SetReadAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadAttribute"))
    );
    function_pointers_
        .SetReadAttributeUInt32 = reinterpret_cast<SetReadAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadAttribute"))
    );
    function_pointers_
        .SetReadAttributeUInt64 = reinterpret_cast<SetReadAttributeUInt64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadAttribute"))
    );
    function_pointers_
        .SetRealTimeAttributeBool = reinterpret_cast<SetRealTimeAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetRealTimeAttribute"))
    );
    function_pointers_
        .SetRealTimeAttributeInt32 = reinterpret_cast<SetRealTimeAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetRealTimeAttribute"))
    );
    function_pointers_
        .SetRealTimeAttributeUInt32 = reinterpret_cast<SetRealTimeAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetRealTimeAttribute"))
    );
    function_pointers_
        .SetRuntimeEnvironment = reinterpret_cast<SetRuntimeEnvironmentPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetRuntimeEnvironment"))
    );
    function_pointers_
        .SetScaleAttributeDouble = reinterpret_cast<SetScaleAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetScaleAttribute"))
    );
    function_pointers_.SetScaleAttributeDoubleArray = reinterpret_cast<
        SetScaleAttributeDoubleArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetScaleAttribute"))
    );
    function_pointers_
        .SetScaleAttributeInt32 = reinterpret_cast<SetScaleAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetScaleAttribute"))
    );
    function_pointers_
        .SetScaleAttributeString = reinterpret_cast<SetScaleAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetScaleAttribute"))
    );
    function_pointers_.SetStartTrigTrigWhen = reinterpret_cast<SetStartTrigTrigWhenPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetStartTrigTrigWhen"))
    );
    function_pointers_.SetSyncPulseTimeWhen = reinterpret_cast<SetSyncPulseTimeWhenPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetSyncPulseTimeWhen"))
    );
    function_pointers_
        .SetTimingAttributeBool = reinterpret_cast<SetTimingAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_
        .SetTimingAttributeDouble = reinterpret_cast<SetTimingAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_
        .SetTimingAttributeExBool = reinterpret_cast<SetTimingAttributeExBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_
        .SetTimingAttributeExDouble = reinterpret_cast<SetTimingAttributeExDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_
        .SetTimingAttributeExInt32 = reinterpret_cast<SetTimingAttributeExInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_
        .SetTimingAttributeExString = reinterpret_cast<SetTimingAttributeExStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_.SetTimingAttributeExTimestamp = reinterpret_cast<
        SetTimingAttributeExTimestampPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_
        .SetTimingAttributeExUInt32 = reinterpret_cast<SetTimingAttributeExUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_
        .SetTimingAttributeExUInt64 = reinterpret_cast<SetTimingAttributeExUInt64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_
        .SetTimingAttributeInt32 = reinterpret_cast<SetTimingAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_
        .SetTimingAttributeString = reinterpret_cast<SetTimingAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_
        .SetTimingAttributeTimestamp = reinterpret_cast<SetTimingAttributeTimestampPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_
        .SetTimingAttributeUInt32 = reinterpret_cast<SetTimingAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_
        .SetTimingAttributeUInt64 = reinterpret_cast<SetTimingAttributeUInt64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_.SetTrigAttributeBool = reinterpret_cast<SetTrigAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeDouble = reinterpret_cast<SetTrigAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeDoubleArray = reinterpret_cast<SetTrigAttributeDoubleArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeInt32 = reinterpret_cast<SetTrigAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeInt32Array = reinterpret_cast<SetTrigAttributeInt32ArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeString = reinterpret_cast<SetTrigAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeTimestamp = reinterpret_cast<SetTrigAttributeTimestampPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeUInt32 = reinterpret_cast<SetTrigAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetWatchdogAttributeBool = reinterpret_cast<SetWatchdogAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWatchdogAttribute"))
    );
    function_pointers_
        .SetWatchdogAttributeDouble = reinterpret_cast<SetWatchdogAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWatchdogAttribute"))
    );
    function_pointers_
        .SetWatchdogAttributeInt32 = reinterpret_cast<SetWatchdogAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWatchdogAttribute"))
    );
    function_pointers_
        .SetWatchdogAttributeString = reinterpret_cast<SetWatchdogAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWatchdogAttribute"))
    );
    function_pointers_
        .SetWriteAttributeBool = reinterpret_cast<SetWriteAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWriteAttribute"))
    );
    function_pointers_
        .SetWriteAttributeDouble = reinterpret_cast<SetWriteAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWriteAttribute"))
    );
    function_pointers_
        .SetWriteAttributeInt32 = reinterpret_cast<SetWriteAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWriteAttribute"))
    );
    function_pointers_
        .SetWriteAttributeString = reinterpret_cast<SetWriteAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWriteAttribute"))
    );
    function_pointers_
        .SetWriteAttributeUInt32 = reinterpret_cast<SetWriteAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWriteAttribute"))
    );
    function_pointers_
        .SetWriteAttributeUInt64 = reinterpret_cast<SetWriteAttributeUInt64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWriteAttribute"))
    );
    function_pointers_.StartNewFile = reinterpret_cast<StartNewFilePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxStartNewFile"))
    );
    function_pointers_.StartTask = reinterpret_cast<StartTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxStartTask"))
    );
    function_pointers_.StopTask = reinterpret_cast<StopTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxStopTask"))
    );
    function_pointers_.TaskControl = reinterpret_cast<TaskControlPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxTaskControl"))
    );
    function_pointers_.TristateOutputTerm = reinterpret_cast<TristateOutputTermPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxTristateOutputTerm"))
    );
    function_pointers_.UnregisterDoneEvent = reinterpret_cast<UnregisterDoneEventPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRegisterDoneEvent"))
    );
    function_pointers_.UnregisterEveryNSamplesEvent = reinterpret_cast<
        UnregisterEveryNSamplesEventPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRegisterEveryNSamplesEvent"))
    );
    function_pointers_.UnregisterSignalEvent = reinterpret_cast<RegisterSignalEventPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRegisterSignalEvent"))
    );
    function_pointers_
        .RemoveCDAQSyncConnection = reinterpret_cast<RemoveCDAQSyncConnectionPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRemoveCDAQSyncConnection"))
    );
    function_pointers_.ReserveNetworkDevice = reinterpret_cast<ReserveNetworkDevicePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxReserveNetworkDevice"))
    );
    function_pointers_.ResetBufferAttribute = reinterpret_cast<ResetBufferAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetBufferAttribute"))
    );
    function_pointers_.ResetChanAttribute = reinterpret_cast<ResetChanAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetChanAttribute"))
    );
    function_pointers_.ResetDevice = reinterpret_cast<ResetDevicePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetDevice"))
    );
    function_pointers_.ResetExportedSignalAttribute = reinterpret_cast<
        ResetExportedSignalAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetExportedSignalAttribute"))
    );
    function_pointers_.ResetReadAttribute = reinterpret_cast<ResetReadAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetReadAttribute"))
    );
    function_pointers_
        .ResetRealTimeAttribute = reinterpret_cast<ResetRealTimeAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetRealTimeAttribute"))
    );
    function_pointers_.ResetTimingAttribute = reinterpret_cast<ResetTimingAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetTimingAttribute"))
    );
    function_pointers_
        .ResetTimingAttributeEx = reinterpret_cast<ResetTimingAttributeExPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetTimingAttributeEx"))
    );
    function_pointers_.ResetTrigAttribute = reinterpret_cast<ResetTrigAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetTrigAttribute"))
    );
    function_pointers_
        .ResetWatchdogAttribute = reinterpret_cast<ResetWatchdogAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetWatchdogAttribute"))
    );
    function_pointers_.ResetWriteAttribute = reinterpret_cast<ResetWriteAttributePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxResetWriteAttribute"))
    );
    function_pointers_
        .RestoreLastExtCalConst = reinterpret_cast<RestoreLastExtCalConstPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRestoreLastExtCalConst"))
    );
    function_pointers_.SaveGlobalChan = reinterpret_cast<SaveGlobalChanPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSaveGlobalChan"))
    );
    function_pointers_.SaveScale = reinterpret_cast<SaveScalePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSaveScale"))
    );
    function_pointers_.SaveTask = reinterpret_cast<SaveTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSaveTask"))
    );
    function_pointers_.SelfCal = reinterpret_cast<SelfCalPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSelfCal"))
    );
    function_pointers_.SelfTestDevice = reinterpret_cast<SelfTestDevicePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSelfTestDevice"))
    );
    function_pointers_.SetAIChanCalCalDate = reinterpret_cast<SetAIChanCalCalDatePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetAIChanCalCalDate"))
    );
    function_pointers_.SetAIChanCalExpDate = reinterpret_cast<SetAIChanCalExpDatePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetAIChanCalExpDate"))
    );
    function_pointers_
        .SetAnalogPowerUpStates = reinterpret_cast<SetAnalogPowerUpStatesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetAnalogPowerUpStates"))
    );
    function_pointers_.SetAnalogPowerUpStatesWithOutputType = reinterpret_cast<
        SetAnalogPowerUpStatesWithOutputTypePtr>(const_cast<void *>(
        lib->get_func_ptr("DAQmxSetAnalogPowerUpStatesWithOutputType")
    ));
    function_pointers_
        .SetArmStartTrigTrigWhen = reinterpret_cast<SetArmStartTrigTrigWhenPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetArmStartTrigTrigWhen"))
    );
    function_pointers_
        .SetBufferAttributeUInt32 = reinterpret_cast<SetBufferAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetBufferAttribute"))
    );
    function_pointers_
        .SetCalInfoAttributeBool = reinterpret_cast<SetCalInfoAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetCalInfoAttribute"))
    );
    function_pointers_
        .SetCalInfoAttributeDouble = reinterpret_cast<SetCalInfoAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetCalInfoAttribute"))
    );
    function_pointers_
        .SetCalInfoAttributeString = reinterpret_cast<SetCalInfoAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetCalInfoAttribute"))
    );
    function_pointers_
        .SetCalInfoAttributeUInt32 = reinterpret_cast<SetCalInfoAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetCalInfoAttribute"))
    );
    function_pointers_.SetChanAttributeBool = reinterpret_cast<SetChanAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetChanAttribute"))
    );
    function_pointers_
        .SetChanAttributeDouble = reinterpret_cast<SetChanAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetChanAttribute"))
    );
    function_pointers_
        .SetChanAttributeDoubleArray = reinterpret_cast<SetChanAttributeDoubleArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetChanAttribute"))
    );
    function_pointers_
        .SetChanAttributeInt32 = reinterpret_cast<SetChanAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetChanAttribute"))
    );
    function_pointers_
        .SetChanAttributeString = reinterpret_cast<SetChanAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetChanAttribute"))
    );
    function_pointers_
        .SetChanAttributeUInt32 = reinterpret_cast<SetChanAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetChanAttribute"))
    );
    function_pointers_.SetDigitalLogicFamilyPowerUpState = reinterpret_cast<
        SetDigitalLogicFamilyPowerUpStatePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetDigitalLogicFamilyPowerUpState"))
    );
    function_pointers_
        .SetDigitalPowerUpStates = reinterpret_cast<SetDigitalPowerUpStatesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetDigitalPowerUpStates"))
    );
    function_pointers_.SetDigitalPullUpPullDownStates = reinterpret_cast<
        SetDigitalPullUpPullDownStatesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetDigitalPullUpPullDownStates"))
    );
    function_pointers_.SetExportedSignalAttributeBool = reinterpret_cast<
        SetExportedSignalAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetExportedSignalAttribute"))
    );
    function_pointers_.SetExportedSignalAttributeDouble = reinterpret_cast<
        SetExportedSignalAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetExportedSignalAttribute"))
    );
    function_pointers_.SetExportedSignalAttributeInt32 = reinterpret_cast<
        SetExportedSignalAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetExportedSignalAttribute"))
    );
    function_pointers_.SetExportedSignalAttributeString = reinterpret_cast<
        SetExportedSignalAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetExportedSignalAttribute"))
    );
    function_pointers_.SetExportedSignalAttributeUInt32 = reinterpret_cast<
        SetExportedSignalAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetExportedSignalAttribute"))
    );
    function_pointers_.SetFirstSampClkWhen = reinterpret_cast<SetFirstSampClkWhenPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetFirstSampClkWhen"))
    );
    function_pointers_.SetReadAttributeBool = reinterpret_cast<SetReadAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadAttribute"))
    );
    function_pointers_
        .SetReadAttributeDouble = reinterpret_cast<SetReadAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadAttribute"))
    );
    function_pointers_
        .SetReadAttributeInt32 = reinterpret_cast<SetReadAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadAttribute"))
    );
    function_pointers_
        .SetReadAttributeString = reinterpret_cast<SetReadAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadAttribute"))
    );
    function_pointers_
        .SetReadAttributeUInt32 = reinterpret_cast<SetReadAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadAttribute"))
    );
    function_pointers_
        .SetReadAttributeUInt64 = reinterpret_cast<SetReadAttributeUInt64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadAttribute"))
    );
    function_pointers_
        .SetRealTimeAttributeBool = reinterpret_cast<SetRealTimeAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetRealTimeAttribute"))
    );
    function_pointers_
        .SetRealTimeAttributeInt32 = reinterpret_cast<SetRealTimeAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetRealTimeAttribute"))
    );
    function_pointers_
        .SetRealTimeAttributeUInt32 = reinterpret_cast<SetRealTimeAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetRealTimeAttribute"))
    );
    function_pointers_
        .SetRuntimeEnvironment = reinterpret_cast<SetRuntimeEnvironmentPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetRuntimeEnvironment"))
    );
    function_pointers_
        .SetScaleAttributeDouble = reinterpret_cast<SetScaleAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetScaleAttribute"))
    );
    function_pointers_.SetScaleAttributeDoubleArray = reinterpret_cast<
        SetScaleAttributeDoubleArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetScaleAttribute"))
    );
    function_pointers_
        .SetScaleAttributeInt32 = reinterpret_cast<SetScaleAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetScaleAttribute"))
    );
    function_pointers_
        .SetScaleAttributeString = reinterpret_cast<SetScaleAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetScaleAttribute"))
    );
    function_pointers_.SetStartTrigTrigWhen = reinterpret_cast<SetStartTrigTrigWhenPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetStartTrigTrigWhen"))
    );
    function_pointers_.SetSyncPulseTimeWhen = reinterpret_cast<SetSyncPulseTimeWhenPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetSyncPulseTimeWhen"))
    );
    function_pointers_
        .SetTimingAttributeBool = reinterpret_cast<SetTimingAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_
        .SetTimingAttributeDouble = reinterpret_cast<SetTimingAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_
        .SetTimingAttributeExBool = reinterpret_cast<SetTimingAttributeExBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_
        .SetTimingAttributeExDouble = reinterpret_cast<SetTimingAttributeExDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_
        .SetTimingAttributeExInt32 = reinterpret_cast<SetTimingAttributeExInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_
        .SetTimingAttributeExString = reinterpret_cast<SetTimingAttributeExStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_.SetTimingAttributeExTimestamp = reinterpret_cast<
        SetTimingAttributeExTimestampPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_
        .SetTimingAttributeExUInt32 = reinterpret_cast<SetTimingAttributeExUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_
        .SetTimingAttributeExUInt64 = reinterpret_cast<SetTimingAttributeExUInt64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttributeEx"))
    );
    function_pointers_
        .SetTimingAttributeInt32 = reinterpret_cast<SetTimingAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_
        .SetTimingAttributeString = reinterpret_cast<SetTimingAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_
        .SetTimingAttributeTimestamp = reinterpret_cast<SetTimingAttributeTimestampPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_
        .SetTimingAttributeUInt32 = reinterpret_cast<SetTimingAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_
        .SetTimingAttributeUInt64 = reinterpret_cast<SetTimingAttributeUInt64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTimingAttribute"))
    );
    function_pointers_.SetTrigAttributeBool = reinterpret_cast<SetTrigAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeDouble = reinterpret_cast<SetTrigAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeDoubleArray = reinterpret_cast<SetTrigAttributeDoubleArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeInt32 = reinterpret_cast<SetTrigAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeInt32Array = reinterpret_cast<SetTrigAttributeInt32ArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeString = reinterpret_cast<SetTrigAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeTimestamp = reinterpret_cast<SetTrigAttributeTimestampPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetTrigAttributeUInt32 = reinterpret_cast<SetTrigAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetTrigAttribute"))
    );
    function_pointers_
        .SetWatchdogAttributeBool = reinterpret_cast<SetWatchdogAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWatchdogAttribute"))
    );
    function_pointers_
        .SetWatchdogAttributeDouble = reinterpret_cast<SetWatchdogAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWatchdogAttribute"))
    );
    function_pointers_
        .SetWatchdogAttributeInt32 = reinterpret_cast<SetWatchdogAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWatchdogAttribute"))
    );
    function_pointers_
        .SetWatchdogAttributeString = reinterpret_cast<SetWatchdogAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWatchdogAttribute"))
    );
    function_pointers_
        .SetWriteAttributeBool = reinterpret_cast<SetWriteAttributeBoolPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWriteAttribute"))
    );
    function_pointers_
        .SetWriteAttributeDouble = reinterpret_cast<SetWriteAttributeDoublePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWriteAttribute"))
    );
    function_pointers_
        .SetWriteAttributeInt32 = reinterpret_cast<SetWriteAttributeInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWriteAttribute"))
    );
    function_pointers_
        .SetWriteAttributeString = reinterpret_cast<SetWriteAttributeStringPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWriteAttribute"))
    );
    function_pointers_
        .SetWriteAttributeUInt32 = reinterpret_cast<SetWriteAttributeUInt32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWriteAttribute"))
    );
    function_pointers_
        .SetWriteAttributeUInt64 = reinterpret_cast<SetWriteAttributeUInt64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetWriteAttribute"))
    );
    function_pointers_.StartNewFile = reinterpret_cast<StartNewFilePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxStartNewFile"))
    );
    function_pointers_.StartTask = reinterpret_cast<StartTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxStartTask"))
    );
    function_pointers_.StopTask = reinterpret_cast<StopTaskPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxStopTask"))
    );
    function_pointers_.TaskControl = reinterpret_cast<TaskControlPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxTaskControl"))
    );
    function_pointers_.TristateOutputTerm = reinterpret_cast<TristateOutputTermPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxTristateOutputTerm"))
    );
    function_pointers_.UnregisterDoneEvent = reinterpret_cast<UnregisterDoneEventPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRegisterDoneEvent"))
    );
    function_pointers_.UnregisterEveryNSamplesEvent = reinterpret_cast<
        UnregisterEveryNSamplesEventPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRegisterEveryNSamplesEvent"))
    );
    function_pointers_
        .UnregisterSignalEvent = reinterpret_cast<UnregisterSignalEventPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxRegisterSignalEvent"))
    );
    function_pointers_
        .UnreserveNetworkDevice = reinterpret_cast<UnreserveNetworkDevicePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxUnreserveNetworkDevice"))
    );
    function_pointers_
        .WaitForNextSampleClock = reinterpret_cast<WaitForNextSampleClockPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWaitForNextSampleClock"))
    );
    function_pointers_
        .WaitForValidTimestamp = reinterpret_cast<WaitForValidTimestampPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWaitForValidTimestamp"))
    );
    function_pointers_.WaitUntilTaskDone = reinterpret_cast<WaitUntilTaskDonePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWaitUntilTaskDone"))
    );
    function_pointers_.WriteAnalogF64 = reinterpret_cast<WriteAnalogF64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteAnalogF64"))
    );
    function_pointers_.WriteAnalogScalarF64 = reinterpret_cast<WriteAnalogScalarF64Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteAnalogScalarF64"))
    );
    function_pointers_.WriteBinaryI16 = reinterpret_cast<WriteBinaryI16Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteBinaryI16"))
    );
    function_pointers_.WriteBinaryI32 = reinterpret_cast<WriteBinaryI32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteBinaryI32"))
    );
    function_pointers_.WriteBinaryU16 = reinterpret_cast<WriteBinaryU16Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteBinaryU16"))
    );
    function_pointers_.WriteBinaryU32 = reinterpret_cast<WriteBinaryU32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteBinaryU32"))
    );
    function_pointers_.WriteCtrFreq = reinterpret_cast<WriteCtrFreqPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteCtrFreq"))
    );
    function_pointers_.WriteCtrFreqScalar = reinterpret_cast<WriteCtrFreqScalarPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteCtrFreqScalar"))
    );
    function_pointers_.WriteCtrTicks = reinterpret_cast<WriteCtrTicksPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteCtrTicks"))
    );
    function_pointers_.WriteCtrTicksScalar = reinterpret_cast<WriteCtrTicksScalarPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteCtrTicksScalar"))
    );
    function_pointers_.WriteCtrTime = reinterpret_cast<WriteCtrTimePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteCtrTime"))
    );
    function_pointers_.WriteCtrTimeScalar = reinterpret_cast<WriteCtrTimeScalarPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteCtrTimeScalar"))
    );
    function_pointers_.WriteDigitalLines = reinterpret_cast<WriteDigitalLinesPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteDigitalLines"))
    );
    function_pointers_
        .WriteDigitalScalarU32 = reinterpret_cast<WriteDigitalScalarU32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteDigitalScalarU32"))
    );
    function_pointers_.WriteDigitalU16 = reinterpret_cast<WriteDigitalU16Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteDigitalU16"))
    );
    function_pointers_.WriteDigitalU32 = reinterpret_cast<WriteDigitalU32Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteDigitalU32"))
    );
    function_pointers_.WriteDigitalU8 = reinterpret_cast<WriteDigitalU8Ptr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteDigitalU8"))
    );
    function_pointers_.WriteRaw = reinterpret_cast<WriteRawPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteRaw"))
    );
    function_pointers_.WriteToTEDSFromArray = reinterpret_cast<WriteToTEDSFromArrayPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteToTEDSFromArray"))
    );
    function_pointers_.WriteToTEDSFromFile = reinterpret_cast<WriteToTEDSFromFilePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxWriteToTEDSFromFile"))
    );
    function_pointers_.SetReadRelativeTo = reinterpret_cast<SetReadRelativeToPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadRelativeTo"))
    );
    function_pointers_.SetReadOffset = reinterpret_cast<SetReadOffsetPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadOffset"))
    );
    function_pointers_.SetReadOverWrite = reinterpret_cast<SetReadOverWritePtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxSetReadOverWrite"))
    );
    function_pointers_.GetReadTotalSampPerChanAcquired = reinterpret_cast<
        GetReadTotalSampPerChanAcquiredPtr>(
        const_cast<void *>(lib->get_func_ptr("DAQmxGetReadTotalSampPerChanAcquired"))
    );
}

ProdAPI::~ProdAPI() {}

int32 ProdAPI::AddCDAQSyncConnection(const char portList[]) {
    return function_pointers_.AddCDAQSyncConnection(portList);
}

int32 ProdAPI::AddGlobalChansToTask(TaskHandle task, const char channelNames[]) {
    return function_pointers_.AddGlobalChansToTask(task, channelNames);
}

int32 ProdAPI::AddNetworkDevice(
    const char ipAddress[],
    const char deviceName[],
    bool32 attemptReservation,
    float64 timeout,
    char deviceNameOut[],
    uInt32 deviceNameOutBufferSize
) {
    return function_pointers_.AddNetworkDevice(
        ipAddress,
        deviceName,
        attemptReservation,
        timeout,
        deviceNameOut,
        deviceNameOutBufferSize
    );
}

int32 ProdAPI::AreConfiguredCDAQSyncPortsDisconnected(
    const char chassisDevicesPorts[],
    float64 timeout,
    bool32 *disconnectedPortsExist
) {
    return function_pointers_.AreConfiguredCDAQSyncPortsDisconnected(
        chassisDevicesPorts,
        timeout,
        disconnectedPortsExist
    );
}

int32 ProdAPI::AutoConfigureCDAQSyncConnections(
    const char chassisDevicesPorts[],
    float64 timeout
) {
    return function_pointers_.AutoConfigureCDAQSyncConnections(
        chassisDevicesPorts,
        timeout
    );
}

int32 ProdAPI::CalculateReversePolyCoeff(
    const float64 forwardCoeffs[],
    uInt32 numForwardCoeffsIn,
    float64 minValX,
    float64 maxValX,
    int32 numPointsToCompute,
    int32 reversePolyOrder,
    float64 reverseCoeffs[]
) {
    return function_pointers_.CalculateReversePolyCoeff(
        forwardCoeffs,
        numForwardCoeffsIn,
        minValX,
        maxValX,
        numPointsToCompute,
        reversePolyOrder,
        reverseCoeffs
    );
}

int32 ProdAPI::CfgAnlgEdgeRefTrig(
    TaskHandle task,
    const char triggerSource[],
    int32 triggerSlope,
    float64 triggerLevel,
    uInt32 pretriggerSamples
) {
    return function_pointers_.CfgAnlgEdgeRefTrig(
        task,
        triggerSource,
        triggerSlope,
        triggerLevel,
        pretriggerSamples
    );
}

int32 ProdAPI::CfgAnlgEdgeStartTrig(
    TaskHandle task,
    const char triggerSource[],
    int32 triggerSlope,
    float64 triggerLevel
) {
    return function_pointers_
        .CfgAnlgEdgeStartTrig(task, triggerSource, triggerSlope, triggerLevel);
}

int32 ProdAPI::CfgAnlgMultiEdgeRefTrig(
    TaskHandle task,
    const char triggerSources[],
    const int32 triggerSlopeArray[],
    const float64 triggerLevelArray[],
    uInt32 pretriggerSamples,
    uInt32 arraySize
) {
    int32 *slopeArray = const_cast<int32 *>(triggerSlopeArray);
    float64 *levelArray = const_cast<float64 *>(triggerLevelArray);

    return function_pointers_.CfgAnlgMultiEdgeRefTrig(
        task,
        triggerSources,
        slopeArray,
        levelArray,
        pretriggerSamples,
        arraySize
    );
}

int32 ProdAPI::CfgAnlgMultiEdgeStartTrig(
    TaskHandle task,
    const char triggerSources[],
    const int32 triggerSlopeArray[],
    const float64 triggerLevelArray[],
    uInt32 arraySize
) {
    int32 *slopeArray = const_cast<int32 *>(triggerSlopeArray);
    float64 *levelArray = const_cast<float64 *>(triggerLevelArray);

    return function_pointers_.CfgAnlgMultiEdgeStartTrig(
        task,
        triggerSources,
        slopeArray,
        levelArray,
        arraySize
    );
}

int32 ProdAPI::CfgAnlgWindowRefTrig(
    TaskHandle task,
    const char triggerSource[],
    int32 triggerWhen,
    float64 windowTop,
    float64 windowBottom,
    uInt32 pretriggerSamples
) {
    return function_pointers_.CfgAnlgWindowRefTrig(
        task,
        triggerSource,
        triggerWhen,
        windowTop,
        windowBottom,
        pretriggerSamples
    );
}

int32 ProdAPI::CfgAnlgWindowStartTrig(
    TaskHandle task,
    const char triggerSource[],
    int32 triggerWhen,
    float64 windowTop,
    float64 windowBottom
) {
    return function_pointers_.CfgAnlgWindowStartTrig(
        task,
        triggerSource,
        triggerWhen,
        windowTop,
        windowBottom
    );
}

int32 ProdAPI::CfgBurstHandshakingTimingExportClock(
    TaskHandle task,
    int32 sampleMode,
    uInt64 sampsPerChan,
    float64 sampleClkRate,
    const char sampleClkOutpTerm[],
    int32 sampleClkPulsePolarity,
    int32 pauseWhen,
    int32 readyEventActiveLevel
) {
    return function_pointers_.CfgBurstHandshakingTimingExportClock(
        task,
        sampleMode,
        sampsPerChan,
        sampleClkRate,
        sampleClkOutpTerm,
        sampleClkPulsePolarity,
        pauseWhen,
        readyEventActiveLevel
    );
}

int32 ProdAPI::CfgBurstHandshakingTimingImportClock(
    TaskHandle task,
    int32 sampleMode,
    uInt64 sampsPerChan,
    float64 sampleClkRate,
    const char sampleClkSrc[],
    int32 sampleClkActiveEdge,
    int32 pauseWhen,
    int32 readyEventActiveLevel
) {
    return function_pointers_.CfgBurstHandshakingTimingImportClock(
        task,
        sampleMode,
        sampsPerChan,
        sampleClkRate,
        sampleClkSrc,
        sampleClkActiveEdge,
        pauseWhen,
        readyEventActiveLevel
    );
}

int32 ProdAPI::CfgChangeDetectionTiming(
    TaskHandle task,
    const char risingEdgeChan[],
    const char fallingEdgeChan[],
    int32 sampleMode,
    uInt64 sampsPerChan
) {
    return function_pointers_.CfgChangeDetectionTiming(
        task,
        risingEdgeChan,
        fallingEdgeChan,
        sampleMode,
        sampsPerChan
    );
}

int32 ProdAPI::CfgDigEdgeRefTrig(
    TaskHandle task,
    const char triggerSource[],
    int32 triggerEdge,
    uInt32 pretriggerSamples
) {
    return function_pointers_
        .CfgDigEdgeRefTrig(task, triggerSource, triggerEdge, pretriggerSamples);
}

int32 ProdAPI::CfgDigEdgeStartTrig(
    TaskHandle task,
    const char triggerSource[],
    int32 triggerEdge
) {
    return function_pointers_.CfgDigEdgeStartTrig(task, triggerSource, triggerEdge);
}

int32 ProdAPI::CfgDigPatternRefTrig(
    TaskHandle task,
    const char triggerSource[],
    const char triggerPattern[],
    int32 triggerWhen,
    uInt32 pretriggerSamples
) {
    return function_pointers_.CfgDigPatternRefTrig(
        task,
        triggerSource,
        triggerPattern,
        triggerWhen,
        pretriggerSamples
    );
}

int32 ProdAPI::CfgDigPatternStartTrig(
    TaskHandle task,
    const char triggerSource[],
    const char triggerPattern[],
    int32 triggerWhen
) {
    return function_pointers_
        .CfgDigPatternStartTrig(task, triggerSource, triggerPattern, triggerWhen);
}

int32 ProdAPI::CfgHandshakingTiming(
    TaskHandle task,
    int32 sampleMode,
    uInt64 sampsPerChan
) {
    return function_pointers_.CfgHandshakingTiming(task, sampleMode, sampsPerChan);
}

int32 ProdAPI::CfgImplicitTiming(
    TaskHandle task,
    int32 sampleMode,
    uInt64 sampsPerChan
) {
    return function_pointers_.CfgImplicitTiming(task, sampleMode, sampsPerChan);
}

int32 ProdAPI::CfgInputBuffer(TaskHandle task, uInt32 numSampsPerChan) {
    return function_pointers_.CfgInputBuffer(task, numSampsPerChan);
}

int32 ProdAPI::CfgOutputBuffer(TaskHandle task, uInt32 numSampsPerChan) {
    return function_pointers_.CfgOutputBuffer(task, numSampsPerChan);
}

int32 ProdAPI::CfgPipelinedSampClkTiming(
    TaskHandle task,
    const char source[],
    float64 rate,
    int32 activeEdge,
    int32 sampleMode,
    uInt64 sampsPerChan
) {
    return function_pointers_.CfgPipelinedSampClkTiming(
        task,
        source,
        rate,
        activeEdge,
        sampleMode,
        sampsPerChan
    );
}

int32 ProdAPI::CfgSampClkTiming(
    TaskHandle task,
    const char source[],
    float64 rate,
    int32 activeEdge,
    int32 sampleMode,
    uInt64 sampsPerChan
) {
    return function_pointers_
        .CfgSampClkTiming(task, source, rate, activeEdge, sampleMode, sampsPerChan);
}

int32 ProdAPI::CfgTimeStartTrig(
    TaskHandle task,
    CVIAbsoluteTime when,
    int32 timescale
) {
    return function_pointers_.CfgTimeStartTrig(task, when, timescale);
}

int32 ProdAPI::CfgWatchdogAOExpirStates(
    TaskHandle task,
    const char channelNames[],
    const float64 expirStateArray[],
    const int32 outputTypeArray[],
    uInt32 arraySize
) {
    return function_pointers_.CfgWatchdogAOExpirStates(
        task,
        channelNames,
        expirStateArray,
        outputTypeArray,
        arraySize
    );
}

int32 ProdAPI::CfgWatchdogCOExpirStates(
    TaskHandle task,
    const char channelNames[],
    const int32 expirStateArray[],
    uInt32 arraySize
) {
    return function_pointers_
        .CfgWatchdogCOExpirStates(task, channelNames, expirStateArray, arraySize);
}

int32 ProdAPI::CfgWatchdogDOExpirStates(
    TaskHandle task,
    const char channelNames[],
    const int32 expirStateArray[],
    uInt32 arraySize
) {
    return function_pointers_
        .CfgWatchdogDOExpirStates(task, channelNames, expirStateArray, arraySize);
}

int32 ProdAPI::ClearTEDS(const char physicalChannel[]) {
    return function_pointers_.ClearTEDS(physicalChannel);
}

int32 ProdAPI::ClearTask(TaskHandle task) {
    return function_pointers_.ClearTask(task);
}

int32 ProdAPI::ConfigureLogging(
    TaskHandle task,
    const char filePath[],
    int32 loggingMode,
    const char groupName[],
    int32 operation
) {
    return function_pointers_
        .ConfigureLogging(task, filePath, loggingMode, groupName, operation);
}

int32 ProdAPI::ConfigureTEDS(const char physicalChannel[], const char filePath[]) {
    return function_pointers_.ConfigureTEDS(physicalChannel, filePath);
}

int32 ProdAPI::ConnectTerms(
    const char sourceTerminal[],
    const char destinationTerminal[],
    int32 signalModifiers
) {
    return function_pointers_
        .ConnectTerms(sourceTerminal, destinationTerminal, signalModifiers);
}

int32 ProdAPI::ControlWatchdogTask(TaskHandle task, int32 action) {
    return function_pointers_.ControlWatchdogTask(task, action);
}

int32 ProdAPI::CreateAIAccel4WireDCVoltageChan(
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
) {
    return function_pointers_.CreateAIAccel4WireDCVoltageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        voltageExcitSource,
        voltageExcitVal,
        useExcitForScaling,
        customScaleName
    );
}

int32 ProdAPI::CreateAIAccelChan(
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
) {
    return function_pointers_.CreateAIAccelChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateAIAccelChargeChan(
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
) {
    return function_pointers_.CreateAIAccelChargeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        customScaleName
    );
}

int32 ProdAPI::CreateAIBridgeChan(
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
) {
    return function_pointers_.CreateAIBridgeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        customScaleName
    );
}

int32 ProdAPI::CreateAIChargeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return function_pointers_.CreateAIChargeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        customScaleName
    );
}

int32 ProdAPI::CreateAICurrentChan(
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
) {
    return function_pointers_.CreateAICurrentChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        shuntResistorLoc,
        extShuntResistorVal,
        customScaleName
    );
}

int32 ProdAPI::CreateAICurrentRMSChan(
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
) {
    return function_pointers_.CreateAICurrentRMSChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        shuntResistorLoc,
        extShuntResistorVal,
        customScaleName
    );
}

int32 ProdAPI::CreateAIForceBridgePolynomialChan(
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
) {
    return function_pointers_.CreateAIForceBridgePolynomialChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        forwardCoeffs,
        numForwardCoeffs,
        reverseCoeffs,
        numReverseCoeffs,
        electricalUnits,
        physicalUnits,
        customScaleName
    );
}

int32 ProdAPI::CreateAIForceBridgeTableChan(
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
) {
    return function_pointers_.CreateAIForceBridgeTableChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        electricalVals,
        numElectricalVals,
        electricalUnits,
        physicalVals,
        numPhysicalVals,
        physicalUnits,
        customScaleName
    );
}

int32 ProdAPI::CreateAIForceBridgeTwoPointLinChan(
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
) {
    return function_pointers_.CreateAIForceBridgeTwoPointLinChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        firstElectricalVal,
        secondElectricalVal,
        electricalUnits,
        firstPhysicalVal,
        secondPhysicalVal,
        physicalUnits,
        customScaleName
    );
}

int32 ProdAPI::CreateAIForceIEPEChan(
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
) {
    return function_pointers_.CreateAIForceIEPEChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateAIFreqVoltageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    float64 thresholdLevel,
    float64 hysteresis,
    const char customScaleName[]
) {
    return function_pointers_.CreateAIFreqVoltageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        thresholdLevel,
        hysteresis,
        customScaleName
    );
}

int32 ProdAPI::CreateAIMicrophoneChan(
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
) {
    return function_pointers_.CreateAIMicrophoneChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        units,
        micSensitivity,
        maxSndPressLevel,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateAIPosEddyCurrProxProbeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    float64 sensitivity,
    int32 sensitivityUnits,
    const char customScaleName[]
) {
    return function_pointers_.CreateAIPosEddyCurrProxProbeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        customScaleName
    );
}

int32 ProdAPI::CreateAIPosLVDTChan(
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
) {
    return function_pointers_.CreateAIPosLVDTChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        voltageExcitSource,
        voltageExcitVal,
        voltageExcitFreq,
        acExcitWireMode,
        customScaleName
    );
}

int32 ProdAPI::CreateAIPosRVDTChan(
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
) {
    return function_pointers_.CreateAIPosRVDTChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        voltageExcitSource,
        voltageExcitVal,
        voltageExcitFreq,
        acExcitWireMode,
        customScaleName
    );
}

int32 ProdAPI::CreateAIPowerChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 voltageSetpoint,
    float64 currentSetpoint,
    bool32 outputEnable
) {
    return 0;
}

int32 ProdAPI::CreateAIPressureBridgePolynomialChan(
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
) {
    return function_pointers_.CreateAIPressureBridgePolynomialChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        forwardCoeffs,
        numForwardCoeffs,
        reverseCoeffs,
        numReverseCoeffs,
        electricalUnits,
        physicalUnits,
        customScaleName
    );
}

int32 ProdAPI::CreateAIPressureBridgeTableChan(
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
) {
    return function_pointers_.CreateAIPressureBridgeTableChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        electricalVals,
        numElectricalVals,
        electricalUnits,
        physicalVals,
        numPhysicalVals,
        physicalUnits,
        customScaleName
    );
}

int32 ProdAPI::CreateAIPressureBridgeTwoPointLinChan(
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
) {
    return function_pointers_.CreateAIPressureBridgeTwoPointLinChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        firstElectricalVal,
        secondElectricalVal,
        electricalUnits,
        firstPhysicalVal,
        secondPhysicalVal,
        physicalUnits,
        customScaleName
    );
}

int32 ProdAPI::CreateAIRTDChan(
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
) {
    return function_pointers_.CreateAIRTDChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        rtdType,
        resistanceConfig,
        currentExcitSource,
        currentExcitVal,
        r0
    );
}

int32 ProdAPI::CreateAIResistanceChan(
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
) {
    return function_pointers_.CreateAIResistanceChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateAIRosetteStrainGageChan(
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
) {
    return function_pointers_.CreateAIRosetteStrainGageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        rosetteType,
        gageOrientation,
        rosetteMeasTypes,
        numRosetteMeasTypes,
        strainConfig,
        voltageExcitSource,
        voltageExcitVal,
        gageFactor,
        nominalGageResistance,
        poissonRatio,
        leadWireResistance
    );
}

int32 ProdAPI::CreateAIStrainGageChan(
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
) {
    return function_pointers_.CreateAIStrainGageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        strainConfig,
        voltageExcitSource,
        voltageExcitVal,
        gageFactor,
        initialBridgeVoltage,
        nominalGageResistance,
        poissonRatio,
        leadWireResistance,
        customScaleName
    );
}

int32 ProdAPI::CreateAITempBuiltInSensorChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 units
) {
    return function_pointers_.CreateAITempBuiltInSensorChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        units
    );
}

int32 ProdAPI::CreateAIThrmcplChan(
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
) {
    return function_pointers_.CreateAIThrmcplChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        thermocoupleType,
        cjcSource,
        cjcVal,
        cjcChannel
    );
}

int32 ProdAPI::CreateAIThrmstrChanIex(
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
) {
    return function_pointers_.CreateAIThrmstrChanIex(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        currentExcitSource,
        currentExcitVal,
        a,
        b,
        c
    );
}

int32 ProdAPI::CreateAIThrmstrChanVex(
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
) {
    return function_pointers_.CreateAIThrmstrChanVex(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        voltageExcitSource,
        voltageExcitVal,
        a,
        b,
        c,
        r1
    );
}

int32 ProdAPI::CreateAITorqueBridgePolynomialChan(
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
) {
    return function_pointers_.CreateAITorqueBridgePolynomialChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        forwardCoeffs,
        numForwardCoeffs,
        reverseCoeffs,
        numReverseCoeffs,
        electricalUnits,
        physicalUnits,
        customScaleName
    );
}

int32 ProdAPI::CreateAITorqueBridgeTableChan(
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
) {
    return function_pointers_.CreateAITorqueBridgeTableChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        electricalVals,
        numElectricalVals,
        electricalUnits,
        physicalVals,
        numPhysicalVals,
        physicalUnits,
        customScaleName
    );
}

int32 ProdAPI::CreateAITorqueBridgeTwoPointLinChan(
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
) {
    return function_pointers_.CreateAITorqueBridgeTwoPointLinChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        firstElectricalVal,
        secondElectricalVal,
        electricalUnits,
        firstPhysicalVal,
        secondPhysicalVal,
        physicalUnits,
        customScaleName
    );
}

int32 ProdAPI::CreateAIVelocityIEPEChan(
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
) {
    return function_pointers_.CreateAIVelocityIEPEChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateAIVoltageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return function_pointers_.CreateAIVoltageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        customScaleName
    );
}

int32 ProdAPI::CreateAIVoltageChanWithExcit(
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
) {
    return function_pointers_.CreateAIVoltageChanWithExcit(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        useExcitForScaling,
        customScaleName
    );
}

int32 ProdAPI::CreateAIVoltageRMSChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return function_pointers_.CreateAIVoltageRMSChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        customScaleName
    );
}

int32 ProdAPI::CreateAOCurrentChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return function_pointers_.CreateAOCurrentChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        customScaleName
    );
}

int32 ProdAPI::CreateAOFuncGenChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 type,
    float64 freq,
    float64 amplitude,
    float64 offset
) {
    return function_pointers_.CreateAOFuncGenChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        type,
        freq,
        amplitude,
        offset
    );
}

int32 ProdAPI::CreateAOVoltageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return function_pointers_.CreateAOVoltageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        customScaleName
    );
}

int32 ProdAPI::CreateCIAngEncoderChan(
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
) {
    return function_pointers_.CreateCIAngEncoderChan(
        task,
        counter,
        nameToAssignToChannel,
        decodingType,
        zidxEnable,
        zidxVal,
        zidxPhase,
        units,
        pulsesPerRev,
        initialAngle,
        customScaleName
    );
}

int32 ProdAPI::CreateCIAngVelocityChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 decodingType,
    int32 units,
    uInt32 pulsesPerRev,
    const char customScaleName[]
) {
    return function_pointers_.CreateCIAngVelocityChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        decodingType,
        units,
        pulsesPerRev,
        customScaleName
    );
}

int32 ProdAPI::CreateCICountEdgesChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    int32 edge,
    uInt32 initialCount,
    int32 countDirection
) {
    return function_pointers_.CreateCICountEdgesChan(
        task,
        counter,
        nameToAssignToChannel,
        edge,
        initialCount,
        countDirection
    );
}

int32 ProdAPI::CreateCIDutyCycleChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minFreq,
    float64 maxFreq,
    int32 edge,
    const char customScaleName[]
) {
    return function_pointers_.CreateCIDutyCycleChan(
        task,
        counter,
        nameToAssignToChannel,
        minFreq,
        maxFreq,
        edge,
        customScaleName
    );
}

int32 ProdAPI::CreateCIFreqChan(
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
) {
    return function_pointers_.CreateCIFreqChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        edge,
        measMethod,
        measTime,
        divisor,
        customScaleName
    );
}

int32 ProdAPI::CreateCIGPSTimestampChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    int32 units,
    int32 syncMethod,
    const char customScaleName[]
) {
    return function_pointers_.CreateCIGPSTimestampChan(
        task,
        counter,
        nameToAssignToChannel,
        units,
        syncMethod,
        customScaleName
    );
}

int32 ProdAPI::CreateCILinEncoderChan(
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
) {
    return function_pointers_.CreateCILinEncoderChan(
        task,
        counter,
        nameToAssignToChannel,
        decodingType,
        zidxEnable,
        zidxVal,
        zidxPhase,
        units,
        distPerPulse,
        initialPos,
        customScaleName
    );
}

int32 ProdAPI::CreateCILinVelocityChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 decodingType,
    int32 units,
    float64 distPerPulse,
    const char customScaleName[]
) {
    return function_pointers_.CreateCILinVelocityChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        decodingType,
        units,
        distPerPulse,
        customScaleName
    );
}

int32 ProdAPI::CreateCIPeriodChan(
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
) {
    return function_pointers_.CreateCIPeriodChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        edge,
        measMethod,
        measTime,
        divisor,
        customScaleName
    );
}

int32 ProdAPI::CreateCIPulseChanFreq(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units
) {
    return function_pointers_.CreateCIPulseChanFreq(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units
    );
}

int32 ProdAPI::CreateCIPulseChanTicks(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    const char sourceTerminal[],
    float64 minVal,
    float64 maxVal
) {
    return function_pointers_.CreateCIPulseChanTicks(
        task,
        counter,
        nameToAssignToChannel,
        sourceTerminal,
        minVal,
        maxVal
    );
}

int32 ProdAPI::CreateCIPulseChanTime(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units
) {
    return function_pointers_.CreateCIPulseChanTime(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units
    );
}

int32 ProdAPI::CreateCIPulseWidthChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 startingEdge,
    const char customScaleName[]
) {
    return function_pointers_.CreateCIPulseWidthChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        startingEdge,
        customScaleName
    );
}

int32 ProdAPI::CreateCISemiPeriodChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return function_pointers_.CreateCISemiPeriodChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        customScaleName
    );
}

int32 ProdAPI::CreateCITwoEdgeSepChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 firstEdge,
    int32 secondEdge,
    const char customScaleName[]
) {
    return function_pointers_.CreateCITwoEdgeSepChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        firstEdge,
        secondEdge,
        customScaleName
    );
}

int32 ProdAPI::CreateCOPulseChanFreq(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    int32 units,
    int32 idleState,
    float64 initialDelay,
    float64 freq,
    float64 dutyCycle
) {
    return function_pointers_.CreateCOPulseChanFreq(
        task,
        counter,
        nameToAssignToChannel,
        units,
        idleState,
        initialDelay,
        freq,
        dutyCycle
    );
}

int32 ProdAPI::CreateCOPulseChanTicks(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    const char sourceTerminal[],
    int32 idleState,
    int32 initialDelay,
    int32 lowTicks,
    int32 highTicks
) {
    return function_pointers_.CreateCOPulseChanTicks(
        task,
        counter,
        nameToAssignToChannel,
        sourceTerminal,
        idleState,
        initialDelay,
        lowTicks,
        highTicks
    );
}

int32 ProdAPI::CreateCOPulseChanTime(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    int32 units,
    int32 idleState,
    float64 initialDelay,
    float64 lowTime,
    float64 highTime
) {
    return function_pointers_.CreateCOPulseChanTime(
        task,
        counter,
        nameToAssignToChannel,
        units,
        idleState,
        initialDelay,
        lowTime,
        highTime
    );
}

int32 ProdAPI::CreateDIChan(
    TaskHandle task,
    const char lines[],
    const char nameToAssignToLines[],
    int32 lineGrouping
) {
    return function_pointers_
        .CreateDIChan(task, lines, nameToAssignToLines, lineGrouping);
}

int32 ProdAPI::CreateDOChan(
    TaskHandle task,
    const char lines[],
    const char nameToAssignToLines[],
    int32 lineGrouping
) {
    return function_pointers_
        .CreateDOChan(task, lines, nameToAssignToLines, lineGrouping);
}

int32 ProdAPI::CreateLinScale(
    const char name[],
    float64 slope,
    float64 yIntercept,
    int32 preScaledUnits,
    const char scaledUnits[]
) {
    return function_pointers_
        .CreateLinScale(name, slope, yIntercept, preScaledUnits, scaledUnits);
}

int32 ProdAPI::CreateMapScale(
    const char name[],
    float64 prescaledMin,
    float64 prescaledMax,
    float64 scaledMin,
    float64 scaledMax,
    int32 preScaledUnits,
    const char scaledUnits[]
) {
    return function_pointers_.CreateMapScale(
        name,
        prescaledMin,
        prescaledMax,
        scaledMin,
        scaledMax,
        preScaledUnits,
        scaledUnits
    );
}

int32 ProdAPI::CreatePolynomialScale(
    const char name[],
    const float64 forwardCoeffs[],
    uInt32 numForwardCoeffsIn,
    const float64 reverseCoeffs[],
    uInt32 numReverseCoeffsIn,
    int32 preScaledUnits,
    const char scaledUnits[]
) {
    return function_pointers_.CreatePolynomialScale(
        name,
        forwardCoeffs,
        numForwardCoeffsIn,
        reverseCoeffs,
        numReverseCoeffsIn,
        preScaledUnits,
        scaledUnits
    );
}

int32 ProdAPI::CreateTEDSAIAccelChan(
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
) {
    return function_pointers_.CreateTEDSAIAccelChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAIBridgeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    const char customScaleName[]
) {
    return function_pointers_.CreateTEDSAIBridgeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAICurrentChan(
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
) {
    return function_pointers_.CreateTEDSAICurrentChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        shuntResistorLoc,
        extShuntResistorVal,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAIForceBridgeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    const char customScaleName[]
) {
    return function_pointers_.CreateTEDSAIForceBridgeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAIForceIEPEChan(
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
) {
    return function_pointers_.CreateTEDSAIForceIEPEChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAIMicrophoneChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    int32 units,
    float64 maxSndPressLevel,
    int32 currentExcitSource,
    float64 currentExcitVal,
    const char customScaleName[]
) {
    return function_pointers_.CreateTEDSAIMicrophoneChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        units,
        maxSndPressLevel,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAIPosLVDTChan(
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
) {
    return function_pointers_.CreateTEDSAIPosLVDTChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        voltageExcitFreq,
        acExcitWireMode,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAIPosRVDTChan(
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
) {
    return function_pointers_.CreateTEDSAIPosRVDTChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        voltageExcitFreq,
        acExcitWireMode,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAIPressureBridgeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    const char customScaleName[]
) {
    return function_pointers_.CreateTEDSAIPressureBridgeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAIRTDChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 resistanceConfig,
    int32 currentExcitSource,
    float64 currentExcitVal
) {
    return function_pointers_.CreateTEDSAIRTDChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        currentExcitSource,
        currentExcitVal
    );
}

int32 ProdAPI::CreateTEDSAIResistanceChan(
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
) {
    return function_pointers_.CreateTEDSAIResistanceChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAIStrainGageChan(
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
) {
    return function_pointers_.CreateTEDSAIStrainGageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        initialBridgeVoltage,
        leadWireResistance,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAIThrmcplChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 cjcSource,
    float64 cjcVal,
    const char cjcChannel[]
) {
    return function_pointers_.CreateTEDSAIThrmcplChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        cjcSource,
        cjcVal,
        cjcChannel
    );
}

int32 ProdAPI::CreateTEDSAIThrmstrChanIex(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 resistanceConfig,
    int32 currentExcitSource,
    float64 currentExcitVal
) {
    return function_pointers_.CreateTEDSAIThrmstrChanIex(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        currentExcitSource,
        currentExcitVal
    );
}

int32 ProdAPI::CreateTEDSAIThrmstrChanVex(
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
) {
    return function_pointers_.CreateTEDSAIThrmstrChanVex(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        voltageExcitSource,
        voltageExcitVal,
        r1
    );
}

int32 ProdAPI::CreateTEDSAITorqueBridgeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    const char customScaleName[]
) {
    return function_pointers_.CreateTEDSAITorqueBridgeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAIVoltageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return function_pointers_.CreateTEDSAIVoltageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        customScaleName
    );
}

int32 ProdAPI::CreateTEDSAIVoltageChanWithExcit(
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
) {
    return function_pointers_.CreateTEDSAIVoltageChanWithExcit(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        customScaleName
    );
}

int32 ProdAPI::CreateTableScale(
    const char name[],
    const float64 prescaledVals[],
    uInt32 numPrescaledValsIn,
    const float64 scaledVals[],
    uInt32 numScaledValsIn,
    int32 preScaledUnits,
    const char scaledUnits[]
) {
    return function_pointers_.CreateTableScale(
        name,
        prescaledVals,
        numPrescaledValsIn,
        scaledVals,
        numScaledValsIn,
        preScaledUnits,
        scaledUnits
    );
}

int32 ProdAPI::CreateTask(const char sessionName[], TaskHandle *task) {
    return function_pointers_.CreateTask(sessionName, task);
}

int32 ProdAPI::CreateWatchdogTimerTask(
    const char deviceName[],
    const char sessionName[],
    TaskHandle *task,
    float64 timeout,
    const char lines[],
    int32 expState,
    const char lines0[],
    int32 expState0,
    const char lines1[],
    int32 expState1,
    const char lines2[],
    int32 expState2,
    const char lines3[],
    int32 expState3,
    const char lines4[],
    int32 expState4,
    const char lines5[],
    int32 expState5,
    const char lines6[],
    int32 expState6,
    const char lines7[],
    int32 expState7,
    const char lines8[],
    int32 expState8,
    const char lines9[],
    int32 expState9,
    const char lines10[],
    int32 expState10,
    const char lines11[],
    int32 expState11,
    const char lines12[],
    int32 expState12,
    const char lines13[],
    int32 expState13,
    const char lines14[],
    int32 expState14,
    const char lines15[],
    int32 expState15,
    const char lines16[],
    int32 expState16,
    const char lines17[],
    int32 expState17,
    const char lines18[],
    int32 expState18,
    const char lines19[],
    int32 expState19,
    const char lines20[],
    int32 expState20,
    const char lines21[],
    int32 expState21,
    const char lines22[],
    int32 expState22,
    const char lines23[],
    int32 expState23,
    const char lines24[],
    int32 expState24,
    const char lines25[],
    int32 expState25,
    const char lines26[],
    int32 expState26,
    const char lines27[],
    int32 expState27,
    const char lines28[],
    int32 expState28,
    const char lines29[],
    int32 expState29,
    const char lines30[],
    int32 expState30,
    const char lines31[],
    int32 expState31,
    const char lines32[],
    int32 expState32,
    const char lines33[],
    int32 expState33,
    const char lines34[],
    int32 expState34,
    const char lines35[],
    int32 expState35,
    const char lines36[],
    int32 expState36,
    const char lines37[],
    int32 expState37,
    const char lines38[],
    int32 expState38,
    const char lines39[],
    int32 expState39,
    const char lines40[],
    int32 expState40,
    const char lines41[],
    int32 expState41,
    const char lines42[],
    int32 expState42,
    const char lines43[],
    int32 expState43,
    const char lines44[],
    int32 expState44,
    const char lines45[],
    int32 expState45,
    const char lines46[],
    int32 expState46,
    const char lines47[],
    int32 expState47,
    const char lines48[],
    int32 expState48,
    const char lines49[],
    int32 expState49,
    const char lines50[],
    int32 expState50,
    const char lines51[],
    int32 expState51,
    const char lines52[],
    int32 expState52,
    const char lines53[],
    int32 expState53,
    const char lines54[],
    int32 expState54,
    const char lines55[],
    int32 expState55,
    const char lines56[],
    int32 expState56,
    const char lines57[],
    int32 expState57,
    const char lines58[],
    int32 expState58,
    const char lines59[],
    int32 expState59,
    const char lines60[],
    int32 expState60,
    const char lines61[],
    int32 expState61,
    const char lines62[],
    int32 expState62,
    const char lines63[],
    int32 expState63,
    const char lines64[],
    int32 expState64,
    const char lines65[],
    int32 expState65,
    const char lines66[],
    int32 expState66,
    const char lines67[],
    int32 expState67,
    const char lines68[],
    int32 expState68,
    const char lines69[],
    int32 expState69,
    const char lines70[],
    int32 expState70,
    const char lines71[],
    int32 expState71,
    const char lines72[],
    int32 expState72,
    const char lines73[],
    int32 expState73,
    const char lines74[],
    int32 expState74,
    const char lines75[],
    int32 expState75,
    const char lines76[],
    int32 expState76,
    const char lines77[],
    int32 expState77,
    const char lines78[],
    int32 expState78,
    const char lines79[],
    int32 expState79,
    const char lines80[],
    int32 expState80,
    const char lines81[],
    int32 expState81,
    const char lines82[],
    int32 expState82,
    const char lines83[],
    int32 expState83,
    const char lines84[],
    int32 expState84,
    const char lines85[],
    int32 expState85,
    const char lines86[],
    int32 expState86,
    const char lines87[],
    int32 expState87,
    const char lines88[],
    int32 expState88,
    const char lines89[],
    int32 expState89,
    const char lines90[],
    int32 expState90,
    const char lines91[],
    int32 expState91,
    const char lines92[],
    int32 expState92,
    const char lines93[],
    int32 expState93,
    const char lines94[],
    int32 expState94,
    const char lines95[],
    int32 expState95
) {
    return function_pointers_.CreateWatchdogTimerTask(
        deviceName,
        sessionName,
        task,
        timeout,
        lines,
        expState,
        lines0,
        expState0,
        lines1,
        expState1,
        lines2,
        expState2,
        lines3,
        expState3,
        lines4,
        expState4,
        lines5,
        expState5,
        lines6,
        expState6,
        lines7,
        expState7,
        lines8,
        expState8,
        lines9,
        expState9,
        lines10,
        expState10,
        lines11,
        expState11,
        lines12,
        expState12,
        lines13,
        expState13,
        lines14,
        expState14,
        lines15,
        expState15,
        lines16,
        expState16,
        lines17,
        expState17,
        lines18,
        expState18,
        lines19,
        expState19,
        lines20,
        expState20,
        lines21,
        expState21,
        lines22,
        expState22,
        lines23,
        expState23,
        lines24,
        expState24,
        lines25,
        expState25,
        lines26,
        expState26,
        lines27,
        expState27,
        lines28,
        expState28,
        lines29,
        expState29,
        lines30,
        expState30,
        lines31,
        expState31,
        lines32,
        expState32,
        lines33,
        expState33,
        lines34,
        expState34,
        lines35,
        expState35,
        lines36,
        expState36,
        lines37,
        expState37,
        lines38,
        expState38,
        lines39,
        expState39,
        lines40,
        expState40,
        lines41,
        expState41,
        lines42,
        expState42,
        lines43,
        expState43,
        lines44,
        expState44,
        lines45,
        expState45,
        lines46,
        expState46,
        lines47,
        expState47,
        lines48,
        expState48,
        lines49,
        expState49,
        lines50,
        expState50,
        lines51,
        expState51,
        lines52,
        expState52,
        lines53,
        expState53,
        lines54,
        expState54,
        lines55,
        expState55,
        lines56,
        expState56,
        lines57,
        expState57,
        lines58,
        expState58,
        lines59,
        expState59,
        lines60,
        expState60,
        lines61,
        expState61,
        lines62,
        expState62,
        lines63,
        expState63,
        lines64,
        expState64,
        lines65,
        expState65,
        lines66,
        expState66,
        lines67,
        expState67,
        lines68,
        expState68,
        lines69,
        expState69,
        lines70,
        expState70,
        lines71,
        expState71,
        lines72,
        expState72,
        lines73,
        expState73,
        lines74,
        expState74,
        lines75,
        expState75,
        lines76,
        expState76,
        lines77,
        expState77,
        lines78,
        expState78,
        lines79,
        expState79,
        lines80,
        expState80,
        lines81,
        expState81,
        lines82,
        expState82,
        lines83,
        expState83,
        lines84,
        expState84,
        lines85,
        expState85,
        lines86,
        expState86,
        lines87,
        expState87,
        lines88,
        expState88,
        lines89,
        expState89,
        lines90,
        expState90,
        lines91,
        expState91,
        lines92,
        expState92,
        lines93,
        expState93,
        lines94,
        expState94,
        lines95,
        expState95
    );
}

int32 ProdAPI::CreateWatchdogTimerTaskEx(
    const char deviceName[],
    const char sessionName[],
    TaskHandle *task,
    float64 timeout
) {
    return function_pointers_
        .CreateWatchdogTimerTaskEx(deviceName, sessionName, task, timeout);
}

int32 ProdAPI::DeleteNetworkDevice(const char deviceName[]) {
    return function_pointers_.DeleteNetworkDevice(deviceName);
}

int32 ProdAPI::DeleteSavedGlobalChan(const char channelName[]) {
    return function_pointers_.DeleteSavedGlobalChan(channelName);
}

int32 ProdAPI::DeleteSavedScale(const char scaleName[]) {
    return function_pointers_.DeleteSavedScale(scaleName);
}

int32 ProdAPI::DeleteSavedTask(const char taskName[]) {
    return function_pointers_.DeleteSavedTask(taskName);
}

int32 ProdAPI::DeviceSupportsCal(const char deviceName[], bool32 *calSupported) {
    return function_pointers_.DeviceSupportsCal(deviceName, calSupported);
}

int32 ProdAPI::DisableRefTrig(TaskHandle task) {
    return function_pointers_.DisableRefTrig(task);
}

int32 ProdAPI::DisableStartTrig(TaskHandle task) {
    return function_pointers_.DisableStartTrig(task);
}

int32 ProdAPI::DisconnectTerms(
    const char sourceTerminal[],
    const char destinationTerminal[]
) {
    return function_pointers_.DisconnectTerms(sourceTerminal, destinationTerminal);
}

int32 ProdAPI::ExportSignal(
    TaskHandle task,
    int32 signalID,
    const char outputTerminal[]
) {
    return function_pointers_.ExportSignal(task, signalID, outputTerminal);
}

int32 ProdAPI::GetAIChanCalCalDate(
    TaskHandle task,
    const char channelName[],
    uInt32 *year,
    uInt32 *month,
    uInt32 *day,
    uInt32 *hour,
    uInt32 *minute
) {
    return function_pointers_
        .GetAIChanCalCalDate(task, channelName, year, month, day, hour, minute);
}

int32 ProdAPI::GetAIChanCalExpDate(
    TaskHandle task,
    const char channelName[],
    uInt32 *year,
    uInt32 *month,
    uInt32 *day,
    uInt32 *hour,
    uInt32 *minute
) {
    return function_pointers_
        .GetAIChanCalExpDate(task, channelName, year, month, day, hour, minute);
}

int32 ProdAPI::GetAnalogPowerUpStates(
    const char deviceName[],
    const char channelName[],
    float64 *state,
    int32 channelType,
    const char channelName0[],
    float64 *state0,
    int32 channelType0,
    const char channelName1[],
    float64 *state1,
    int32 channelType1,
    const char channelName2[],
    float64 *state2,
    int32 channelType2,
    const char channelName3[],
    float64 *state3,
    int32 channelType3,
    const char channelName4[],
    float64 *state4,
    int32 channelType4,
    const char channelName5[],
    float64 *state5,
    int32 channelType5,
    const char channelName6[],
    float64 *state6,
    int32 channelType6,
    const char channelName7[],
    float64 *state7,
    int32 channelType7,
    const char channelName8[],
    float64 *state8,
    int32 channelType8,
    const char channelName9[],
    float64 *state9,
    int32 channelType9,
    const char channelName10[],
    float64 *state10,
    int32 channelType10,
    const char channelName11[],
    float64 *state11,
    int32 channelType11,
    const char channelName12[],
    float64 *state12,
    int32 channelType12,
    const char channelName13[],
    float64 *state13,
    int32 channelType13,
    const char channelName14[],
    float64 *state14,
    int32 channelType14,
    const char channelName15[],
    float64 *state15,
    int32 channelType15,
    const char channelName16[],
    float64 *state16,
    int32 channelType16,
    const char channelName17[],
    float64 *state17,
    int32 channelType17,
    const char channelName18[],
    float64 *state18,
    int32 channelType18,
    const char channelName19[],
    float64 *state19,
    int32 channelType19,
    const char channelName20[],
    float64 *state20,
    int32 channelType20,
    const char channelName21[],
    float64 *state21,
    int32 channelType21,
    const char channelName22[],
    float64 *state22,
    int32 channelType22,
    const char channelName23[],
    float64 *state23,
    int32 channelType23,
    const char channelName24[],
    float64 *state24,
    int32 channelType24,
    const char channelName25[],
    float64 *state25,
    int32 channelType25,
    const char channelName26[],
    float64 *state26,
    int32 channelType26,
    const char channelName27[],
    float64 *state27,
    int32 channelType27,
    const char channelName28[],
    float64 *state28,
    int32 channelType28,
    const char channelName29[],
    float64 *state29,
    int32 channelType29,
    const char channelName30[],
    float64 *state30,
    int32 channelType30,
    const char channelName31[],
    float64 *state31,
    int32 channelType31,
    const char channelName32[],
    float64 *state32,
    int32 channelType32,
    const char channelName33[],
    float64 *state33,
    int32 channelType33,
    const char channelName34[],
    float64 *state34,
    int32 channelType34,
    const char channelName35[],
    float64 *state35,
    int32 channelType35,
    const char channelName36[],
    float64 *state36,
    int32 channelType36,
    const char channelName37[],
    float64 *state37,
    int32 channelType37,
    const char channelName38[],
    float64 *state38,
    int32 channelType38,
    const char channelName39[],
    float64 *state39,
    int32 channelType39,
    const char channelName40[],
    float64 *state40,
    int32 channelType40,
    const char channelName41[],
    float64 *state41,
    int32 channelType41,
    const char channelName42[],
    float64 *state42,
    int32 channelType42,
    const char channelName43[],
    float64 *state43,
    int32 channelType43,
    const char channelName44[],
    float64 *state44,
    int32 channelType44,
    const char channelName45[],
    float64 *state45,
    int32 channelType45,
    const char channelName46[],
    float64 *state46,
    int32 channelType46,
    const char channelName47[],
    float64 *state47,
    int32 channelType47,
    const char channelName48[],
    float64 *state48,
    int32 channelType48,
    const char channelName49[],
    float64 *state49,
    int32 channelType49,
    const char channelName50[],
    float64 *state50,
    int32 channelType50,
    const char channelName51[],
    float64 *state51,
    int32 channelType51,
    const char channelName52[],
    float64 *state52,
    int32 channelType52,
    const char channelName53[],
    float64 *state53,
    int32 channelType53,
    const char channelName54[],
    float64 *state54,
    int32 channelType54,
    const char channelName55[],
    float64 *state55,
    int32 channelType55,
    const char channelName56[],
    float64 *state56,
    int32 channelType56,
    const char channelName57[],
    float64 *state57,
    int32 channelType57,
    const char channelName58[],
    float64 *state58,
    int32 channelType58,
    const char channelName59[],
    float64 *state59,
    int32 channelType59,
    const char channelName60[],
    float64 *state60,
    int32 channelType60,
    const char channelName61[],
    float64 *state61,
    int32 channelType61,
    const char channelName62[],
    float64 *state62,
    int32 channelType62,
    const char channelName63[],
    float64 *state63,
    int32 channelType63,
    const char channelName64[],
    float64 *state64,
    int32 channelType64,
    const char channelName65[],
    float64 *state65,
    int32 channelType65,
    const char channelName66[],
    float64 *state66,
    int32 channelType66,
    const char channelName67[],
    float64 *state67,
    int32 channelType67,
    const char channelName68[],
    float64 *state68,
    int32 channelType68,
    const char channelName69[],
    float64 *state69,
    int32 channelType69,
    const char channelName70[],
    float64 *state70,
    int32 channelType70,
    const char channelName71[],
    float64 *state71,
    int32 channelType71,
    const char channelName72[],
    float64 *state72,
    int32 channelType72,
    const char channelName73[],
    float64 *state73,
    int32 channelType73,
    const char channelName74[],
    float64 *state74,
    int32 channelType74,
    const char channelName75[],
    float64 *state75,
    int32 channelType75,
    const char channelName76[],
    float64 *state76,
    int32 channelType76,
    const char channelName77[],
    float64 *state77,
    int32 channelType77,
    const char channelName78[],
    float64 *state78,
    int32 channelType78,
    const char channelName79[],
    float64 *state79,
    int32 channelType79,
    const char channelName80[],
    float64 *state80,
    int32 channelType80,
    const char channelName81[],
    float64 *state81,
    int32 channelType81,
    const char channelName82[],
    float64 *state82,
    int32 channelType82,
    const char channelName83[],
    float64 *state83,
    int32 channelType83,
    const char channelName84[],
    float64 *state84,
    int32 channelType84,
    const char channelName85[],
    float64 *state85,
    int32 channelType85,
    const char channelName86[],
    float64 *state86,
    int32 channelType86,
    const char channelName87[],
    float64 *state87,
    int32 channelType87,
    const char channelName88[],
    float64 *state88,
    int32 channelType88,
    const char channelName89[],
    float64 *state89,
    int32 channelType89,
    const char channelName90[],
    float64 *state90,
    int32 channelType90,
    const char channelName91[],
    float64 *state91,
    int32 channelType91,
    const char channelName92[],
    float64 *state92,
    int32 channelType92,
    const char channelName93[],
    float64 *state93,
    int32 channelType93,
    const char channelName94[],
    float64 *state94,
    int32 channelType94,
    const char channelName95[],
    float64 *state95,
    int32 channelType95
) {
    return function_pointers_.GetAnalogPowerUpStates(
        deviceName,
        channelName,
        state,
        channelType,
        channelName0,
        state0,
        channelType0,
        channelName1,
        state1,
        channelType1,
        channelName2,
        state2,
        channelType2,
        channelName3,
        state3,
        channelType3,
        channelName4,
        state4,
        channelType4,
        channelName5,
        state5,
        channelType5,
        channelName6,
        state6,
        channelType6,
        channelName7,
        state7,
        channelType7,
        channelName8,
        state8,
        channelType8,
        channelName9,
        state9,
        channelType9,
        channelName10,
        state10,
        channelType10,
        channelName11,
        state11,
        channelType11,
        channelName12,
        state12,
        channelType12,
        channelName13,
        state13,
        channelType13,
        channelName14,
        state14,
        channelType14,
        channelName15,
        state15,
        channelType15,
        channelName16,
        state16,
        channelType16,
        channelName17,
        state17,
        channelType17,
        channelName18,
        state18,
        channelType18,
        channelName19,
        state19,
        channelType19,
        channelName20,
        state20,
        channelType20,
        channelName21,
        state21,
        channelType21,
        channelName22,
        state22,
        channelType22,
        channelName23,
        state23,
        channelType23,
        channelName24,
        state24,
        channelType24,
        channelName25,
        state25,
        channelType25,
        channelName26,
        state26,
        channelType26,
        channelName27,
        state27,
        channelType27,
        channelName28,
        state28,
        channelType28,
        channelName29,
        state29,
        channelType29,
        channelName30,
        state30,
        channelType30,
        channelName31,
        state31,
        channelType31,
        channelName32,
        state32,
        channelType32,
        channelName33,
        state33,
        channelType33,
        channelName34,
        state34,
        channelType34,
        channelName35,
        state35,
        channelType35,
        channelName36,
        state36,
        channelType36,
        channelName37,
        state37,
        channelType37,
        channelName38,
        state38,
        channelType38,
        channelName39,
        state39,
        channelType39,
        channelName40,
        state40,
        channelType40,
        channelName41,
        state41,
        channelType41,
        channelName42,
        state42,
        channelType42,
        channelName43,
        state43,
        channelType43,
        channelName44,
        state44,
        channelType44,
        channelName45,
        state45,
        channelType45,
        channelName46,
        state46,
        channelType46,
        channelName47,
        state47,
        channelType47,
        channelName48,
        state48,
        channelType48,
        channelName49,
        state49,
        channelType49,
        channelName50,
        state50,
        channelType50,
        channelName51,
        state51,
        channelType51,
        channelName52,
        state52,
        channelType52,
        channelName53,
        state53,
        channelType53,
        channelName54,
        state54,
        channelType54,
        channelName55,
        state55,
        channelType55,
        channelName56,
        state56,
        channelType56,
        channelName57,
        state57,
        channelType57,
        channelName58,
        state58,
        channelType58,
        channelName59,
        state59,
        channelType59,
        channelName60,
        state60,
        channelType60,
        channelName61,
        state61,
        channelType61,
        channelName62,
        state62,
        channelType62,
        channelName63,
        state63,
        channelType63,
        channelName64,
        state64,
        channelType64,
        channelName65,
        state65,
        channelType65,
        channelName66,
        state66,
        channelType66,
        channelName67,
        state67,
        channelType67,
        channelName68,
        state68,
        channelType68,
        channelName69,
        state69,
        channelType69,
        channelName70,
        state70,
        channelType70,
        channelName71,
        state71,
        channelType71,
        channelName72,
        state72,
        channelType72,
        channelName73,
        state73,
        channelType73,
        channelName74,
        state74,
        channelType74,
        channelName75,
        state75,
        channelType75,
        channelName76,
        state76,
        channelType76,
        channelName77,
        state77,
        channelType77,
        channelName78,
        state78,
        channelType78,
        channelName79,
        state79,
        channelType79,
        channelName80,
        state80,
        channelType80,
        channelName81,
        state81,
        channelType81,
        channelName82,
        state82,
        channelType82,
        channelName83,
        state83,
        channelType83,
        channelName84,
        state84,
        channelType84,
        channelName85,
        state85,
        channelType85,
        channelName86,
        state86,
        channelType86,
        channelName87,
        state87,
        channelType87,
        channelName88,
        state88,
        channelType88,
        channelName89,
        state89,
        channelType89,
        channelName90,
        state90,
        channelType90,
        channelName91,
        state91,
        channelType91,
        channelName92,
        state92,
        channelType92,
        channelName93,
        state93,
        channelType93,
        channelName94,
        state94,
        channelType94,
        channelName95,
        state95,
        channelType95
    );
}

int32 ProdAPI::GetAnalogPowerUpStatesWithOutputType(
    const char channelNames[],
    float64 stateArray[],
    int32 channelTypeArray[],
    uInt32 *arraySize
) {
    return function_pointers_.GetAnalogPowerUpStatesWithOutputType(
        channelNames,
        stateArray,
        channelTypeArray,
        arraySize
    );
}

int32 ProdAPI::GetArmStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) {
    return function_pointers_.GetArmStartTrigTimestampVal(task, data);
}

int32 ProdAPI::GetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data) {
    return function_pointers_.GetArmStartTrigTrigWhen(task, data);
}

int32 ProdAPI::GetAutoConfiguredCDAQSyncConnections(
    char portList[],
    uInt32 portListSize
) {
    return function_pointers_.GetAutoConfiguredCDAQSyncConnections(
        portList,
        portListSize
    );
}

int32 ProdAPI::GetBufferAttributeUInt32(
    TaskHandle task,
    int32 attribute,
    uInt32 *value
) {
    return function_pointers_.GetBufferAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::GetCalInfoAttributeBool(
    const char deviceName[],
    int32 attribute,
    bool32 *value
) {
    return function_pointers_.GetCalInfoAttributeBool(deviceName, attribute, value);
}

int32 ProdAPI::GetCalInfoAttributeDouble(
    const char deviceName[],
    int32 attribute,
    float64 *value
) {
    return function_pointers_.GetCalInfoAttributeDouble(deviceName, attribute, value);
}

int32 ProdAPI::GetCalInfoAttributeString(
    const char deviceName[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_
        .GetCalInfoAttributeString(deviceName, attribute, value, size);
}

int32 ProdAPI::GetCalInfoAttributeUInt32(
    const char deviceName[],
    int32 attribute,
    uInt32 *value
) {
    return function_pointers_.GetCalInfoAttributeUInt32(deviceName, attribute, value);
}

int32 ProdAPI::GetChanAttributeBool(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    bool32 *value
) {
    return function_pointers_.GetChanAttributeBool(task, channel, attribute, value);
}

int32 ProdAPI::GetChanAttributeDouble(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    float64 *value
) {
    return function_pointers_.GetChanAttributeDouble(task, channel, attribute, value);
}

int32 ProdAPI::GetChanAttributeDoubleArray(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    float64 value[],
    uInt32 size
) {
    return function_pointers_
        .GetChanAttributeDoubleArray(task, channel, attribute, value, size);
}

int32 ProdAPI::GetChanAttributeInt32(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    int32 *value
) {
    return function_pointers_.GetChanAttributeInt32(task, channel, attribute, value);
}

int32 ProdAPI::GetChanAttributeString(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_
        .GetChanAttributeString(task, channel, attribute, value, size);
}

int32 ProdAPI::GetChanAttributeUInt32(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    uInt32 *value
) {
    return function_pointers_.GetChanAttributeUInt32(task, channel, attribute, value);
}

int32 ProdAPI::GetDeviceAttributeBool(
    const char deviceName[],
    int32 attribute,
    bool32 *value
) {
    return function_pointers_.GetDeviceAttributeBool(deviceName, attribute, value);
}

int32 ProdAPI::GetDeviceAttributeDouble(
    const char deviceName[],
    int32 attribute,
    float64 *value
) {
    return function_pointers_.GetDeviceAttributeDouble(deviceName, attribute, value);
}

int32 ProdAPI::GetDeviceAttributeDoubleArray(
    const char deviceName[],
    int32 attribute,
    float64 value[],
    uInt32 size
) {
    return function_pointers_
        .GetDeviceAttributeDoubleArray(deviceName, attribute, value, size);
}

int32 ProdAPI::GetDeviceAttributeInt32(
    const char deviceName[],
    int32 attribute,
    int32 *value
) {
    return function_pointers_.GetDeviceAttributeInt32(deviceName, attribute, value);
}

int32 ProdAPI::GetDeviceAttributeInt32Array(
    const char deviceName[],
    int32 attribute,
    int32 value[],
    uInt32 size
) {
    return function_pointers_
        .GetDeviceAttributeInt32Array(deviceName, attribute, value, size);
}

int32 ProdAPI::GetDeviceAttributeString(
    const char deviceName[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_
        .GetDeviceAttributeString(deviceName, attribute, value, size);
}

int32 ProdAPI::GetDeviceAttributeUInt32(
    const char deviceName[],
    int32 attribute,
    uInt32 *value
) {
    return function_pointers_.GetDeviceAttributeUInt32(deviceName, attribute, value);
}

int32 ProdAPI::GetDeviceAttributeUInt32Array(
    const char deviceName[],
    int32 attribute,
    uInt32 value[],
    uInt32 size
) {
    return function_pointers_
        .GetDeviceAttributeUInt32Array(deviceName, attribute, value, size);
}

int32 ProdAPI::GetDigitalLogicFamilyPowerUpState(
    const char deviceName[],
    int32 *logicFamily
) {
    return function_pointers_.GetDigitalLogicFamilyPowerUpState(
        deviceName,
        logicFamily
    );
}

int32 ProdAPI::GetDigitalPowerUpStates(
    const char deviceName[],
    const char channelName[],
    int32 *state,
    const char channelName0[],
    int32 *state0,
    const char channelName1[],
    int32 *state1,
    const char channelName2[],
    int32 *state2,
    const char channelName3[],
    int32 *state3,
    const char channelName4[],
    int32 *state4,
    const char channelName5[],
    int32 *state5,
    const char channelName6[],
    int32 *state6,
    const char channelName7[],
    int32 *state7,
    const char channelName8[],
    int32 *state8,
    const char channelName9[],
    int32 *state9,
    const char channelName10[],
    int32 *state10,
    const char channelName11[],
    int32 *state11,
    const char channelName12[],
    int32 *state12,
    const char channelName13[],
    int32 *state13,
    const char channelName14[],
    int32 *state14,
    const char channelName15[],
    int32 *state15,
    const char channelName16[],
    int32 *state16,
    const char channelName17[],
    int32 *state17,
    const char channelName18[],
    int32 *state18,
    const char channelName19[],
    int32 *state19,
    const char channelName20[],
    int32 *state20,
    const char channelName21[],
    int32 *state21,
    const char channelName22[],
    int32 *state22,
    const char channelName23[],
    int32 *state23,
    const char channelName24[],
    int32 *state24,
    const char channelName25[],
    int32 *state25,
    const char channelName26[],
    int32 *state26,
    const char channelName27[],
    int32 *state27,
    const char channelName28[],
    int32 *state28,
    const char channelName29[],
    int32 *state29,
    const char channelName30[],
    int32 *state30,
    const char channelName31[],
    int32 *state31,
    const char channelName32[],
    int32 *state32,
    const char channelName33[],
    int32 *state33,
    const char channelName34[],
    int32 *state34,
    const char channelName35[],
    int32 *state35,
    const char channelName36[],
    int32 *state36,
    const char channelName37[],
    int32 *state37,
    const char channelName38[],
    int32 *state38,
    const char channelName39[],
    int32 *state39,
    const char channelName40[],
    int32 *state40,
    const char channelName41[],
    int32 *state41,
    const char channelName42[],
    int32 *state42,
    const char channelName43[],
    int32 *state43,
    const char channelName44[],
    int32 *state44,
    const char channelName45[],
    int32 *state45,
    const char channelName46[],
    int32 *state46,
    const char channelName47[],
    int32 *state47,
    const char channelName48[],
    int32 *state48,
    const char channelName49[],
    int32 *state49,
    const char channelName50[],
    int32 *state50,
    const char channelName51[],
    int32 *state51,
    const char channelName52[],
    int32 *state52,
    const char channelName53[],
    int32 *state53,
    const char channelName54[],
    int32 *state54,
    const char channelName55[],
    int32 *state55,
    const char channelName56[],
    int32 *state56,
    const char channelName57[],
    int32 *state57,
    const char channelName58[],
    int32 *state58,
    const char channelName59[],
    int32 *state59,
    const char channelName60[],
    int32 *state60,
    const char channelName61[],
    int32 *state61,
    const char channelName62[],
    int32 *state62,
    const char channelName63[],
    int32 *state63,
    const char channelName64[],
    int32 *state64,
    const char channelName65[],
    int32 *state65,
    const char channelName66[],
    int32 *state66,
    const char channelName67[],
    int32 *state67,
    const char channelName68[],
    int32 *state68,
    const char channelName69[],
    int32 *state69,
    const char channelName70[],
    int32 *state70,
    const char channelName71[],
    int32 *state71,
    const char channelName72[],
    int32 *state72,
    const char channelName73[],
    int32 *state73,
    const char channelName74[],
    int32 *state74,
    const char channelName75[],
    int32 *state75,
    const char channelName76[],
    int32 *state76,
    const char channelName77[],
    int32 *state77,
    const char channelName78[],
    int32 *state78,
    const char channelName79[],
    int32 *state79,
    const char channelName80[],
    int32 *state80,
    const char channelName81[],
    int32 *state81,
    const char channelName82[],
    int32 *state82,
    const char channelName83[],
    int32 *state83,
    const char channelName84[],
    int32 *state84,
    const char channelName85[],
    int32 *state85,
    const char channelName86[],
    int32 *state86,
    const char channelName87[],
    int32 *state87,
    const char channelName88[],
    int32 *state88,
    const char channelName89[],
    int32 *state89,
    const char channelName90[],
    int32 *state90,
    const char channelName91[],
    int32 *state91,
    const char channelName92[],
    int32 *state92,
    const char channelName93[],
    int32 *state93,
    const char channelName94[],
    int32 *state94,
    const char channelName95[],
    int32 *state95
) {
    return function_pointers_.GetDigitalPowerUpStates(
        deviceName,
        channelName,
        state,
        channelName0,
        state0,
        channelName1,
        state1,
        channelName2,
        state2,
        channelName3,
        state3,
        channelName4,
        state4,
        channelName5,
        state5,
        channelName6,
        state6,
        channelName7,
        state7,
        channelName8,
        state8,
        channelName9,
        state9,
        channelName10,
        state10,
        channelName11,
        state11,
        channelName12,
        state12,
        channelName13,
        state13,
        channelName14,
        state14,
        channelName15,
        state15,
        channelName16,
        state16,
        channelName17,
        state17,
        channelName18,
        state18,
        channelName19,
        state19,
        channelName20,
        state20,
        channelName21,
        state21,
        channelName22,
        state22,
        channelName23,
        state23,
        channelName24,
        state24,
        channelName25,
        state25,
        channelName26,
        state26,
        channelName27,
        state27,
        channelName28,
        state28,
        channelName29,
        state29,
        channelName30,
        state30,
        channelName31,
        state31,
        channelName32,
        state32,
        channelName33,
        state33,
        channelName34,
        state34,
        channelName35,
        state35,
        channelName36,
        state36,
        channelName37,
        state37,
        channelName38,
        state38,
        channelName39,
        state39,
        channelName40,
        state40,
        channelName41,
        state41,
        channelName42,
        state42,
        channelName43,
        state43,
        channelName44,
        state44,
        channelName45,
        state45,
        channelName46,
        state46,
        channelName47,
        state47,
        channelName48,
        state48,
        channelName49,
        state49,
        channelName50,
        state50,
        channelName51,
        state51,
        channelName52,
        state52,
        channelName53,
        state53,
        channelName54,
        state54,
        channelName55,
        state55,
        channelName56,
        state56,
        channelName57,
        state57,
        channelName58,
        state58,
        channelName59,
        state59,
        channelName60,
        state60,
        channelName61,
        state61,
        channelName62,
        state62,
        channelName63,
        state63,
        channelName64,
        state64,
        channelName65,
        state65,
        channelName66,
        state66,
        channelName67,
        state67,
        channelName68,
        state68,
        channelName69,
        state69,
        channelName70,
        state70,
        channelName71,
        state71,
        channelName72,
        state72,
        channelName73,
        state73,
        channelName74,
        state74,
        channelName75,
        state75,
        channelName76,
        state76,
        channelName77,
        state77,
        channelName78,
        state78,
        channelName79,
        state79,
        channelName80,
        state80,
        channelName81,
        state81,
        channelName82,
        state82,
        channelName83,
        state83,
        channelName84,
        state84,
        channelName85,
        state85,
        channelName86,
        state86,
        channelName87,
        state87,
        channelName88,
        state88,
        channelName89,
        state89,
        channelName90,
        state90,
        channelName91,
        state91,
        channelName92,
        state92,
        channelName93,
        state93,
        channelName94,
        state94,
        channelName95,
        state95
    );
}

int32 ProdAPI::GetDigitalPullUpPullDownStates(
    const char deviceName[],
    const char channelName[],
    int32 *state,
    const char channelName0[],
    int32 *state0,
    const char channelName1[],
    int32 *state1,
    const char channelName2[],
    int32 *state2,
    const char channelName3[],
    int32 *state3,
    const char channelName4[],
    int32 *state4,
    const char channelName5[],
    int32 *state5,
    const char channelName6[],
    int32 *state6,
    const char channelName7[],
    int32 *state7,
    const char channelName8[],
    int32 *state8,
    const char channelName9[],
    int32 *state9,
    const char channelName10[],
    int32 *state10,
    const char channelName11[],
    int32 *state11,
    const char channelName12[],
    int32 *state12,
    const char channelName13[],
    int32 *state13,
    const char channelName14[],
    int32 *state14,
    const char channelName15[],
    int32 *state15,
    const char channelName16[],
    int32 *state16,
    const char channelName17[],
    int32 *state17,
    const char channelName18[],
    int32 *state18,
    const char channelName19[],
    int32 *state19,
    const char channelName20[],
    int32 *state20,
    const char channelName21[],
    int32 *state21,
    const char channelName22[],
    int32 *state22,
    const char channelName23[],
    int32 *state23,
    const char channelName24[],
    int32 *state24,
    const char channelName25[],
    int32 *state25,
    const char channelName26[],
    int32 *state26,
    const char channelName27[],
    int32 *state27,
    const char channelName28[],
    int32 *state28,
    const char channelName29[],
    int32 *state29,
    const char channelName30[],
    int32 *state30,
    const char channelName31[],
    int32 *state31,
    const char channelName32[],
    int32 *state32,
    const char channelName33[],
    int32 *state33,
    const char channelName34[],
    int32 *state34,
    const char channelName35[],
    int32 *state35,
    const char channelName36[],
    int32 *state36,
    const char channelName37[],
    int32 *state37,
    const char channelName38[],
    int32 *state38,
    const char channelName39[],
    int32 *state39,
    const char channelName40[],
    int32 *state40,
    const char channelName41[],
    int32 *state41,
    const char channelName42[],
    int32 *state42,
    const char channelName43[],
    int32 *state43,
    const char channelName44[],
    int32 *state44,
    const char channelName45[],
    int32 *state45,
    const char channelName46[],
    int32 *state46,
    const char channelName47[],
    int32 *state47,
    const char channelName48[],
    int32 *state48,
    const char channelName49[],
    int32 *state49,
    const char channelName50[],
    int32 *state50,
    const char channelName51[],
    int32 *state51,
    const char channelName52[],
    int32 *state52,
    const char channelName53[],
    int32 *state53,
    const char channelName54[],
    int32 *state54,
    const char channelName55[],
    int32 *state55,
    const char channelName56[],
    int32 *state56,
    const char channelName57[],
    int32 *state57,
    const char channelName58[],
    int32 *state58,
    const char channelName59[],
    int32 *state59,
    const char channelName60[],
    int32 *state60,
    const char channelName61[],
    int32 *state61,
    const char channelName62[],
    int32 *state62,
    const char channelName63[],
    int32 *state63,
    const char channelName64[],
    int32 *state64,
    const char channelName65[],
    int32 *state65,
    const char channelName66[],
    int32 *state66,
    const char channelName67[],
    int32 *state67,
    const char channelName68[],
    int32 *state68,
    const char channelName69[],
    int32 *state69,
    const char channelName70[],
    int32 *state70,
    const char channelName71[],
    int32 *state71,
    const char channelName72[],
    int32 *state72,
    const char channelName73[],
    int32 *state73,
    const char channelName74[],
    int32 *state74,
    const char channelName75[],
    int32 *state75,
    const char channelName76[],
    int32 *state76,
    const char channelName77[],
    int32 *state77,
    const char channelName78[],
    int32 *state78,
    const char channelName79[],
    int32 *state79,
    const char channelName80[],
    int32 *state80,
    const char channelName81[],
    int32 *state81,
    const char channelName82[],
    int32 *state82,
    const char channelName83[],
    int32 *state83,
    const char channelName84[],
    int32 *state84,
    const char channelName85[],
    int32 *state85,
    const char channelName86[],
    int32 *state86,
    const char channelName87[],
    int32 *state87,
    const char channelName88[],
    int32 *state88,
    const char channelName89[],
    int32 *state89,
    const char channelName90[],
    int32 *state90,
    const char channelName91[],
    int32 *state91,
    const char channelName92[],
    int32 *state92,
    const char channelName93[],
    int32 *state93,
    const char channelName94[],
    int32 *state94,
    const char channelName95[],
    int32 *state95
) {
    return function_pointers_.GetDigitalPullUpPullDownStates(
        deviceName,
        channelName,
        state,
        channelName0,
        state0,
        channelName1,
        state1,
        channelName2,
        state2,
        channelName3,
        state3,
        channelName4,
        state4,
        channelName5,
        state5,
        channelName6,
        state6,
        channelName7,
        state7,
        channelName8,
        state8,
        channelName9,
        state9,
        channelName10,
        state10,
        channelName11,
        state11,
        channelName12,
        state12,
        channelName13,
        state13,
        channelName14,
        state14,
        channelName15,
        state15,
        channelName16,
        state16,
        channelName17,
        state17,
        channelName18,
        state18,
        channelName19,
        state19,
        channelName20,
        state20,
        channelName21,
        state21,
        channelName22,
        state22,
        channelName23,
        state23,
        channelName24,
        state24,
        channelName25,
        state25,
        channelName26,
        state26,
        channelName27,
        state27,
        channelName28,
        state28,
        channelName29,
        state29,
        channelName30,
        state30,
        channelName31,
        state31,
        channelName32,
        state32,
        channelName33,
        state33,
        channelName34,
        state34,
        channelName35,
        state35,
        channelName36,
        state36,
        channelName37,
        state37,
        channelName38,
        state38,
        channelName39,
        state39,
        channelName40,
        state40,
        channelName41,
        state41,
        channelName42,
        state42,
        channelName43,
        state43,
        channelName44,
        state44,
        channelName45,
        state45,
        channelName46,
        state46,
        channelName47,
        state47,
        channelName48,
        state48,
        channelName49,
        state49,
        channelName50,
        state50,
        channelName51,
        state51,
        channelName52,
        state52,
        channelName53,
        state53,
        channelName54,
        state54,
        channelName55,
        state55,
        channelName56,
        state56,
        channelName57,
        state57,
        channelName58,
        state58,
        channelName59,
        state59,
        channelName60,
        state60,
        channelName61,
        state61,
        channelName62,
        state62,
        channelName63,
        state63,
        channelName64,
        state64,
        channelName65,
        state65,
        channelName66,
        state66,
        channelName67,
        state67,
        channelName68,
        state68,
        channelName69,
        state69,
        channelName70,
        state70,
        channelName71,
        state71,
        channelName72,
        state72,
        channelName73,
        state73,
        channelName74,
        state74,
        channelName75,
        state75,
        channelName76,
        state76,
        channelName77,
        state77,
        channelName78,
        state78,
        channelName79,
        state79,
        channelName80,
        state80,
        channelName81,
        state81,
        channelName82,
        state82,
        channelName83,
        state83,
        channelName84,
        state84,
        channelName85,
        state85,
        channelName86,
        state86,
        channelName87,
        state87,
        channelName88,
        state88,
        channelName89,
        state89,
        channelName90,
        state90,
        channelName91,
        state91,
        channelName92,
        state92,
        channelName93,
        state93,
        channelName94,
        state94,
        channelName95,
        state95
    );
}

int32 ProdAPI::GetDisconnectedCDAQSyncPorts(char portList[], uInt32 portListSize) {
    return function_pointers_.GetDisconnectedCDAQSyncPorts(portList, portListSize);
}

int32 ProdAPI::GetErrorString(int32 errorCode, char errorString[], uInt32 bufferSize) {
    return function_pointers_.GetErrorString(errorCode, errorString, bufferSize);
}

int32 ProdAPI::GetExportedSignalAttributeBool(
    TaskHandle task,
    int32 attribute,
    bool32 *value
) {
    return function_pointers_.GetExportedSignalAttributeBool(task, attribute, value);
}

int32 ProdAPI::GetExportedSignalAttributeDouble(
    TaskHandle task,
    int32 attribute,
    float64 *value
) {
    return function_pointers_.GetExportedSignalAttributeDouble(task, attribute, value);
}

int32 ProdAPI::GetExportedSignalAttributeInt32(
    TaskHandle task,
    int32 attribute,
    int32 *value
) {
    return function_pointers_.GetExportedSignalAttributeInt32(task, attribute, value);
}

int32 ProdAPI::GetExportedSignalAttributeString(
    TaskHandle task,
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_
        .GetExportedSignalAttributeString(task, attribute, value, size);
}

int32 ProdAPI::GetExportedSignalAttributeUInt32(
    TaskHandle task,
    int32 attribute,
    uInt32 *value
) {
    return function_pointers_.GetExportedSignalAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::GetExtCalLastDateAndTime(
    const char deviceName[],
    uInt32 *year,
    uInt32 *month,
    uInt32 *day,
    uInt32 *hour,
    uInt32 *minute
) {
    return function_pointers_
        .GetExtCalLastDateAndTime(deviceName, year, month, day, hour, minute);
}

int32 ProdAPI::GetExtendedErrorInfo(char errorString[], uInt32 bufferSize) {
    return function_pointers_.GetExtendedErrorInfo(errorString, bufferSize);
}

int32 ProdAPI::GetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime *data) {
    return function_pointers_.GetFirstSampClkWhen(task, data);
}

int32 ProdAPI::GetFirstSampTimestampVal(TaskHandle task, CVIAbsoluteTime *data) {
    return function_pointers_.GetFirstSampTimestampVal(task, data);
}

int32 ProdAPI::GetNthTaskChannel(
    TaskHandle task,
    uInt32 index,
    char buffer[],
    int32 bufferSize
) {
    return function_pointers_.GetNthTaskChannel(task, index, buffer, bufferSize);
}

int32 ProdAPI::GetNthTaskDevice(
    TaskHandle task,
    uInt32 index,
    char buffer[],
    int32 bufferSize
) {
    return function_pointers_.GetNthTaskDevice(task, index, buffer, bufferSize);
}

int32 ProdAPI::GetNthTaskReadChannel(
    TaskHandle task,
    uInt32 index,
    char buffer[],
    int32 bufferSize
) {
    return function_pointers_.GetNthTaskReadChannel(task, index, buffer, bufferSize);
}

int32 ProdAPI::GetPersistedChanAttributeBool(
    const char channel[],
    int32 attribute,
    bool32 *value
) {
    return function_pointers_.GetPersistedChanAttributeBool(channel, attribute, value);
}

int32 ProdAPI::GetPersistedChanAttributeString(
    const char channel[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_
        .GetPersistedChanAttributeString(channel, attribute, value, size);
}

int32 ProdAPI::GetPersistedScaleAttributeBool(
    const char scaleName[],
    int32 attribute,
    bool32 *value
) {
    return function_pointers_
        .GetPersistedScaleAttributeBool(scaleName, attribute, value);
}

int32 ProdAPI::GetPersistedScaleAttributeString(
    const char scaleName[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_
        .GetPersistedScaleAttributeString(scaleName, attribute, value, size);
}

int32 ProdAPI::GetPersistedTaskAttributeBool(
    const char taskName[],
    int32 attribute,
    bool32 *value
) {
    return function_pointers_.GetPersistedTaskAttributeBool(taskName, attribute, value);
}

int32 ProdAPI::GetPersistedTaskAttributeString(
    const char taskName[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_
        .GetPersistedTaskAttributeString(taskName, attribute, value, size);
}

int32 ProdAPI::GetPhysicalChanAttributeBool(
    const char physicalChannel[],
    int32 attribute,
    bool32 *value
) {
    return function_pointers_
        .GetPhysicalChanAttributeBool(physicalChannel, attribute, value);
}

int32 ProdAPI::GetPhysicalChanAttributeBytes(
    const char physicalChannel[],
    int32 attribute,
    uInt8 value[],
    uInt32 size
) {
    return function_pointers_
        .GetPhysicalChanAttributeBytes(physicalChannel, attribute, value, size);
}

int32 ProdAPI::GetPhysicalChanAttributeDouble(
    const char physicalChannel[],
    int32 attribute,
    float64 *value
) {
    return function_pointers_
        .GetPhysicalChanAttributeDouble(physicalChannel, attribute, value);
}

int32 ProdAPI::GetPhysicalChanAttributeDoubleArray(
    const char physicalChannel[],
    int32 attribute,
    float64 value[],
    uInt32 size
) {
    return function_pointers_
        .GetPhysicalChanAttributeDoubleArray(physicalChannel, attribute, value, size);
}

int32 ProdAPI::GetPhysicalChanAttributeInt32(
    const char physicalChannel[],
    int32 attribute,
    int32 *value
) {
    return function_pointers_
        .GetPhysicalChanAttributeInt32(physicalChannel, attribute, value);
}

int32 ProdAPI::GetPhysicalChanAttributeInt32Array(
    const char physicalChannel[],
    int32 attribute,
    int32 value[],
    uInt32 size
) {
    return function_pointers_
        .GetPhysicalChanAttributeInt32Array(physicalChannel, attribute, value, size);
}

int32 ProdAPI::GetPhysicalChanAttributeString(
    const char physicalChannel[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_
        .GetPhysicalChanAttributeString(physicalChannel, attribute, value, size);
}

int32 ProdAPI::GetPhysicalChanAttributeUInt32(
    const char physicalChannel[],
    int32 attribute,
    uInt32 *value
) {
    return function_pointers_
        .GetPhysicalChanAttributeUInt32(physicalChannel, attribute, value);
}

int32 ProdAPI::GetPhysicalChanAttributeUInt32Array(
    const char physicalChannel[],
    int32 attribute,
    uInt32 value[],
    uInt32 size
) {
    return function_pointers_
        .GetPhysicalChanAttributeUInt32Array(physicalChannel, attribute, value, size);
}

int32 ProdAPI::GetReadAttributeBool(TaskHandle task, int32 attribute, bool32 *value) {
    return function_pointers_.GetReadAttributeBool(task, attribute, value);
}

int32 ProdAPI::GetReadAttributeDouble(
    TaskHandle task,
    int32 attribute,
    float64 *value
) {
    return function_pointers_.GetReadAttributeDouble(task, attribute, value);
}

int32 ProdAPI::GetReadAttributeInt32(TaskHandle task, int32 attribute, int32 *value) {
    return function_pointers_.GetReadAttributeInt32(task, attribute, value);
}

int32 ProdAPI::GetReadAttributeString(
    TaskHandle task,
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_.GetReadAttributeString(task, attribute, value, size);
}

int32 ProdAPI::GetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) {
    return function_pointers_.GetReadAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::GetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) {
    return function_pointers_.GetReadAttributeUInt64(task, attribute, value);
}

int32 ProdAPI::GetRealTimeAttributeBool(
    TaskHandle task,
    int32 attribute,
    bool32 *value
) {
    return function_pointers_.GetRealTimeAttributeBool(task, attribute, value);
}

int32 ProdAPI::GetRealTimeAttributeInt32(
    TaskHandle task,
    int32 attribute,
    int32 *value
) {
    return function_pointers_.GetRealTimeAttributeInt32(task, attribute, value);
}

int32 ProdAPI::GetRealTimeAttributeUInt32(
    TaskHandle task,
    int32 attribute,
    uInt32 *value
) {
    return function_pointers_.GetRealTimeAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::GetRefTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) {
    return function_pointers_.GetRefTrigTimestampVal(task, data);
}

int32 ProdAPI::GetScaleAttributeDouble(
    const char scaleName[],
    int32 attribute,
    float64 *value
) {
    return function_pointers_.GetScaleAttributeDouble(scaleName, attribute, value);
}

int32 ProdAPI::GetScaleAttributeDoubleArray(
    const char scaleName[],
    int32 attribute,
    float64 value[],
    uInt32 size
) {
    return function_pointers_
        .GetScaleAttributeDoubleArray(scaleName, attribute, value, size);
}

int32 ProdAPI::GetScaleAttributeInt32(
    const char scaleName[],
    int32 attribute,
    int32 *value
) {
    return function_pointers_.GetScaleAttributeInt32(scaleName, attribute, value);
}

int32 ProdAPI::GetScaleAttributeString(
    const char scaleName[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_
        .GetScaleAttributeString(scaleName, attribute, value, size);
}

int32 ProdAPI::GetSelfCalLastDateAndTime(
    const char deviceName[],
    uInt32 *year,
    uInt32 *month,
    uInt32 *day,
    uInt32 *hour,
    uInt32 *minute
) {
    return function_pointers_
        .GetSelfCalLastDateAndTime(deviceName, year, month, day, hour, minute);
}

int32 ProdAPI::GetStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) {
    return function_pointers_.GetStartTrigTimestampVal(task, data);
}

int32 ProdAPI::GetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data) {
    return function_pointers_.GetStartTrigTrigWhen(task, data);
}

int32 ProdAPI::GetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime *data) {
    return function_pointers_.GetSyncPulseTimeWhen(task, data);
}

int32 ProdAPI::GetSystemInfoAttributeString(
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_.GetSystemInfoAttributeString(attribute, value, size);
}

int32 ProdAPI::GetSystemInfoAttributeUInt32(int32 attribute, uInt32 *value) {
    return function_pointers_.GetSystemInfoAttributeUInt32(attribute, value);
}

int32 ProdAPI::GetTaskAttributeBool(TaskHandle task, int32 attribute, bool32 *value) {
    return function_pointers_.GetTaskAttributeBool(task, attribute, value);
}

int32 ProdAPI::GetTaskAttributeString(
    TaskHandle task,
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_.GetTaskAttributeString(task, attribute, value, size);
}

int32 ProdAPI::GetTaskAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) {
    return function_pointers_.GetTaskAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::GetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 *value) {
    return function_pointers_.GetTimingAttributeBool(task, attribute, value);
}

int32 ProdAPI::GetTimingAttributeDouble(
    TaskHandle task,
    int32 attribute,
    float64 *value
) {
    return function_pointers_.GetTimingAttributeDouble(task, attribute, value);
}

int32 ProdAPI::GetTimingAttributeExBool(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    bool32 *value
) {
    return function_pointers_
        .GetTimingAttributeExBool(task, deviceNames, attribute, value);
}

int32 ProdAPI::GetTimingAttributeExDouble(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    float64 *value
) {
    return function_pointers_
        .GetTimingAttributeExDouble(task, deviceNames, attribute, value);
}

int32 ProdAPI::GetTimingAttributeExInt32(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    int32 *value
) {
    return function_pointers_
        .GetTimingAttributeExInt32(task, deviceNames, attribute, value);
}

int32 ProdAPI::GetTimingAttributeExString(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_
        .GetTimingAttributeExString(task, deviceNames, attribute, value, size);
}

int32 ProdAPI::GetTimingAttributeExTimestamp(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    CVIAbsoluteTime *value
) {
    return function_pointers_
        .GetTimingAttributeExTimestamp(task, deviceNames, attribute, value);
}

int32 ProdAPI::GetTimingAttributeExUInt32(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    uInt32 *value
) {
    return function_pointers_
        .GetTimingAttributeExUInt32(task, deviceNames, attribute, value);
}

int32 ProdAPI::GetTimingAttributeExUInt64(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    uInt64 *value
) {
    return function_pointers_
        .GetTimingAttributeExUInt64(task, deviceNames, attribute, value);
}

int32 ProdAPI::GetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 *value) {
    return function_pointers_.GetTimingAttributeInt32(task, attribute, value);
}

int32 ProdAPI::GetTimingAttributeString(
    TaskHandle task,
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_.GetTimingAttributeString(task, attribute, value, size);
}

int32 ProdAPI::GetTimingAttributeTimestamp(
    TaskHandle task,
    int32 attribute,
    CVIAbsoluteTime *value
) {
    return function_pointers_.GetTimingAttributeTimestamp(task, attribute, value);
}

int32 ProdAPI::GetTimingAttributeUInt32(
    TaskHandle task,
    int32 attribute,
    uInt32 *value
) {
    return function_pointers_.GetTimingAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::GetTimingAttributeUInt64(
    TaskHandle task,
    int32 attribute,
    uInt64 *value
) {
    return function_pointers_.GetTimingAttributeUInt64(task, attribute, value);
}

int32 ProdAPI::GetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 *value) {
    return function_pointers_.GetTrigAttributeBool(task, attribute, value);
}

int32 ProdAPI::GetTrigAttributeDouble(
    TaskHandle task,
    int32 attribute,
    float64 *value
) {
    return function_pointers_.GetTrigAttributeDouble(task, attribute, value);
}

int32 ProdAPI::GetTrigAttributeDoubleArray(
    TaskHandle task,
    int32 attribute,
    float64 value[],
    uInt32 size
) {
    return function_pointers_.GetTrigAttributeDoubleArray(task, attribute, value, size);
}

int32 ProdAPI::GetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 *value) {
    return function_pointers_.GetTrigAttributeInt32(task, attribute, value);
}

int32 ProdAPI::GetTrigAttributeInt32Array(
    TaskHandle task,
    int32 attribute,
    int32 value[],
    uInt32 size
) {
    return function_pointers_.GetTrigAttributeInt32Array(task, attribute, value, size);
}

int32 ProdAPI::GetTrigAttributeString(
    TaskHandle task,
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_.GetTrigAttributeString(task, attribute, value, size);
}

int32 ProdAPI::GetTrigAttributeTimestamp(
    TaskHandle task,
    int32 attribute,
    CVIAbsoluteTime *value
) {
    return function_pointers_.GetTrigAttributeTimestamp(task, attribute, value);
}

int32 ProdAPI::GetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) {
    return function_pointers_.GetTrigAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::GetWatchdogAttributeBool(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    bool32 *value
) {
    return function_pointers_.GetWatchdogAttributeBool(task, lines, attribute, value);
}

int32 ProdAPI::GetWatchdogAttributeDouble(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    float64 *value
) {
    return function_pointers_.GetWatchdogAttributeDouble(task, lines, attribute, value);
}

int32 ProdAPI::GetWatchdogAttributeInt32(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    int32 *value
) {
    return function_pointers_.GetWatchdogAttributeInt32(task, lines, attribute, value);
}

int32 ProdAPI::GetWatchdogAttributeString(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_
        .GetWatchdogAttributeString(task, lines, attribute, value, size);
}

int32 ProdAPI::GetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 *value) {
    return function_pointers_.GetWriteAttributeBool(task, attribute, value);
}

int32 ProdAPI::GetWriteAttributeDouble(
    TaskHandle task,
    int32 attribute,
    float64 *value
) {
    return function_pointers_.GetWriteAttributeDouble(task, attribute, value);
}

int32 ProdAPI::GetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 *value) {
    return function_pointers_.GetWriteAttributeInt32(task, attribute, value);
}

int32 ProdAPI::GetWriteAttributeString(
    TaskHandle task,
    int32 attribute,
    char value[],
    uInt32 size
) {
    return function_pointers_.GetWriteAttributeString(task, attribute, value, size);
}

int32 ProdAPI::GetWriteAttributeUInt32(
    TaskHandle task,
    int32 attribute,
    uInt32 *value
) {
    return function_pointers_.GetWriteAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::GetWriteAttributeUInt64(
    TaskHandle task,
    int32 attribute,
    uInt64 *value
) {
    return function_pointers_.GetWriteAttributeUInt64(task, attribute, value);
}

int32 ProdAPI::IsTaskDone(TaskHandle task, bool32 *isTaskDone) {
    return function_pointers_.IsTaskDone(task, isTaskDone);
}

int32 ProdAPI::LoadTask(const char sessionName[], TaskHandle *task) {
    return function_pointers_.LoadTask(sessionName, task);
}

int32 ProdAPI::PerformBridgeOffsetNullingCalEx(
    TaskHandle task,
    const char channel[],
    bool32 skipUnsupportedChannels
) {
    return function_pointers_
        .PerformBridgeOffsetNullingCalEx(task, channel, skipUnsupportedChannels);
}

int32 ProdAPI::PerformBridgeShuntCalEx(
    TaskHandle task,
    const char channel[],
    float64 shuntResistorValue,
    int32 shuntResistorLocation,
    int32 shuntResistorSelect,
    int32 shuntResistorSource,
    float64 bridgeResistance,
    bool32 skipUnsupportedChannels
) {
    return function_pointers_.PerformBridgeShuntCalEx(
        task,
        channel,
        shuntResistorValue,
        shuntResistorLocation,
        shuntResistorSelect,
        shuntResistorSource,
        bridgeResistance,
        skipUnsupportedChannels
    );
}

int32 ProdAPI::PerformStrainShuntCalEx(
    TaskHandle task,
    const char channel[],
    float64 shuntResistorValue,
    int32 shuntResistorLocation,
    int32 shuntResistorSelect,
    int32 shuntResistorSource,
    bool32 skipUnsupportedChannels
) {
    return function_pointers_.PerformStrainShuntCalEx(
        task,
        channel,
        shuntResistorValue,
        shuntResistorLocation,
        shuntResistorSelect,
        shuntResistorSource,
        skipUnsupportedChannels
    );
}

int32 ProdAPI::PerformThrmcplLeadOffsetNullingCal(
    TaskHandle task,
    const char channel[],
    bool32 skipUnsupportedChannels
) {
    return function_pointers_
        .PerformThrmcplLeadOffsetNullingCal(task, channel, skipUnsupportedChannels);
}

int32 ProdAPI::ReadAnalogF64(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    float64 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadAnalogF64(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadAnalogScalarF64(
    TaskHandle task,
    float64 timeout,
    float64 *value,
    bool32 *reserved
) {
    return function_pointers_.ReadAnalogScalarF64(task, timeout, value, reserved);
}

int32 ProdAPI::ReadBinaryI16(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    int16 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadBinaryI16(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadBinaryI32(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    int32 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadBinaryI32(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadBinaryU16(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt16 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadBinaryU16(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadBinaryU32(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt32 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadBinaryU32(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadCounterF64(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    float64 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadCounterF64(
        task,
        numSampsPerChan,
        timeout,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadCounterF64Ex(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    float64 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadCounterF64Ex(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadCounterScalarF64(
    TaskHandle task,
    float64 timeout,
    float64 *value,
    bool32 *reserved
) {
    return function_pointers_.ReadCounterScalarF64(task, timeout, value, reserved);
}

int32 ProdAPI::ReadCounterScalarU32(
    TaskHandle task,
    float64 timeout,
    uInt32 *value,
    bool32 *reserved
) {
    return function_pointers_.ReadCounterScalarU32(task, timeout, value, reserved);
}

int32 ProdAPI::ReadCounterU32(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    uInt32 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadCounterU32(
        task,
        numSampsPerChan,
        timeout,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadCounterU32Ex(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt32 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadCounterU32Ex(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadCtrFreq(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 interleaved,
    float64 readArrayFrequency[],
    float64 readArrayDutyCycle[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadCtrFreq(
        task,
        numSampsPerChan,
        timeout,
        interleaved,
        readArrayFrequency,
        readArrayDutyCycle,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadCtrFreqScalar(
    TaskHandle task,
    float64 timeout,
    float64 *frequency,
    float64 *dutyCycle,
    bool32 *reserved
) {
    return function_pointers_
        .ReadCtrFreqScalar(task, timeout, frequency, dutyCycle, reserved);
}

int32 ProdAPI::ReadCtrTicks(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 interleaved,
    uInt32 readArrayHighTicks[],
    uInt32 readArrayLowTicks[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadCtrTicks(
        task,
        numSampsPerChan,
        timeout,
        interleaved,
        readArrayHighTicks,
        readArrayLowTicks,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadCtrTicksScalar(
    TaskHandle task,
    float64 timeout,
    uInt32 *highTicks,
    uInt32 *lowTicks,
    bool32 *reserved
) {
    return function_pointers_
        .ReadCtrTicksScalar(task, timeout, highTicks, lowTicks, reserved);
}

int32 ProdAPI::ReadCtrTime(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 interleaved,
    float64 readArrayHighTime[],
    float64 readArrayLowTime[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadCtrTime(
        task,
        numSampsPerChan,
        timeout,
        interleaved,
        readArrayHighTime,
        readArrayLowTime,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadCtrTimeScalar(
    TaskHandle task,
    float64 timeout,
    float64 *highTime,
    float64 *lowTime,
    bool32 *reserved
) {
    return function_pointers_
        .ReadCtrTimeScalar(task, timeout, highTime, lowTime, reserved);
}

int32 ProdAPI::ReadDigitalLines(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt8 readArray[],
    uInt32 arraySizeInBytes,
    int32 *sampsPerChanRead,
    int32 *numBytesPerSamp,
    bool32 *reserved
) {
    return function_pointers_.ReadDigitalLines(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInBytes,
        sampsPerChanRead,
        numBytesPerSamp,
        reserved
    );
}

int32 ProdAPI::ReadDigitalScalarU32(
    TaskHandle task,
    float64 timeout,
    uInt32 *value,
    bool32 *reserved
) {
    return function_pointers_.ReadDigitalScalarU32(task, timeout, value, reserved);
}

int32 ProdAPI::ReadDigitalU16(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt16 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadDigitalU16(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadDigitalU32(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt32 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadDigitalU32(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadDigitalU8(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt8 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return function_pointers_.ReadDigitalU8(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    );
}

int32 ProdAPI::ReadPowerBinaryI16(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    int16 readArrayVoltage[],
    int16 readArrayCurrent[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return 0;
}

int32 ProdAPI::ReadPowerF64(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    float64 readArrayVoltage[],
    float64 readArrayCurrent[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return 0;
}

int32 ProdAPI::ReadPowerScalarF64(
    TaskHandle task,
    float64 timeout,
    float64 *voltage,
    float64 *current,
    bool32 *reserved
) {
    return 0;
}

int32 ProdAPI::ReadRaw(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    uInt8 readArray[],
    uInt32 arraySizeInBytes,
    int32 *sampsRead,
    int32 *numBytesPerSamp,
    bool32 *reserved
) {
    return function_pointers_.ReadRaw(
        task,
        numSampsPerChan,
        timeout,
        readArray,
        arraySizeInBytes,
        sampsRead,
        numBytesPerSamp,
        reserved
    );
}

int32 ProdAPI::RegisterDoneEvent(
    TaskHandle task,
    uInt32 options,
    DAQmxDoneEventCallbackPtr callbackFunction,
    void *callbackData
) {
    return function_pointers_
        .RegisterDoneEvent(task, options, callbackFunction, callbackData);
}

int32 ProdAPI::RegisterEveryNSamplesEvent(
    TaskHandle task,
    int32 everyNSamplesEventType,
    uInt32 nSamples,
    uInt32 options,
    DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
    void *callbackData
) {
    return function_pointers_.RegisterEveryNSamplesEvent(
        task,
        everyNSamplesEventType,
        nSamples,
        options,
        callbackFunction,
        callbackData
    );
}

int32 ProdAPI::RegisterSignalEvent(
    TaskHandle task,
    int32 signalID,
    uInt32 options,
    DAQmxSignalEventCallbackPtr callbackFunction,
    void *callbackData
) {
    return function_pointers_
        .RegisterSignalEvent(task, signalID, options, callbackFunction, callbackData);
}

int32 ProdAPI::RemoveCDAQSyncConnection(const char portList[]) {
    return function_pointers_.RemoveCDAQSyncConnection(portList);
}

int32 ProdAPI::ReserveNetworkDevice(
    const char deviceName[],
    bool32 overrideReservation
) {
    return function_pointers_.ReserveNetworkDevice(deviceName, overrideReservation);
}

int32 ProdAPI::ResetBufferAttribute(TaskHandle task, int32 attribute) {
    return function_pointers_.ResetBufferAttribute(task, attribute);
}

int32 ProdAPI::ResetChanAttribute(
    TaskHandle task,
    const char channel[],
    int32 attribute
) {
    return function_pointers_.ResetChanAttribute(task, channel, attribute);
}

int32 ProdAPI::ResetDevice(const char deviceName[]) {
    return function_pointers_.ResetDevice(deviceName);
}

int32 ProdAPI::ResetExportedSignalAttribute(TaskHandle task, int32 attribute) {
    return function_pointers_.ResetExportedSignalAttribute(task, attribute);
}

int32 ProdAPI::ResetReadAttribute(TaskHandle task, int32 attribute) {
    return function_pointers_.ResetReadAttribute(task, attribute);
}

int32 ProdAPI::ResetRealTimeAttribute(TaskHandle task, int32 attribute) {
    return function_pointers_.ResetRealTimeAttribute(task, attribute);
}

int32 ProdAPI::ResetTimingAttribute(TaskHandle task, int32 attribute) {
    return function_pointers_.ResetTimingAttribute(task, attribute);
}

int32 ProdAPI::ResetTimingAttributeEx(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute
) {
    return function_pointers_.ResetTimingAttributeEx(task, deviceNames, attribute);
}

int32 ProdAPI::ResetTrigAttribute(TaskHandle task, int32 attribute) {
    return function_pointers_.ResetTrigAttribute(task, attribute);
}

int32 ProdAPI::ResetWatchdogAttribute(
    TaskHandle task,
    const char lines[],
    int32 attribute
) {
    return function_pointers_.ResetWatchdogAttribute(task, lines, attribute);
}

int32 ProdAPI::ResetWriteAttribute(TaskHandle task, int32 attribute) {
    return function_pointers_.ResetWriteAttribute(task, attribute);
}

int32 ProdAPI::RestoreLastExtCalConst(const char deviceName[]) {
    return function_pointers_.RestoreLastExtCalConst(deviceName);
}

int32 ProdAPI::SaveGlobalChan(
    TaskHandle task,
    const char channelName[],
    const char saveAs[],
    const char author[],
    uInt32 options
) {
    return function_pointers_
        .SaveGlobalChan(task, channelName, saveAs, author, options);
}

int32 ProdAPI::SaveScale(
    const char scaleName[],
    const char saveAs[],
    const char author[],
    uInt32 options
) {
    return function_pointers_.SaveScale(scaleName, saveAs, author, options);
}

int32 ProdAPI::SaveTask(
    TaskHandle task,
    const char saveAs[],
    const char author[],
    uInt32 options
) {
    return function_pointers_.SaveTask(task, saveAs, author, options);
}

int32 ProdAPI::SelfCal(const char deviceName[]) {
    return function_pointers_.SelfCal(deviceName);
}

int32 ProdAPI::SelfTestDevice(const char deviceName[]) {
    return function_pointers_.SelfTestDevice(deviceName);
}

int32 ProdAPI::SetAIChanCalCalDate(
    TaskHandle task,
    const char channelName[],
    uInt32 year,
    uInt32 month,
    uInt32 day,
    uInt32 hour,
    uInt32 minute
) {
    return function_pointers_
        .SetAIChanCalCalDate(task, channelName, year, month, day, hour, minute);
}

int32 ProdAPI::SetAIChanCalExpDate(
    TaskHandle task,
    const char channelName[],
    uInt32 year,
    uInt32 month,
    uInt32 day,
    uInt32 hour,
    uInt32 minute
) {
    return function_pointers_
        .SetAIChanCalExpDate(task, channelName, year, month, day, hour, minute);
}

int32 ProdAPI::SetAnalogPowerUpStates(
    const char deviceName[],
    const char channelNames[],
    float64 state,
    int32 channelType,
    const char channelNames0[],
    float64 state0,
    int32 channelType0,
    const char channelNames1[],
    float64 state1,
    int32 channelType1,
    const char channelNames2[],
    float64 state2,
    int32 channelType2,
    const char channelNames3[],
    float64 state3,
    int32 channelType3,
    const char channelNames4[],
    float64 state4,
    int32 channelType4,
    const char channelNames5[],
    float64 state5,
    int32 channelType5,
    const char channelNames6[],
    float64 state6,
    int32 channelType6,
    const char channelNames7[],
    float64 state7,
    int32 channelType7,
    const char channelNames8[],
    float64 state8,
    int32 channelType8,
    const char channelNames9[],
    float64 state9,
    int32 channelType9,
    const char channelNames10[],
    float64 state10,
    int32 channelType10,
    const char channelNames11[],
    float64 state11,
    int32 channelType11,
    const char channelNames12[],
    float64 state12,
    int32 channelType12,
    const char channelNames13[],
    float64 state13,
    int32 channelType13,
    const char channelNames14[],
    float64 state14,
    int32 channelType14,
    const char channelNames15[],
    float64 state15,
    int32 channelType15,
    const char channelNames16[],
    float64 state16,
    int32 channelType16,
    const char channelNames17[],
    float64 state17,
    int32 channelType17,
    const char channelNames18[],
    float64 state18,
    int32 channelType18,
    const char channelNames19[],
    float64 state19,
    int32 channelType19,
    const char channelNames20[],
    float64 state20,
    int32 channelType20,
    const char channelNames21[],
    float64 state21,
    int32 channelType21,
    const char channelNames22[],
    float64 state22,
    int32 channelType22,
    const char channelNames23[],
    float64 state23,
    int32 channelType23,
    const char channelNames24[],
    float64 state24,
    int32 channelType24,
    const char channelNames25[],
    float64 state25,
    int32 channelType25,
    const char channelNames26[],
    float64 state26,
    int32 channelType26,
    const char channelNames27[],
    float64 state27,
    int32 channelType27,
    const char channelNames28[],
    float64 state28,
    int32 channelType28,
    const char channelNames29[],
    float64 state29,
    int32 channelType29,
    const char channelNames30[],
    float64 state30,
    int32 channelType30,
    const char channelNames31[],
    float64 state31,
    int32 channelType31,
    const char channelNames32[],
    float64 state32,
    int32 channelType32,
    const char channelNames33[],
    float64 state33,
    int32 channelType33,
    const char channelNames34[],
    float64 state34,
    int32 channelType34,
    const char channelNames35[],
    float64 state35,
    int32 channelType35,
    const char channelNames36[],
    float64 state36,
    int32 channelType36,
    const char channelNames37[],
    float64 state37,
    int32 channelType37,
    const char channelNames38[],
    float64 state38,
    int32 channelType38,
    const char channelNames39[],
    float64 state39,
    int32 channelType39,
    const char channelNames40[],
    float64 state40,
    int32 channelType40,
    const char channelNames41[],
    float64 state41,
    int32 channelType41,
    const char channelNames42[],
    float64 state42,
    int32 channelType42,
    const char channelNames43[],
    float64 state43,
    int32 channelType43,
    const char channelNames44[],
    float64 state44,
    int32 channelType44,
    const char channelNames45[],
    float64 state45,
    int32 channelType45,
    const char channelNames46[],
    float64 state46,
    int32 channelType46,
    const char channelNames47[],
    float64 state47,
    int32 channelType47,
    const char channelNames48[],
    float64 state48,
    int32 channelType48,
    const char channelNames49[],
    float64 state49,
    int32 channelType49,
    const char channelNames50[],
    float64 state50,
    int32 channelType50,
    const char channelNames51[],
    float64 state51,
    int32 channelType51,
    const char channelNames52[],
    float64 state52,
    int32 channelType52,
    const char channelNames53[],
    float64 state53,
    int32 channelType53,
    const char channelNames54[],
    float64 state54,
    int32 channelType54,
    const char channelNames55[],
    float64 state55,
    int32 channelType55,
    const char channelNames56[],
    float64 state56,
    int32 channelType56,
    const char channelNames57[],
    float64 state57,
    int32 channelType57,
    const char channelNames58[],
    float64 state58,
    int32 channelType58,
    const char channelNames59[],
    float64 state59,
    int32 channelType59,
    const char channelNames60[],
    float64 state60,
    int32 channelType60,
    const char channelNames61[],
    float64 state61,
    int32 channelType61,
    const char channelNames62[],
    float64 state62,
    int32 channelType62,
    const char channelNames63[],
    float64 state63,
    int32 channelType63,
    const char channelNames64[],
    float64 state64,
    int32 channelType64,
    const char channelNames65[],
    float64 state65,
    int32 channelType65,
    const char channelNames66[],
    float64 state66,
    int32 channelType66,
    const char channelNames67[],
    float64 state67,
    int32 channelType67,
    const char channelNames68[],
    float64 state68,
    int32 channelType68,
    const char channelNames69[],
    float64 state69,
    int32 channelType69,
    const char channelNames70[],
    float64 state70,
    int32 channelType70,
    const char channelNames71[],
    float64 state71,
    int32 channelType71,
    const char channelNames72[],
    float64 state72,
    int32 channelType72,
    const char channelNames73[],
    float64 state73,
    int32 channelType73,
    const char channelNames74[],
    float64 state74,
    int32 channelType74,
    const char channelNames75[],
    float64 state75,
    int32 channelType75,
    const char channelNames76[],
    float64 state76,
    int32 channelType76,
    const char channelNames77[],
    float64 state77,
    int32 channelType77,
    const char channelNames78[],
    float64 state78,
    int32 channelType78,
    const char channelNames79[],
    float64 state79,
    int32 channelType79,
    const char channelNames80[],
    float64 state80,
    int32 channelType80,
    const char channelNames81[],
    float64 state81,
    int32 channelType81,
    const char channelNames82[],
    float64 state82,
    int32 channelType82,
    const char channelNames83[],
    float64 state83,
    int32 channelType83,
    const char channelNames84[],
    float64 state84,
    int32 channelType84,
    const char channelNames85[],
    float64 state85,
    int32 channelType85,
    const char channelNames86[],
    float64 state86,
    int32 channelType86,
    const char channelNames87[],
    float64 state87,
    int32 channelType87,
    const char channelNames88[],
    float64 state88,
    int32 channelType88,
    const char channelNames89[],
    float64 state89,
    int32 channelType89,
    const char channelNames90[],
    float64 state90,
    int32 channelType90,
    const char channelNames91[],
    float64 state91,
    int32 channelType91,
    const char channelNames92[],
    float64 state92,
    int32 channelType92,
    const char channelNames93[],
    float64 state93,
    int32 channelType93,
    const char channelNames94[],
    float64 state94,
    int32 channelType94,
    const char channelNames95[],
    float64 state95,
    int32 channelType95
) {
    return function_pointers_.SetAnalogPowerUpStates(
        deviceName,
        channelNames,
        state,
        channelType,
        channelNames0,
        state0,
        channelType0,
        channelNames1,
        state1,
        channelType1,
        channelNames2,
        state2,
        channelType2,
        channelNames3,
        state3,
        channelType3,
        channelNames4,
        state4,
        channelType4,
        channelNames5,
        state5,
        channelType5,
        channelNames6,
        state6,
        channelType6,
        channelNames7,
        state7,
        channelType7,
        channelNames8,
        state8,
        channelType8,
        channelNames9,
        state9,
        channelType9,
        channelNames10,
        state10,
        channelType10,
        channelNames11,
        state11,
        channelType11,
        channelNames12,
        state12,
        channelType12,
        channelNames13,
        state13,
        channelType13,
        channelNames14,
        state14,
        channelType14,
        channelNames15,
        state15,
        channelType15,
        channelNames16,
        state16,
        channelType16,
        channelNames17,
        state17,
        channelType17,
        channelNames18,
        state18,
        channelType18,
        channelNames19,
        state19,
        channelType19,
        channelNames20,
        state20,
        channelType20,
        channelNames21,
        state21,
        channelType21,
        channelNames22,
        state22,
        channelType22,
        channelNames23,
        state23,
        channelType23,
        channelNames24,
        state24,
        channelType24,
        channelNames25,
        state25,
        channelType25,
        channelNames26,
        state26,
        channelType26,
        channelNames27,
        state27,
        channelType27,
        channelNames28,
        state28,
        channelType28,
        channelNames29,
        state29,
        channelType29,
        channelNames30,
        state30,
        channelType30,
        channelNames31,
        state31,
        channelType31,
        channelNames32,
        state32,
        channelType32,
        channelNames33,
        state33,
        channelType33,
        channelNames34,
        state34,
        channelType34,
        channelNames35,
        state35,
        channelType35,
        channelNames36,
        state36,
        channelType36,
        channelNames37,
        state37,
        channelType37,
        channelNames38,
        state38,
        channelType38,
        channelNames39,
        state39,
        channelType39,
        channelNames40,
        state40,
        channelType40,
        channelNames41,
        state41,
        channelType41,
        channelNames42,
        state42,
        channelType42,
        channelNames43,
        state43,
        channelType43,
        channelNames44,
        state44,
        channelType44,
        channelNames45,
        state45,
        channelType45,
        channelNames46,
        state46,
        channelType46,
        channelNames47,
        state47,
        channelType47,
        channelNames48,
        state48,
        channelType48,
        channelNames49,
        state49,
        channelType49,
        channelNames50,
        state50,
        channelType50,
        channelNames51,
        state51,
        channelType51,
        channelNames52,
        state52,
        channelType52,
        channelNames53,
        state53,
        channelType53,
        channelNames54,
        state54,
        channelType54,
        channelNames55,
        state55,
        channelType55,
        channelNames56,
        state56,
        channelType56,
        channelNames57,
        state57,
        channelType57,
        channelNames58,
        state58,
        channelType58,
        channelNames59,
        state59,
        channelType59,
        channelNames60,
        state60,
        channelType60,
        channelNames61,
        state61,
        channelType61,
        channelNames62,
        state62,
        channelType62,
        channelNames63,
        state63,
        channelType63,
        channelNames64,
        state64,
        channelType64,
        channelNames65,
        state65,
        channelType65,
        channelNames66,
        state66,
        channelType66,
        channelNames67,
        state67,
        channelType67,
        channelNames68,
        state68,
        channelType68,
        channelNames69,
        state69,
        channelType69,
        channelNames70,
        state70,
        channelType70,
        channelNames71,
        state71,
        channelType71,
        channelNames72,
        state72,
        channelType72,
        channelNames73,
        state73,
        channelType73,
        channelNames74,
        state74,
        channelType74,
        channelNames75,
        state75,
        channelType75,
        channelNames76,
        state76,
        channelType76,
        channelNames77,
        state77,
        channelType77,
        channelNames78,
        state78,
        channelType78,
        channelNames79,
        state79,
        channelType79,
        channelNames80,
        state80,
        channelType80,
        channelNames81,
        state81,
        channelType81,
        channelNames82,
        state82,
        channelType82,
        channelNames83,
        state83,
        channelType83,
        channelNames84,
        state84,
        channelType84,
        channelNames85,
        state85,
        channelType85,
        channelNames86,
        state86,
        channelType86,
        channelNames87,
        state87,
        channelType87,
        channelNames88,
        state88,
        channelType88,
        channelNames89,
        state89,
        channelType89,
        channelNames90,
        state90,
        channelType90,
        channelNames91,
        state91,
        channelType91,
        channelNames92,
        state92,
        channelType92,
        channelNames93,
        state93,
        channelType93,
        channelNames94,
        state94,
        channelType94,
        channelNames95,
        state95,
        channelType95
    );
}

int32 ProdAPI::SetAnalogPowerUpStatesWithOutputType(
    const char channelNames[],
    const float64 stateArray[],
    const int32 channelTypeArray[],
    uInt32 arraySize
) {
    return function_pointers_.SetAnalogPowerUpStatesWithOutputType(
        channelNames,
        stateArray,
        channelTypeArray,
        arraySize
    );
}

int32 ProdAPI::SetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data) {
    return function_pointers_.SetArmStartTrigTrigWhen(task, data);
}

int32 ProdAPI::SetBufferAttributeUInt32(
    TaskHandle task,
    int32 attribute,
    uInt32 value
) {
    return function_pointers_.SetBufferAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::SetCalInfoAttributeBool(
    const char deviceName[],
    int32 attribute,
    bool32 value
) {
    return function_pointers_.SetCalInfoAttributeBool(deviceName, attribute, value);
}

int32 ProdAPI::SetCalInfoAttributeDouble(
    const char deviceName[],
    int32 attribute,
    float64 value
) {
    return function_pointers_.SetCalInfoAttributeDouble(deviceName, attribute, value);
}

int32 ProdAPI::SetCalInfoAttributeString(
    const char deviceName[],
    int32 attribute,
    const char value[]
) {
    return function_pointers_.SetCalInfoAttributeString(deviceName, attribute, value);
}

int32 ProdAPI::SetCalInfoAttributeUInt32(
    const char deviceName[],
    int32 attribute,
    uInt32 value
) {
    return function_pointers_.SetCalInfoAttributeUInt32(deviceName, attribute, value);
}

int32 ProdAPI::SetChanAttributeBool(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    bool32 value
) {
    return function_pointers_.SetChanAttributeBool(task, channel, attribute, value);
}

int32 ProdAPI::SetChanAttributeDouble(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    float64 value
) {
    return function_pointers_.SetChanAttributeDouble(task, channel, attribute, value);
}

int32 ProdAPI::SetChanAttributeDoubleArray(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    const float64 value[],
    uInt32 size
) {
    return function_pointers_
        .SetChanAttributeDoubleArray(task, channel, attribute, value, size);
}

int32 ProdAPI::SetChanAttributeInt32(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    int32 value
) {
    return function_pointers_.SetChanAttributeInt32(task, channel, attribute, value);
}

int32 ProdAPI::SetChanAttributeString(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    const char value[]
) {
    return function_pointers_.SetChanAttributeString(task, channel, attribute, value);
}

int32 ProdAPI::SetChanAttributeUInt32(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    uInt32 value
) {
    return function_pointers_.SetChanAttributeUInt32(task, channel, attribute, value);
}

int32 ProdAPI::SetDigitalLogicFamilyPowerUpState(
    const char deviceName[],
    int32 logicFamily
) {
    return function_pointers_.SetDigitalLogicFamilyPowerUpState(
        deviceName,
        logicFamily
    );
}

int32 ProdAPI::SetDigitalPowerUpStates(
    const char deviceName[],
    const char channelNames[],
    int32 state,
    const char channelNames0[],
    int32 state0,
    const char channelNames1[],
    int32 state1,
    const char channelNames2[],
    int32 state2,
    const char channelNames3[],
    int32 state3,
    const char channelNames4[],
    int32 state4,
    const char channelNames5[],
    int32 state5,
    const char channelNames6[],
    int32 state6,
    const char channelNames7[],
    int32 state7,
    const char channelNames8[],
    int32 state8,
    const char channelNames9[],
    int32 state9,
    const char channelNames10[],
    int32 state10,
    const char channelNames11[],
    int32 state11,
    const char channelNames12[],
    int32 state12,
    const char channelNames13[],
    int32 state13,
    const char channelNames14[],
    int32 state14,
    const char channelNames15[],
    int32 state15,
    const char channelNames16[],
    int32 state16,
    const char channelNames17[],
    int32 state17,
    const char channelNames18[],
    int32 state18,
    const char channelNames19[],
    int32 state19,
    const char channelNames20[],
    int32 state20,
    const char channelNames21[],
    int32 state21,
    const char channelNames22[],
    int32 state22,
    const char channelNames23[],
    int32 state23,
    const char channelNames24[],
    int32 state24,
    const char channelNames25[],
    int32 state25,
    const char channelNames26[],
    int32 state26,
    const char channelNames27[],
    int32 state27,
    const char channelNames28[],
    int32 state28,
    const char channelNames29[],
    int32 state29,
    const char channelNames30[],
    int32 state30,
    const char channelNames31[],
    int32 state31,
    const char channelNames32[],
    int32 state32,
    const char channelNames33[],
    int32 state33,
    const char channelNames34[],
    int32 state34,
    const char channelNames35[],
    int32 state35,
    const char channelNames36[],
    int32 state36,
    const char channelNames37[],
    int32 state37,
    const char channelNames38[],
    int32 state38,
    const char channelNames39[],
    int32 state39,
    const char channelNames40[],
    int32 state40,
    const char channelNames41[],
    int32 state41,
    const char channelNames42[],
    int32 state42,
    const char channelNames43[],
    int32 state43,
    const char channelNames44[],
    int32 state44,
    const char channelNames45[],
    int32 state45,
    const char channelNames46[],
    int32 state46,
    const char channelNames47[],
    int32 state47,
    const char channelNames48[],
    int32 state48,
    const char channelNames49[],
    int32 state49,
    const char channelNames50[],
    int32 state50,
    const char channelNames51[],
    int32 state51,
    const char channelNames52[],
    int32 state52,
    const char channelNames53[],
    int32 state53,
    const char channelNames54[],
    int32 state54,
    const char channelNames55[],
    int32 state55,
    const char channelNames56[],
    int32 state56,
    const char channelNames57[],
    int32 state57,
    const char channelNames58[],
    int32 state58,
    const char channelNames59[],
    int32 state59,
    const char channelNames60[],
    int32 state60,
    const char channelNames61[],
    int32 state61,
    const char channelNames62[],
    int32 state62,
    const char channelNames63[],
    int32 state63,
    const char channelNames64[],
    int32 state64,
    const char channelNames65[],
    int32 state65,
    const char channelNames66[],
    int32 state66,
    const char channelNames67[],
    int32 state67,
    const char channelNames68[],
    int32 state68,
    const char channelNames69[],
    int32 state69,
    const char channelNames70[],
    int32 state70,
    const char channelNames71[],
    int32 state71,
    const char channelNames72[],
    int32 state72,
    const char channelNames73[],
    int32 state73,
    const char channelNames74[],
    int32 state74,
    const char channelNames75[],
    int32 state75,
    const char channelNames76[],
    int32 state76,
    const char channelNames77[],
    int32 state77,
    const char channelNames78[],
    int32 state78,
    const char channelNames79[],
    int32 state79,
    const char channelNames80[],
    int32 state80,
    const char channelNames81[],
    int32 state81,
    const char channelNames82[],
    int32 state82,
    const char channelNames83[],
    int32 state83,
    const char channelNames84[],
    int32 state84,
    const char channelNames85[],
    int32 state85,
    const char channelNames86[],
    int32 state86,
    const char channelNames87[],
    int32 state87,
    const char channelNames88[],
    int32 state88,
    const char channelNames89[],
    int32 state89,
    const char channelNames90[],
    int32 state90,
    const char channelNames91[],
    int32 state91,
    const char channelNames92[],
    int32 state92,
    const char channelNames93[],
    int32 state93,
    const char channelNames94[],
    int32 state94,
    const char channelNames95[],
    int32 state95
) {
    return function_pointers_.SetDigitalPowerUpStates(
        deviceName,
        channelNames,
        state,
        channelNames0,
        state0,
        channelNames1,
        state1,
        channelNames2,
        state2,
        channelNames3,
        state3,
        channelNames4,
        state4,
        channelNames5,
        state5,
        channelNames6,
        state6,
        channelNames7,
        state7,
        channelNames8,
        state8,
        channelNames9,
        state9,
        channelNames10,
        state10,
        channelNames11,
        state11,
        channelNames12,
        state12,
        channelNames13,
        state13,
        channelNames14,
        state14,
        channelNames15,
        state15,
        channelNames16,
        state16,
        channelNames17,
        state17,
        channelNames18,
        state18,
        channelNames19,
        state19,
        channelNames20,
        state20,
        channelNames21,
        state21,
        channelNames22,
        state22,
        channelNames23,
        state23,
        channelNames24,
        state24,
        channelNames25,
        state25,
        channelNames26,
        state26,
        channelNames27,
        state27,
        channelNames28,
        state28,
        channelNames29,
        state29,
        channelNames30,
        state30,
        channelNames31,
        state31,
        channelNames32,
        state32,
        channelNames33,
        state33,
        channelNames34,
        state34,
        channelNames35,
        state35,
        channelNames36,
        state36,
        channelNames37,
        state37,
        channelNames38,
        state38,
        channelNames39,
        state39,
        channelNames40,
        state40,
        channelNames41,
        state41,
        channelNames42,
        state42,
        channelNames43,
        state43,
        channelNames44,
        state44,
        channelNames45,
        state45,
        channelNames46,
        state46,
        channelNames47,
        state47,
        channelNames48,
        state48,
        channelNames49,
        state49,
        channelNames50,
        state50,
        channelNames51,
        state51,
        channelNames52,
        state52,
        channelNames53,
        state53,
        channelNames54,
        state54,
        channelNames55,
        state55,
        channelNames56,
        state56,
        channelNames57,
        state57,
        channelNames58,
        state58,
        channelNames59,
        state59,
        channelNames60,
        state60,
        channelNames61,
        state61,
        channelNames62,
        state62,
        channelNames63,
        state63,
        channelNames64,
        state64,
        channelNames65,
        state65,
        channelNames66,
        state66,
        channelNames67,
        state67,
        channelNames68,
        state68,
        channelNames69,
        state69,
        channelNames70,
        state70,
        channelNames71,
        state71,
        channelNames72,
        state72,
        channelNames73,
        state73,
        channelNames74,
        state74,
        channelNames75,
        state75,
        channelNames76,
        state76,
        channelNames77,
        state77,
        channelNames78,
        state78,
        channelNames79,
        state79,
        channelNames80,
        state80,
        channelNames81,
        state81,
        channelNames82,
        state82,
        channelNames83,
        state83,
        channelNames84,
        state84,
        channelNames85,
        state85,
        channelNames86,
        state86,
        channelNames87,
        state87,
        channelNames88,
        state88,
        channelNames89,
        state89,
        channelNames90,
        state90,
        channelNames91,
        state91,
        channelNames92,
        state92,
        channelNames93,
        state93,
        channelNames94,
        state94,
        channelNames95,
        state95
    );
}

int32 ProdAPI::SetDigitalPullUpPullDownStates(
    const char deviceName[],
    const char channelNames[],
    int32 state,
    const char channelNames0[],
    int32 state0,
    const char channelNames1[],
    int32 state1,
    const char channelNames2[],
    int32 state2,
    const char channelNames3[],
    int32 state3,
    const char channelNames4[],
    int32 state4,
    const char channelNames5[],
    int32 state5,
    const char channelNames6[],
    int32 state6,
    const char channelNames7[],
    int32 state7,
    const char channelNames8[],
    int32 state8,
    const char channelNames9[],
    int32 state9,
    const char channelNames10[],
    int32 state10,
    const char channelNames11[],
    int32 state11,
    const char channelNames12[],
    int32 state12,
    const char channelNames13[],
    int32 state13,
    const char channelNames14[],
    int32 state14,
    const char channelNames15[],
    int32 state15,
    const char channelNames16[],
    int32 state16,
    const char channelNames17[],
    int32 state17,
    const char channelNames18[],
    int32 state18,
    const char channelNames19[],
    int32 state19,
    const char channelNames20[],
    int32 state20,
    const char channelNames21[],
    int32 state21,
    const char channelNames22[],
    int32 state22,
    const char channelNames23[],
    int32 state23,
    const char channelNames24[],
    int32 state24,
    const char channelNames25[],
    int32 state25,
    const char channelNames26[],
    int32 state26,
    const char channelNames27[],
    int32 state27,
    const char channelNames28[],
    int32 state28,
    const char channelNames29[],
    int32 state29,
    const char channelNames30[],
    int32 state30,
    const char channelNames31[],
    int32 state31,
    const char channelNames32[],
    int32 state32,
    const char channelNames33[],
    int32 state33,
    const char channelNames34[],
    int32 state34,
    const char channelNames35[],
    int32 state35,
    const char channelNames36[],
    int32 state36,
    const char channelNames37[],
    int32 state37,
    const char channelNames38[],
    int32 state38,
    const char channelNames39[],
    int32 state39,
    const char channelNames40[],
    int32 state40,
    const char channelNames41[],
    int32 state41,
    const char channelNames42[],
    int32 state42,
    const char channelNames43[],
    int32 state43,
    const char channelNames44[],
    int32 state44,
    const char channelNames45[],
    int32 state45,
    const char channelNames46[],
    int32 state46,
    const char channelNames47[],
    int32 state47,
    const char channelNames48[],
    int32 state48,
    const char channelNames49[],
    int32 state49,
    const char channelNames50[],
    int32 state50,
    const char channelNames51[],
    int32 state51,
    const char channelNames52[],
    int32 state52,
    const char channelNames53[],
    int32 state53,
    const char channelNames54[],
    int32 state54,
    const char channelNames55[],
    int32 state55,
    const char channelNames56[],
    int32 state56,
    const char channelNames57[],
    int32 state57,
    const char channelNames58[],
    int32 state58,
    const char channelNames59[],
    int32 state59,
    const char channelNames60[],
    int32 state60,
    const char channelNames61[],
    int32 state61,
    const char channelNames62[],
    int32 state62,
    const char channelNames63[],
    int32 state63,
    const char channelNames64[],
    int32 state64,
    const char channelNames65[],
    int32 state65,
    const char channelNames66[],
    int32 state66,
    const char channelNames67[],
    int32 state67,
    const char channelNames68[],
    int32 state68,
    const char channelNames69[],
    int32 state69,
    const char channelNames70[],
    int32 state70,
    const char channelNames71[],
    int32 state71,
    const char channelNames72[],
    int32 state72,
    const char channelNames73[],
    int32 state73,
    const char channelNames74[],
    int32 state74,
    const char channelNames75[],
    int32 state75,
    const char channelNames76[],
    int32 state76,
    const char channelNames77[],
    int32 state77,
    const char channelNames78[],
    int32 state78,
    const char channelNames79[],
    int32 state79,
    const char channelNames80[],
    int32 state80,
    const char channelNames81[],
    int32 state81,
    const char channelNames82[],
    int32 state82,
    const char channelNames83[],
    int32 state83,
    const char channelNames84[],
    int32 state84,
    const char channelNames85[],
    int32 state85,
    const char channelNames86[],
    int32 state86,
    const char channelNames87[],
    int32 state87,
    const char channelNames88[],
    int32 state88,
    const char channelNames89[],
    int32 state89,
    const char channelNames90[],
    int32 state90,
    const char channelNames91[],
    int32 state91,
    const char channelNames92[],
    int32 state92,
    const char channelNames93[],
    int32 state93,
    const char channelNames94[],
    int32 state94,
    const char channelNames95[],
    int32 state95
) {
    return function_pointers_.SetDigitalPullUpPullDownStates(
        deviceName,
        channelNames,
        state,
        channelNames0,
        state0,
        channelNames1,
        state1,
        channelNames2,
        state2,
        channelNames3,
        state3,
        channelNames4,
        state4,
        channelNames5,
        state5,
        channelNames6,
        state6,
        channelNames7,
        state7,
        channelNames8,
        state8,
        channelNames9,
        state9,
        channelNames10,
        state10,
        channelNames11,
        state11,
        channelNames12,
        state12,
        channelNames13,
        state13,
        channelNames14,
        state14,
        channelNames15,
        state15,
        channelNames16,
        state16,
        channelNames17,
        state17,
        channelNames18,
        state18,
        channelNames19,
        state19,
        channelNames20,
        state20,
        channelNames21,
        state21,
        channelNames22,
        state22,
        channelNames23,
        state23,
        channelNames24,
        state24,
        channelNames25,
        state25,
        channelNames26,
        state26,
        channelNames27,
        state27,
        channelNames28,
        state28,
        channelNames29,
        state29,
        channelNames30,
        state30,
        channelNames31,
        state31,
        channelNames32,
        state32,
        channelNames33,
        state33,
        channelNames34,
        state34,
        channelNames35,
        state35,
        channelNames36,
        state36,
        channelNames37,
        state37,
        channelNames38,
        state38,
        channelNames39,
        state39,
        channelNames40,
        state40,
        channelNames41,
        state41,
        channelNames42,
        state42,
        channelNames43,
        state43,
        channelNames44,
        state44,
        channelNames45,
        state45,
        channelNames46,
        state46,
        channelNames47,
        state47,
        channelNames48,
        state48,
        channelNames49,
        state49,
        channelNames50,
        state50,
        channelNames51,
        state51,
        channelNames52,
        state52,
        channelNames53,
        state53,
        channelNames54,
        state54,
        channelNames55,
        state55,
        channelNames56,
        state56,
        channelNames57,
        state57,
        channelNames58,
        state58,
        channelNames59,
        state59,
        channelNames60,
        state60,
        channelNames61,
        state61,
        channelNames62,
        state62,
        channelNames63,
        state63,
        channelNames64,
        state64,
        channelNames65,
        state65,
        channelNames66,
        state66,
        channelNames67,
        state67,
        channelNames68,
        state68,
        channelNames69,
        state69,
        channelNames70,
        state70,
        channelNames71,
        state71,
        channelNames72,
        state72,
        channelNames73,
        state73,
        channelNames74,
        state74,
        channelNames75,
        state75,
        channelNames76,
        state76,
        channelNames77,
        state77,
        channelNames78,
        state78,
        channelNames79,
        state79,
        channelNames80,
        state80,
        channelNames81,
        state81,
        channelNames82,
        state82,
        channelNames83,
        state83,
        channelNames84,
        state84,
        channelNames85,
        state85,
        channelNames86,
        state86,
        channelNames87,
        state87,
        channelNames88,
        state88,
        channelNames89,
        state89,
        channelNames90,
        state90,
        channelNames91,
        state91,
        channelNames92,
        state92,
        channelNames93,
        state93,
        channelNames94,
        state94,
        channelNames95,
        state95
    );
}

int32 ProdAPI::SetExportedSignalAttributeBool(
    TaskHandle task,
    int32 attribute,
    bool32 value
) {
    return function_pointers_.SetExportedSignalAttributeBool(task, attribute, value);
}

int32 ProdAPI::SetExportedSignalAttributeDouble(
    TaskHandle task,
    int32 attribute,
    float64 value
) {
    return function_pointers_.SetExportedSignalAttributeDouble(task, attribute, value);
}

int32 ProdAPI::SetExportedSignalAttributeInt32(
    TaskHandle task,
    int32 attribute,
    int32 value
) {
    return function_pointers_.SetExportedSignalAttributeInt32(task, attribute, value);
}

int32 ProdAPI::SetExportedSignalAttributeString(
    TaskHandle task,
    int32 attribute,
    const char value[]
) {
    return function_pointers_.SetExportedSignalAttributeString(task, attribute, value);
}

int32 ProdAPI::SetExportedSignalAttributeUInt32(
    TaskHandle task,
    int32 attribute,
    uInt32 value
) {
    return function_pointers_.SetExportedSignalAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::SetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime data) {
    return function_pointers_.SetFirstSampClkWhen(task, data);
}

int32 ProdAPI::SetReadAttributeBool(TaskHandle task, int32 attribute, bool32 value) {
    return function_pointers_.SetReadAttributeBool(task, attribute, value);
}

int32 ProdAPI::SetReadAttributeDouble(TaskHandle task, int32 attribute, float64 value) {
    return function_pointers_.SetReadAttributeDouble(task, attribute, value);
}

int32 ProdAPI::SetReadAttributeInt32(TaskHandle task, int32 attribute, int32 value) {
    return function_pointers_.SetReadAttributeInt32(task, attribute, value);
}

int32 ProdAPI::SetReadAttributeString(
    TaskHandle task,
    int32 attribute,
    const char value[]
) {
    return function_pointers_.SetReadAttributeString(task, attribute, value);
}

int32 ProdAPI::SetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) {
    return function_pointers_.SetReadAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::SetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) {
    return function_pointers_.SetReadAttributeUInt64(task, attribute, value);
}

int32 ProdAPI::SetRealTimeAttributeBool(
    TaskHandle task,
    int32 attribute,
    bool32 value
) {
    return function_pointers_.SetRealTimeAttributeBool(task, attribute, value);
}

int32 ProdAPI::SetRealTimeAttributeInt32(
    TaskHandle task,
    int32 attribute,
    int32 value
) {
    return function_pointers_.SetRealTimeAttributeInt32(task, attribute, value);
}

int32 ProdAPI::SetRealTimeAttributeUInt32(
    TaskHandle task,
    int32 attribute,
    uInt32 value
) {
    return function_pointers_.SetRealTimeAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::SetRuntimeEnvironment(
    const char environment[],
    const char environmentVersion[],
    const char reserved1[],
    const char reserved2[]
) {
    return function_pointers_
        .SetRuntimeEnvironment(environment, environmentVersion, reserved1, reserved2);
}

int32 ProdAPI::SetScaleAttributeDouble(
    const char scaleName[],
    int32 attribute,
    float64 value
) {
    return function_pointers_.SetScaleAttributeDouble(scaleName, attribute, value);
}

int32 ProdAPI::SetScaleAttributeDoubleArray(
    const char scaleName[],
    int32 attribute,
    const float64 value[],
    uInt32 size
) {
    return function_pointers_
        .SetScaleAttributeDoubleArray(scaleName, attribute, value, size);
}

int32 ProdAPI::SetScaleAttributeInt32(
    const char scaleName[],
    int32 attribute,
    int32 value
) {
    return function_pointers_.SetScaleAttributeInt32(scaleName, attribute, value);
}

int32 ProdAPI::SetScaleAttributeString(
    const char scaleName[],
    int32 attribute,
    const char value[]
) {
    return function_pointers_.SetScaleAttributeString(scaleName, attribute, value);
}

int32 ProdAPI::SetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data) {
    return function_pointers_.SetStartTrigTrigWhen(task, data);
}

int32 ProdAPI::SetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime data) {
    return function_pointers_.SetSyncPulseTimeWhen(task, data);
}

int32 ProdAPI::SetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 value) {
    return function_pointers_.SetTimingAttributeBool(task, attribute, value);
}

int32 ProdAPI::SetTimingAttributeDouble(
    TaskHandle task,
    int32 attribute,
    float64 value
) {
    return function_pointers_.SetTimingAttributeDouble(task, attribute, value);
}

int32 ProdAPI::SetTimingAttributeExBool(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    bool32 value
) {
    return function_pointers_
        .SetTimingAttributeExBool(task, deviceNames, attribute, value);
}

int32 ProdAPI::SetTimingAttributeExDouble(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    float64 value
) {
    return function_pointers_
        .SetTimingAttributeExDouble(task, deviceNames, attribute, value);
}

int32 ProdAPI::SetTimingAttributeExInt32(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    int32 value
) {
    return function_pointers_
        .SetTimingAttributeExInt32(task, deviceNames, attribute, value);
}

int32 ProdAPI::SetTimingAttributeExString(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    const char value[]
) {
    return function_pointers_
        .SetTimingAttributeExString(task, deviceNames, attribute, value);
}

int32 ProdAPI::SetTimingAttributeExTimestamp(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    CVIAbsoluteTime value
) {
    return function_pointers_
        .SetTimingAttributeExTimestamp(task, deviceNames, attribute, value);
}

int32 ProdAPI::SetTimingAttributeExUInt32(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    uInt32 value
) {
    return function_pointers_
        .SetTimingAttributeExUInt32(task, deviceNames, attribute, value);
}

int32 ProdAPI::SetTimingAttributeExUInt64(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    uInt64 value
) {
    return function_pointers_
        .SetTimingAttributeExUInt64(task, deviceNames, attribute, value);
}

int32 ProdAPI::SetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 value) {
    return function_pointers_.SetTimingAttributeInt32(task, attribute, value);
}

int32 ProdAPI::SetTimingAttributeString(
    TaskHandle task,
    int32 attribute,
    const char value[]
) {
    return function_pointers_.SetTimingAttributeString(task, attribute, value);
}

int32 ProdAPI::SetTimingAttributeTimestamp(
    TaskHandle task,
    int32 attribute,
    CVIAbsoluteTime value
) {
    return function_pointers_.SetTimingAttributeTimestamp(task, attribute, value);
}

int32 ProdAPI::SetTimingAttributeUInt32(
    TaskHandle task,
    int32 attribute,
    uInt32 value
) {
    return function_pointers_.SetTimingAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::SetTimingAttributeUInt64(
    TaskHandle task,
    int32 attribute,
    uInt64 value
) {
    return function_pointers_.SetTimingAttributeUInt64(task, attribute, value);
}

int32 ProdAPI::SetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 value) {
    return function_pointers_.SetTrigAttributeBool(task, attribute, value);
}

int32 ProdAPI::SetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 value) {
    return function_pointers_.SetTrigAttributeDouble(task, attribute, value);
}

int32 ProdAPI::SetTrigAttributeDoubleArray(
    TaskHandle task,
    int32 attribute,
    const float64 value[],
    uInt32 size
) {
    return function_pointers_.SetTrigAttributeDoubleArray(task, attribute, value, size);
}

int32 ProdAPI::SetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 value) {
    return function_pointers_.SetTrigAttributeInt32(task, attribute, value);
}

int32 ProdAPI::SetTrigAttributeInt32Array(
    TaskHandle task,
    int32 attribute,
    const int32 value[],
    uInt32 size
) {
    return function_pointers_.SetTrigAttributeInt32Array(task, attribute, value, size);
}

int32 ProdAPI::SetTrigAttributeString(
    TaskHandle task,
    int32 attribute,
    const char value[]
) {
    return function_pointers_.SetTrigAttributeString(task, attribute, value);
}

int32 ProdAPI::SetTrigAttributeTimestamp(
    TaskHandle task,
    int32 attribute,
    CVIAbsoluteTime value
) {
    return function_pointers_.SetTrigAttributeTimestamp(task, attribute, value);
}

int32 ProdAPI::SetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) {
    return function_pointers_.SetTrigAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::SetWatchdogAttributeBool(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    bool32 value
) {
    return function_pointers_.SetWatchdogAttributeBool(task, lines, attribute, value);
}

int32 ProdAPI::SetWatchdogAttributeDouble(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    float64 value
) {
    return function_pointers_.SetWatchdogAttributeDouble(task, lines, attribute, value);
}

int32 ProdAPI::SetWatchdogAttributeInt32(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    int32 value
) {
    return function_pointers_.SetWatchdogAttributeInt32(task, lines, attribute, value);
}

int32 ProdAPI::SetWatchdogAttributeString(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    const char value[]
) {
    return function_pointers_.SetWatchdogAttributeString(task, lines, attribute, value);
}

int32 ProdAPI::SetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 value) {
    return function_pointers_.SetWriteAttributeBool(task, attribute, value);
}

int32 ProdAPI::SetWriteAttributeDouble(
    TaskHandle task,
    int32 attribute,
    float64 value
) {
    return function_pointers_.SetWriteAttributeDouble(task, attribute, value);
}

int32 ProdAPI::SetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 value) {
    return function_pointers_.SetWriteAttributeInt32(task, attribute, value);
}

int32 ProdAPI::SetWriteAttributeString(
    TaskHandle task,
    int32 attribute,
    const char value[]
) {
    return function_pointers_.SetWriteAttributeString(task, attribute, value);
}

int32 ProdAPI::SetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) {
    return function_pointers_.SetWriteAttributeUInt32(task, attribute, value);
}

int32 ProdAPI::SetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) {
    return function_pointers_.SetWriteAttributeUInt64(task, attribute, value);
}

int32 ProdAPI::StartNewFile(TaskHandle task, const char filePath[]) {
    return function_pointers_.StartNewFile(task, filePath);
}

int32 ProdAPI::StartTask(TaskHandle task) {
    return function_pointers_.StartTask(task);
}

int32 ProdAPI::StopTask(TaskHandle task) {
    return function_pointers_.StopTask(task);
}

int32 ProdAPI::TaskControl(TaskHandle task, int32 action) {
    return function_pointers_.TaskControl(task, action);
}

int32 ProdAPI::TristateOutputTerm(const char outputTerminal[]) {
    return function_pointers_.TristateOutputTerm(outputTerminal);
}

int32 ProdAPI::UnregisterDoneEvent(
    TaskHandle task,
    uInt32 options,
    DAQmxDoneEventCallbackPtr callbackFunction,
    void *callbackData
) {
    return function_pointers_
        .UnregisterDoneEvent(task, options, callbackFunction, callbackData);
}

int32 ProdAPI::UnregisterEveryNSamplesEvent(
    TaskHandle task,
    int32 everyNSamplesEventType,
    uInt32 nSamples,
    uInt32 options,
    DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
    void *callbackData
) {
    return function_pointers_.UnregisterEveryNSamplesEvent(
        task,
        everyNSamplesEventType,
        nSamples,
        options,
        callbackFunction,
        callbackData
    );
}

int32 ProdAPI::UnregisterSignalEvent(
    TaskHandle task,
    int32 signalID,
    uInt32 options,
    DAQmxSignalEventCallbackPtr callbackFunction,
    void *callbackData
) {
    return function_pointers_
        .UnregisterSignalEvent(task, signalID, options, callbackFunction, callbackData);
}

int32 ProdAPI::UnreserveNetworkDevice(const char deviceName[]) {
    return function_pointers_.UnreserveNetworkDevice(deviceName);
}

int32 ProdAPI::WaitForNextSampleClock(
    TaskHandle task,
    float64 timeout,
    bool32 *isLate
) {
    return function_pointers_.WaitForNextSampleClock(task, timeout, isLate);
}

int32 ProdAPI::WaitForValidTimestamp(
    TaskHandle task,
    int32 timestampEvent,
    float64 timeout,
    CVIAbsoluteTime *timestamp
) {
    return function_pointers_
        .WaitForValidTimestamp(task, timestampEvent, timeout, timestamp);
}

int32 ProdAPI::WaitUntilTaskDone(TaskHandle task, float64 timeToWait) {
    return function_pointers_.WaitUntilTaskDone(task, timeToWait);
}

int32 ProdAPI::WriteAnalogF64(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const float64 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteAnalogF64(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteAnalogScalarF64(
    TaskHandle task,
    bool32 autoStart,
    float64 timeout,
    float64 value,
    bool32 *reserved
) {
    return function_pointers_
        .WriteAnalogScalarF64(task, autoStart, timeout, value, reserved);
}

int32 ProdAPI::WriteBinaryI16(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const int16 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteBinaryI16(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteBinaryI32(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const int32 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteBinaryI32(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteBinaryU16(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt16 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteBinaryU16(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteBinaryU32(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt32 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteBinaryU32(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteCtrFreq(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const float64 frequency[],
    const float64 dutyCycle[],
    int32 *numSampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteCtrFreq(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        frequency,
        dutyCycle,
        numSampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteCtrFreqScalar(
    TaskHandle task,
    bool32 autoStart,
    float64 timeout,
    float64 frequency,
    float64 dutyCycle,
    bool32 *reserved
) {
    return function_pointers_
        .WriteCtrFreqScalar(task, autoStart, timeout, frequency, dutyCycle, reserved);
}

int32 ProdAPI::WriteCtrTicks(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt32 highTicks[],
    const uInt32 lowTicks[],
    int32 *numSampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteCtrTicks(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        highTicks,
        lowTicks,
        numSampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteCtrTicksScalar(
    TaskHandle task,
    bool32 autoStart,
    float64 timeout,
    uInt32 highTicks,
    uInt32 lowTicks,
    bool32 *reserved
) {
    return function_pointers_
        .WriteCtrTicksScalar(task, autoStart, timeout, highTicks, lowTicks, reserved);
}

int32 ProdAPI::WriteCtrTime(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const float64 highTime[],
    const float64 lowTime[],
    int32 *numSampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteCtrTime(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        highTime,
        lowTime,
        numSampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteCtrTimeScalar(
    TaskHandle task,
    bool32 autoStart,
    float64 timeout,
    float64 highTime,
    float64 lowTime,
    bool32 *reserved
) {
    return function_pointers_
        .WriteCtrTimeScalar(task, autoStart, timeout, highTime, lowTime, reserved);
}

int32 ProdAPI::WriteDigitalLines(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt8 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteDigitalLines(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteDigitalScalarU32(
    TaskHandle task,
    bool32 autoStart,
    float64 timeout,
    uInt32 value,
    bool32 *reserved
) {
    return function_pointers_
        .WriteDigitalScalarU32(task, autoStart, timeout, value, reserved);
}

int32 ProdAPI::WriteDigitalU16(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt16 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteDigitalU16(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteDigitalU32(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt32 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteDigitalU32(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteDigitalU8(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt8 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteDigitalU8(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteRaw(
    TaskHandle task,
    int32 numSamps,
    bool32 autoStart,
    float64 timeout,
    const uInt8 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return function_pointers_.WriteRaw(
        task,
        numSamps,
        autoStart,
        timeout,
        writeArray,
        sampsPerChanWritten,
        reserved
    );
}

int32 ProdAPI::WriteToTEDSFromArray(
    const char physicalChannel[],
    const uInt8 bitStream[],
    uInt32 arraySize,
    int32 basicTEDSOptions
) {
    return function_pointers_
        .WriteToTEDSFromArray(physicalChannel, bitStream, arraySize, basicTEDSOptions);
}

int32 ProdAPI::WriteToTEDSFromFile(
    const char physicalChannel[],
    const char filePath[],
    int32 basicTEDSOptions
) {
    return function_pointers_
        .WriteToTEDSFromFile(physicalChannel, filePath, basicTEDSOptions);
}

int32 ProdAPI::SetReadRelativeTo(TaskHandle taskHandle, int32 data) {
    return function_pointers_.SetReadRelativeTo(taskHandle, data);
};
int32 ProdAPI::SetReadOffset(TaskHandle taskHandle, int32 data) {
    return function_pointers_.SetReadOffset(taskHandle, data);
};
int32 ProdAPI::SetReadOverWrite(TaskHandle taskHandle, int32 data) {
    return function_pointers_.SetReadOverWrite(taskHandle, data);
};

int32 ProdAPI::GetReadTotalSampPerChanAcquired(TaskHandle taskHandle, uInt64 *data) {
    return function_pointers_.GetReadTotalSampPerChanAcquired(taskHandle, data);
}
}
