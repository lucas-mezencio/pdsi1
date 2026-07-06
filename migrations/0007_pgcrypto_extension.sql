-- pgcrypto provides pgp_sym_encrypt / pgp_sym_decrypt / digest() used by the
-- LGPD column-level encryption layer added in 0008 and 0009.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
