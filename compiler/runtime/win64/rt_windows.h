#ifndef AU_RUNTIME_H
#define AU_RUNTIME_H

#include <stdint.h>

void au_rt_setconsoleoutput(void);
void au_rt_sleep(uint32_t ms);
void au_rt_print_str(const char *ptr, int64_t len);
void au_rt_println_str(const char *ptr, int64_t len);
void au_rt_println_int(int64_t value);
void au_rt_print_int(int64_t value);

#endif