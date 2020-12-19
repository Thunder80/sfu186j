


# Design Notes

## minimalist logging philiosophy
// two levels: debug, fatal
// debug goes to stdout: fmt.print....
// fatal goes to stderr: panic, etc.
// thus we don't need to import log
// https://dave.cheney.net/2015/11/05/lets-talk-about-logging