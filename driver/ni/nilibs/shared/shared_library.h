//
// Created by Emiliano Bonilla on 1/22/25.
//

#pragma once

#include <string>
#ifdef _WIN32
typedef HMODULE LibraryHandle;
#else
typedef void *LibraryHandle;
#endif

#if defined(__GNUC__)
#include <dlfcn.h>
#endif

class SharedLibrary {
public:
    // not included because we have to templatize the class in order to make this virtual
    // and then can't specify a pointer to SharedLibraryInterface for our Library classes (because we would need a template specifier)
    // virtual void swap(SharedLibraryT& other) = 0;
    SharedLibrary() : handle_(nullptr) {
    }

    explicit SharedLibrary(const char *library_name)
        : library_name_(library_name), handle_(nullptr) {
    }

    SharedLibrary(const SharedLibrary &other)
        : library_name_(other.library_name_), handle_(nullptr) {
        if (other.handle_) {
            load();
        }
    }

    ~SharedLibrary() {
        unload();
    }

    void swap(SharedLibrary &other) noexcept {
        library_name_.swap(other.library_name_);
        std::swap(handle_, other.handle_);
    }

    [[nodiscard]] bool is_loaded() const {
        return handle_ != nullptr;
    }

    [[nodiscard]] LibraryHandle get_handle() const {
        return handle_;
    }

    void load() {
        if (handle_) {
            return;
        }
        if (!library_name_.empty()) {
#ifdef _WIN32
            handle_ = ::LoadLibraryA(library_name_.c_str());
#else
            handle_ = ::dlopen(library_name_.c_str(), RTLD_NOW | RTLD_GLOBAL);
#endif
        }
    }

    void unload() {
        if (handle_) {
#ifdef _WIN32
            ::FreeLibrary(handle_);
#else
            ::dlclose(handle_);
#endif
            handle_ = nullptr;
        }
    }

    const void *get_function_pointer(const char *name) const {
        if (!handle_) {
            return nullptr;
        }
#ifdef _WIN32
        return ::GetProcAddress(handle_, name);
#else
        return ::dlsym(handle_, name);
#endif
    }

    bool function_exists(const char *name) const {
        return get_function_pointer(name) != nullptr;
    }

    void set_library_name(const char *library_name) {
        if (!is_loaded())
            library_name_ = library_name;
    }

    [[nodiscard]] std::string get_library_name() const {
        return library_name_;
    }

private:
    std::string library_name_;
    LibraryHandle handle_;
};
