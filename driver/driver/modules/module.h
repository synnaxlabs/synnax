//
// Created by Emiliano Bonilla on 1/5/24.
//

#include "synnax/synnax.h"
#include <memory>


namespace module {
class Module {
private:
    synnax::Module internal;
public:
    void command(const std::string data);

    virtual freighter::Error stop() = 0;
};

class Factory {
public:
    virtual std::pair<std::unique_ptr<Module>, freighter::Error> configure(
            const std::shared_ptr<synnax::Synnax> &client, const synnax::Module &module) = 0;
};
}