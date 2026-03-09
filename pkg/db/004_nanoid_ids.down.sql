-- Migration rollback for NanoID
-- Since we are moving forward and this changes types from INTEGER to TEXT,
-- a true rollback is complex. We rely on the initial migration's DROP TABLE
-- if a full reset is needed.
PRAGMA foreign_keys=OFF;
