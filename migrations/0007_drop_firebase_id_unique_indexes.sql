-- firebase_id is server-managed: it is linked to the local entity at
-- create time by the Firebase Admin SDK and on login backfill. The
-- partial unique indexes created by migrations 0004 / 0005 were a
-- defence-in-depth measure that backfired in production: when consecutive
-- POST /users requests each stored firebase_id as an empty string (the
-- pre-DisallowUnknownFields code path), the partial index treated ''
-- as a distinct value and the second insert violated the constraint,
-- surfacing as 500 internal server error.
--
-- Firebase Auth guarantees UID uniqueness upstream. The local FindBy
-- FirebaseID queries fall back to a sequential scan, which is acceptable
-- at current row counts. Re-introduce these indexes (or a stricter
-- alternative) in a dedicated PR if cardinality warrants it.

DROP INDEX IF EXISTS idx_users_firebase_id_unique;
DROP INDEX IF EXISTS idx_doctors_firebase_id_unique;
