//
// Created by Emiliano Bonilla on 1/22/25.
//

#pragma once
#include <memory>

#include "nidaqmx/nidaqmx_api.h"

struct NILibs {
    NILibs() : daqmx(std::make_shared<NiDAQmxLibraryInterface>()) {};
    std::shared_ptr<NiDAQmxLibraryInterface> daqmx;
};

