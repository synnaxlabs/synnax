//
// Created by Synnax on 3/24/2024.
//

#ifndef DRIVER_NI_SCANNER_H
#define DRIVER_NI_SCANNER_H

#endif //DRIVER_NI_SCANNER_H

#include "nlohmann/json.hpp"

usign json = nlohmann::json;

namespace ni {
    class NiScanner {
    public:
        NiScanner();
        ~NiScanner();
        json getDevices();
    private:

    };
}