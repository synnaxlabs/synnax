#include <iomanip>
#include <iostream>

#include "arc/cpp/module/module.h"

int main() {
    std::string test = "AGFzbQEAAAA";
    auto decoded = arc::module::decode_base64(test);

    std::cout << "Decoded " << decoded.size() << " bytes:" << std::endl;
    for (size_t i = 0; i < decoded.size() && i < 16; i++) {
        std::cout << std::hex << std::setw(2) << std::setfill('0') << (int) decoded[i]
                  << " ";
    }
    std::cout << std::endl;

    return 0;
}
