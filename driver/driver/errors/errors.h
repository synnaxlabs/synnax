//
// Created by Emiliano Bonilla on 1/3/24.
//

#pragma once

#include <string>

const std::string ERROR_PREFIX = "sy.driver.";
const std::string TYPE_CRITICAL_HARDWARE_ERROR = ERROR_PREFIX + "hardware.critical";
const std::string TYPE_TRANSIENT_HARDWARE_ERROR = ERROR_PREFIX + "hardware.temporary";
const std::string TYPE_CONFIGURATION_ERROR = ERROR_PREFIX + "configuration";