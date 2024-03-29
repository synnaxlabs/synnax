#include "driver/driver/opcua/opcua.h"
#include "driver/driver/opcua/scanner.h"

std::pair<std::unique_ptr<task::Task>, bool> opcua::Factory::configureTask(
    const std::shared_ptr<task::Context>& ctx,
    const synnax::Task& task
) {
    if (task.type == "opcuaScanner") {
        auto scanner = std::make_unique<Scanner>(ctx, task);
        std::cout << "opcuaScanner" << std::endl;
        return {std::move(scanner), true};
    } else {
        return {nullptr, false};
    }
}
