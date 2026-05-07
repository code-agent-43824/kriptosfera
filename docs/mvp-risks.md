# MVP technical risks

1. Can CryptoPro extension load as an unpacked extension in the chosen Chromium runtime?
2. Can extension ID remain stable?
3. Can native messaging be registered purely in HKCU without admin rights?
4. Can the native host and crypto libraries work from a user-space deployed payload?
5. Can Rutoken + certificate be seen without system-wide CryptoPro CSP installation?
6. Will antivirus or OS security controls dislike a self-extracting single-file launcher?
7. How large can the final executable become before first-run UX becomes unacceptable?
