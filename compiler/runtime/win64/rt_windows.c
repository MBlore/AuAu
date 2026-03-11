#include "rt_windows.h"

#include <windows.h>
#include <stdio.h>

void au_rt_setconsoleoutput(void) {
    SetConsoleOutputCP(CP_UTF8);
}

void au_rt_sleep(uint32_t ms) {
    Sleep(ms);
}

void au_rt_print_str(const char *ptr, int64_t len) {
    if (ptr == NULL) {
        return;
    }

    if (len <= 0) {
        return;
    }

    fwrite(ptr, 1, (size_t)len, stdout);
}

void au_rt_println_str(const char* ptr, int64_t len) {
    au_rt_print_str(ptr, len);
    fputc('\n', stdout);
}

void au_rt_println_int(int64_t value) {
    printf("%lld\n", value);
}

void au_rt_print_int(int64_t value) {
    printf("%lld", value);
}
