#ifndef GOPASS_CORE_H
#define GOPASS_CORE_H

#include <stddef.h>
#include <stdint.h>

#define GOPASS_DERIVATION_ROUNDS 100000

typedef struct {
    int length;
    uint64_t counter;
    int lowercase;
    int uppercase;
    int digits;
    int symbols;
    const char *exclude; /* NUL-terminated; may be NULL or empty */
} GopassOptions;

/* Derives the 32-byte PBKDF2-HMAC-SHA256 key (one block, 100000 rounds),
 * the byte-exact counterpart of the Go package's deriveKey. */
void gopass_derive_key(const char *password, size_t password_len,
                       const char *salt, size_t salt_len,
                       uint8_t out[32]);

/* Generates a LessPass-compatible password. Returns 0 on success and writes
 * the NUL-terminated password into out. Returns nonzero on invalid options
 * and writes the reason into err (English, matches the Go error strings). */
int gopass_generate(const char *site, const char *login, const char *master,
                    const GopassOptions *options,
                    char *out, size_t out_size,
                    char *err, size_t err_size);

#endif
