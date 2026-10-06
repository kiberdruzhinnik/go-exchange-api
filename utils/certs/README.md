Bundled CA certificates for outbound API HTTPS requests. These public CA
certificates are embedded in the Go binaries; no runtime downloads or host trust
store changes are needed. Standard system roots remain trusted.

Official landing page: https://www.gosuslugi.ru/landing/crt
Downloaded on 2026-10-06 over verified HTTPS:

- Root: https://gu-st.ru/content/lending/russian_trusted_root_ca_pem.crt
  SHA-256: D26D2D0231B7C39F92CC738512BA54103519E4405D68B5BD703E9788CA8ECF31
  Expires: 2032-02-27 21:04:15 UTC
- Subordinate: https://gu-st.ru/content/lending/russian_trusted_sub_ca_pem.crt
  SHA-256: BBBDE2103E790B999EC62BD03CF625A5A2E7C316E10AFE6A490EEDEAD8B3FD9B
  Expires: 2027-03-06 11:25:19 UTC

The subordinate signature was verified against the bundled root with OpenSSL.
Refresh the files from the official source and rebuild before their expiry or
when the issuing authority replaces them. Certificates are trusted by the
application's HTTP clients, not installed system-wide.
